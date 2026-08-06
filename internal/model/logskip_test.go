// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"testing"

	"tgp/internal/tags"
)

func TestMethodLogSkip_parsesSidesAndNames(t *testing.T) {

	method := &Method{Annotations: tags.DocTags{TagLogSkip: "request, secret, response,name"}}
	sides, names := MethodLogSkip(nil, nil, method)
	if !sides.Request || !sides.Response {
		t.Fatalf("sides = %+v, want request+response", sides)
	}
	if _, ok := names["secret"]; !ok {
		t.Fatalf("names = %#v, want secret", names)
	}
	if _, ok := names["name"]; !ok {
		t.Fatalf("names = %#v, want name", names)
	}
	if _, ok := names["request"]; ok {
		t.Fatalf("request must not be in names: %#v", names)
	}
}

func TestVariableLogSkipped_doesNotInheritMethodListAsFlag(t *testing.T) {

	method := &Method{
		Annotations: tags.DocTags{TagLogSkip: "secret"},
		Args: []*Variable{
			{Name: "secret", TypeID: "string"},
			{Name: "name", TypeID: "string"},
		},
	}
	_, names := MethodLogSkip(nil, nil, method)
	if !VariableLogSkipped(method, method.Args[0], names) {
		t.Fatal("secret must be skipped by name list")
	}
	if VariableLogSkipped(method, method.Args[1], names) {
		t.Fatal("name must not be skipped via method IsSet inheritance")
	}
}

func TestVariableLogSkipped_paramFlagAndSubAnnotation(t *testing.T) {

	method := &Method{
		Annotations: tags.DocTags{"token.log-skip": ""},
		Args: []*Variable{
			{Name: "token", TypeID: "string"},
			{Name: "password", TypeID: "string", Annotations: tags.DocTags{TagLogSkip: ""}},
			{Name: "ok", TypeID: "string"},
		},
	}
	if !VariableLogSkipped(method, method.Args[0], nil) {
		t.Fatal("token.log-skip sub-annotation must skip")
	}
	if !VariableLogSkipped(method, method.Args[1], nil) {
		t.Fatal("param @tg log-skip must skip")
	}
	if VariableLogSkipped(method, method.Args[2], nil) {
		t.Fatal("ok must not be skipped")
	}
}

func TestVariableLogSkipped_dumperHideOnMethodTags(t *testing.T) {

	method := &Method{
		Annotations: tags.DocTags{"user.tags": "dumper:hide"},
		Args:        []*Variable{{Name: "user", TypeID: "string"}},
	}
	if !VariableLogSkipped(method, method.Args[0], nil) {
		t.Fatal("dumper:hide on method tags must skip")
	}
}

func TestStructFieldLogSkipped_logSkipAndDumper(t *testing.T) {

	if !StructFieldLogSkipped(&StructField{Name: "Secret", Annotations: tags.DocTags{TagLogSkip: ""}}) {
		t.Fatal("log-skip flag must skip")
	}
	if !StructFieldLogSkipped(&StructField{Name: "Internal", Annotations: tags.DocTags{TagParamTags: "dumper:hide"}}) {
		t.Fatal("tags=dumper:hide must skip")
	}
	if !StructFieldLogSkipped(&StructField{Name: "Hidden", Tags: map[string][]string{"dumper": {"hide"}}}) {
		t.Fatal("struct tag dumper:\"hide\" must skip")
	}
	if StructFieldLogSkipped(&StructField{Name: "Name"}) {
		t.Fatal("plain field must not skip")
	}
	if StructFieldLogSkipped(&StructField{Name: "Partial", Annotations: tags.DocTags{TagParamTags: "dumper:hide,fh"}}) {
		t.Fatal("formula hide must not be full omit for projection")
	}
}

func TestDumperHidesFully(t *testing.T) {

	cases := []struct {
		in   string
		want bool
	}{
		{"hide", true},
		{"hide,-", true},
		{"hide,", true},
		{"hide,fh", false},
		{"hide,1:2", false},
		{"", false},
		{"other", false},
	}
	for _, tc := range cases {
		if got := DumperHidesFully(tc.in); got != tc.want {
			t.Fatalf("DumperHidesFully(%q)=%v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestTypeNeedsLogProjection_nested(t *testing.T) {

	project := &Project{
		Types: map[string]*Type{
			"pkg:Outer": {
				Kind:     TypeKindStruct,
				TypeName: "Outer",
				StructFields: []*StructField{
					{Name: "Inner", TypeID: "pkg:Inner"},
				},
			},
			"pkg:Inner": {
				Kind:     TypeKindStruct,
				TypeName: "Inner",
				StructFields: []*StructField{
					{Name: "Secret", TypeID: "string", Annotations: tags.DocTags{TagLogSkip: ""}},
					{Name: "OK", TypeID: "string"},
				},
			},
			"pkg:Plain": {
				Kind:     TypeKindStruct,
				TypeName: "Plain",
				StructFields: []*StructField{
					{Name: "OK", TypeID: "string"},
				},
			},
		},
	}
	if !TypeNeedsLogProjection(project, "pkg:Outer") {
		t.Fatal("Outer must need projection via Inner.Secret")
	}
	if !TypeRefNeedsLogProjection(project, &TypeRef{TypeID: "pkg:Inner", NumberOfPointers: 1}) {
		t.Fatal("pointer to Inner must need projection")
	}
	if TypeNeedsLogProjection(project, "pkg:Plain") {
		t.Fatal("Plain must not need projection")
	}
}
