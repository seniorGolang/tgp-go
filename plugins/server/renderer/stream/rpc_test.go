// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"context"
	"encoding/json"
	"testing"
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
	raw, _ := json.Marshal(msg)
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
	sess := NewSession(conn, handlers)
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
