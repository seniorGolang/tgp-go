// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package validate

import (
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func TestMethodHTTPAnnotations_rejectsInvalidHttpArgs(t *testing.T) {

	project := &model.Project{ModulePath: "example"}
	contract := &model.Contract{
		Name: "Http",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
		},
		Methods: []*model.Method{{
			Name: "Get",
			Annotations: tags.DocTags{
				model.TagHTTPMethod: "GET",
				model.TagHttpArg:    "broken",
			},
		}},
	}

	err := methodHTTPAnnotations(project, contract, contract.Methods[0])
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "http-args") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMethodHTTPAnnotations_acceptsValidMapping(t *testing.T) {

	project := &model.Project{ModulePath: "example"}
	contract := &model.Contract{
		Name: "Http",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
		},
		Methods: []*model.Method{{
			Name: "Get",
			Annotations: tags.DocTags{
				model.TagHTTPMethod: "GET",
				model.TagHttpArg:    "q|q",
			},
		}},
	}

	if err := methodHTTPAnnotations(project, contract, contract.Methods[0]); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMethodHTTPAnnotations_rejectsFormArgWithoutFormTag(t *testing.T) {

	project := &model.Project{ModulePath: "example"}
	contract := &model.Contract{
		Name: "Annotations",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
		},
		Methods: []*model.Method{{
			Name: "FormBody",
			Annotations: tags.DocTags{
				model.TagHTTPMethod:         "POST",
				model.TagRequestContentType: "application/x-www-form-urlencoded",
				"name.tags":                 "form:displayName",
				"optionalNote.tags":         "json:note,omitempty",
			},
			Args: []*model.Variable{
				{Name: "name", TypeRef: model.TypeRef{TypeID: "string"}},
				{Name: "optionalNote", TypeRef: model.TypeRef{TypeID: "string", NumberOfPointers: 1}},
			},
		}},
	}

	err := methodHTTPAnnotations(project, contract, contract.Methods[0])
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "optionalNote") || !strings.Contains(err.Error(), "form:<name>") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMethodHTTPAnnotations_acceptsFormArgWithFormTag(t *testing.T) {

	project := &model.Project{ModulePath: "example"}
	contract := &model.Contract{
		Name: "Annotations",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
		},
		Methods: []*model.Method{{
			Name: "FormBody",
			Annotations: tags.DocTags{
				model.TagHTTPMethod:         "POST",
				model.TagRequestContentType: "application/x-www-form-urlencoded",
				"name.tags":                 "form:displayName",
				"optionalNote.tags":         "form:note",
			},
			Args: []*model.Variable{
				{Name: "name", TypeRef: model.TypeRef{TypeID: "string"}},
				{Name: "optionalNote", TypeRef: model.TypeRef{TypeID: "string", NumberOfPointers: 1}},
			},
		}},
	}

	if err := methodHTTPAnnotations(project, contract, contract.Methods[0]); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMethodHTTPAnnotations_acceptsFormInlineStructArg(t *testing.T) {

	const tokenParamsTypeID = "example/contracts/dto:TokenParams"

	project := &model.Project{
		ModulePath: "example",
		Types: map[string]*model.Type{
			tokenParamsTypeID: {
				Kind:     model.TypeKindStruct,
				TypeName: "TokenParams",
			},
		},
	}
	contract := &model.Contract{
		Name: "OAuth2",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
		},
		Methods: []*model.Method{{
			Name: "Token",
			Annotations: tags.DocTags{
				model.TagHTTPMethod:         "POST",
				model.TagRequestContentType: "application/x-www-form-urlencoded",
				"params.tags":               "json:inline",
			},
			Args: []*model.Variable{
				{Name: "params", TypeRef: model.TypeRef{TypeID: tokenParamsTypeID}},
			},
		}},
	}

	if err := methodHTTPAnnotations(project, contract, contract.Methods[0]); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMethodHTTPAnnotations_rejectsFormNonInlineStructArg(t *testing.T) {

	const tokenParamsTypeID = "example/contracts/dto:TokenParams"

	project := &model.Project{
		ModulePath: "example",
		Types: map[string]*model.Type{
			tokenParamsTypeID: {
				Kind:     model.TypeKindStruct,
				TypeName: "TokenParams",
			},
		},
	}
	contract := &model.Contract{
		Name: "OAuth2",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
		},
		Methods: []*model.Method{{
			Name: "Token",
			Annotations: tags.DocTags{
				model.TagHTTPMethod:         "POST",
				model.TagRequestContentType: "application/x-www-form-urlencoded",
			},
			Args: []*model.Variable{
				{Name: "params", TypeRef: model.TypeRef{TypeID: tokenParamsTypeID}},
			},
		}},
	}

	err := methodHTTPAnnotations(project, contract, contract.Methods[0])
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "params") || !strings.Contains(err.Error(), "form:<name>") {
		t.Fatalf("unexpected error: %v", err)
	}
}
