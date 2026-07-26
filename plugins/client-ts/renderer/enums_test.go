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

func TestRenderExchange_namedStringEnum(t *testing.T) {

	const (
		modulePath = "example"
		dtoPkg     = modulePath + "/contracts/dto"
		roleID     = dtoPkg + ":Role"
		userID     = dtoPkg + ":User"
	)

	project := &model.Project{
		ModulePath: modulePath,
		Types: map[string]*model.Type{
			"string": {Kind: model.TypeKindString},
			roleID: {
				Kind:          model.TypeKindString,
				TypeName:      "Role",
				ImportPkgPath: dtoPkg,
				PkgName:       "dto",
				Enums: []*model.EnumValue{
					{Name: "RoleAdmin", Value: "admin"},
					{Name: "RoleUser", Value: "user"},
				},
			},
			userID: {
				Kind:          model.TypeKindStruct,
				TypeName:      "User",
				ImportPkgPath: dtoPkg,
				PkgName:       "dto",
				StructFields: []*model.StructField{
					{
						Name:    "Role",
						TypeRef: model.TypeRef{TypeID: roleID},
						Tags:    map[string][]string{"json": {"role"}},
					},
				},
			},
		},
		Contracts: []*model.Contract{{
			Name:    "Users",
			PkgPath: modulePath + "/contracts",
			Annotations: tags.DocTags{
				model.TagServerJsonRPC: "",
			},
			Methods: []*model.Method{{
				Name: "Get",
				Args: []*model.Variable{
					{Name: "id", TypeRef: model.TypeRef{TypeID: "string"}},
				},
				Results: []*model.Variable{
					{Name: "user", TypeRef: model.TypeRef{TypeID: userID}},
				},
			}},
		}},
	}

	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderExchangeTypes(project.Contracts[0]); err != nil {
		t.Fatalf("RenderExchangeTypes: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "users-exchange.ts"))
	if err != nil {
		t.Fatalf("read exchange: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, `export type Role =`) || !strings.Contains(source, `"admin"`) || !strings.Contains(source, `"user"`) {
		t.Fatalf("expected Role union type, got:\n%s", source)
	}
	if !strings.Contains(source, "role:dto.Role") && !strings.Contains(source, "role: dto.Role") {
		t.Fatalf("expected field typed as dto.Role, got:\n%s", source)
	}
	if strings.Contains(source, "role:string") || strings.Contains(source, "role: string") {
		t.Fatalf("Role field must not collapse to string:\n%s", source)
	}
}

func TestRenderExchange_namedIntEnum(t *testing.T) {

	const (
		modulePath = "example"
		dtoPkg     = modulePath + "/contracts/dto"
		statusID   = dtoPkg + ":Status"
	)

	project := &model.Project{
		ModulePath: modulePath,
		Types: map[string]*model.Type{
			"int": {Kind: model.TypeKindInt},
			statusID: {
				Kind:          model.TypeKindInt,
				TypeName:      "Status",
				ImportPkgPath: dtoPkg,
				PkgName:       "dto",
				Enums: []*model.EnumValue{
					{Name: "StatusPending", Value: "0"},
					{Name: "StatusActive", Value: "1"},
				},
			},
		},
		Contracts: []*model.Contract{{
			Name:    "Jobs",
			PkgPath: modulePath + "/contracts",
			Annotations: tags.DocTags{
				model.TagServerJsonRPC: "",
			},
			Methods: []*model.Method{{
				Name: "GetStatus",
				Results: []*model.Variable{
					{Name: "status", TypeRef: model.TypeRef{TypeID: statusID}},
				},
			}},
		}},
	}

	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderExchangeTypes(project.Contracts[0]); err != nil {
		t.Fatalf("RenderExchangeTypes: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "jobs-exchange.ts"))
	if err != nil {
		t.Fatalf("read exchange: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "export type Status =") || !strings.Contains(source, "0 | 1") {
		t.Fatalf("expected numeric Status union, got:\n%s", source)
	}
}

func TestTSEnumUnionHelpers(t *testing.T) {

	union := tsEnumUnion([]*model.EnumValue{
		{Value: "a"},
		{Value: "b"},
	}, &model.Type{Kind: model.TypeKindString})
	if union != `"a" | "b"` {
		t.Fatalf("string union = %q", union)
	}

	union = tsEnumUnion([]*model.EnumValue{
		{Value: "0"},
		{Value: "1"},
	}, &model.Type{Kind: model.TypeKindInt})
	if union != "0 | 1" {
		t.Fatalf("int union = %q", union)
	}

	if tsEnumUnion(nil, &model.Type{Kind: model.TypeKindString}) != "string" {
		t.Fatal("empty enums must fallback to string")
	}
}
