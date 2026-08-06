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
	)

	project := &model.Project{
		ModulePath: modulePath,
		Types: map[string]*model.Type{
			contourID: {
				Kind:          model.TypeKindStruct,
				TypeName:      "Contour",
				ImportPkgPath: dtoPkg,
				PkgName:       "dto",
			},
			problemErr: {
				Kind:          model.TypeKindStruct,
				TypeName:      "Error",
				ImportPkgPath: modulePath + "/pkg/problem",
				PkgName:       "problem",
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
				},
				Annotations: tags.DocTags{
					"defaultError": modulePath + "/pkg/problem:Error",
				},
			},
		},
	}

	renderer := NewClientRenderer(project, t.TempDir(), false, "", false)
	collected := renderer.CollectTypeIDsForExchange(contract)

	if !collected[contourID] {
		t.Fatalf("expected wire type %s in collected set: %#v", contourID, collected)
	}
	if collected[problemErr] {
		t.Fatalf("method.Errors type must not enter dto collection: %#v", collected)
	}
	if collected["error"] {
		t.Fatalf("builtin error must not enter dto collection: %#v", collected)
	}
}
