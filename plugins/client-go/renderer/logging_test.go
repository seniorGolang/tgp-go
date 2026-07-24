// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderHTTP_LogOnErrorSnapshotsCurlBeforeDo(t *testing.T) {

	project := httpClientTestProject()
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, "example", "client")
	if err := renderer.RenderHTTP(); err != nil {
		t.Fatalf("RenderHTTP: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "http.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	assertCurlSnapshotBeforeDo(t, string(content), "func (cli *Client) doRoundTrip(", "cli.httpClient.Do(")
}

func TestRenderJsonRPC_LogOnErrorSnapshotsCurlBeforeDo(t *testing.T) {

	project := httpClientTestProject()
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, "example", "client")
	if err := renderer.RenderJsonRPCPackage(dir); err != nil {
		t.Fatalf("RenderJsonRPCPackage: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "jsonrpc", "internal.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)
	assertCurlSnapshotBeforeDo(t, source, "func (client *ClientRPC) doCall(", "client.httpClient.Do(")
	assertCurlSnapshotBeforeDo(t, source, "func (client *ClientRPC) doBatchCall(", "client.httpClient.Do(")
}

func assertCurlSnapshotBeforeDo(t *testing.T, source string, funcPrefix string, doCall string) {

	t.Helper()
	body := extractFuncBody(t, source, funcPrefix)
	curlIdx := strings.Index(body, "var curlCmd string")
	toCurlIdx := strings.Index(body, "ToCurl(")
	errorLogIdx := strings.Index(body, "ErrorContext")
	doIdx := strings.Index(body, doCall)
	if curlIdx < 0 {
		t.Fatalf("%s must declare curlCmd snapshot:\n%s", funcPrefix, body)
	}
	if toCurlIdx < 0 {
		t.Fatalf("%s must call ToCurl before Do:\n%s", funcPrefix, body)
	}
	if errorLogIdx < 0 {
		t.Fatalf("%s must log errors via ErrorContext:\n%s", funcPrefix, body)
	}
	if doIdx < 0 {
		t.Fatalf("%s must call Do:\n%s", funcPrefix, body)
	}
	if curlIdx >= toCurlIdx || toCurlIdx >= doIdx || errorLogIdx >= doIdx {
		t.Fatalf("%s must snapshot curl and register error log defer before Do (curl=%d ToCurl=%d ErrorContext=%d Do=%d):\n%s",
			funcPrefix, curlIdx, toCurlIdx, errorLogIdx, doIdx, body)
	}
	if strings.Count(body, "ToCurl(") != 1 {
		t.Fatalf("%s must call ToCurl exactly once (before Do), got %d:\n%s",
			funcPrefix, strings.Count(body, "ToCurl("), body)
	}
	afterDo := body[doIdx:]
	if strings.Contains(afterDo, "ToCurl(") {
		t.Fatalf("%s must not call ToCurl after Do:\n%s", funcPrefix, body)
	}
	if !strings.Contains(body, `slog.String("curl", curlCmd)`) {
		t.Fatalf("%s LogOnError must log snapshotted curlCmd:\n%s", funcPrefix, body)
	}
	if !strings.Contains(body, "logRequests ||") && !strings.Contains(body, "logRequests||") {
		t.Fatalf("%s must snapshot curl when logRequests or logOnError:\n%s", funcPrefix, body)
	}
}

func extractFuncBody(t *testing.T, source string, funcPrefix string) (body string) {

	t.Helper()
	start := strings.Index(source, funcPrefix)
	if start < 0 {
		t.Fatalf("function %q not found in generated source", funcPrefix)
	}
	brace := strings.Index(source[start:], "{")
	if brace < 0 {
		t.Fatalf("opening brace not found for %q", funcPrefix)
	}
	i := start + brace
	depth := 0
	for j := i; j < len(source); j++ {
		switch source[j] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				body = source[i : j+1]
				return
			}
		}
	}
	t.Fatalf("closing brace not found for %q", funcPrefix)
	return
}
