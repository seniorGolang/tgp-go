// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func TestCollectTypeIDsForExchange_WireTypesOnly(t *testing.T) {

	const (
		modulePath = "example.com/svc"
		dtoPkg     = modulePath + "/contracts/dto"
		contourID  = dtoPkg + ":Contour"
		problemErr = modulePath + "/pkg/problem:Error"
		tokenErr   = modulePath + "/pkg/errs:TokenEndpointError"
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
					{Name: "Name", TypeID: "string", Tags: map[string][]string{"json": {"name"}}},
				},
			},
			problemErr: {
				Kind:          model.TypeKindStruct,
				TypeName:      "Error",
				ImportPkgPath: modulePath + "/pkg/problem",
				PkgName:       "problem",
			},
			tokenErr: {
				Kind:          model.TypeKindStruct,
				TypeName:      "TokenEndpointError",
				ImportPkgPath: modulePath + "/pkg/errs",
				PkgName:       "errs",
				StructFields: []*model.StructField{
					{Name: "ErrCode", TypeID: "string", Tags: map[string][]string{"json": {"error"}}},
				},
			},
		},
	}

	contract := &model.Contract{
		Name: "Catalog",
		Methods: []*model.Method{
			{
				Name: "Show",
				Args: []*model.Variable{
					{Name: "ctx", TypeID: "context:Context"},
					{Name: "name", TypeID: "string"},
				},
				Results: []*model.Variable{
					{Name: "contour", TypeID: contourID},
					{Name: "err", TypeID: "error"},
				},
				Errors: []*model.ErrorInfo{
					{TypeID: problemErr, TypeName: "Error", PkgPath: modulePath + "/pkg/problem"},
					{TypeID: tokenErr, TypeName: "TokenEndpointError", PkgPath: modulePath + "/pkg/errs"},
				},
				Annotations: tags.DocTags{
					"defaultError": modulePath + "/pkg/problem:Error",
					"404":          modulePath + "/pkg/problem:Error",
				},
			},
		},
	}

	renderer := NewClientRenderer(project, t.TempDir(), modulePath, "client")
	collected := renderer.CollectTypeIDsForExchange(contract)

	if !collected[contourID] {
		t.Fatalf("expected wire type %s in collected set: %#v", contourID, collected)
	}
	if collected[problemErr] {
		t.Fatalf("method.Errors type must not enter dto collection: %#v", collected)
	}
	if collected[tokenErr] {
		t.Fatalf("method.Errors type must not enter dto collection: %#v", collected)
	}
	if collected["error"] {
		t.Fatalf("builtin error must not enter dto collection: %#v", collected)
	}
	if collected["context:Context"] {
		t.Fatalf("context.Context must not enter dto collection: %#v", collected)
	}
}

func TestCollectTypeIDsForExchange_DoesNotRequireAliasTargetInProject(t *testing.T) {

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

	contract := &model.Contract{
		Name: "Contour",
		Methods: []*model.Method{
			{
				Name: "Get",
				Results: []*model.Variable{
					{Name: "contour", TypeID: contourID},
					{Name: "err", TypeID: "error"},
				},
			},
		},
	}

	renderer := NewClientRenderer(project, t.TempDir(), modulePath, "client")
	collected := renderer.CollectTypeIDsForExchange(contract)
	if !collected[contourID] {
		t.Fatalf("expected contour in collected set")
	}
}
