// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func TestIsRequiredGeneratedRequestField(t *testing.T) {

	methodTags := tags.DocTags{
		"OptionalByTag.tags": "json:optionalByTag,omitempty",
	}

	tests := []struct {
		name       string
		variable   *model.Variable
		methodTags tags.DocTags
		want       bool
	}{
		{
			name: "required value type",
			variable: &model.Variable{
				TypeRef: model.TypeRef{TypeID: "string"},
				Name:    "UserID",
			},
			want: true,
		},
		{
			name: "optional pointer",
			variable: &model.Variable{
				TypeRef: model.TypeRef{TypeID: "string", NumberOfPointers: 1},
				Name:    "Filter",
			},
			want: false,
		},
		{
			name: "optional by omitempty annotation",
			variable: &model.Variable{
				TypeRef:     model.TypeRef{TypeID: "string"},
				Name:        "Filter",
				Annotations: tags.DocTags{"json": "filter,omitempty"},
			},
			want: false,
		},
		{
			name: "optional by method var tag",
			variable: &model.Variable{
				TypeRef: model.TypeRef{TypeID: "string"},
				Name:    "OptionalByTag",
			},
			methodTags: methodTags,
			want:       false,
		},
		{
			name: "forced required pointer",
			variable: &model.Variable{
				TypeRef:     model.TypeRef{TypeID: "string", NumberOfPointers: 1},
				Name:        "Forced",
				Annotations: tags.DocTags{"required": ""},
			},
			want: true,
		},
		{
			name:     "nil variable",
			variable: nil,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRequiredGeneratedRequestField(tt.variable, tt.methodTags)
			if got != tt.want {
				t.Fatalf("isRequiredGeneratedRequestField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRequiredGeneratedResponseField(t *testing.T) {

	tests := []struct {
		name     string
		variable *model.Variable
		want     bool
	}{
		{
			name: "required value type",
			variable: &model.Variable{
				TypeRef: model.TypeRef{TypeID: "int"},
				Name:    "TotalCount",
			},
			want: true,
		},
		{
			name: "required pointer",
			variable: &model.Variable{
				TypeRef: model.TypeRef{TypeID: "string", NumberOfPointers: 1},
				Name:    "Response",
			},
			want: true,
		},
		{
			name: "optional by omitempty",
			variable: &model.Variable{
				TypeRef:     model.TypeRef{TypeID: "string"},
				Name:        "Message",
				Annotations: tags.DocTags{"json": "message,omitempty"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRequiredGeneratedResponseField(tt.variable, nil)
			if got != tt.want {
				t.Fatalf("isRequiredGeneratedResponseField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRequiredStructField(t *testing.T) {

	tests := []struct {
		name  string
		field *model.StructField
		want  bool
	}{
		{
			name: "required without json tag",
			field: &model.StructField{
				TypeRef: model.TypeRef{TypeID: "string"},
				Name:    "RequiredWithoutJSONTag",
			},
			want: true,
		},
		{
			name: "optional by omitempty",
			field: &model.StructField{
				TypeRef: model.TypeRef{TypeID: "string"},
				Name:    "OptionalByOmitEmpty",
				Tags:    map[string][]string{"json": {"optionalByOmitEmpty", "omitempty"}},
			},
			want: false,
		},
		{
			name: "required pointer without omitempty",
			field: &model.StructField{
				TypeRef: model.TypeRef{TypeID: "string", NumberOfPointers: 1},
				Name:    "OptionalPointer",
				Tags:    map[string][]string{"json": {"optionalPointer"}},
			},
			want: true,
		},
		{
			name: "forced required pointer",
			field: &model.StructField{
				TypeRef:     model.TypeRef{TypeID: "string", NumberOfPointers: 1},
				Name:        "ForcedPointer",
				Tags:        map[string][]string{"json": {"forcedPointer"}},
				Annotations: tags.DocTags{"required": ""},
			},
			want: true,
		},
		{
			name:  "nil field",
			field: nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRequiredStructField(tt.field)
			if got != tt.want {
				t.Fatalf("isRequiredStructField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRequiredQueryParameter(t *testing.T) {

	tests := []struct {
		name     string
		variable *model.Variable
		want     bool
	}{
		{
			name: "required value type",
			variable: &model.Variable{
				TypeRef: model.TypeRef{TypeID: "string"},
				Name:    "Filter",
			},
			want: true,
		},
		{
			name: "optional pointer",
			variable: &model.Variable{
				TypeRef: model.TypeRef{TypeID: "string", NumberOfPointers: 1},
				Name:    "Filter",
			},
			want: false,
		},
		{
			name: "forced required pointer",
			variable: &model.Variable{
				TypeRef:     model.TypeRef{TypeID: "string", NumberOfPointers: 1},
				Name:        "Filter",
				Annotations: tags.DocTags{"required": ""},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRequiredQueryParameter(tt.variable)
			if got != tt.want {
				t.Fatalf("isRequiredQueryParameter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRequiredHeaderOrCookie(t *testing.T) {

	requiredVariable := &model.Variable{
		TypeRef:     model.TypeRef{TypeID: "string"},
		Name:        "Token",
		Annotations: tags.DocTags{"required": ""},
	}
	optionalVariable := &model.Variable{
		TypeRef: model.TypeRef{TypeID: "string"},
		Name:    "Token",
	}

	if !isRequiredHeaderOrCookie(requiredVariable) {
		t.Fatal("expected header parameter with @tg required to be required")
	}
	if isRequiredHeaderOrCookie(optionalVariable) {
		t.Fatal("expected header parameter without @tg required to be optional")
	}
	if isRequiredHeaderOrCookie(nil) {
		t.Fatal("expected nil variable to be optional")
	}
}

func TestHasJSONOmitemptyTag(t *testing.T) {

	if !hasJSONOmitemptyTag("name,omitempty") {
		t.Fatal("expected omitempty in comma-separated json tag")
	}
	if hasJSONOmitemptyTag("name") {
		t.Fatal("expected tag without omitempty to return false")
	}
	if hasJSONOmitemptyTag("") {
		t.Fatal("expected empty tag to return false")
	}
}

func TestHasJSONOmitemptyForStructFieldNegative(t *testing.T) {

	field := &model.StructField{
		Tags: map[string][]string{"json": {"name"}},
	}
	if hasJSONOmitemptyForStructField(field) {
		t.Fatal("expected field without omitempty option to return false")
	}
}
