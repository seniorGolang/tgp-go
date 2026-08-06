// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

type memConn struct {
	in  chan Message
	out chan Message
}

func (c *memConn) ReadJSON(v any) (err error) {

	msg, ok := <-c.in
	if !ok {
		return ioEOF{}
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}
	return json.Unmarshal(raw, v)
}

func (c *memConn) WriteJSON(v any) (err error) {

	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	var msg Message
	if err = json.Unmarshal(raw, &msg); err != nil {
		return
	}
	c.out <- msg
	return
}

func (c *memConn) Close() (err error) {

	close(c.in)
	return
}

type ioEOF struct{}

func (ioEOF) Error() string { return "EOF" }

func TestSessionServerStream(t *testing.T) {

	conn := &memConn{in: make(chan Message, 8), out: make(chan Message, 8)}
	handlers := map[string]Handler{
		"live.subscribe": func(ctx context.Context, req Message, sess *Session) (result json.RawMessage, err error) {
			out := make(chan string, 2)
			out <- "a"
			out <- "b"
			close(out)
			if err = PumpOutTyped(ctx, sess, req.ID, out); err != nil {
				return
			}
			return EmptyResult(), nil
		},
	}
	sess := NewSession(conn, handlers, 0, 0)
	go sess.Serve(context.Background())

	id := json.RawMessage(`1`)
	conn.in <- Message{ID: id, Version: Version, Method: "live.subscribe", Params: json.RawMessage(`{"symbol":"X"}`)}

	gotChunks := 0
	for gotChunks < 2 {
		msg := <-conn.out
		if msg.Method == MethodStream {
			gotChunks++
		}
	}
	final := <-conn.out
	if final.Error != nil {
		t.Fatalf("unexpected error: %+v", final.Error)
	}
	if string(final.Result) != "{}" {
		t.Fatalf("unexpected result: %s", final.Result)
	}
	sess.Close()
}

func TestSessionMaxOpenStreams(t *testing.T) {

	conn := &memConn{in: make(chan Message, 8), out: make(chan Message, 8)}
	handlers := map[string]Handler{
		"live.hold": func(ctx context.Context, _ Message, _ *Session) (result json.RawMessage, err error) {
			<-ctx.Done()
			return EmptyResult(), nil
		},
	}
	sess := NewSession(conn, handlers, 1, 0)
	go sess.Serve(context.Background())

	id1 := json.RawMessage(`1`)
	id2 := json.RawMessage(`2`)
	conn.in <- Message{ID: id1, Version: Version, Method: "live.hold", Params: json.RawMessage(`{}`)}
	conn.in <- Message{ID: id2, Version: Version, Method: "live.hold", Params: json.RawMessage(`{}`)}

	msg := <-conn.out
	if msg.Error == nil {
		t.Fatalf("expected too many open streams error, got: %+v", msg)
	}
	if msg.Error.Message != "too many open streams" {
		t.Fatalf("unexpected error message: %q", msg.Error.Message)
	}
	sess.Close()
}

func TestSessionDuplicateStreamID(t *testing.T) {

	conn := &memConn{in: make(chan Message, 8), out: make(chan Message, 8)}
	handlers := map[string]Handler{
		"live.hold": func(ctx context.Context, _ Message, _ *Session) (result json.RawMessage, err error) {
			<-ctx.Done()
			return EmptyResult(), nil
		},
	}
	sess := NewSession(conn, handlers, 0, 0)
	go sess.Serve(context.Background())

	id := json.RawMessage(`1`)
	conn.in <- Message{ID: id, Version: Version, Method: "live.hold", Params: json.RawMessage(`{}`)}
	conn.in <- Message{ID: id, Version: Version, Method: "live.hold", Params: json.RawMessage(`{}`)}

	dup := <-conn.out
	if dup.Error == nil {
		t.Fatalf("expected duplicate stream id error, got: %+v", dup)
	}
	if dup.Error.Code != invalidRequestError {
		t.Fatalf("unexpected error code: %d", dup.Error.Code)
	}
	if dup.Error.Message != "duplicate stream id" {
		t.Fatalf("unexpected error message: %q", dup.Error.Message)
	}

	cancelParams, err := json.Marshal(CancelParams{ID: id})
	if err != nil {
		t.Fatal(err)
	}
	conn.in <- Message{Version: Version, Method: MethodCancel, Params: cancelParams}

	final := <-conn.out
	if final.Error != nil {
		t.Fatalf("first stream must complete after cancel, got error: %+v", final.Error)
	}
	if string(final.Result) != "{}" {
		t.Fatalf("unexpected result after cancel: %s", final.Result)
	}

	conn.in <- Message{ID: id, Version: Version, Method: "live.hold", Params: json.RawMessage(`{}`)}
	conn.in <- Message{Version: Version, Method: MethodCancel, Params: cancelParams}

	reused := <-conn.out
	if reused.Error != nil {
		t.Fatalf("stream id must be reusable after previous stream closed, got error: %+v", reused.Error)
	}
	if string(reused.Result) != "{}" {
		t.Fatalf("unexpected result for reused stream: %s", reused.Result)
	}
	sess.Close()
}

func TestHandleChunkOverflowFailsStream(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	streamID := `"s1"`
	st := &openStream{ctx: ctx, cancel: cancel, in: make(chan json.RawMessage, 1)}
	st.in <- json.RawMessage(`"first"`)
	conn := &memConn{in: make(chan Message, 4), out: make(chan Message, 4)}
	sess := &Session{conn: conn, streams: map[string]*openStream{streamID: st}}
	params, err := json.Marshal(ChunkParams{ID: json.RawMessage(streamID), Item: json.RawMessage(`"second"`)})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		sess.handleChunk(Message{Params: params})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handleChunk must not block when buffer is full")
	}
	if st.ctx.Err() == nil {
		t.Fatal("overflow must cancel stream")
	}
	select {
	case msg := <-conn.out:
		if msg.Error == nil {
			t.Fatalf("expected overflow error, got: %+v", msg)
		}
		if msg.Error.Code != invalidRequestError {
			t.Fatalf("unexpected error code: %d", msg.Error.Code)
		}
		if msg.Error.Message != incomingBufferOverflow {
			t.Fatalf("unexpected error message: %q", msg.Error.Message)
		}
	case <-time.After(time.Second):
		t.Fatal("expected overflow error write")
	}
	sess.handleChunk(Message{Params: params})
	select {
	case msg := <-conn.out:
		t.Fatalf("second overflow must not write another final: %+v", msg)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHandleChunkIgnoresFullBufferAfterCancel(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	streamID := `"s2"`
	st := &openStream{ctx: ctx, cancel: cancel, in: make(chan json.RawMessage, 1)}
	st.in <- json.RawMessage(`"first"`)
	sess := &Session{streams: map[string]*openStream{streamID: st}}
	params, err := json.Marshal(ChunkParams{ID: json.RawMessage(streamID), Item: json.RawMessage(`"second"`)})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	done := make(chan struct{})
	go func() {
		sess.handleChunk(Message{Params: params})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handleChunk must return immediately when stream is cancelled")
	}
}

func TestHandleChunkAfterEndDoesNotPanic(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	streamID := `"s-end"`
	st := &openStream{ctx: ctx, cancel: cancel, in: make(chan json.RawMessage, 1)}
	sess := &Session{streams: map[string]*openStream{streamID: st}}
	endParams, err := json.Marshal(EndParams{ID: json.RawMessage(streamID)})
	if err != nil {
		t.Fatal(err)
	}
	sess.handleEnd(Message{Params: endParams})
	chunkParams, err := json.Marshal(ChunkParams{ID: json.RawMessage(streamID), Item: json.RawMessage(`"late"`)})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		sess.handleChunk(Message{Params: chunkParams})
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handleChunk after end must return without panic")
	}
	select {
	case _, open := <-st.in:
		if open {
			t.Fatal("incoming must stay closed after end")
		}
	default:
		t.Fatal("incoming must be closed after end")
	}
}

func TestHandleChunkConcurrentCloseDoesNotPanic(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	streamID := `"s-race"`
	st := &openStream{ctx: ctx, cancel: cancel, in: make(chan json.RawMessage, 32)}
	conn := &memConn{in: make(chan Message, 4), out: make(chan Message, 4)}
	sess := &Session{conn: conn, streams: map[string]*openStream{streamID: st}}
	params, err := json.Marshal(ChunkParams{ID: json.RawMessage(streamID), Item: json.RawMessage(`"x"`)})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			sess.handleChunk(Message{Params: params})
		})
	}
	wg.Go(func() {
		sess.closeIncoming(st)
		cancel()
	})
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent close and handleChunk must finish without panic")
	}
}

func TestFeedInInvalidItem(t *testing.T) {

	ctx := context.Background()
	raw := make(chan json.RawMessage, 1)
	dst := make(chan string, 1)
	raw <- json.RawMessage(`{`)
	close(raw)
	if err := FeedIn[string](ctx, raw, dst); err == nil {
		t.Fatal("expected decode error")
	}
	if _, open := <-dst; open {
		t.Fatal("dst must be closed after FeedIn error")
	}
}

func TestServeSurvivesIncomingOverflow(t *testing.T) {

	conn := &memConn{in: make(chan Message, 16), out: make(chan Message, 16)}
	handlers := map[string]Handler{
		"live.ingest": func(ctx context.Context, req Message, sess *Session) (result json.RawMessage, err error) {
			if _, ok := sess.Incoming(req.ID); !ok {
				return nil, &Error{Code: internalError, Message: "no incoming"}
			}
			<-ctx.Done()
			return EmptyResult(), nil
		},
		"live.ping": func(_ context.Context, _ Message, _ *Session) (result json.RawMessage, err error) {
			return EmptyResult(), nil
		},
	}
	sess := NewSession(conn, handlers, 0, 0)
	sess.incomingBuffer = 1
	go sess.Serve(context.Background())

	idA := json.RawMessage(`"a"`)
	idB := json.RawMessage(`"b"`)
	conn.in <- Message{ID: idA, Version: Version, Method: "live.ingest", Params: json.RawMessage(`{}`)}

	chunk1, err := json.Marshal(ChunkParams{ID: idA, Item: json.RawMessage(`"one"`)})
	if err != nil {
		t.Fatal(err)
	}
	chunk2, err := json.Marshal(ChunkParams{ID: idA, Item: json.RawMessage(`"two"`)})
	if err != nil {
		t.Fatal(err)
	}
	conn.in <- Message{Version: Version, Method: MethodStream, Params: chunk1}
	conn.in <- Message{Version: Version, Method: MethodStream, Params: chunk2}

	deadline := time.After(2 * time.Second)
	var overflow Message
	for {
		select {
		case overflow = <-conn.out:
			if overflow.Error != nil && overflow.Error.Message == incomingBufferOverflow {
				if string(overflow.ID) != string(idA) {
					t.Fatalf("overflow must target stream a, got id %s", overflow.ID)
				}
				goto gotOverflow
			}
		case <-deadline:
			t.Fatal("expected incoming buffer overflow error")
		}
	}
gotOverflow:

	conn.in <- Message{ID: idB, Version: Version, Method: "live.ping", Params: json.RawMessage(`{}`)}
	select {
	case ping := <-conn.out:
		if ping.Error != nil {
			t.Fatalf("serve must stay alive after overflow, got: %+v", ping.Error)
		}
		if string(ping.ID) != string(idB) {
			t.Fatalf("expected ping result for b, got id %s", ping.ID)
		}
		if string(ping.Result) != "{}" {
			t.Fatalf("unexpected ping result: %s", ping.Result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve must still accept other RPC after incoming overflow")
	}
	sess.Close()
}
