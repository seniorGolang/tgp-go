// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	Version                = "2.0"
	MethodStream           = "$/stream"
	MethodStreamEnd        = "$/stream.end"
	MethodCancel           = "$/cancel"
	DefaultMaxOpenStreams  = 64
	DefaultIncomingBuffer  = 32
	incomingBufferOverflow = "incoming buffer overflow"
	parseError             = -32700
	invalidRequestError    = -32600
	methodNotFoundError    = -32601
	invalidParamsError     = -32602
	internalError          = -32603
)

// Message — JSON-RPC 2.0 envelope (streaming profile).
type Message struct {
	ID      json.RawMessage `json:"id,omitempty"`
	Version string          `json:"jsonrpc"`
	Method  string          `json:"method,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// Error — JSON-RPC error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ChunkParams — params for $/stream notifications.
type ChunkParams struct {
	ID   json.RawMessage `json:"id"`
	Seq  int64           `json:"seq"`
	Item json.RawMessage `json:"item"`
}

// EndParams — params for $/stream.end.
type EndParams struct {
	ID json.RawMessage `json:"id"`
}

// CancelParams — params for $/cancel.
type CancelParams struct {
	ID json.RawMessage `json:"id"`
}

// Conn — duplex JSON message transport (WebSocket adapter).
type Conn interface {
	ReadJSON(v any) (err error)
	WriteJSON(v any) (err error)
	Close() (err error)
}

type writeDeadliner interface {
	SetWriteDeadline(deadline time.Time) (err error)
}

// Handler обрабатывает открытие stream-RPC.
type Handler func(ctx context.Context, req Message, sess *Session) (result json.RawMessage, err error)

// Session — multiplex-сессия поверх одного соединения.
type Session struct {
	conn           Conn
	handlers       map[string]Handler
	mu             sync.Mutex
	streams        map[string]*openStream
	writeMu        sync.Mutex
	closed         atomic.Bool
	maxOpenStreams int
	incomingBuffer int
	writeTimeout   time.Duration
}

type openStream struct {
	ctx       context.Context
	cancel    context.CancelFunc
	in        chan json.RawMessage
	inMu      sync.Mutex
	inClosed  bool
	finalized atomic.Bool
	seq       atomic.Int64
}

func (st *openStream) claimFinal() (ok bool) {

	return st.finalized.CompareAndSwap(false, true)
}

// NewSession создаёт session с картой method→handler.
// maxOpenStreams ≤ 0 означает DefaultMaxOpenStreams.
// writeTimeout ≤ 0 отключает write deadline.
func NewSession(conn Conn, handlers map[string]Handler, maxOpenStreams int, writeTimeout time.Duration) (sess *Session) {

	if maxOpenStreams <= 0 {
		maxOpenStreams = DefaultMaxOpenStreams
	}
	sess = &Session{
		conn:           conn,
		handlers:       handlers,
		streams:        make(map[string]*openStream),
		maxOpenStreams: maxOpenStreams,
		incomingBuffer: DefaultIncomingBuffer,
		writeTimeout:   writeTimeout,
	}
	return
}

// Serve читает сообщения до закрытия соединения.
func (s *Session) Serve(ctx context.Context) {

	defer s.Close()
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			s.Close()
		case <-stop:
		}
	}()
	for {
		if ctx.Err() != nil {
			return
		}
		var msg Message
		if err := s.conn.ReadJSON(&msg); err != nil {
			return
		}
		s.dispatch(ctx, msg)
	}
}

// Close закрывает все потоки и соединение.
func (s *Session) Close() {

	if !s.closed.CompareAndSwap(false, true) {
		return
	}
	s.mu.Lock()
	for _, st := range s.streams {
		st.cancel()
		s.closeIncoming(st)
	}
	s.streams = make(map[string]*openStream)
	s.mu.Unlock()
	_ = s.conn.Close()
}

func (s *Session) dispatch(ctx context.Context, msg Message) {

	method := msg.Method
	switch method {
	case MethodStream:
		s.handleChunk(msg)
		return
	case MethodStreamEnd:
		s.handleEnd(msg)
		return
	case MethodCancel:
		s.handleCancel(msg)
		return
	}
	if method == "" {
		return
	}
	handler, ok := s.handlers[strings.ToLower(method)]
	if !ok {
		if len(msg.ID) > 0 {
			_ = s.Write(errorMessage(msg.ID, methodNotFoundError, "method not found: "+method))
		}
		return
	}
	if len(msg.ID) == 0 {
		return
	}
	streamID := string(msg.ID)
	s.mu.Lock()
	if _, exists := s.streams[streamID]; exists {
		s.mu.Unlock()
		_ = s.Write(errorMessage(msg.ID, invalidRequestError, "duplicate stream id"))
		return
	}
	if len(s.streams) >= s.maxOpenStreams {
		s.mu.Unlock()
		_ = s.Write(errorMessage(msg.ID, invalidRequestError, "too many open streams"))
		return
	}
	streamCtx, cancel := context.WithCancel(ctx)
	buf := s.incomingBuffer
	if buf <= 0 {
		buf = DefaultIncomingBuffer
	}
	st := &openStream{ctx: streamCtx, cancel: cancel, in: make(chan json.RawMessage, buf)}
	s.streams[streamID] = st
	s.mu.Unlock()

	go func() {
		defer func() {
			cancel()
			s.mu.Lock()
			if s.streams[streamID] == st {
				delete(s.streams, streamID)
			}
			s.mu.Unlock()
		}()
		defer func() {
			if rec := recover(); rec != nil {
				if !st.claimFinal() {
					return
				}
				_ = s.Write(errorMessage(msg.ID, internalError, "internal error"))
			}
		}()
		result, err := handler(streamCtx, msg, s)
		if !st.claimFinal() {
			return
		}
		if err != nil {
			code, message := publicRPCError(err)
			_ = s.Write(errorMessage(msg.ID, code, message))
			return
		}
		_ = s.Write(Message{ID: msg.ID, Version: Version, Result: result})
	}()
}

func (s *Session) handleChunk(msg Message) {

	var params ChunkParams
	if err := json.Unmarshal(msg.Params, &params); err != nil || len(params.ID) == 0 {
		return
	}
	s.mu.Lock()
	st := s.streams[string(params.ID)]
	s.mu.Unlock()
	if st == nil || st.in == nil {
		return
	}
	overflow, ignored := st.enqueue(params.Item)
	if ignored {
		return
	}
	if overflow {
		s.failIncoming(st, params.ID)
	}
}

// enqueue кладёт item во входящий канал. overflow — буфер полон; ignored — поток закрыт/отменён.
func (st *openStream) enqueue(item json.RawMessage) (overflow bool, ignored bool) {

	st.inMu.Lock()
	defer st.inMu.Unlock()
	if st.inClosed || st.ctx.Err() != nil {
		ignored = true
		return
	}
	select {
	case st.in <- item:
	case <-st.ctx.Done():
		ignored = true
	default:
		overflow = true
	}
	return
}

func (s *Session) failIncoming(st *openStream, id json.RawMessage) {

	st.cancel()
	s.closeIncoming(st)
	if !st.claimFinal() {
		return
	}
	_ = s.Write(errorMessage(id, invalidRequestError, incomingBufferOverflow))
}

func (s *Session) handleEnd(msg Message) {

	var params EndParams
	if err := json.Unmarshal(msg.Params, &params); err != nil || len(params.ID) == 0 {
		return
	}
	s.mu.Lock()
	st := s.streams[string(params.ID)]
	s.mu.Unlock()
	if st == nil {
		return
	}
	s.closeIncoming(st)
}

func (s *Session) closeIncoming(st *openStream) {

	if st == nil || st.in == nil {
		return
	}
	st.inMu.Lock()
	defer st.inMu.Unlock()
	if st.inClosed {
		return
	}
	st.inClosed = true
	close(st.in)
}

func (s *Session) handleCancel(msg Message) {

	var params CancelParams
	if err := json.Unmarshal(msg.Params, &params); err != nil || len(params.ID) == 0 {
		return
	}
	s.mu.Lock()
	st := s.streams[string(params.ID)]
	s.mu.Unlock()
	if st != nil {
		st.cancel()
	}
}

// Write сериализует сообщение в соединение.
func (s *Session) Write(msg Message) (err error) {

	if s.closed.Load() {
		return io.ErrClosedPipe
	}
	if msg.Version == "" {
		msg.Version = Version
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.writeTimeout > 0 {
		if deadliner, ok := s.conn.(writeDeadliner); ok {
			if err = deadliner.SetWriteDeadline(time.Now().Add(s.writeTimeout)); err != nil {
				return
			}
		}
	}
	return s.conn.WriteJSON(msg)
}

// publicRPCError возвращает код/сообщение, безопасные для клиента.
func publicRPCError(err error) (code int, message string) {

	if rpcErr, ok := errors.AsType[*Error](err); ok {
		return rpcErr.Code, rpcErr.Message
	}
	code = internalError
	if coder, ok := err.(interface{ Code() int }); ok {
		code = coder.Code()
	}
	if code == internalError {
		return internalError, "internal error"
	}
	message = err.Error()
	return
}

// SendChunk отправляет $/stream notification с item.
func (s *Session) SendChunk(id json.RawMessage, item any) (err error) {

	s.mu.Lock()
	st := s.streams[string(id)]
	s.mu.Unlock()
	var seq int64 = 1
	if st != nil {
		seq = st.seq.Add(1)
	}
	var raw json.RawMessage
	switch typed := item.(type) {
	case json.RawMessage:
		raw = typed
	case []byte:
		raw = json.RawMessage(typed)
	default:
		if raw, err = json.Marshal(item); err != nil {
			return
		}
	}
	var params json.RawMessage
	if params, err = json.Marshal(ChunkParams{ID: id, Seq: seq, Item: raw}); err != nil {
		return
	}
	return s.Write(Message{Version: Version, Method: MethodStream, Params: params})
}

// Incoming возвращает канал входных chunk'ов для stream id.
func (s *Session) Incoming(id json.RawMessage) (items <-chan json.RawMessage, ok bool) {

	s.mu.Lock()
	st := s.streams[string(id)]
	s.mu.Unlock()
	if st == nil || st.in == nil {
		return nil, false
	}
	return st.in, true
}

// PumpOut читает out и шлёт chunks, пока канал не закроется или ctx не отменится.
func PumpOut(ctx context.Context, sess *Session, id json.RawMessage, out <-chan json.RawMessage) (err error) {

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item, open := <-out:
			if !open {
				return nil
			}
			if err = sess.SendChunk(id, item); err != nil {
				return
			}
		}
	}
}

// PumpOutTyped читает типизированный канал и шлёт chunks.
func PumpOutTyped[T any](ctx context.Context, sess *Session, id json.RawMessage, out <-chan T) (err error) {

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item, open := <-out:
			if !open {
				return nil
			}
			if err = sess.SendChunk(id, item); err != nil {
				return
			}
		}
	}
}

// FeedIn типизирует входящие raw chunks в канал.
func FeedIn[T any](ctx context.Context, raw <-chan json.RawMessage, dst chan<- T) (err error) {

	defer close(dst)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item, open := <-raw:
			if !open {
				return nil
			}
			var value T
			if err = json.Unmarshal(item, &value); err != nil {
				return fmt.Errorf("invalid stream item: %w", err)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case dst <- value:
			}
		}
	}
}

func errorMessage(id json.RawMessage, code int, message string) (msg Message) {

	if id == nil {
		id = json.RawMessage("null")
	}
	return Message{
		ID:      id,
		Version: Version,
		Error:   &Error{Code: code, Message: message},
	}
}

// Error реализует error.
func (e *Error) Error() (s string) {

	if e == nil {
		return ""
	}
	return e.Message
}

// EmptyResult — пустой JSON object result.
func EmptyResult() (raw json.RawMessage) {

	return json.RawMessage(`{}`)
}

// MarshalResult маршалит value в result.
func MarshalResult(value any) (raw json.RawMessage, err error) {

	if value == nil {
		return EmptyResult(), nil
	}
	return json.Marshal(value)
}
