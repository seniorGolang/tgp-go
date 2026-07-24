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

func TestRenderServiceClient_GeneratesWebSocketStream(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Live",
				PkgPath: "example/live",
				Annotations: tags.DocTags{
					model.TagServerWS: "",
				},
				Methods: []*model.Method{
					{
						Name:        "Subscribe",
						Annotations: tags.DocTags{model.TagStream: model.StreamModeServer},
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "symbol", TypeRef: model.TypeRef{TypeID: "string"}},
						},
						Results: []*model.Variable{
							{Name: "ticks", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, "example", "client")

	if err := renderer.RenderStreamHelpers(); err != nil {
		t.Fatalf("RenderStreamHelpers: %v", err)
	}
	if err := renderer.RenderServiceClient(project.Contracts[0]); err != nil {
		t.Fatalf("RenderServiceClient: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-client.go"))
	if err != nil {
		t.Fatalf("read generated client: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{"websocket.DefaultDialer", `$/stream`, "func (cli *ClientLive) Subscribe"} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("generated stream client does not contain %q:\n%s", fragment, source)
		}
	}
	if _, err = os.Stat(filepath.Join(dir, "stream.go")); err != nil {
		t.Fatalf("expected package stream helpers: %v", err)
	}
}

func TestRenderServiceClient_SkipsImplicitDialAndDropsSyncResults(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Live",
				PkgPath: "example/live",
				Annotations: tags.DocTags{
					model.TagServerWS:   "",
					model.TagServerSSE:  "",
					model.TagHttpPrefix: "api/v1",
					model.TagWSPath:     "/ws/live",
				},
				Methods: []*model.Method{
					{
						Name: "Subscribe",
						Annotations: tags.DocTags{
							model.TagStream:     model.StreamModeServer,
							model.TagSSEPath:    "/sse/live/subscribe",
							model.TagHttpHeader: "locale|X-Locale|implicit",
						},
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "locale", TypeRef: model.TypeRef{TypeID: "string"}},
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
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, "example", "client")
	if err := renderer.RenderStreamHelpers(); err != nil {
		t.Fatalf("RenderStreamHelpers: %v", err)
	}
	if err := renderer.RenderServiceClient(project.Contracts[0]); err != nil {
		t.Fatalf("RenderServiceClient: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-client.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, "func (cli *ClientLive) Subscribe(ctx context.Context, symbol string) (ticks <-chan string, err error)") {
		t.Fatalf("expected client signature without locale/count, got:\n%s", source)
	}
	if strings.Contains(source, "locale") {
		t.Fatalf("implicit locale must not appear in client stream method:\n%s", source)
	}
	if strings.Contains(source, "dialHeader.Set(\"X-Locale\"") {
		t.Fatal("implicit header must not be dialed")
	}
}
