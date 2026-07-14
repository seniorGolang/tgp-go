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

func TestRenderClientTypes_NamedSlice(t *testing.T) {

	const (
		modulePath = "example.com/svc"
		dtoPkg     = modulePath + "/contracts/dto"
		stateID    = dtoPkg + ":State"
		detailsID  = dtoPkg + ":StateDetails"
	)

	project := &model.Project{
		ModulePath: modulePath,
		Types: map[string]*model.Type{
			stateID: {
				Kind:          model.TypeKindStruct,
				TypeName:      "State",
				ImportPkgPath: dtoPkg,
				PkgName:       "dto",
				StructFields: []*model.StructField{
					{Name: "scope", TypeRef: model.TypeRef{TypeID: "string"}},
				},
			},
			detailsID: {
				Kind:          model.TypeKindArray,
				TypeName:      "StateDetails",
				ImportPkgPath: dtoPkg,
				PkgName:       "dto",
				IsSlice:       true,
				ArrayOfID:     stateID,
			},
		},
	}

	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, modulePath, "client")
	if err := renderer.RenderClientTypes(map[string]bool{stateID: true, detailsID: true}); err != nil {
		t.Fatalf("RenderClientTypes: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "dto", "statedetails.go"))
	if err != nil {
		t.Fatalf("read statedetails.go: %v", err)
	}
	source := string(content)

	if strings.Contains(source, "type StateDetails array") {
		t.Fatalf("named slice must not use Kind as type name:\n%s", source)
	}
	if !strings.Contains(source, "type StateDetails []State") {
		t.Fatalf("expected type StateDetails []State, got:\n%s", source)
	}
}

func TestRenderClientTypes_NamedFixedArray(t *testing.T) {

	const (
		modulePath = "example.com/svc"
		dtoPkg     = modulePath + "/contracts/dto"
		hashID     = dtoPkg + ":Hash"
	)

	project := &model.Project{
		ModulePath: modulePath,
		Types: map[string]*model.Type{
			hashID: {
				Kind:          model.TypeKindArray,
				TypeName:      "Hash",
				ImportPkgPath: dtoPkg,
				PkgName:       "dto",
				ArrayLen:      16,
				ArrayOfID:     "byte",
			},
		},
	}

	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, modulePath, "client")
	if err := renderer.RenderClientTypes(map[string]bool{hashID: true}); err != nil {
		t.Fatalf("RenderClientTypes: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "dto", "hash.go"))
	if err != nil {
		t.Fatalf("read hash.go: %v", err)
	}
	source := string(content)

	if strings.Contains(source, "type Hash array") {
		t.Fatalf("named array must not use Kind as type name:\n%s", source)
	}
	if !strings.Contains(source, "type Hash [16]byte") {
		t.Fatalf("expected type Hash [16]byte, got:\n%s", source)
	}
}

func TestRenderClientTypes_NamedBuiltin(t *testing.T) {

	const (
		modulePath = "example.com/svc"
		dtoPkg     = modulePath + "/contracts/dto"
		userIDType = dtoPkg + ":UserID"
	)

	project := &model.Project{
		ModulePath: modulePath,
		Types: map[string]*model.Type{
			userIDType: {
				Kind:          model.TypeKindUint64,
				TypeName:      "UserID",
				ImportPkgPath: dtoPkg,
				PkgName:       "dto",
			},
		},
	}

	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, modulePath, "client")
	if err := renderer.RenderClientTypes(map[string]bool{userIDType: true}); err != nil {
		t.Fatalf("RenderClientTypes: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "dto", "userid.go"))
	if err != nil {
		t.Fatalf("read userid.go: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "type UserID uint64") {
		t.Fatalf("expected type UserID uint64, got:\n%s", source)
	}
}

func TestRenderClientTypes_NamedSlicePointerElem(t *testing.T) {

	const (
		modulePath = "example.com/svc"
		dtoPkg     = modulePath + "/contracts/dto"
		stateID    = dtoPkg + ":State"
		listID     = dtoPkg + ":StateList"
	)

	project := &model.Project{
		ModulePath: modulePath,
		Types: map[string]*model.Type{
			stateID: {
				Kind:          model.TypeKindStruct,
				TypeName:      "State",
				ImportPkgPath: dtoPkg,
				PkgName:       "dto",
			},
			listID: {
				Kind:            model.TypeKindArray,
				TypeName:        "StateList",
				ImportPkgPath:   dtoPkg,
				PkgName:         "dto",
				IsSlice:         true,
				ArrayOfID:       stateID,
				ElementPointers: 1,
			},
		},
	}

	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, modulePath, "client")
	if err := renderer.RenderClientTypes(map[string]bool{stateID: true, listID: true}); err != nil {
		t.Fatalf("RenderClientTypes: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "dto", "statelist.go"))
	if err != nil {
		t.Fatalf("read statelist.go: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "type StateList []*State") {
		t.Fatalf("expected type StateList []*State, got:\n%s", source)
	}
}

func TestRenderClientTypes_UnsupportedNamedKindSkipped(t *testing.T) {

	const (
		modulePath = "example.com/svc"
		dtoPkg     = modulePath + "/contracts/dto"
		chanID     = dtoPkg + ":EventChan"
	)

	project := &model.Project{
		ModulePath: modulePath,
		Types: map[string]*model.Type{
			chanID: {
				Kind:          model.TypeKindChan,
				TypeName:      "EventChan",
				ImportPkgPath: dtoPkg,
				PkgName:       "dto",
				ChanOfID:      "string",
			},
		},
	}

	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, modulePath, "client")
	if err := renderer.RenderClientTypes(map[string]bool{chanID: true}); err != nil {
		t.Fatalf("RenderClientTypes: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "dto", "eventchan.go")); !os.IsNotExist(err) {
		t.Fatalf("unsupported named kind must not generate dto file, err=%v", err)
	}
}
