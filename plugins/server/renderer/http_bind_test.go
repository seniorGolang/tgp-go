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

func TestRenderREST_FormUsesBodyParserOnly(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Annotations",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					model.TagHttpPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "FormBody",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "name", TypeRef: model.TypeRef{TypeID: "string"}},
						},
						Results: []*model.Variable{
							{Name: "displayName", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod:         "POST",
							model.TagHttpPath:           "/annotations/form",
							model.TagRequestContentType: "application/x-www-form-urlencoded",
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderExchange(); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	if err := renderer.RenderREST(); err != nil {
		t.Fatalf("RenderREST: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "annotations-rest.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, "BodyParser(&request)") {
		t.Fatalf("form must use BodyParser:\n%s", source)
	}
	if strings.Contains(source, "SetBodyRaw") || strings.Contains(source, "ReadAll(bodyStream)") {
		t.Fatalf("form must not ReadAll+SetBodyRaw:\n%s", source)
	}
}

func TestRenderREST_HeaderBindingUsesPeek(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Modes",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					model.TagHttpPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "BodyHeader",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "token", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "body", TypeRef: model.TypeRef{TypeID: "string"}},
						},
						Results: []*model.Variable{
							{Name: "ok", TypeRef: model.TypeRef{TypeID: "bool"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod: "POST",
							model.TagHttpPath:   "/modes/body-header",
							model.TagHttpHeader: "token|X-Token",
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderExchange(); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	if err := renderer.RenderREST(); err != nil {
		t.Fatalf("RenderREST: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "modes-rest.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, `Header.Peek("X-Token")`) {
		t.Fatalf("expected Peek for header presence:\n%s", source)
	}
	if strings.Contains(source, `ftx.Get("X-Token")`) {
		t.Fatalf("must not use Get+empty check for header overlay:\n%s", source)
	}
}

func TestRenderREST_MultiMultipartUsesOrderedStream(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Http",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					model.TagHttpPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "MultipartUploadMulti",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "partA", TypeRef: model.TypeRef{TypeID: "io:Reader"}},
							{Name: "partB", TypeRef: model.TypeRef{TypeID: "io:Reader"}},
						},
						Results: []*model.Variable{
							{Name: "id", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod:      "POST",
							model.TagHttpPath:        "/upload-multi",
							model.TagHttpPartName:    "partA|partA,partB|partB",
							model.TagHttpPartContent: "partA|text/plain,partB|application/octet-stream",
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	tr := NewTransportRenderer(project, dir)
	if err := tr.RenderTransportContext(); err != nil {
		t.Fatalf("RenderTransportContext: %v", err)
	}
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderExchange(); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	if err := renderer.RenderREST(); err != nil {
		t.Fatalf("RenderREST: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "http-rest.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if !strings.Contains(source, "stream.NewParts(") {
		t.Fatalf("multi multipart must use stream.NewParts:\n%s", source)
	}
	if strings.Contains(source, "partBodies") || strings.Contains(source, "ReadAll(p)") {
		t.Fatalf("multi must not buffer part bodies:\n%s", source)
	}
	if _, err = os.Stat(filepath.Join(dir, "stream", "parts.go")); err != nil {
		t.Fatalf("stream package must be rendered: %v", err)
	}
}

func TestRenderREST_HandlerSkipsServeMethod(t *testing.T) {

	project, contract := overridesTestContract()
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, contract, dir)
	if err := renderer.RenderExchange(); err != nil {
		t.Fatalf("RenderExchange: %v", err)
	}
	if err := renderer.RenderREST(); err != nil {
		t.Fatalf("RenderREST: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "overrides-rest.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	if strings.Contains(source, "func (http *httpOverrides) serveCustomPing(") {
		t.Fatalf("handler= method must not generate serve*:\n%s", source)
	}
}
