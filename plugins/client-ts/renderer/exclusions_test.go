// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

const (
	testUUIDPkg    = "github.com/google/uuid"
	testUIDPkg     = "example/libs/uid"
	testDtoPkg     = "example/dto"
	testFiltersPkg = "example/libs/filters"
)

func testProjectWithDtoID() (project *model.Project) {

	uuidTypeID := testUUIDPkg + ":UUID"
	uidTypeID := testUIDPkg + ":ID"
	dtoIDTypeID := testDtoPkg + ":ID"
	attrFilterTypeID := testFiltersPkg + ":AttrFilter"

	return &model.Project{
		ModulePath: "example",
		Types: map[string]*model.Type{
			uuidTypeID: {
				Kind:                 model.TypeKindAlias,
				TypeName:             "UUID",
				ImportPkgPath:        testUUIDPkg,
				PkgName:              "uuid",
				AliasOf:              "builtin:string",
				ImplementsInterfaces: []string{"encoding/json:Marshaler", "encoding/json:Unmarshaler", "encoding/text:Marshaler", "encoding/text:Unmarshaler"},
			},
			dtoIDTypeID: {
				Kind:          model.TypeKindAlias,
				TypeName:      "ID",
				ImportPkgPath: testDtoPkg,
				PkgName:       "dto",
				AliasOf:       uuidTypeID,
			},
			attrFilterTypeID: {
				Kind:          model.TypeKindStruct,
				TypeName:      "AttrFilter",
				ImportPkgPath: testFiltersPkg,
				PkgName:       "filters",
				StructFields: []*model.StructField{
					{
						Name:    "scope",
						TypeRef: model.TypeRef{TypeID: "string"},
						Tags:    map[string][]string{"json": {"scope"}},
					},
					{
						Name: "objects",
						TypeRef: model.TypeRef{
							TypeID:          uidTypeID,
							IsSlice:         true,
							ElementPointers: 1,
						},
						Tags: map[string][]string{"json": {"objects"}},
					},
					{
						Name: "subjects",
						TypeRef: model.TypeRef{
							TypeID:          uidTypeID,
							IsSlice:         true,
							ElementPointers: 1,
						},
						Tags: map[string][]string{"json": {"subjects"}},
					},
				},
			},
		},
	}
}

func TestIsExplicitlyExcludedType_uuidAliasChain(t *testing.T) {

	project := testProjectWithDtoID()
	renderer := NewClientRenderer(project, t.TempDir(), false, "", true)
	dtoID := project.Types[testDtoPkg+":ID"]
	if !renderer.isExplicitlyExcludedType(dtoID) {
		t.Fatal("dto.ID alias to uuid must be excluded from opaque marshaler path")
	}
}

func TestWalkTypeRef_dtoIDMapsToNamedType(t *testing.T) {

	project := testProjectWithDtoID()
	renderer := NewClientRenderer(project, t.TempDir(), false, "", true)

	schema := renderer.walkTypeRefWithVisited(
		"id",
		testDtoPkg,
		&model.TypeRef{TypeID: testDtoPkg + ":ID"},
		nil,
		make(map[string]bool),
		false,
	)

	got := schema.typeLink()
	want := "dto.ID"
	if got != want {
		t.Fatalf("typeLink() = %q, want %q", got, want)
	}
	if schema.typeName != "string" {
		t.Fatalf("typeName = %q, want %q", schema.typeName, "string")
	}
}

func TestWalkTypeRef_dtoIDMapsToNamedTypeWithoutBaseInProject(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types: map[string]*model.Type{
			testDtoPkg + ":ID": {
				Kind:          model.TypeKindAlias,
				TypeName:      "ID",
				ImportPkgPath: testDtoPkg,
				PkgName:       "dto",
				AliasOf:       testUUIDPkg + ":UUID",
			},
		},
	}
	renderer := NewClientRenderer(project, t.TempDir(), false, "", true)

	schema := renderer.walkTypeRefWithVisited(
		"id",
		testDtoPkg,
		&model.TypeRef{TypeID: testDtoPkg + ":ID"},
		nil,
		make(map[string]bool),
		false,
	)

	got := schema.typeLink()
	want := "dto.ID"
	if got != want {
		t.Fatalf("typeLink() = %q, want %q", got, want)
	}
	if schema.typeName != "string" {
		t.Fatalf("typeName = %q, want %q", schema.typeName, "string")
	}
}

func TestRenderExchange_declaresDtoIDWithoutUUIDInProject(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types: map[string]*model.Type{
			testDtoPkg + ":ID": {
				Kind:          model.TypeKindAlias,
				TypeName:      "ID",
				ImportPkgPath: testDtoPkg,
				PkgName:       "dto",
				AliasOf:       testUUIDPkg + ":UUID",
			},
			testDtoPkg + ":Entity": {
				Kind:          model.TypeKindStruct,
				TypeName:      "Entity",
				ImportPkgPath: testDtoPkg,
				PkgName:       "dto",
				StructFields: []*model.StructField{
					{
						Name:    "ID",
						TypeRef: model.TypeRef{TypeID: testDtoPkg + ":ID"},
						Tags:    map[string][]string{"json": {"id"}},
					},
				},
			},
		},
	}
	project.Contracts = []*model.Contract{{
		Name:    "Shapes",
		PkgPath: "example/contracts",
		Annotations: tags.DocTags{
			model.TagServerJsonRPC: "",
		},
		Methods: []*model.Method{{
			Name: "EchoEntity",
			Args: []*model.Variable{
				{Name: "entity", TypeRef: model.TypeRef{TypeID: testDtoPkg + ":Entity"}},
			},
			Results: []*model.Variable{
				{Name: "out", TypeRef: model.TypeRef{TypeID: testDtoPkg + ":Entity"}},
			},
		}},
	}}

	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderExchangeTypes(project.Contracts[0]); err != nil {
		t.Fatalf("RenderExchangeTypes: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "shapes-exchange.ts"))
	if err != nil {
		t.Fatalf("read exchange: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "export type ID =") {
		t.Fatalf("expected export type ID in dto namespace, got:\n%s", source)
	}
	if !strings.Contains(source, "id:dto.ID") && !strings.Contains(source, "id: dto.ID") {
		t.Fatalf("expected id: dto.ID in Entity struct, got:\n%s", source)
	}
}

func TestWalkTypeRef_uuidWithMarshalerMapsToString(t *testing.T) {

	project := testProjectWithDtoID()
	renderer := NewClientRenderer(project, t.TempDir(), false, "", true)

	schema := renderer.walkTypeRefWithVisited(
		"id",
		testUUIDPkg,
		&model.TypeRef{TypeID: testUUIDPkg + ":UUID"},
		nil,
		make(map[string]bool),
		false,
	)

	got := schema.typeLink()
	want := "string"
	if got != want {
		t.Fatalf("typeLink() = %q, want %q", got, want)
	}
}

func TestWalkTypeRef_externalUIDSliceMapsToString(t *testing.T) {

	project := testProjectWithDtoID()
	renderer := NewClientRenderer(project, t.TempDir(), false, "", true)

	uidTypeID := testUIDPkg + ":ID"
	schema := renderer.walkTypeRefWithVisited(
		"objects",
		testFiltersPkg,
		&model.TypeRef{TypeID: uidTypeID, IsSlice: true, ElementPointers: 1},
		nil,
		make(map[string]bool),
		true,
	)

	got := schema.def()
	want := "string[]"
	if got != want {
		t.Fatalf("def() = %q, want %q", got, want)
	}
}

func TestWalkTypeRef_dtoAliasToStructNamespace(t *testing.T) {

	project := testProjectWithDtoID()
	project.Types[testDtoPkg+":AttrFilter"] = &model.Type{
		Kind:          model.TypeKindAlias,
		TypeName:      "AttrFilter",
		ImportPkgPath: testDtoPkg,
		PkgName:       "dto",
		AliasOf:       testFiltersPkg + ":AttrFilter",
	}
	renderer := NewClientRenderer(project, t.TempDir(), false, "", true)

	schema := renderer.walkTypeRefWithVisited(
		"filter",
		testDtoPkg,
		&model.TypeRef{TypeID: testDtoPkg + ":AttrFilter"},
		nil,
		make(map[string]bool),
		false,
	)

	got := schema.typeLink()
	want := "dto.AttrFilter"
	if got != want {
		t.Fatalf("typeLink() = %q, want %q", got, want)
	}
	if schema.typeName != "filters.AttrFilter" {
		t.Fatalf("typeName = %q, want %q", schema.typeName, "filters.AttrFilter")
	}
}

func TestWalkTypeRef_opaqueMarshalerMapsToAny(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types: map[string]*model.Type{
			"example/custom:Payload": {
				Kind:                 model.TypeKindStruct,
				TypeName:             "Payload",
				ImportPkgPath:        "example/custom",
				PkgName:              "custom",
				ImplementsInterfaces: []string{"encoding/json:Marshaler", "encoding/json:Unmarshaler"},
				StructFields: []*model.StructField{
					{Name: "value", TypeRef: model.TypeRef{TypeID: "string"}},
				},
			},
		},
	}
	renderer := NewClientRenderer(project, t.TempDir(), false, "", true)

	schema := renderer.walkTypeRefWithVisited(
		"payload",
		"example/custom",
		&model.TypeRef{TypeID: "example/custom:Payload"},
		nil,
		make(map[string]bool),
		true,
	)

	got := schema.typeLink()
	want := "any"
	if got != want {
		t.Fatalf("typeLink() = %q, want %q", got, want)
	}
}

func TestWalkTypeRef_emptySliceWithoutTypeIDMapsToAny(t *testing.T) {

	project := testProjectWithDtoID()
	renderer := NewClientRenderer(project, t.TempDir(), false, "", true)

	schema := renderer.walkTypeRefWithVisited(
		"objects",
		testFiltersPkg,
		&model.TypeRef{IsSlice: true},
		nil,
		make(map[string]bool),
		true,
	)

	got := schema.def()
	want := "any[]"
	if got != want {
		t.Fatalf("def() = %q, want %q", got, want)
	}
}

func TestRenderExchange_declaresDtoIDAndUsesIt(t *testing.T) {

	project := testProjectWithDtoID()
	project.Types[testDtoPkg+":Attr"] = &model.Type{
		Kind:          model.TypeKindStruct,
		TypeName:      "Attr",
		ImportPkgPath: testDtoPkg,
		PkgName:       "dto",
		StructFields: []*model.StructField{
			{
				Name:    "id",
				TypeRef: model.TypeRef{TypeID: testDtoPkg + ":ID"},
				Tags:    map[string][]string{"json": {"id"}},
			},
		},
	}
	project.Contracts = []*model.Contract{{
		Name:    "Attr",
		PkgPath: "example/contracts",
		Annotations: tags.DocTags{
			model.TagServerJsonRPC: "",
		},
		Methods: []*model.Method{{
			Name: "List",
			Args: []*model.Variable{
				{Name: "filter", TypeRef: model.TypeRef{TypeID: testFiltersPkg + ":AttrFilter"}},
				{Name: "limit", TypeRef: model.TypeRef{TypeID: "int"}},
				{Name: "offset", TypeRef: model.TypeRef{TypeID: "int"}},
			},
			Results: []*model.Variable{
				{Name: "attrs", TypeRef: model.TypeRef{TypeID: testDtoPkg + ":Attr", IsSlice: true, NumberOfPointers: 1}},
			},
		}},
	}}

	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderExchangeTypes(project.Contracts[0]); err != nil {
		t.Fatalf("RenderExchangeTypes: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "attr-exchange.ts"))
	if err != nil {
		t.Fatalf("read exchange: %v", err)
	}
	source := string(content)

	if strings.Contains(source, "objects?: any[]") {
		t.Fatalf("must not contain objects?: any[], got:\n%s", source)
	}
	if !strings.Contains(source, "export type ID =") {
		t.Fatalf("expected export type ID = string in dto namespace, got:\n%s", source)
	}
	if !strings.Contains(source, "objects?:string[]") && !strings.Contains(source, "objects?: string[]") {
		t.Fatalf("expected objects?: string[] for external uid.ID, got:\n%s", source)
	}
	if !strings.Contains(source, "id:dto.ID") && !strings.Contains(source, "id: dto.ID") {
		t.Fatalf("expected id: dto.ID in Attr struct, got:\n%s", source)
	}
}

func TestRenderJsonRPCClient_skipsNamespaceTypeDuplicates(t *testing.T) {

	project := testProjectWithDtoID()
	project.Types[testDtoPkg+":AttrFilter"] = &model.Type{
		Kind:          model.TypeKindAlias,
		TypeName:      "AttrFilter",
		ImportPkgPath: testDtoPkg,
		PkgName:       "dto",
		AliasOf:       testFiltersPkg + ":AttrFilter",
	}
	project.Contracts = []*model.Contract{{
		Name:    "Attr",
		PkgPath: "example/contracts",
		Annotations: tags.DocTags{
			model.TagServerJsonRPC: "",
		},
		Methods: []*model.Method{{
			Name: "List",
			Args: []*model.Variable{
				{Name: "filter", TypeRef: model.TypeRef{TypeID: testDtoPkg + ":AttrFilter"}},
				{Name: "limit", TypeRef: model.TypeRef{TypeID: "int"}},
				{Name: "offset", TypeRef: model.TypeRef{TypeID: "int"}},
			},
			Results: []*model.Variable{
				{Name: "attrs", TypeRef: model.TypeRef{TypeID: testDtoPkg + ":Attr", IsSlice: true, NumberOfPointers: 1}},
			},
		}},
	}}

	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderExchangeTypes(project.Contracts[0]); err != nil {
		t.Fatalf("RenderExchangeTypes: %v", err)
	}
	if err := renderer.RenderJsonRPCClientClass(project.Contracts[0]); err != nil {
		t.Fatalf("RenderJsonRPCClientClass: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "attr.ts"))
	if err != nil {
		t.Fatalf("read client: %v", err)
	}
	source := string(content)

	if strings.Contains(source, "export interface AttrFilter") {
		t.Fatalf("must not duplicate AttrFilter interface in client file, got:\n%s", source)
	}
}
