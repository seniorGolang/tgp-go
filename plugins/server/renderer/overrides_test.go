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

func TestRenderHTTP_HandlerAnnotationCallsFunction(t *testing.T) {

	project, contract := overridesTestContract()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewContractRenderer(project, contract, dir)
	if err := renderer.RenderHTTP(); err != nil {
		t.Fatalf("RenderHTTP: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "overrides-http.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if strings.Contains(source, "PingHandler, (") {
		t.Fatalf("handler must be invoked, not returned as separate value:\n%s", source)
	}
	if !strings.Contains(source, "fiberhooks.PingHandler(ftx, http.base)") {
		t.Fatalf("expected PingHandler call, got:\n%s", source)
	}
}

func TestRenderREST_HttpResponseAnnotationCallsFunction(t *testing.T) {

	project, contract := overridesTestContract()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewContractRenderer(project, contract, dir)
	if err := renderer.RenderREST(); err != nil {
		t.Fatalf("RenderREST: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "overrides-rest.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if strings.Contains(source, "EchoResponseHandler, (") {
		t.Fatalf("http-response must be invoked, not returned as separate value:\n%s", source)
	}
	if !strings.Contains(source, "fiberhooks.EchoResponseHandler(ftx, http.base, request.Text)") {
		t.Fatalf("expected EchoResponseHandler call, got:\n%s", source)
	}
	if strings.Contains(source, "func (http *httpOverrides) customEcho(") {
		t.Fatalf("http-response= must not generate unused svc wrapper:\n%s", source)
	}
	if strings.Contains(source, "func (http *httpOverrides) customPing(") {
		t.Fatalf("handler= must not generate unused svc wrapper:\n%s", source)
	}
}

func TestRenderHTTP_HandlerAnnotationRejectsMalformedReturn(t *testing.T) {

	project, contract := overridesTestContract()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewContractRenderer(project, contract, dir)
	if err := renderer.RenderHTTP(); err != nil {
		t.Fatalf("RenderHTTP: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "overrides-http.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	if strings.Contains(string(content), "return fiberhooks.PingHandler, ftx") {
		t.Fatal("must not generate comma-separated handler and args return")
	}
}

func overridesTestContract() (project *model.Project, contract *model.Contract) {

	contract = &model.Contract{
		Name:    "Overrides",
		PkgPath: "example/internal/services",
		ID:      "Overrides",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
			model.TagHttpPrefix: "api/v1",
		},
		Methods: []*model.Method{
			{
				Name: "CustomPing",
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
				},
				Results: []*model.Variable{
					{Name: "message", TypeRef: model.TypeRef{TypeID: "string"}},
					{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
				},
				Annotations: tags.DocTags{
					model.TagHTTPMethod: "GET",
					model.TagHttpPath:   "/overrides/ping",
					TagHandler:          "example/internal/fiberhooks:PingHandler",
				},
			},
			{
				Name: "CustomEcho",
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
					{Name: "text", TypeRef: model.TypeRef{TypeID: "string"}},
				},
				Results: []*model.Variable{
					{Name: "echo", TypeRef: model.TypeRef{TypeID: "string"}},
					{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
				},
				Annotations: tags.DocTags{
					model.TagHTTPMethod: "GET",
					model.TagHttpPath:   "/overrides/echo",
					model.TagHttpArg:    "text|text",
					TagHttpResponse:     "example/internal/fiberhooks:EchoResponseHandler",
				},
			},
		},
	}

	project = &model.Project{
		ModulePath: "example",
		Contracts:  []*model.Contract{contract},
		Types:      map[string]*model.Type{},
	}
	return project, contract
}
