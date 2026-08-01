// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func TestRenderWebSocket_UniqueConnAndMixedResults(t *testing.T) {

	project := streamTestProject("Live", model.TagServerWS)
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderWebSocket(); err != nil {
		t.Fatalf("RenderWebSocket: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-websocket.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{
		"type wsLiveConn struct",
		"ticks, response.Count, err = http.svc.Subscribe",
		"stream.MarshalResult(response)",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("missing %q in:\n%s", fragment, source)
		}
	}
	if strings.Contains(source, "type wsConn struct") {
		t.Fatal("generic wsConn must not be generated")
	}
}

func TestRenderSSE_MixedResultsFinalMarshal(t *testing.T) {

	project := streamTestProject("Live", model.TagServerSSE)
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderSSE(); err != nil {
		t.Fatalf("RenderSSE: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-sse.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{
		"conn := ftx.Context().Conn()",
		"conn.SetWriteDeadline(time.Time{})",
		"stream.OpenSSE(writer)",
		"stream.PumpSSEServerStreamTyped[string]",
		"stream.MarshalResult(response)",
		"stream.WriteSSEError(writer, requestBase.ID, err)",
		"stream.DefaultSSEHeartbeat",
		"X-Accel-Buffering",
		"ticks, response.Count, err = http.svc.Subscribe(streamCtx",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("missing %q in:\n%s", fragment, source)
		}
	}
	if idxConn := strings.Index(source, "conn.SetWriteDeadline(time.Time{})"); idxConn < 0 {
		t.Fatal("missing SetWriteDeadline")
	} else if idxOpen := strings.Index(source, "stream.OpenSSE(writer)"); idxOpen < idxConn {
		t.Fatal("SetWriteDeadline must run before OpenSSE")
	}
	if strings.Contains(source, "ticks, err = http.svc.Subscribe") {
		t.Fatal("SSE must not drop non-chan results")
	}
	if strings.Contains(source, `fmt.Fprintf(writer, "data: %s`) {
		t.Fatal("SSE must use stream runtime instead of inline fmt.Fprintf")
	}
}

func TestRenderSSE_HTTPContractWiresServerRef(t *testing.T) {

	project := streamTestProject("Live", model.TagServerSSE)
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderHTTP(); err != nil {
		t.Fatalf("RenderHTTP: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-http.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(content), "srv") || !strings.Contains(string(content), "*Server") {
		t.Fatalf("SSE contract http wrapper must expose srv *Server:\n%s", content)
	}
}

func TestRenderTransportOptions_SSEHeartbeat(t *testing.T) {

	project := streamTestProject("Live", model.TagServerSSE)
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportOptions(); err != nil {
		t.Fatalf("RenderTransportOptions: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "options.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{
		"func SetSSEHeartbeat(interval time.Duration) Option",
		"srv.sseHeartbeat = interval",
		"httpSvc.srv = srv",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("missing %q in:\n%s", fragment, source)
		}
	}
}

func streamTestProject(name string, transportTag string) (project *model.Project) {

	return &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    name,
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					transportTag:        "",
					model.TagHttpPrefix: "api/v1",
					model.TagWSPath:     "/ws/live",
				},
				Methods: []*model.Method{
					{
						Name: "Subscribe",
						Annotations: tags.DocTags{
							model.TagStream:  model.StreamModeServer,
							model.TagSSEPath: "/sse/live/subscribe",
						},
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "symbol", TypeRef: model.TypeRef{TypeID: "string"}},
						},
						Results: []*model.Variable{
							{Name: "ticks", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2}},
							{Name: "count", TypeRef: model.TypeRef{TypeID: "int"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
				},
			},
		},
	}
}
