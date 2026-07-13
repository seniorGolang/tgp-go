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
	renderer := NewClientRenderer(project, dir, false, "")
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
	renderer := NewClientRenderer(project, dir, false, "")
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
