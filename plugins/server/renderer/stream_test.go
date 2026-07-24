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
		"ticks, response.Count, err = http.svc.Subscribe",
		"stream.MarshalResult(response)",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("missing %q in:\n%s", fragment, source)
		}
	}
	if strings.Contains(source, "ticks, err = http.svc.Subscribe") {
		t.Fatal("SSE must not drop non-chan results")
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
