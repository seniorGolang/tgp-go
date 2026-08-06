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
			model.TagHTTPPrefix: "api/v1",
		},
		Methods: []*model.Method{{
			Name: "QueryOptional",
			Annotations: tags.DocTags{
				model.TagHTTPMethod: "GET",
				model.TagHTTPPath:   "/annotations/query-optional",
				model.TagHTTPArg:    "filter|filter",
			},
			Args: []*model.Variable{
				{Name: "filter", TypeID: "string", NumberOfPointers: 1},
			},
			Results: []*model.Variable{
				{Name: "value", TypeID: "string"},
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
	if !strings.Contains(source, `let _path_ = "/api/v1/annotations/query-optional"`) {
		t.Fatalf("HTTP path must keep absolute route with http-prefix:\n%s", source)
	}
	if !strings.Contains(source, "this.baseClient.resolveURL(_path_)") {
		t.Fatalf("HTTP client must resolve URL via baseClient.resolveURL:\n%s", source)
	}
	if strings.Contains(source, "getEndpoint()") || strings.Contains(source, "endsWith(\"/\")") {
		t.Fatalf("HTTP client must not concatenate endpoint with path:\n%s", source)
	}
}

func TestRenderHTTPClient_formBodyUsesFormFieldName(t *testing.T) {

	project := &model.Project{ModulePath: "example"}
	contract := &model.Contract{
		Name:    "Annotations",
		PkgPath: "example/contracts",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
			model.TagHTTPPrefix: "api/v1",
		},
		Methods: []*model.Method{{
			Name: "FormBody",
			Annotations: tags.DocTags{
				model.TagHTTPMethod:         "POST",
				model.TagHTTPPath:           "/annotations/form",
				model.TagRequestContentType: "application/x-www-form-urlencoded",
				"name.tags":                 "form:displayName",
				"optionalNote.tags":         "form:note",
			},
			Args: []*model.Variable{
				{Name: "name", TypeID: "string"},
				{Name: "optionalNote", TypeID: "string", NumberOfPointers: 1},
			},
			Results: []*model.Variable{
				{Name: "displayName", TypeID: "string"},
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
			model.TagHTTPPrefix: "api/v1",
		},
		Methods: []*model.Method{{
			Name: "HeaderRequired",
			Annotations: tags.DocTags{
				model.TagHTTPMethod: "GET",
				model.TagHTTPPath:   "/annotations/header-required",
				model.TagHTTPHeader: "token|X-Auth-Token|explicit",
			},
			Args: []*model.Variable{
				{Name: "token", TypeID: "string"},
			},
			Results: []*model.Variable{
				{Name: "ok", TypeID: "bool"},
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
			model.TagHTTPPrefix: "api/v1",
		},
		Methods: []*model.Method{{
			Name: "XmlRoundTrip",
			Annotations: tags.DocTags{
				model.TagHTTPMethod:          "POST",
				model.TagHTTPPath:            "/xml/echo",
				model.TagRequestContentType:  "application/xml",
				model.TagResponseContentType: "application/xml",
			},
			Args: []*model.Variable{
				{Name: "payload", TypeID: "string"},
			},
			Results: []*model.Variable{
				{Name: "out", TypeID: "string"},
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

func TestRenderHTTPClient_resultHeaderSliceParsesCSVNumbers(t *testing.T) {

	project := &model.Project{ModulePath: "example"}
	contract := &model.Contract{
		Name:    "Http",
		PkgPath: "example/contracts",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
			model.TagHTTPPrefix: "api/v1",
		},
		Methods: []*model.Method{{
			Name: "EchoIDHeader",
			Annotations: tags.DocTags{
				model.TagHTTPMethod: "GET",
				model.TagHTTPPath:   "/items/echo-ids-header",
				model.TagHTTPHeader: "ids|X-Item-Ids",
			},
			Results: []*model.Variable{
				{Name: "ids", TypeID: "int64", IsSlice: true},
			},
		}, {
			Name: "GetByIDs",
			Annotations: tags.DocTags{
				model.TagHTTPMethod: "GET",
				model.TagHTTPPath:   "/items/by-ids",
				model.TagHTTPArg:    "ids|ids",
			},
			Args: []*model.Variable{
				{Name: "ids", TypeID: "int64", IsSlice: true},
			},
			Results: []*model.Variable{
				{Name: "items", TypeID: "string", IsSlice: true},
			},
		}},
	}

	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderForm(); err != nil {
		t.Fatalf("RenderForm: %v", err)
	}
	if err := renderer.RenderHTTPClientClass(contract); err != nil {
		t.Fatalf("RenderHTTPClientClass: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "http-http.ts"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)
	if strings.Contains(source, "function parseCSVFormValue") {
		t.Fatalf("helpers must live in form.ts, not http client:\n%s", source)
	}
	if !strings.Contains(source, `parseCSVFormValue(_response_.headers.get("X-Item-Ids"), "number")`) {
		t.Fatalf("EchoIDHeader must CSV-parse X-Item-Ids as number[], got:\n%s", source)
	}
	if strings.Contains(source, `parseFormValue(_response_.headers.get("X-Item-Ids"), "string")`) {
		t.Fatalf("EchoIDHeader must not treat CSV []int64 header as string:\n%s", source)
	}
	if !strings.Contains(source, `_params_.ids.map(String).join(",")`) {
		t.Fatalf("GetByIDs query must CSV-join number[], got:\n%s", source)
	}
}
