// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// WriteSSE пишет одно SSE-событие с JSON-RPC notification payload.
func WriteSSE(w http.ResponseWriter, flusher http.Flusher, msg Message) (err error) {

	if msg.Version == "" {
		msg.Version = Version
	}
	var raw []byte
	if raw, err = json.Marshal(msg); err != nil {
		return
	}
	if _, err = fmt.Fprintf(w, "data: %s\n\n", raw); err != nil {
		return
	}
	flusher.Flush()
	return
}

// ServeSSEServerStream открывает server-stream: шлёт chunks из out, затем final result.
func ServeSSEServerStream(ctx context.Context, w http.ResponseWriter, id json.RawMessage, out <-chan json.RawMessage, final json.RawMessage) (err error) {

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	var seq int64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item, open := <-out:
			if !open {
				if final == nil {
					final = EmptyResult()
				}
				return WriteSSE(w, flusher, Message{ID: id, Version: Version, Result: final})
			}
			seq++
			var params json.RawMessage
			if params, err = json.Marshal(ChunkParams{ID: id, Seq: seq, Item: item}); err != nil {
				return
			}
			if err = WriteSSE(w, flusher, Message{Version: Version, Method: MethodStream, Params: params}); err != nil {
				return
			}
		}
	}
}

// DecodeSSEMessages читает SSE data: lines как JSON-RPC Message.
func DecodeSSEMessages(r io.Reader, emit func(msg Message) error) (err error) {

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 6 || line[:5] != "data:" {
			continue
		}
		payload := line[5:]
		if len(payload) > 0 && payload[0] == ' ' {
			payload = payload[1:]
		}
		var msg Message
		if err = json.Unmarshal([]byte(payload), &msg); err != nil {
			return
		}
		if err = emit(msg); err != nil {
			return
		}
	}
	return scanner.Err()
}
