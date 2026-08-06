// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/types/typeutil"

	"tgp/internal/model"
)

func TestTypeHasJSONCodec(t *testing.T) {

	const src = `package sample

type Opaque struct {
	Secret string
	Nested Nested
}

func (o Opaque) MarshalJSON() ([]byte, error) { return nil, nil }

type Nested struct {
	Value string
}

type Plain struct {
	Name string
}

type PointerCodec struct {
	X int
}

func (p *PointerCodec) UnmarshalJSON(data []byte) error { return nil }
`

	pkg := mustTypecheckPackage(t, "sample", src)
	opaque := pkg.Scope().Lookup("Opaque").Type()
	plain := pkg.Scope().Lookup("Plain").Type()
	pointerCodec := pkg.Scope().Lookup("PointerCodec").Type()

	if !typeHasJSONCodec(opaque) {
		t.Fatal("Opaque with MarshalJSON must be JSON codec")
	}
	if typeHasJSONCodec(plain) {
		t.Fatal("Plain without codec must not be JSON codec")
	}
	if !typeHasJSONCodec(pointerCodec) {
		t.Fatal("PointerCodec with UnmarshalJSON on pointer must be JSON codec")
	}
}

func TestConvertTypeFromGoTypes_opaqueSkipsFields(t *testing.T) {

	const src = `package sample

type Opaque struct {
	Secret string
	Nested Nested
}

func (o Opaque) MarshalJSON() ([]byte, error) { return nil, nil }

type Nested struct {
	Value string
}

type Plain struct {
	Name string
}
`

	pkg, files, fset, typeInfo := mustTypecheckPackageFull(t, "sample", src)
	loader := testLoaderWithPackage(t, "sample", "sample", files, pkg, typeInfo, fset)
	project := &model.Project{Types: map[string]*model.Type{}}

	opaqueObj := pkg.Scope().Lookup("Opaque").(*types.TypeName)
	opaqueType, err := convertTypeFromGoTypes(opaqueObj.Type(), "sample", nil, project, loader)
	if err != nil {
		t.Fatalf("convert opaque: %v", err)
	}
	if opaqueType.Kind != model.TypeKindStruct {
		t.Fatalf("opaque Kind=%v, want struct", opaqueType.Kind)
	}
	if len(opaqueType.StructFields) != 0 {
		t.Fatalf("opaque must not expand fields, got %#v", opaqueType.StructFields)
	}

	plainObj := pkg.Scope().Lookup("Plain").(*types.TypeName)
	plainType, err := convertTypeFromGoTypes(plainObj.Type(), "sample", nil, project, loader)
	if err != nil {
		t.Fatalf("convert plain: %v", err)
	}
	if len(plainType.StructFields) != 1 || plainType.StructFields[0].Name != "Name" {
		t.Fatalf("plain fields = %#v", plainType.StructFields)
	}
}

func TestForEachReachableType_wireClosureStopsAtJSONCodec(t *testing.T) {

	const src = `package jose

type JSONWebKeySet struct {
	Keys []JSONWebKey
}

type JSONWebKey struct {
	KeyID        string
	Certificates []*Certificate
}

func (k JSONWebKey) MarshalJSON() ([]byte, error) { return nil, nil }

type Certificate struct {
	Raw []byte
	Pool *CertPool
}

func (c *Certificate) Verify() (*CertPool, error) { return nil, nil }

type CertPool struct {
	getCert func() (*Certificate, error)
}
`

	pkg, files, fset, typeInfo := mustTypecheckPackageFull(t, "example/jose", src)
	loader := testLoaderWithPackage(t, "example/jose", "jose", files, pkg, typeInfo, fset)
	project := &model.Project{Types: map[string]*model.Type{}}
	seen := &typeutil.Map{}

	setObj := pkg.Scope().Lookup("JSONWebKeySet").(*types.TypeName)
	forEachReachableType(setObj.Type(), project, seen, loader)

	if _, ok := project.Types["example/jose:JSONWebKeySet"]; !ok {
		t.Fatal("JSONWebKeySet must be materialized")
	}
	keyType, ok := project.Types["example/jose:JSONWebKey"]
	if !ok {
		t.Fatal("JSONWebKey must be materialized as leaf")
	}
	if len(keyType.StructFields) != 0 {
		t.Fatalf("JSONWebKey must stay opaque, fields=%#v", keyType.StructFields)
	}
	if _, ok = project.Types["example/jose:Certificate"]; ok {
		t.Fatal("Certificate must not be pulled through opaque JSONWebKey")
	}
	if _, ok = project.Types["example/jose:CertPool"]; ok {
		t.Fatal("CertPool must not be pulled through method set")
	}

	setType := project.Types["example/jose:JSONWebKeySet"]
	if len(setType.StructFields) != 1 || setType.StructFields[0].Name != "Keys" {
		t.Fatalf("JSONWebKeySet fields = %#v", setType.StructFields)
	}
}

func TestForEachReachableType_skipsUnexportedFields(t *testing.T) {

	const src = `package sample

type Visible struct {
	Public  string
	hidden  *Hidden
}

type Hidden struct {
	Value string
}
`

	pkg, files, fset, typeInfo := mustTypecheckPackageFull(t, "sample", src)
	loader := testLoaderWithPackage(t, "sample", "sample", files, pkg, typeInfo, fset)
	project := &model.Project{Types: map[string]*model.Type{}}
	seen := &typeutil.Map{}

	forEachReachableType(pkg.Scope().Lookup("Visible").Type(), project, seen, loader)
	if _, ok := project.Types["sample:Hidden"]; ok {
		t.Fatal("unexported field type must not be expanded")
	}
}

func testLoaderWithPackage(t *testing.T, pkgPath string, pkgName string, files []*ast.File, pkg *types.Package, typeInfo *types.Info, fset *token.FileSet) (loader *AutonomousPackageLoader) {

	t.Helper()
	loader, err := NewAutonomousPackageLoader(testModuleFile(t, "example"))
	if err != nil {
		t.Fatalf("loader: %v", err)
	}
	loader.cache[pkgPath] = &PackageInfo{
		PkgPath:     pkgPath,
		PackageName: pkgName,
		Files:       files,
		Types:       pkg,
		TypeInfo:    typeInfo,
		Fset:        fset,
		Imports:     map[string]string{},
	}
	return
}

func mustTypecheckPackage(t *testing.T, pkgPath string, src string) (pkg *types.Package) {

	t.Helper()
	pkg, _, _, _ = mustTypecheckPackageFull(t, pkgPath, src)
	return
}

func mustTypecheckPackageFull(t *testing.T, pkgPath string, src string) (pkg *types.Package, files []*ast.File, fset *token.FileSet, typeInfo *types.Info) {

	t.Helper()
	fset = token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	files = []*ast.File{file}
	typeInfo = createTypeInfo()
	cfg := &types.Config{Error: func(error) {}}
	pkg, err = cfg.Check(pkgPath, fset, files, typeInfo)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	return
}
