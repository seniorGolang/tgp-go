// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal/model"
)

func TestRenderClientTypes_StringEnum(t *testing.T) {

	const (
		modulePath = "example.com/svc"
		dtoPkg     = modulePath + "/contracts/dto"
		roleID     = dtoPkg + ":Role"
	)

	project := &model.Project{
		ModulePath: modulePath,
		Types: map[string]*model.Type{
			roleID: {
				Kind:          model.TypeKindString,
				TypeName:      "Role",
				ImportPkgPath: dtoPkg,
				PkgName:       "dto",
				Enums: []*model.EnumValue{
					{Name: "RoleAdmin", Value: "admin"},
					{Name: "RoleUser", Value: "user"},
					{Name: "roleHidden", Value: "hidden"},
				},
			},
		},
	}

	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, modulePath, "client")
	if err := renderer.RenderClientTypes(map[string]bool{roleID: true}); err != nil {
		t.Fatalf("RenderClientTypes: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "dto", "role.go"))
	if err != nil {
		t.Fatalf("read role.go: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "type Role string") {
		t.Fatalf("expected defined type Role string, got:\n%s", source)
	}
	if strings.Contains(source, "type Role =") {
		t.Fatalf("enum must not be type alias:\n%s", source)
	}
	if !strings.Contains(source, "RoleAdmin") || !strings.Contains(source, `"admin"`) {
		t.Fatalf("expected RoleAdmin const, got:\n%s", source)
	}
	if !strings.Contains(source, "RoleUser") || !strings.Contains(source, `"user"`) {
		t.Fatalf("expected RoleUser const, got:\n%s", source)
	}
	if strings.Contains(source, "roleHidden") {
		t.Fatalf("unexported const must not be generated:\n%s", source)
	}
}

func TestRenderClientTypes_IntEnum(t *testing.T) {

	const (
		modulePath = "example.com/svc"
		dtoPkg     = modulePath + "/contracts/dto"
		statusID   = dtoPkg + ":Status"
	)

	project := &model.Project{
		ModulePath: modulePath,
		Types: map[string]*model.Type{
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
	}

	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, modulePath, "client")
	if err := renderer.RenderClientTypes(map[string]bool{statusID: true}); err != nil {
		t.Fatalf("RenderClientTypes: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "dto", "status.go"))
	if err != nil {
		t.Fatalf("read status.go: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "type Status int") {
		t.Fatalf("expected type Status int, got:\n%s", source)
	}
	if !strings.Contains(source, "StatusPending") || !strings.Contains(source, "StatusActive") {
		t.Fatalf("expected status const names, got:\n%s", source)
	}
	if !strings.Contains(source, "= 0") || !strings.Contains(source, "= 1") {
		t.Fatalf("expected int const values, got:\n%s", source)
	}
}

func TestRenderClientTypes_WithoutEnumsKeepsAlias(t *testing.T) {

	const (
		modulePath = "example.com/svc"
		dtoPkg     = modulePath + "/contracts/dto"
		idType     = dtoPkg + ":UserID"
	)

	project := &model.Project{
		ModulePath: modulePath,
		Types: map[string]*model.Type{
			idType: {
				Kind:             model.TypeKindAlias,
				TypeName:         "UserID",
				ImportPkgPath:    dtoPkg,
				PkgName:          "dto",
				UnderlyingKind:   model.TypeKindString,
				UnderlyingTypeID: "string",
			},
		},
	}

	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, modulePath, "client")
	if err := renderer.RenderClientTypes(map[string]bool{idType: true}); err != nil {
		t.Fatalf("RenderClientTypes: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "dto", "userid.go"))
	if err != nil {
		t.Fatalf("read userid.go: %v", err)
	}
	source := string(content)
	if strings.Contains(source, "const (") {
		t.Fatalf("type without Enums must not emit const block:\n%s", source)
	}
}
