// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"tgp/internal/model"
)

func TestExtractImportsFromTypeDefinition_transitiveSamePackage(t *testing.T) {

	const src = `package jose

import (
	x509 "crypto/x509"
	url "net/url"
)

type JSONWebKeySet struct {
	Keys []JSONWebKey ` + "`json:\"keys\"`" + `
}

type JSONWebKey struct {
	KeyID           string
	Certificates    []*x509.Certificate
	CertificatesURL *url.URL
}
`

	files := mustParsePackageFiles(t, "jose.go", src)
	resolver := &PackageResolver{
		modulePath: "example",
		resolveCache: map[string]string{
			"crypto/x509": "/x509",
			"net/url":     "/url",
		},
	}

	required := extractImportsFromTypeDefinition(files, "JSONWebKeySet", resolver)
	if !required["crypto/x509"] {
		t.Fatalf("JSONWebKeySet must pull crypto/x509 via JSONWebKey, got %#v", required)
	}
	if !required["net/url"] {
		t.Fatalf("JSONWebKeySet must pull net/url via JSONWebKey, got %#v", required)
	}
}

func TestExtractImportsFromTypeDefinition_noTransitiveLeak(t *testing.T) {

	const src = `package jose

import (
	x509 "crypto/x509"
	url "net/url"
)

type JSONWebKeySet struct {
	Keys []JSONWebKey ` + "`json:\"keys\"`" + `
}

type JSONWebKey struct {
	KeyID string
}

type Unused struct {
	Certificates []*x509.Certificate
	URL          *url.URL
}
`

	files := mustParsePackageFiles(t, "jose.go", src)
	resolver := &PackageResolver{
		modulePath: "example",
		resolveCache: map[string]string{
			"crypto/x509": "/x509",
			"net/url":     "/url",
		},
	}

	required := extractImportsFromTypeDefinition(files, "JSONWebKeySet", resolver)
	if required["crypto/x509"] || required["net/url"] {
		t.Fatalf("unused type imports must not leak: %#v", required)
	}
}

func TestExtractImportsFromTypeDefinition_cycle(t *testing.T) {

	const src = `package jose

import x509 "crypto/x509"

type A struct {
	B *B
}

type B struct {
	A    *A
	Cert *x509.Certificate
}
`

	files := mustParsePackageFiles(t, "jose.go", src)
	resolver := &PackageResolver{
		modulePath:   "example",
		resolveCache: map[string]string{"crypto/x509": "/x509"},
	}

	required := extractImportsFromTypeDefinition(files, "A", resolver)
	if !required["crypto/x509"] {
		t.Fatalf("cycle must still collect nested imports, got %#v", required)
	}
}

func TestExtractImportsFromErrorType_transitiveAndMethods(t *testing.T) {

	const src = `package errs

import (
	codes "example/codes"
	details "example/details"
)

type AppError struct {
	Detail details.Info
}

func (e AppError) Error() string { return "" }

func (e AppError) Code() codes.Code { return codes.Code{} }
`

	files := mustParsePackageFiles(t, "errs.go", src)
	resolver := &PackageResolver{
		modulePath: "example",
		resolveCache: map[string]string{
			"example/codes":   "/codes",
			"example/details": "/details",
		},
	}

	required := extractImportsFromErrorType(files, "AppError", resolver)
	if !required["example/details"] {
		t.Fatalf("missing details import: %#v", required)
	}
	if !required["example/codes"] {
		t.Fatalf("missing codes import from Code method: %#v", required)
	}
}

func TestLoadPackageForType_transitiveSamePackageImports(t *testing.T) {

	root := t.TempDir()
	joseDir := filepath.Join(root, "jose")
	certsDir := filepath.Join(root, "certs")
	if err := os.MkdirAll(joseDir, 0o755); err != nil {
		t.Fatalf("mkdir jose: %v", err)
	}
	if err := os.MkdirAll(certsDir, 0o755); err != nil {
		t.Fatalf("mkdir certs: %v", err)
	}

	joseSrc := `package jose

import "example/certs"

type JSONWebKeySet struct {
	Keys []JSONWebKey ` + "`json:\"keys\"`" + `
}

type JSONWebKey struct {
	KeyID        string
	Certificates []*certs.Certificate
}
`
	certsSrc := `package certs

type Certificate struct {
	Raw []byte
}
`
	if err := os.WriteFile(filepath.Join(joseDir, "jose.go"), []byte(joseSrc), 0o644); err != nil {
		t.Fatalf("write jose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(certsDir, "certs.go"), []byte(certsSrc), 0o644); err != nil {
		t.Fatalf("write certs: %v", err)
	}

	modFile := testModuleFile(t, "example")
	loader, err := NewAutonomousPackageLoader(modFile)
	if err != nil {
		t.Fatalf("loader: %v", err)
	}
	loader.resolver.resolveCacheMu.Lock()
	loader.resolver.resolveCache["example/jose"] = joseDir
	loader.resolver.resolveCache["example/certs"] = certsDir
	loader.resolver.resolveCacheMu.Unlock()

	pkgInfo, err := loader.LoadPackageForType("example/jose", "JSONWebKeySet")
	if err != nil {
		t.Fatalf("LoadPackageForType: %v", err)
	}

	obj := pkgInfo.Types.Scope().Lookup("JSONWebKey")
	if obj == nil {
		t.Fatal("JSONWebKey not found")
	}
	typeName, ok := obj.(*types.TypeName)
	if !ok {
		t.Fatalf("unexpected object %T", obj)
	}

	project := &model.Project{Types: map[string]*model.Type{}}
	coreType, convErr := convertTypeFromGoTypes(typeName.Type(), "example/jose", pkgInfo.Imports, project, loader)
	if convErr != nil {
		t.Fatalf("convertTypeFromGoTypes: %v", convErr)
	}
	if coreType == nil {
		t.Fatal("coreType is nil")
	}

	var certificates *model.StructField
	for _, field := range coreType.StructFields {
		if field.Name == "Certificates" {
			certificates = field
			break
		}
	}
	if certificates == nil {
		t.Fatalf("Certificates field missing (invalid type was likely skipped): %#v", coreType.StructFields)
	}
	if certificates.TypeID != "example/certs:Certificate" {
		t.Fatalf("Certificates TypeID=%q, want example/certs:Certificate", certificates.TypeID)
	}
	if !certificates.IsSlice || certificates.ElementPointers != 1 {
		t.Fatalf("Certificates slice/pointers = IsSlice=%v ElementPointers=%d", certificates.IsSlice, certificates.ElementPointers)
	}
}

func mustParsePackageFiles(t *testing.T, filename string, src string) (files []*ast.File) {

	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return []*ast.File{file}
}
