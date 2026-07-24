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

func TestRenderHTTPClient_optionalQueryParamSkipsUndefined(t *testing.T) {

	project := &model.Project{ModulePath: "example"}
	contract := &model.Contract{
		Name:    "Annotations",
		PkgPath: "example/contracts",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
			model.TagHttpPrefix: "api/v1",
		},
		Methods: []*model.Method{{
			Name: "QueryOptional",
			Annotations: tags.DocTags{
				model.TagHTTPMethod: "GET",
				model.TagHttpPath:   "/annotations/query-optional",
				model.TagHttpArg:    "filter|filter",
			},
			Args: []*model.Variable{
				{Name: "filter", TypeRef: model.TypeRef{TypeID: "string", NumberOfPointers: 1}},
			},
			Results: []*model.Variable{
				{Name: "value", TypeRef: model.TypeRef{TypeID: "string"}},
			},
		}},
	}

	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderHTTPClientClass(contract); err != nil {
		t.Fatalf("RenderHTTPClientClass: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "annotations-http.ts"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "filter?:") {
		t.Fatalf("expected optional filter parameter, got:\n%s", source)
	}
	if !strings.Contains(source, "_params_.filter !== undefined") {
		t.Fatalf("expected undefined guard for optional query param, got:\n%s", source)
	}
}

func TestRenderHTTPClient_formBodyUsesFormFieldName(t *testing.T) {

	project := &model.Project{ModulePath: "example"}
	contract := &model.Contract{
		Name:    "Annotations",
		PkgPath: "example/contracts",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
			model.TagHttpPrefix: "api/v1",
		},
		Methods: []*model.Method{{
			Name: "FormBody",
			Annotations: tags.DocTags{
				model.TagHTTPMethod:         "POST",
				model.TagHttpPath:           "/annotations/form",
				model.TagRequestContentType: "application/x-www-form-urlencoded",
				"name.tags":                 "form:displayName",
				"optionalNote.tags":         "form:note",
			},
			Args: []*model.Variable{
				{Name: "name", TypeRef: model.TypeRef{TypeID: "string"}},
				{Name: "optionalNote", TypeRef: model.TypeRef{TypeID: "string", NumberOfPointers: 1}},
			},
			Results: []*model.Variable{
				{Name: "displayName", TypeRef: model.TypeRef{TypeID: "string"}},
			},
		}},
	}

	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderHTTPClientClass(contract); err != nil {
		t.Fatalf("RenderHTTPClientClass: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "annotations-http.ts"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, `"displayName"`) {
		t.Fatalf("expected displayName form field, got:\n%s", source)
	}
	if !strings.Contains(source, `"note"`) {
		t.Fatalf("expected note form field, got:\n%s", source)
	}
	if strings.Contains(source, `"optionalNote"`) {
		t.Fatalf("must not use fallback optionalNote form field name, got:\n%s", source)
	}
}

func TestRenderHTTPClient_infersParamsTypeForHeaderMethod(t *testing.T) {

	project := &model.Project{ModulePath: "example"}
	contract := &model.Contract{
		Name:    "Annotations",
		PkgPath: "example/contracts",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
			model.TagHttpPrefix: "api/v1",
		},
		Methods: []*model.Method{{
			Name: "HeaderRequired",
			Annotations: tags.DocTags{
				model.TagHTTPMethod: "GET",
				model.TagHttpPath:   "/annotations/header-required",
				model.TagHttpHeader: "token|X-Auth-Token|explicit",
			},
			Args: []*model.Variable{
				{Name: "token", TypeRef: model.TypeRef{TypeID: "string"}},
			},
			Results: []*model.Variable{
				{Name: "ok", TypeRef: model.TypeRef{TypeID: "bool"}},
			},
		}},
	}

	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderHTTPClientClass(contract); err != nil {
		t.Fatalf("RenderHTTPClientClass: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "annotations-http.ts"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if strings.Contains(source, ":RequestAnnotationsHeaderRequired") {
		t.Fatalf("must not annotate params with exchange request type, got:\n%s", source)
	}
	if !strings.Contains(source, "const _params_ =") {
		t.Fatalf("expected inferred params object, got:\n%s", source)
	}
}

func TestRenderHTTPClient_xmlResponseUnwrapsRoot(t *testing.T) {

	project := &model.Project{ModulePath: "example"}
	contract := &model.Contract{
		Name:    "Http",
		PkgPath: "example/contracts",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
			model.TagHttpPrefix: "api/v1",
		},
		Methods: []*model.Method{{
			Name: "XmlRoundTrip",
			Annotations: tags.DocTags{
				model.TagHTTPMethod:          "POST",
				model.TagHttpPath:            "/xml/echo",
				model.TagRequestContentType:  "application/xml",
				model.TagResponseContentType: "application/xml",
			},
			Args: []*model.Variable{
				{Name: "payload", TypeRef: model.TypeRef{TypeID: "string"}},
			},
			Results: []*model.Variable{
				{Name: "out", TypeRef: model.TypeRef{TypeID: "string"}},
			},
		}},
	}

	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderHTTPClientClass(contract); err != nil {
		t.Fatalf("RenderHTTPClientClass: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "http-http.ts"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, "new XMLParser().parse") {
		t.Fatalf("expected XMLParser.parse, got:\n%s", source)
	}
	if !strings.Contains(source, "Object.keys") || !strings.Contains(source, "_rootKeys_") {
		t.Fatalf("expected XML root unwrap via Object.keys, got:\n%s", source)
	}
	if !strings.Contains(source, "new XMLBuilder().build") {
		t.Fatalf("expected XMLBuilder.build for request, got:\n%s", source)
	}
	if !strings.Contains(source, "requestHttpXmlRoundTrip") && !strings.Contains(source, "requestHTTPXmlRoundTrip") {
		t.Fatalf("expected XML request wrapped in Go exchange root, got:\n%s", source)
	}
}
