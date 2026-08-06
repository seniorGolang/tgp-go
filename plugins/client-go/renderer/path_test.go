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

func TestRenderPathHelpers_EmitsJoinEndpointPath(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Http",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					model.TagHTTPPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Get",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
						},
						Results: []*model.Variable{
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod: "GET",
							model.TagHTTPPath:   "/item",
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, "example", "client")
	if err := renderer.RenderPathHelpers(); err != nil {
		t.Fatalf("RenderPathHelpers: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "path.go"))
	if err != nil {
		t.Fatalf("read path.go: %v", err)
	}
	source := string(content)
	for _, want := range []string{
		"func joinEndpointPath(basePath string, routePath string) (joined string)",
		"func splitEndpointPath(raw string) (segments []string)",
		"func endpointPathOverlap(base []string, route []string) (overlap int)",
		"func endpointPathSegmentsEqual(left []string, right []string) (equal bool)",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("path.go missing %q:\n%s", want, source)
		}
	}
}

func TestRenderPathHelpers_SkippedWithoutHTTPOrStreams(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Rpc",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
				},
				Methods: []*model.Method{
					{
						Name: "Ping",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
						},
						Results: []*model.Variable{
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, "example", "client")
	if err := renderer.RenderPathHelpers(); err != nil {
		t.Fatalf("RenderPathHelpers: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "path.go")); !os.IsNotExist(err) {
		t.Fatalf("path.go must not be generated for JSON-RPC-only client, err=%v", err)
	}
}
