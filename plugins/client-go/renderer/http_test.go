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

func TestRenderHTTP_GeneratesSharedHelpersOnce(t *testing.T) {

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
	source := string(content)

	for _, fn := range []string{
		"func (cli *Client) applyHeadersFromCtx(",
		"func (cli *Client) doRoundTrip(",
		"func (cli *Client) recordHTTPMetrics(",
	} {
		if strings.Count(source, fn) != 1 {
			t.Fatalf("expected exactly one %q, got %d in:\n%s", fn, strings.Count(source, fn), source)
		}
	}
	if !strings.Contains(source, "serviceLabel string") {
		t.Fatalf("recordHTTPMetrics must accept serviceLabel parameter:\n%s", source)
	}
}

func TestRenderServiceClient_DoesNotDuplicateHTTPHelpers(t *testing.T) {

	project := httpClientTestProject()
	dir := filepath.Join(t.TempDir(), "client")

	renderer := NewClientRenderer(project, dir, "example", "client")
	if err := renderer.RenderJsonRPCPackage(dir); err != nil {
		t.Fatalf("RenderJsonRPCPackage: %v", err)
	}
	if err := renderer.RenderHTTP(); err != nil {
		t.Fatalf("RenderHTTP: %v", err)
	}

	contract := project.Contracts[0]
	if err := renderer.RenderServiceClient(contract); err != nil {
		t.Fatalf("RenderServiceClient: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "modes-client.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	for _, fn := range []string{
		"applyHeadersFromCtx",
		"doRoundTrip",
		"recordHTTPMetrics",
	} {
		if strings.Contains(source, "func (cli *ClientModes) "+fn) {
			t.Fatalf("must not generate per-contract helper %s in service client:\n%s", fn, source)
		}
	}
	if !strings.Contains(source, "cli.doRoundTrip(") {
		t.Fatalf("HTTP method must call shared doRoundTrip:\n%s", source)
	}
	if !strings.Contains(source, `cli.recordHTTPMetrics("modes",`) {
		t.Fatalf("HTTP method must call shared recordHTTPMetrics with service label:\n%s", source)
	}
}

func TestRenderHTTP_SkippedWithoutHTTPContracts(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Rpc",
				PkgPath: "example/contracts",
				ID:      "Rpc",
				Annotations: tags.DocTags{
					model.TagServerJsonRPC: "",
				},
				Methods: []*model.Method{
					{
						Name: "Ping",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
						},
						Results: []*model.Variable{
							{Name: "message", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "client")

	renderer := NewClientRenderer(project, dir, "example", "client")
	if renderer.HasHTTP() {
		t.Fatal("project without HTTP contracts must not require HTTP helpers")
	}
}

func httpClientTestProject() (project *model.Project) {

	project = &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Modes",
				PkgPath: "example/contracts",
				ID:      "Modes",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					TagMetrics:          "",
				},
				Methods: []*model.Method{
					{
						Name: "BodyHeader",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "token", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "body", TypeRef: model.TypeRef{TypeID: "example/contracts/dto:Item"}},
						},
						Results: []*model.Variable{
							{Name: "ok", TypeRef: model.TypeRef{TypeID: "bool"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod: "POST",
							model.TagHttpPath:   "/modes/body-header",
						},
					},
				},
			},
		},
	}
	return project
}
