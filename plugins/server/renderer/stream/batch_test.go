// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"errors"
	"io"
	"strings"
	"testing"
)

var decodeCount int

type counted struct{}

func (c *counted) UnmarshalJSON(_ []byte) (err error) {

	decodeCount++
	return
}

func TestDecodeEnvelope_Object(t *testing.T) {

	type req struct {
		Method string `json:"method"`
	}
	items, single, err := DecodeEnvelope[req](strings.NewReader(`{"method":"ping"}`), 100)
	if err != nil {
		t.Fatalf("DecodeEnvelope: %v", err)
	}
	if !single || len(items) != 1 || items[0].Method != "ping" {
		t.Fatalf("got single=%v items=%+v", single, items)
	}
}

func TestDecodeEnvelope_BatchOK(t *testing.T) {

	type req struct {
		Method string `json:"method"`
	}
	items, single, err := DecodeEnvelope[req](strings.NewReader(`[{"method":"a"},{"method":"b"}]`), 100)
	if err != nil {
		t.Fatalf("DecodeEnvelope: %v", err)
	}
	if single || len(items) != 2 {
		t.Fatalf("got single=%v len=%d", single, len(items))
	}
}

func TestDecodeEnvelope_BatchExactLimit(t *testing.T) {

	body := `[` + strings.Repeat(`{},`, 99) + `{}]`
	items, single, err := DecodeEnvelope[struct{}](strings.NewReader(body), 100)
	if err != nil {
		t.Fatalf("DecodeEnvelope: %v", err)
	}
	if single || len(items) != 100 {
		t.Fatalf("got single=%v len=%d", single, len(items))
	}
}

func TestDecodeEnvelope_BatchTooLargeAbortsDecode(t *testing.T) {

	decodeCount = 0
	body := `[` + strings.Repeat(`{},`, 999) + `{}]`
	items, single, err := DecodeEnvelope[counted](strings.NewReader(body), 100)
	if !errors.Is(err, ErrBatchTooLarge) {
		t.Fatalf("err=%v want ErrBatchTooLarge", err)
	}
	if items != nil || single {
		t.Fatalf("items=%v single=%v", items, single)
	}
	if decodeCount != 100 {
		t.Fatalf("decodes=%d want 100 (abort before 101st)", decodeCount)
	}
}

func TestDecodeEnvelope_EmptyBatch(t *testing.T) {

	items, _, err := DecodeEnvelope[struct{}](strings.NewReader(`[]`), 100)
	if !errors.Is(err, ErrEmptyBatch) {
		t.Fatalf("err=%v want ErrEmptyBatch", err)
	}
	if items != nil {
		t.Fatalf("items=%v", items)
	}
}

func TestDecodeEnvelope_EmptyBody(t *testing.T) {

	_, _, err := DecodeEnvelope[struct{}](strings.NewReader(``), 100)
	if !errors.Is(err, ErrEmptyBody) {
		t.Fatalf("err=%v want ErrEmptyBody", err)
	}
	_, _, err = DecodeEnvelope[struct{}](strings.NewReader("   \n\t"), 100)
	if !errors.Is(err, ErrEmptyBody) {
		t.Fatalf("whitespace err=%v want ErrEmptyBody", err)
	}
}

func TestDecodeEnvelope_BadEnvelope(t *testing.T) {

	_, _, err := DecodeEnvelope[struct{}](strings.NewReader(`"x"`), 100)
	if !errors.Is(err, ErrBadEnvelope) {
		t.Fatalf("err=%v want ErrBadEnvelope", err)
	}
}

func TestDecodeEnvelope_SingleExceedsMaxBatch(t *testing.T) {

	_, _, err := DecodeEnvelope[struct{}](strings.NewReader(`{}`), 0)
	if !errors.Is(err, ErrBatchTooLarge) {
		t.Fatalf("err=%v want ErrBatchTooLarge", err)
	}
}

func TestDecodeEnvelope_LeadingWhitespace(t *testing.T) {

	type req struct {
		Method string `json:"method"`
	}
	items, single, err := DecodeEnvelope[req](strings.NewReader("\n  {\"method\":\"x\"}"), 1)
	if err != nil {
		t.Fatalf("DecodeEnvelope: %v", err)
	}
	if !single || items[0].Method != "x" {
		t.Fatalf("got single=%v items=%+v", single, items)
	}
}

func TestDecodeEnvelope_InvalidJSON(t *testing.T) {

	_, _, err := DecodeEnvelope[struct{}](strings.NewReader(`{`), 100)
	if err == nil || errors.Is(err, ErrEmptyBody) || errors.Is(err, ErrBadEnvelope) {
		t.Fatalf("err=%v want json syntax error", err)
	}
}

type limitedReader struct {
	r     io.Reader
	limit int
	read  int
}

func (l *limitedReader) Read(p []byte) (n int, err error) {

	if l.read >= l.limit {
		return 0, errors.New("read past abort budget")
	}
	remain := l.limit - l.read
	if len(p) > remain {
		p = p[:remain]
	}
	n, err = l.r.Read(p)
	l.read += n
	return
}

func TestDecodeEnvelope_DoesNotConsumeHugeTail(t *testing.T) {

	// 100 tiny objects then a huge 101st; abort must not require reading the whole pad.
	pad := strings.Repeat("x", 256*1024)
	body := `[` + strings.Repeat(`{},`, 100) + `{"p":"` + pad + `"}]`
	lr := &limitedReader{r: strings.NewReader(body), limit: 64 * 1024}
	decodeCount = 0
	_, _, err := DecodeEnvelope[counted](lr, 100)
	if !errors.Is(err, ErrBatchTooLarge) {
		t.Fatalf("err=%v want ErrBatchTooLarge (read=%d)", err, lr.read)
	}
	if decodeCount != 100 {
		t.Fatalf("decodes=%d want 100", decodeCount)
	}
}
