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

func TestRenderClientTypes_AnonymousInlineEmbed(t *testing.T) {

	const (
		modulePath = "example.com/svc"
		dtoPkg     = modulePath + "/contracts/dto"
		summaryID  = dtoPkg + ":LayerSummary"
		detailID   = dtoPkg + ":LayerDetail"
	)

	project := &model.Project{
		ModulePath: modulePath,
		Types: map[string]*model.Type{
			summaryID: {
				Kind:          model.TypeKindStruct,
				TypeName:      "LayerSummary",
				ImportPkgPath: dtoPkg,
				PkgName:       "dto",
				StructFields: []*model.StructField{
					{Name: "LayerID", TypeID: "string", Tags: map[string][]string{"json": {"layerId"}}},
					{Name: "Name", TypeID: "string", Tags: map[string][]string{"json": {"name"}}},
				},
			},
			detailID: {
				Kind:          model.TypeKindStruct,
				TypeName:      "LayerDetail",
				ImportPkgPath: dtoPkg,
				PkgName:       "dto",
				StructFields: []*model.StructField{
					{
						TypeID:    summaryID,
						Name:      "LayerSummary",
						Anonymous: true,
						Tags:      map[string][]string{"json": {"", "inline"}},
					},
					{
						Name:   "NeedsRefresh",
						TypeID: "bool",
						Tags:   map[string][]string{"json": {"needsRefresh"}},
					},
				},
			},
		},
	}

	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, modulePath, "client")
	if err := renderer.RenderClientTypes(map[string]bool{summaryID: true, detailID: true}); err != nil {
		t.Fatalf("RenderClientTypes: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "dto", "layerdetail.go"))
	if err != nil {
		t.Fatalf("read layerdetail.go: %v", err)
	}
	source := string(content)

	if strings.Contains(source, "LayerSummary LayerSummary") {
		t.Fatalf("anonymous embed must not duplicate field name:\n%s", source)
	}
	if !strings.Contains(source, "LayerSummary `json:\",inline\"`") {
		t.Fatalf("expected anonymous inline embed:\n%s", source)
	}
}

func TestRenderClientTypes_SkipsExternalAlias(t *testing.T) {

	const (
		modulePath = "example.com/svc"
		dtoPkg     = modulePath + "/contracts/dto"
		contourID  = dtoPkg + ":Contour"
		rawMsgID   = "encoding/json:RawMessage"
		jsontextID = "encoding/json/jsontext:Value"
	)

	project := &model.Project{
		ModulePath: modulePath,
		Types: map[string]*model.Type{
			contourID: {
				Kind:          model.TypeKindStruct,
				TypeName:      "Contour",
				ImportPkgPath: dtoPkg,
				PkgName:       "dto",
				StructFields: []*model.StructField{
					{Name: "Params", TypeID: rawMsgID, Tags: map[string][]string{"json": {"params"}}},
				},
			},
			rawMsgID: {
				Kind:          model.TypeKindAlias,
				TypeName:      "RawMessage",
				ImportPkgPath: "encoding/json",
				PkgName:       "json",
				AliasOf:       jsontextID,
			},
		},
	}

	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, modulePath, "client")
	if err := renderer.RenderClientTypes(map[string]bool{contourID: true, rawMsgID: true}); err != nil {
		t.Fatalf("RenderClientTypes: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "dto", "rawmessage.go")); !os.IsNotExist(err) {
		t.Fatalf("external RawMessage alias must not be materialized in dto, err=%v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "dto", "contour.go"))
	if err != nil {
		t.Fatalf("read contour.go: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, "json.RawMessage") {
		t.Fatalf("expected field typed as encoding/json.RawMessage:\n%s", source)
	}
}

func TestRenderClientTypes_SkipsOpaqueProjectErrorStruct(t *testing.T) {

	const (
		modulePath = "example.com/svc"
		problemErr = modulePath + "/pkg/problem:Error"
	)

	project := &model.Project{
		ModulePath: modulePath,
		Types: map[string]*model.Type{
			problemErr: {
				Kind:          model.TypeKindStruct,
				TypeName:      "Error",
				ImportPkgPath: modulePath + "/pkg/problem",
				PkgName:       "problem",
			},
		},
	}

	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, modulePath, "client")
	collected := renderer.CollectTypeIDsForExchange(&model.Contract{
		Name: "X",
		Methods: []*model.Method{{
			Name: "Do",
			Results: []*model.Variable{
				{Name: "err", TypeID: "error"},
			},
			Errors: []*model.ErrorInfo{{TypeID: problemErr, TypeName: "Error"}},
		}},
	})
	if collected[problemErr] {
		t.Fatalf("opaque error must not be collected: %#v", collected)
	}
	if err := renderer.RenderClientTypes(collected); err != nil {
		t.Fatalf("RenderClientTypes: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "dto", "error.go")); !os.IsNotExist(err) {
		t.Fatalf("error.go must not be generated when only method.Errors references it, err=%v", err)
	}
}
