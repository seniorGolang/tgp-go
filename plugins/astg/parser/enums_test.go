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

func TestDetectEnums_stringTypedConsts(t *testing.T) {

	loader, pkgPath := mustEnumPackage(t, `package dto

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
	RoleGuest Role = "guest"
)

type Color string

const (
	ColorRed   Color = "red"
	ColorGreen Color = "green"
)
`)

	role := &model.Type{
		Kind:          model.TypeKindString,
		TypeName:      "Role",
		ImportPkgPath: pkgPath,
		PkgName:       "dto",
	}
	detectEnums(role, loader)
	if len(role.Enums) != 3 {
		t.Fatalf("Role.Enums=%v, want 3", role.Enums)
	}
	if role.Enums[0].Name != "RoleAdmin" || role.Enums[0].Value != "admin" {
		t.Fatalf("first enum = %#v", role.Enums[0])
	}
	if role.Enums[1].Name != "RoleGuest" || role.Enums[1].Value != "guest" {
		t.Fatalf("second enum = %#v", role.Enums[1])
	}
	if role.Enums[2].Name != "RoleUser" || role.Enums[2].Value != "user" {
		t.Fatalf("third enum = %#v", role.Enums[2])
	}
}

func TestDetectEnums_intIota(t *testing.T) {

	loader, pkgPath := mustEnumPackage(t, `package dto

type Status int

const (
	StatusPending Status = iota
	StatusActive
	StatusDone
)
`)

	status := &model.Type{
		Kind:          model.TypeKindInt,
		TypeName:      "Status",
		ImportPkgPath: pkgPath,
		PkgName:       "dto",
	}
	detectEnums(status, loader)
	if len(status.Enums) != 3 {
		t.Fatalf("Status.Enums=%v", status.Enums)
	}
	byName := map[string]string{}
	for _, item := range status.Enums {
		byName[item.Name] = item.Value
	}
	if byName["StatusPending"] != "0" || byName["StatusActive"] != "1" || byName["StatusDone"] != "2" {
		t.Fatalf("iota values = %#v", byName)
	}
}

func TestDetectEnums_ignoresUntypedConsts(t *testing.T) {

	loader, pkgPath := mustEnumPackage(t, `package dto

type Role string

const (
	UntypedAdmin = "admin"
	UntypedUser  = "user"
)
`)

	role := &model.Type{
		Kind:          model.TypeKindString,
		TypeName:      "Role",
		ImportPkgPath: pkgPath,
	}
	detectEnums(role, loader)
	if len(role.Enums) != 0 {
		t.Fatalf("untyped consts must not produce enums, got %#v", role.Enums)
	}
}

func TestDetectEnums_requiresAtLeastTwo(t *testing.T) {

	loader, pkgPath := mustEnumPackage(t, `package dto

type Role string

const RoleAdmin Role = "admin"
`)

	role := &model.Type{
		Kind:          model.TypeKindString,
		TypeName:      "Role",
		ImportPkgPath: pkgPath,
	}
	detectEnums(role, loader)
	if len(role.Enums) != 0 {
		t.Fatalf("single const must not produce enums, got %#v", role.Enums)
	}
}

func TestDetectEnums_skipsIneligibleAndBuiltin(t *testing.T) {

	loader, pkgPath := mustEnumPackage(t, `package dto

type User struct {
	Name string
}
`)

	detectEnums(&model.Type{Kind: model.TypeKindStruct, TypeName: "User", ImportPkgPath: pkgPath}, loader)
	detectEnums(&model.Type{Kind: model.TypeKindString, TypeName: "string"}, loader)
	detectEnums(nil, loader)
	detectEnums(&model.Type{Kind: model.TypeKindString, TypeName: "Role"}, nil)
}

func TestDetectEnums_doesNotOverwriteExisting(t *testing.T) {

	loader, pkgPath := mustEnumPackage(t, `package dto

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)
`)

	role := &model.Type{
		Kind:          model.TypeKindString,
		TypeName:      "Role",
		ImportPkgPath: pkgPath,
		Enums: []*model.EnumValue{
			{Name: "Custom", Value: "custom"},
		},
	}
	detectEnums(role, loader)
	if len(role.Enums) != 1 || role.Enums[0].Value != "custom" {
		t.Fatalf("existing Enums must be preserved, got %#v", role.Enums)
	}
}

func TestAttachContractEnums_onlyProjectTypes(t *testing.T) {

	loader, pkgPath := mustEnumPackage(t, `package dto

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type Color string

const (
	ColorRed   Color = "red"
	ColorGreen Color = "green"
)
`)

	roleID := pkgPath + ":Role"
	project := &model.Project{
		Types: map[string]*model.Type{
			roleID: {
				Kind:          model.TypeKindString,
				TypeName:      "Role",
				ImportPkgPath: pkgPath,
				PkgName:       "dto",
			},
		},
	}
	attachContractEnums(project, loader)

	if len(project.Types[roleID].Enums) != 2 {
		t.Fatalf("Role enums = %#v", project.Types[roleID].Enums)
	}
	if _, exists := project.Types[pkgPath+":Color"]; exists {
		t.Fatalf("Color must not appear in project.Types")
	}
}

func TestIsEnumEligibleType(t *testing.T) {

	if !isEnumEligibleType(&model.Type{Kind: model.TypeKindString, TypeName: "Role", ImportPkgPath: "pkg"}) {
		t.Fatal("string named type must be eligible")
	}
	if !isEnumEligibleType(&model.Type{Kind: model.TypeKindAlias, UnderlyingKind: model.TypeKindInt, TypeName: "Status", ImportPkgPath: "pkg"}) {
		t.Fatal("alias with underlying int must be eligible")
	}
	if isEnumEligibleType(&model.Type{Kind: model.TypeKindStruct, TypeName: "User", ImportPkgPath: "pkg"}) {
		t.Fatal("struct must not be eligible")
	}
	if isEnumEligibleType(&model.Type{Kind: model.TypeKindString, TypeName: "string", ImportPkgPath: "pkg"}) {
		t.Fatal("builtin name must not be eligible")
	}
}

func mustEnumPackage(t *testing.T, src string) (loader *AutonomousPackageLoader, pkgPath string) {

	t.Helper()

	const path = "example/contracts/dto"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "dto.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	typeInfo := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	cfg := &types.Config{Error: func(error) {}}
	pkg, err := cfg.Check(path, fset, []*ast.File{file}, typeInfo)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}

	modFile := testModuleFile(t, "example")
	if loader, err = NewAutonomousPackageLoader(modFile); err != nil {
		t.Fatalf("loader: %v", err)
	}
	loader.mu.Lock()
	loader.cache[path] = &PackageInfo{
		PkgPath:     path,
		PackageName: "dto",
		Types:       pkg,
		TypeInfo:    typeInfo,
		Fset:        fset,
	}
	loader.mu.Unlock()
	return loader, path
}
