// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"testing"

	"tgp/internal/tags"
)

func TestEffectiveVariableMergesMethodSubAnnotations(t *testing.T) {

	method := &Method{
		Annotations: tags.DocTags{
			"token.required": "",
			"token.desc":     "from method",
		},
	}
	variable := &Variable{
		Name: "token",
		Annotations: tags.DocTags{
			"format": "uuid",
		},
	}

	effective := EffectiveVariable(method, variable)
	if effective == nil {
		t.Fatal("expected effective variable")
	}
	if !effective.Annotations.IsSet(TagRequired) {
		t.Fatal("expected required from token.required")
	}
	if effective.Annotations.Value("desc", "") != "from method" {
		t.Fatalf("desc = %q, want %q", effective.Annotations.Value("desc", ""), "from method")
	}
	if effective.Annotations.Value("format", "") != "uuid" {
		t.Fatalf("format = %q, want uuid", effective.Annotations.Value("format", ""))
	}
}

func TestEffectiveVariablePrefersVariableAnnotationsOverMethodSub(t *testing.T) {

	method := &Method{
		Annotations: tags.DocTags{
			"token.required": "",
			"token.desc":     "from method",
		},
	}
	variable := &Variable{
		Name: "token",
		Annotations: tags.DocTags{
			"desc": "from argument",
		},
	}

	effective := EffectiveVariable(method, variable)
	if effective.Annotations.Value("desc", "") != "from argument" {
		t.Fatalf("desc = %q, want %q", effective.Annotations.Value("desc", ""), "from argument")
	}
}

func TestEffectiveVariableReturnsOriginalWithoutMethodSub(t *testing.T) {

	variable := &Variable{Name: "token"}
	method := &Method{Annotations: tags.DocTags{"other.required": ""}}

	effective := EffectiveVariable(method, variable)
	if effective != variable {
		t.Fatal("expected original variable when method has no sub annotations")
	}
}

func TestEffectiveVariableNilVariable(t *testing.T) {

	if EffectiveVariable(&Method{}, nil) != nil {
		t.Fatal("expected nil for nil variable")
	}
}

func TestIsAnnotationSetResolvesMethodSubAnnotations(t *testing.T) {

	method := &Method{
		Annotations: tags.DocTags{"token.required": ""},
	}
	variable := &Variable{Name: "token"}

	if !IsAnnotationSet(nil, nil, method, variable, TagRequired) {
		t.Fatal("expected required from token.required via IsAnnotationSet")
	}
}

func TestGetAnnotationValueResolvesMethodSubAnnotations(t *testing.T) {

	method := &Method{
		Annotations: tags.DocTags{"token.desc": "header token"},
	}
	variable := &Variable{Name: "token"}

	if got := GetAnnotationValue(nil, nil, method, variable, "desc", ""); got != "header token" {
		t.Fatalf("desc = %q, want %q", got, "header token")
	}
}

func TestIsAnnotationSetPrefersVariableAnnotationsOverMethodSub(t *testing.T) {

	method := &Method{
		Annotations: tags.DocTags{
			"token.desc": "from method",
		},
	}
	variable := &Variable{
		Name: "token",
		Annotations: tags.DocTags{
			"desc": "from argument",
		},
	}

	if got := GetAnnotationValue(nil, nil, method, variable, "desc", ""); got != "from argument" {
		t.Fatalf("desc = %q, want %q", got, "from argument")
	}
}
