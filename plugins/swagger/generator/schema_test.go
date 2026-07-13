// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"reflect"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func TestRegisterStructKeepsPointerFieldsOptional(t *testing.T) {

	gen := newGenerator(&model.Project{Types: map[string]*model.Type{"string": {Kind: model.TypeKindString}}})

	gen.registerStruct("requestSample", "example/service", nil, []*model.Variable{
		{
			TypeRef: model.TypeRef{TypeID: "string"},
			Name:    "Required",
			Annotations: tags.DocTags{
				"json": "required",
			},
		},
		{
			TypeRef: model.TypeRef{TypeID: "string", NumberOfPointers: 1},
			Name:    "OptionalPointer",
			Annotations: tags.DocTags{
				"json": "optionalPointer",
			},
		},
		{
			TypeRef: model.TypeRef{TypeID: "string"},
			Name:    "OptionalByOmitEmpty",
			Annotations: tags.DocTags{
				"json": "optionalByOmitEmpty,omitempty",
			},
		},
	}, contentJSON, true)

	want := []string{"required"}
	if got := gen.schemas["requestSample"].Required; !reflect.DeepEqual(got, want) {
		t.Fatalf("required fields = %v, want %v", got, want)
	}
}

func TestRegisterStructMarksResponsePointerFieldsRequired(t *testing.T) {

	gen := newGenerator(&model.Project{Types: map[string]*model.Type{
		"string": {Kind: model.TypeKindString},
		"int":    {Kind: model.TypeKindInt},
	}})

	gen.registerStruct("responseSample", "example/service", nil, []*model.Variable{
		{
			TypeRef: model.TypeRef{TypeID: "int"},
			Name:    "TotalCount",
			Annotations: tags.DocTags{
				"json": "totalCount",
			},
		},
		{
			TypeRef: model.TypeRef{TypeID: "string", NumberOfPointers: 1},
			Name:    "Response",
			Annotations: tags.DocTags{
				"json": "response",
			},
		},
	}, contentJSON, false)

	want := []string{"totalCount", "response"}
	if got := gen.schemas["responseSample"].Required; !reflect.DeepEqual(got, want) {
		t.Fatalf("required fields = %v, want %v", got, want)
	}
}

func TestStructTypeToSchemaMarksFieldsRequiredByJSONPresence(t *testing.T) {

	sampleType := &model.Type{
		Kind:     model.TypeKindStruct,
		TypeName: "Sample",
		StructFields: []*model.StructField{
			{
				TypeRef: model.TypeRef{TypeID: "string"},
				Name:    "Required",
				Tags:    map[string][]string{"json": {"required"}},
			},
			{
				TypeRef: model.TypeRef{TypeID: "string"},
				Name:    "RequiredWithoutJSONTag",
			},
			{
				TypeRef: model.TypeRef{TypeID: "string"},
				Name:    "OptionalByOmitEmpty",
				Tags:    map[string][]string{"json": {"optionalByOmitEmpty", "omitempty"}},
			},
			{
				TypeRef: model.TypeRef{TypeID: "string", NumberOfPointers: 1},
				Name:    "OptionalPointer",
				Tags:    map[string][]string{"json": {"optionalPointer"}},
			},
			{
				TypeRef:     model.TypeRef{TypeID: "string", NumberOfPointers: 1},
				Name:        "ForcedPointer",
				Tags:        map[string][]string{"json": {"forcedPointer"}},
				Annotations: tags.DocTags{"required": ""},
			},
			{
				TypeRef: model.TypeRef{TypeID: "string"},
				Name:    "Ignored",
				Tags:    map[string][]string{"json": {"-"}},
			},
		},
	}

	gen := newGenerator(&model.Project{Types: map[string]*model.Type{
		"string": {Kind: model.TypeKindString},
		"Sample": sampleType,
	}})

	schemaRef := gen.structTypeToSchema(sampleType, nil)
	if schemaRef == nil || schemaRef.Ref == "" {
		t.Fatal("expected schema ref")
	}

	schema := gen.schemas["Sample"]
	want := []string{"required", "requiredWithoutJSONTag", "optionalPointer", "forcedPointer"}
	if !reflect.DeepEqual(schema.Required, want) {
		t.Fatalf("required fields = %v, want %v", schema.Required, want)
	}
	if _, found := schema.Properties["Ignored"]; found {
		t.Fatal("expected ignored field to be omitted from properties")
	}
}

func TestRegisterStructMarksMethodSubAnnotationsRequired(t *testing.T) {

	gen := newGenerator(&model.Project{Types: map[string]*model.Type{"string": {Kind: model.TypeKindString}}})
	method := &model.Method{
		Annotations: tags.DocTags{
			"Forced.required": "",
		},
	}

	gen.registerStruct("requestSample", "example/service", method, []*model.Variable{
		{
			TypeRef: model.TypeRef{TypeID: "string", NumberOfPointers: 1},
			Name:    "Forced",
			Annotations: tags.DocTags{
				"json": "forced",
			},
		},
	}, contentJSON, true)

	want := []string{"forced"}
	if got := gen.schemas["requestSample"].Required; !reflect.DeepEqual(got, want) {
		t.Fatalf("required fields = %v, want %v", got, want)
	}
}

func TestRegisterStructRequestMarksUnannotatedExportedFieldRequired(t *testing.T) {

	gen := newGenerator(&model.Project{Types: map[string]*model.Type{"string": {Kind: model.TypeKindString}}})

	gen.registerStruct("requestSample", "example/service", nil, []*model.Variable{
		{
			TypeRef: model.TypeRef{TypeID: "string"},
			Name:    "UserID",
		},
	}, contentJSON, true)

	want := []string{"userID"}
	if got := gen.schemas["requestSample"].Required; !reflect.DeepEqual(got, want) {
		t.Fatalf("required fields = %v, want %v", got, want)
	}
}
