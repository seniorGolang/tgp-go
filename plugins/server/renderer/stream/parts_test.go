// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package stream

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"
)

func buildBody(t *testing.T, order []string, values map[string]string, contentTypes map[string]string) (raw *bytes.Buffer, boundary string) {

	t.Helper()
	raw = &bytes.Buffer{}
	writer := multipart.NewWriter(raw)
	boundary = writer.Boundary()
	for _, name := range order {
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", `form-data; name="`+name+`"`)
		if ct := contentTypes[name]; ct != "" {
			header.Set("Content-Type", ct)
		}
		part, err := writer.CreatePart(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = io.WriteString(part, values[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return raw, boundary
}

func TestNewParts_SequentialRead(t *testing.T) {

	raw, boundary := buildBody(t, []string{"partA", "partB"}, map[string]string{"partA": "AAA", "partB": "BBB"}, nil)
	reader := multipart.NewReader(raw, boundary)
	parts := NewParts(reader, []string{"partA", "partB"}, nil)

	a, err := io.ReadAll(parts[0])
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != "AAA" || string(b) != "BBB" {
		t.Fatalf("got %q %q", a, b)
	}
}

func TestNewParts_OutOfOrder(t *testing.T) {

	raw, boundary := buildBody(t, []string{"partA", "partB"}, map[string]string{"partA": "AAA", "partB": "BBB"}, nil)
	reader := multipart.NewReader(raw, boundary)
	parts := NewParts(reader, []string{"partA", "partB"}, nil)

	_, err := io.ReadAll(parts[1])
	if err == nil || !strings.Contains(err.Error(), "out of order") {
		t.Fatalf("expected out of order, got %v", err)
	}
}

func TestNewParts_ContentTypeMismatch(t *testing.T) {

	raw, boundary := buildBody(t, []string{"partA"}, map[string]string{"partA": "AAA"}, map[string]string{"partA": "text/plain"})
	reader := multipart.NewReader(raw, boundary)
	parts := NewParts(reader, []string{"partA"}, []string{"application/octet-stream"})

	_, err := io.ReadAll(parts[0])
	if err == nil || !strings.Contains(err.Error(), "invalid content-type") {
		t.Fatalf("expected content-type error, got %v", err)
	}
}

func TestNewParts_MissingPart(t *testing.T) {

	raw, boundary := buildBody(t, []string{"partA"}, map[string]string{"partA": "AAA"}, nil)
	reader := multipart.NewReader(raw, boundary)
	parts := NewParts(reader, []string{"partA", "partB"}, nil)

	_, err := io.ReadAll(parts[0])
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(parts[1])
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found, got %v", err)
	}
}
