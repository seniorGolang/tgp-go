// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/mod/modfile"
)

func TestAllPackageImports_includesBodyOnlyImports(t *testing.T) {

	const src = `package http

import errs "example/errs"

type Service struct{}

func (svc *Service) ReturnsError() error {
	return errs.NotFound
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "handlers.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	files := []*ast.File{file}
	resolver := &PackageResolver{
		modulePath:   "example",
		resolveCache: map[string]string{"example/errs": "/errs"},
	}

	allImports := allPackageImports(files, resolver)
	if !allImports["example/errs"] {
		t.Fatalf("allPackageImports missing example/errs: %#v", allImports)
	}

	partialImports := extractImportsFromExportedAndAliases(files, resolver)
	if partialImports["example/errs"] {
		t.Fatalf("extractImportsFromExportedAndAliases must not include body-only import: %#v", partialImports)
	}
}

func TestLoadPackageFromFiles_typechecksBodyImports(t *testing.T) {

	const errsSrc = `package errs

type Error struct{}

func (Error) Error() string { return "" }

var NotFound = Error{}
`

	const httpSrc = `package http

import errs "example/errs"

type Service struct{}

func (svc *Service) ReturnsError() error {
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

	modFile := testModuleFile(t, "example")

	loader, err := NewAutonomousPackageLoader(modFile)
	if err != nil {
		t.Fatalf("loader: %v", err)
	}

	errsTypeInfo := createTypeInfo()
	errsCfg := &types.Config{Error: func(error) {}}
	errsPkg, err := errsCfg.Check("example/errs", fset, []*ast.File{errsFile}, errsTypeInfo)
	if err != nil {
		t.Fatalf("type check errs: %v", err)
	}

	loader.mu.Lock()
	loader.cache["example/errs"] = &PackageInfo{
		PkgPath:  "example/errs",
		Types:    errsPkg,
		TypeInfo: errsTypeInfo,
		Fset:     fset,
	}
	loader.mu.Unlock()

	pkgInfo, err := loader.LoadPackageFromFiles("example/http", "/http", fset, []*ast.File{httpFile})
	if err != nil {
		t.Fatalf("LoadPackageFromFiles: %v", err)
	}

	funcDecl := findFuncDeclInFiles([]*ast.File{httpFile}, "ReturnsError")
	signature := funcSignatureFromDecl(funcDecl, pkgInfo.TypeInfo)

	refs := findErrorTypesInMethodBody(funcDecl.Body, signature, pkgInfo)
	if len(refs) != 1 {
		t.Fatalf("unexpected refs count: %#v", refs)
	}
	if refs[0].Name != "NotFound" || refs[0].TypeName != "Error" || refs[0].PkgPath != "example/errs" {
		t.Fatalf("unexpected refs: %#v", refs)
	}
}

func testModuleFile(t *testing.T, modulePath string) *modfile.File {

	t.Helper()

	modFile, err := modfile.Parse("go.mod", []byte("module "+modulePath+"\n"), nil)
	if err != nil {
		t.Fatalf("modfile: %v", err)
	}
	return modFile
}
