// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestOpenSSE_WritesConnectedComment(t *testing.T) {

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	if err := OpenSSE(writer); err != nil {
		t.Fatalf("OpenSSE: %v", err)
	}
	if got := buf.String(); got != ": connected\n\n" {
		t.Fatalf("OpenSSE output = %q, want %q", got, ": connected\n\n")
	}
}

func TestWriteSSEComment_FlushesComment(t *testing.T) {

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	if err := WriteSSEComment(writer, "heartbeat"); err != nil {
		t.Fatalf("WriteSSEComment: %v", err)
	}
	if got := buf.String(); got != ": heartbeat\n\n" {
		t.Fatalf("WriteSSEComment output = %q", got)
	}
}

func TestWriteSSEError_WritesJSONRPCError(t *testing.T) {

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	id := json.RawMessage(`"req-1"`)
	if err := WriteSSEError(writer, id, &Error{Code: 400, Message: "bad input"}); err != nil {
		t.Fatalf("WriteSSEError: %v", err)
	}
	if !strings.Contains(buf.String(), `"code":400`) || !strings.Contains(buf.String(), `"message":"bad input"`) {
		t.Fatalf("WriteSSEError output = %q", buf.String())
	}
}

func TestPumpSSEServerStreamTyped_SendsChunksAndFinal(t *testing.T) {

	out := make(chan string, 2)
	out <- "a"
	out <- "b"
	close(out)

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	ctx := context.Background()
	id := json.RawMessage(`"stream-1"`)
	if err := PumpSSEServerStreamTyped(ctx, writer, id, out, json.RawMessage(`{"count":2}`), 0); err != nil {
		t.Fatalf("PumpSSEServerStreamTyped: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, `"method":"$/stream"`) {
		t.Fatalf("missing stream chunk in %q", output)
	}
	if !strings.Contains(output, `"result":{"count":2}`) {
		t.Fatalf("missing final result in %q", output)
	}
}

func TestPumpSSEServerStreamTyped_HeartbeatWhileIdle(t *testing.T) {

	out := make(chan int)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	done := make(chan struct{})
	go func() {
		_ = PumpSSEServerStreamTyped(ctx, writer, json.RawMessage(`"id"`), out, nil, 20*time.Millisecond)
		close(done)
	}()

	time.Sleep(60 * time.Millisecond)
	cancel()
	<-done

	if !strings.Contains(buf.String(), ": heartbeat") {
		t.Fatalf("expected heartbeat comment in %q", buf.String())
	}
}

func TestPumpSSEServerStreamTyped_ContextCancel(t *testing.T) {

	out := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	done := make(chan error, 1)
	go func() {
		done <- PumpSSEServerStreamTyped(ctx, writer, json.RawMessage(`"id"`), out, nil, 0)
	}()

	cancel()
	if err := <-done; err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
