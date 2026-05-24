// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"tgp/internal/model"
)

type testPackageImporter map[string]*types.Package

func (imp testPackageImporter) Import(path string) (pkg *types.Package, err error) {

	pkg, ok := imp[path]
	if !ok {
		err = fmt.Errorf("package not found: %s", path)
		return
	}
	return
}

func TestFindErrorTypesInMethodBody(t *testing.T) {

	const src = `package svc

type Error struct{}

func (Error) Error() string { return "" }

var NotFound = Error{}

type SimpleErr struct{}

func (SimpleErr) Error() string { return "" }

type Item struct {
	ID string
}

type ReadCloser interface {
	Close() error
}

type stream struct{}

func (stream) Close() error { return nil }

func ReturnsError() error {
	return NotFound
}

func ReturnsSimple() error {
	return SimpleErr{}
}

func ReturnsNil() error {
	return nil
}

func ReturnsIface(err error) error {
	return err
}

func Download() (out ReadCloser, err error) {
	return stream{}, nil
}

func PostBody() (item *Item, err error) {
	return &Item{ID: "1"}, nil
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "svc.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	typeInfo := createTypeInfo()
	cfg := &types.Config{Error: func(error) {}}
	pkg, err := cfg.Check("svc", fset, []*ast.File{file}, typeInfo)
	if err != nil {
		t.Fatalf("type check: %v", err)
	}

	pkgInfo := &PackageInfo{
		PkgPath:  "example/svc",
		Types:    pkg,
		TypeInfo: typeInfo,
		Fset:     fset,
	}

	tests := []struct {
		name     string
		funcName string
		want     []errorTypeRefExpectation
	}{
		{
			name:     "package error var",
			funcName: "ReturnsError",
			want: []errorTypeRefExpectation{
				{pkgPath: "svc", name: "NotFound", typeName: "Error"},
			},
		},
		{
			name:     "error without Code method",
			funcName: "ReturnsSimple",
			want: []errorTypeRefExpectation{
				{pkgPath: "svc", name: "SimpleErr", typeName: "SimpleErr"},
			},
		},
		{
			name:     "nil return",
			funcName: "ReturnsNil",
			want:     nil,
		},
		{
			name:     "error interface",
			funcName: "ReturnsIface",
			want:     nil,
		},
		{
			name:     "success and error return values",
			funcName: "Download",
			want:     nil,
		},
		{
			name:     "dto and error return values",
			funcName: "PostBody",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runFindErrorTypesTest(t, []*ast.File{file}, typeInfo, pkgInfo, tt.funcName, tt.want)
		})
	}
}

func TestFindErrorTypesInMethodBody_crossPackage(t *testing.T) {

	const errsSrc = `package errs

type Error struct{}

func (Error) Error() string { return "" }

var NotFound = Error{}
`

	const httpSrc = `package http

import "example/errs"

func ReturnsError() error {
	return errs.NotFound
}
`

	fset := token.NewFileSet()

	errsFile, err := parser.ParseFile(fset, "errs.go", errsSrc, 0)
	if err != nil {
		t.Fatalf("parse errs: %v", err)
	}

	httpFile, err := parser.ParseFile(fset, "handlers.go", httpSrc, 0)
	if err != nil {
		t.Fatalf("parse http: %v", err)
	}

	errsTypeInfo := createTypeInfo()
	errsCfg := &types.Config{Error: func(error) {}}
	errsPkg, err := errsCfg.Check("example/errs", fset, []*ast.File{errsFile}, errsTypeInfo)
	if err != nil {
		t.Fatalf("type check errs: %v", err)
	}

	httpTypeInfo := createTypeInfo()
	httpCfg := &types.Config{
		Importer: testPackageImporter{
			"example/errs": errsPkg,
		},
		Error: func(error) {},
	}
	httpPkg, err := httpCfg.Check("example/http", fset, []*ast.File{httpFile}, httpTypeInfo)
	if err != nil {
		t.Fatalf("type check http: %v", err)
	}

	pkgInfo := &PackageInfo{
		PkgPath:  "example/http",
		Types:    httpPkg,
		TypeInfo: httpTypeInfo,
		Fset:     fset,
	}

	runFindErrorTypesTest(t, []*ast.File{httpFile}, httpTypeInfo, pkgInfo, "ReturnsError", []errorTypeRefExpectation{
		{pkgPath: "example/errs", name: "NotFound", typeName: "Error"},
	})
}

type errorTypeRefExpectation struct {
	pkgPath  string
	name     string
	typeName string
}

func runFindErrorTypesTest(
	t *testing.T,
	files []*ast.File,
	typeInfo *types.Info,
	pkgInfo *PackageInfo,
	funcName string,
	want []errorTypeRefExpectation,
) {

	t.Helper()

	funcDecl := findFuncDeclInFiles(files, funcName)
	if funcDecl == nil || funcDecl.Body == nil {
		t.Fatalf("body not found for %s", funcName)
	}

	signature := funcSignatureFromDecl(funcDecl, typeInfo)
	if signature == nil {
		t.Fatalf("signature not found for %s", funcName)
	}

	got := findErrorTypesInMethodBody(funcDecl.Body, signature, pkgInfo)
	if len(got) != len(want) {
		t.Fatalf("got %d refs, want %d: %#v", len(got), len(want), got)
	}

	for i, ref := range got {
		if ref.PkgPath != want[i].pkgPath {
			t.Fatalf("ref[%d].PkgPath = %q, want %q", i, ref.PkgPath, want[i].pkgPath)
		}
		if ref.Name != want[i].name {
			t.Fatalf("ref[%d].Name = %q, want %q", i, ref.Name, want[i].name)
		}
		if ref.TypeName != want[i].typeName {
			t.Fatalf("ref[%d].TypeName = %q, want %q", i, ref.TypeName, want[i].typeName)
		}
		fullName := fmt.Sprintf("%s.%s", want[i].pkgPath, want[i].name)
		if ref.FullName != fullName {
			t.Fatalf("ref[%d].FullName = %q, want %q", i, ref.FullName, fullName)
		}
	}
}

func TestIsErrorResultType(t *testing.T) {

	const src = `package svc

type ReadCloser interface {
	Close() error
}

type Item struct{}

func A() (out ReadCloser, err error) { return nil, nil }
func B() (item *Item, err error) { return nil, nil }
func C() (err error) { return nil }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "svc.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	typeInfo := createTypeInfo()
	cfg := &types.Config{Error: func(error) {}}
	pkg, err := cfg.Check("svc", fset, []*ast.File{file}, typeInfo)
	if err != nil {
		t.Fatalf("type check: %v", err)
	}

	tests := []struct {
		funcName string
		index    int
		want     bool
	}{
		{funcName: "A", index: 0, want: false},
		{funcName: "A", index: 1, want: true},
		{funcName: "B", index: 0, want: false},
		{funcName: "B", index: 1, want: true},
		{funcName: "C", index: 0, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.funcName, func(t *testing.T) {
			obj := pkg.Scope().Lookup(tt.funcName)
			fn, ok := obj.(*types.Func)
			if !ok {
				t.Fatalf("func %s not found", tt.funcName)
			}
			signature := fn.Type().(*types.Signature)
			got := isErrorResultType(signature.Results().At(tt.index).Type())
			if got != tt.want {
				t.Fatalf("isErrorResultType(%s[%d]) = %v, want %v", tt.funcName, tt.index, got, tt.want)
			}
		})
	}
}

func TestNamedConcreteErrorType(t *testing.T) {

	errorIface := createErrorInterface()

	tests := []struct {
		name string
		typ  types.Type
		want bool
	}{
		{
			name: "struct with Error method",
			typ: func() types.Type {
				typ := types.NewStruct([]*types.Var{
					types.NewField(token.NoPos, nil, "msg", types.Typ[types.String], false),
				}, nil)
				method := types.NewFunc(token.NoPos, types.NewPackage("p", "p"), "Error",
					types.NewSignatureType(nil, nil, nil, nil,
						types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])),
						false))
				return types.NewNamed(types.NewTypeName(token.NoPos, types.NewPackage("p", "p"), "Err", nil), typ, []*types.Func{method})
			}(),
			want: true,
		},
		{
			name: "predeclared error interface",
			typ:  types.Universe.Lookup("error").Type(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			named := namedConcreteErrorType(tt.typ)
			got := named != nil
			if got != tt.want {
				t.Fatalf("namedConcreteErrorType() presence = %v, want %v (type %v implements error: %v)",
					got, tt.want, tt.typ, types.Implements(tt.typ, errorIface))
			}
		})
	}
}

func TestErrorInfoFromReference(t *testing.T) {

	errorRef := &model.ErrorTypeReference{
		PkgPath:  "example/errs",
		Name:     "NotFound",
		TypeName: "Error",
		FullName: "example/errs.NotFound",
	}

	errInfo := errorInfoFromReference(errorRef)
	if errInfo == nil {
		t.Fatal("errorInfoFromReference returned nil")
	}
	if errInfo.TypeName != "NotFound" {
		t.Fatalf("TypeName = %q, want NotFound", errInfo.TypeName)
	}
	if errInfo.TypeID != "example/errs:Error" {
		t.Fatalf("TypeID = %q, want example/errs:Error", errInfo.TypeID)
	}
}
