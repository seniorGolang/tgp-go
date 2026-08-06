// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"tgp/internal/model"
)

func TestExtractTagsFromASTStruct_AnonymousInline(t *testing.T) {

	const src = `package p

type LayerDetail struct {
	LayerSummary ` + "`json:\",inline\"`" + `
	Foo string ` + "`json:\"foo\"`" + `
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "layer.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	var structType *ast.StructType
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "LayerDetail" {
				continue
			}
			structType, _ = typeSpec.Type.(*ast.StructType)
		}
	}
	if structType == nil {
		t.Fatal("struct type not found")
	}

	tags := extractTagsFromASTStruct(structType, "LayerSummary", true)
	if got := tags["json"]; len(got) != 2 || got[0] != "" || got[1] != "inline" {
		t.Fatalf("anonymous inline tags = %#v", tags)
	}

	if tags = extractTagsFromASTStruct(structType, "LayerSummary", false); len(tags) != 0 {
		t.Fatalf("non-anonymous lookup must not match anonymous field, got %#v", tags)
	}

	tags = extractTagsFromASTStruct(structType, "Foo", false)
	if got := tags["json"]; len(got) != 1 || got[0] != "foo" {
		t.Fatalf("named field tags = %#v", tags)
	}
}

func TestAstFieldMatches(t *testing.T) {

	field := &ast.Field{Names: nil}
	if !astFieldMatches(field, "LayerSummary", true) {
		t.Fatal("anonymous field must match when anonymous=true")
	}
	if astFieldMatches(field, "LayerSummary", false) {
		t.Fatal("anonymous field must not match when anonymous=false")
	}

	named := &ast.Field{Names: []*ast.Ident{{Name: "Foo"}}}
	if !astFieldMatches(named, "Foo", false) {
		t.Fatal("named field must match by name")
	}
}

func TestFillStructFields_skipsInvalidFieldType(t *testing.T) {

	pkg := types.NewPackage("example.com/jose", "jose")
	invalidSlice := types.NewSlice(types.Typ[types.Invalid])
	fields := []*types.Var{
		types.NewField(token.NoPos, pkg, "KeyID", types.Typ[types.String], false),
		types.NewField(token.NoPos, pkg, "Certificates", invalidSlice, false),
	}
	structType := types.NewStruct(fields, []string{"", ""})

	coreType := &model.Type{TypeName: "JSONWebKey"}
	project := &model.Project{Types: map[string]*model.Type{}}

	fillStructFields(structType, "example.com/jose", nil, project, coreType, &AutonomousPackageLoader{})

	if len(coreType.StructFields) != 1 {
		t.Fatalf("expected only valid field, got %#v", coreType.StructFields)
	}
	if coreType.StructFields[0].Name != "KeyID" || coreType.StructFields[0].TypeID != "string" {
		t.Fatalf("unexpected field: %#v", coreType.StructFields[0])
	}
}

func TestFillStructFields_nilStruct(t *testing.T) {

	coreType := &model.Type{TypeName: "Empty"}
	fillStructFields(nil, "example", nil, &model.Project{Types: map[string]*model.Type{}}, coreType, &AutonomousPackageLoader{})
	if coreType.StructFields != nil {
		t.Fatalf("nil struct must leave StructFields untouched, got %#v", coreType.StructFields)
	}
}
