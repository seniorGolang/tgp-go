// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal/model"
)

func TestRenderTransportLimits_NormalizeClientID(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportLimits(); err != nil {
		t.Fatalf("RenderTransportLimits: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "limits.go"))
	if err != nil {
		t.Fatalf("read limits.go: %v", err)
	}
	source := string(content)
	for _, want := range []string{
		"func normalizeClientID(raw string) (id string)",
		"func hashClientID(id string) (h uint64)",
		"func (srv *Server) boundClientID(id string) (out string)",
		"func clipString(value string, maxLen int) (out string)",
		"func metricRoutePath(ftx *fiber.Ctx) (path string)",
		"maxClientIDLength",
		`"unknown"`,
		`"invalid"`,
		`"other"`,
		"maphash.String",
		`id == "unknown" || id == "invalid"`,
		"len(srv.clientIDs) >= srv.maxClientIDs",
		"srv.maxClientIDs == 0",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("limits.go must contain %q, got:\n%s", want, source)
		}
	}
	if strings.Contains(source, "strings.Clone") {
		t.Fatalf("limits.go must not Clone under Immutable, got:\n%s", source)
	}
	if !strings.Contains(source, "path = ftx.Route().Path") {
		t.Fatalf("metricRoutePath must use Route().Path directly, got:\n%s", source)
	}
}

func TestRenderTransportServer_ClientIDUsesNormalize(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportLimits(); err != nil {
		t.Fatalf("RenderTransportLimits: %v", err)
	}
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read server.go: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, `clientID := srv.boundClientID(normalizeClientID(ftx.Get("X-Client-Id")))`) {
		t.Fatalf("clientIDMiddleware must bound+normalize X-Client-Id, got:\n%s", source)
	}
	for _, want := range []string{
		"maxClientIDs",
		"clientIDs",
		"clientIDsMu",
		"defaultMaxClientIDs",
		"maxClientIDs:",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("server.go must contain %q, got:\n%s", want, source)
		}
	}
	if strings.Contains(source, `ftx.Path()`) && strings.Contains(source, "RequestsInFlight") {
		if !strings.Contains(source, "metricRoutePath(ftx)") {
			t.Fatalf("inFlight must use metricRoutePath, got:\n%s", source)
		}
	}
	if !strings.Contains(source, "metricRoutePath(ftx)") {
		t.Fatalf("inFlight middleware must use metricRoutePath, got:\n%s", source)
	}
}

func TestRenderTransportOptions_MaxClientIDs(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportOptions(); err != nil {
		t.Fatalf("RenderTransportOptions: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "options.go"))
	if err != nil {
		t.Fatalf("read options.go: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, "func MaxClientIDs(n int) Option") {
		t.Fatalf("MaxClientIDs option must be generated, got:\n%s", source)
	}
	if !strings.Contains(source, "srv.maxClientIDs = n") {
		t.Fatalf("MaxClientIDs must assign maxClientIDs, got:\n%s", source)
	}
	if !strings.Contains(source, "if n < 0") {
		t.Fatalf("MaxClientIDs must ignore negative n (0 disables bound), got:\n%s", source)
	}
	if strings.Contains(source, "if n <= 0") {
		t.Fatalf("MaxClientIDs must accept n == 0, got:\n%s", source)
	}
	if !strings.Contains(source, "func AllowClientLogLevel()") {
		t.Fatalf("AllowClientLogLevel option must be generated, got:\n%s", source)
	}
}

func TestRenderTransportOptions_SetFiberCfgForcesImmutable(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportOptions(); err != nil {
		t.Fatalf("RenderTransportOptions: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "options.go"))
	if err != nil {
		t.Fatalf("read options.go: %v", err)
	}
	source := string(content)
	setFiberStart := strings.Index(source, "func SetFiberCfg(")
	if setFiberStart < 0 {
		t.Fatalf("SetFiberCfg must be generated, got:\n%s", source)
	}
	setFiber := source[setFiberStart:]
	if end := strings.Index(setFiber[len("func SetFiberCfg("):], "\nfunc "); end >= 0 {
		setFiber = setFiber[:len("func SetFiberCfg(")+end]
	}
	for _, want := range []string{
		"srv.config.Immutable = true",
		"srv.config.StreamRequestBody = true",
		"srv.config.DisableStartupMessage = true",
		"srv.config.DisablePreParseMultipartForm = true",
	} {
		if !strings.Contains(setFiber, want) {
			t.Fatalf("SetFiberCfg must force %q, got:\n%s", want, setFiber)
		}
	}
}

func TestRenderTransportOptions_MaxOpenStreams(t *testing.T) {

	project := streamTestProject("Live", model.TagServerWS)
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportOptions(); err != nil {
		t.Fatalf("RenderTransportOptions: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "options.go"))
	if err != nil {
		t.Fatalf("read options.go: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, "func MaxOpenStreams(size int) Option") {
		t.Fatalf("MaxOpenStreams option must be generated for WS, got:\n%s", source)
	}
	if !strings.Contains(source, "func MaxWSConnections(n int) Option") {
		t.Fatalf("MaxWSConnections option must be generated for WS, got:\n%s", source)
	}
	if !strings.Contains(source, "srv.maxWSConnections = n") {
		t.Fatalf("MaxWSConnections must assign including 0, got:\n%s", source)
	}
	if !strings.Contains(source, "if n < 0") {
		t.Fatalf("MaxWSConnections must ignore negative n, got:\n%s", source)
	}
	if !strings.Contains(source, "httpSvc.srv = srv") {
		t.Fatalf("WS contract must wire httpSvc.srv, got:\n%s", source)
	}
	for _, fragment := range []string{
		"func MaxStreamBodySize(size int64) Option",
		"srv.maxStreamBodySize = size",
		"func StreamWriteTimeout(timeout time.Duration) Option",
		"srv.streamWriteTimeout = timeout",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("missing stream option %q, got:\n%s", fragment, source)
		}
	}
}

func TestRenderTransportOptions_MaxSSEConnections(t *testing.T) {

	project := streamTestProject("Live", model.TagServerSSE)
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportOptions(); err != nil {
		t.Fatalf("RenderTransportOptions: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "options.go"))
	if err != nil {
		t.Fatalf("read options.go: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, "func MaxSSEConnections(n int) Option") {
		t.Fatalf("MaxSSEConnections option must be generated for SSE, got:\n%s", source)
	}
	if !strings.Contains(source, "srv.maxSSEConnections = n") {
		t.Fatalf("MaxSSEConnections must assign including 0, got:\n%s", source)
	}
	if !strings.Contains(source, "if n < 0") {
		t.Fatalf("MaxSSEConnections must ignore negative n, got:\n%s", source)
	}
}

func TestRenderTransportServer_WSConnectionDefaults(t *testing.T) {

	project := streamTestProject("Live", model.TagServerWS)
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read server.go: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{
		"defaultMaxWSConnections = 512",
		"maxWSConnections:",
		"defaultMaxWSMessageSize = defaultBodyLimit",
		"maxWSMessageSize:",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("missing %q in:\n%s", fragment, source)
		}
	}
}

func TestRenderTransportServer_SSEConnectionDefaults(t *testing.T) {

	project := streamTestProject("Live", model.TagServerSSE)
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read server.go: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{
		"defaultMaxSSEConnections = 512",
		"maxSSEConnections:",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("missing %q in:\n%s", fragment, source)
		}
	}
}
