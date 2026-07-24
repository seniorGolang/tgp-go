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

func TestTypeRefFromExpr_pointerCounts(t *testing.T) {

	typeInfo, exprs := mustTypecheckExprs(t, `
package sample

type ID [16]byte

type Service interface {
	One(key *ID)
	Two(key **ID)
	Plain(key ID)
}
`)

	cases := []struct {
		name     string
		expr     ast.Expr
		wantPtr  int
		wantType string
	}{
		{name: "one", expr: exprs["One"], wantPtr: 1, wantType: "sample:ID"},
		{name: "two", expr: exprs["Two"], wantPtr: 2, wantType: "sample:ID"},
		{name: "plain", expr: exprs["Plain"], wantPtr: 0, wantType: "sample:ID"},
	}

	project := &model.Project{Types: map[string]*model.Type{
		"sample:ID": {TypeName: "ID", ImportPkgPath: "sample"},
	}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info, ok := typeRefFromExpr(tc.expr, "sample", nil, project, &AutonomousPackageLoader{}, typeInfo)
			if !ok {
				t.Fatalf("typeRefFromExpr failed")
			}
			if info.NumberOfPointers != tc.wantPtr {
				t.Fatalf("NumberOfPointers=%d, want %d", info.NumberOfPointers, tc.wantPtr)
			}
			if info.TypeID != tc.wantType {
				t.Fatalf("TypeID=%q, want %q", info.TypeID, tc.wantType)
			}
		})
	}
}

func TestTypeRefFromExpr_ellipsisMap(t *testing.T) {

	typeInfo, exprs := mustTypecheckExprs(t, `
package sample

type Service interface {
	Attrs(attributes ...map[string]any)
}
`)

	info, ok := typeRefFromExpr(exprs["Attrs"], "sample", nil, &model.Project{Types: make(map[string]*model.Type)}, &AutonomousPackageLoader{}, typeInfo)
	if !ok {
		t.Fatalf("typeRefFromExpr failed")
	}
	if !info.IsEllipsis || !info.IsSlice {
		t.Fatalf("expected ellipsis slice, got IsEllipsis=%v IsSlice=%v", info.IsEllipsis, info.IsSlice)
	}
	if info.MapKey == nil || info.MapKey.TypeID != "string" {
		t.Fatalf("MapKey=%v", info.MapKey)
	}
	if info.MapValue == nil || info.MapValue.TypeID != "any" {
		t.Fatalf("MapValue=%v", info.MapValue)
	}
	if info.TypeID != "" {
		t.Fatalf("TypeID should be empty for map, got %q", info.TypeID)
	}
}

func TestTypeRefFromExpr_slicePointers(t *testing.T) {

	typeInfo, exprs := mustTypecheckExprs(t, `
package sample

type ID [16]byte

type Service interface {
	Keys(keys []*ID)
}
`)

	info, ok := typeRefFromExpr(exprs["Keys"], "sample", nil, &model.Project{Types: map[string]*model.Type{
		"sample:ID": {TypeName: "ID", ImportPkgPath: "sample"},
	}}, &AutonomousPackageLoader{}, typeInfo)
	if !ok {
		t.Fatalf("typeRefFromExpr failed")
	}
	if !info.IsSlice {
		t.Fatalf("expected slice")
	}
	if info.TypeID != "sample:ID" {
		t.Fatalf("TypeID=%q", info.TypeID)
	}
	if info.ElementPointers != 1 {
		t.Fatalf("ElementPointers=%d, want 1", info.ElementPointers)
	}
	if info.NumberOfPointers != 0 {
		t.Fatalf("NumberOfPointers=%d, want 0", info.NumberOfPointers)
	}
}

func TestTypeRefFromExpr_nilInfo(t *testing.T) {

	_, ok := typeRefFromExpr(&ast.Ident{Name: "string"}, "sample", nil, &model.Project{Types: make(map[string]*model.Type)}, &AutonomousPackageLoader{}, nil)
	if ok {
		t.Fatalf("expected failure for nil types.Info")
	}
}

func TestTypeRefFromTypes_nil(t *testing.T) {

	_, ok := typeRefFromTypes(nil, "sample", nil, &model.Project{Types: make(map[string]*model.Type)}, &AutonomousPackageLoader{})
	if ok {
		t.Fatalf("expected failure for nil type")
	}
}

func TestTypeRefFromTypes_builtins(t *testing.T) {

	typeInfo, exprs := mustTypecheckExprs(t, `
package sample

type Service interface {
	A(v *uint64)
	B(v string)
}
`)

	project := &model.Project{Types: make(map[string]*model.Type)}
	loader := &AutonomousPackageLoader{}

	info, ok := typeRefFromExpr(exprs["A"], "sample", nil, project, loader, typeInfo)
	if !ok {
		t.Fatalf("uint64 pointer failed")
	}
	if info.TypeID != "uint64" || info.NumberOfPointers != 1 {
		t.Fatalf("got TypeID=%q ptr=%d", info.TypeID, info.NumberOfPointers)
	}

	info, ok = typeRefFromExpr(exprs["B"], "sample", nil, project, loader, typeInfo)
	if !ok {
		t.Fatalf("string failed")
	}
	if info.TypeID != "string" || info.NumberOfPointers != 0 {
		t.Fatalf("got TypeID=%q ptr=%d", info.TypeID, info.NumberOfPointers)
	}
}

func mustTypecheckExprs(t *testing.T, src string) (typeInfo *types.Info, argExprs map[string]ast.Expr) {

	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	conf := types.Config{Importer: nil}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	if _, err = conf.Check("sample", fset, []*ast.File{file}, info); err != nil {
		t.Fatalf("typecheck: %v", err)
	}

	argExprs = make(map[string]ast.Expr)
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			iface, ok := ts.Type.(*ast.InterfaceType)
			if !ok || iface.Methods == nil {
				continue
			}
			for _, method := range iface.Methods.List {
				if len(method.Names) == 0 {
					continue
				}
				ft, ok := method.Type.(*ast.FuncType)
				if !ok || ft.Params == nil || len(ft.Params.List) == 0 {
					continue
				}
				argExprs[method.Names[0].Name] = ft.Params.List[0].Type
			}
		}
	}
	if len(argExprs) == 0 {
		t.Fatalf("no method args found")
	}
	return info, argExprs
}
