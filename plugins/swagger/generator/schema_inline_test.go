// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"reflect"
	"testing"

	"tgp/internal/model"
)

func TestStructTypeToSchemaInlinesEmbeddedStruct(t *testing.T) {

	const (
		summaryID = "example/contracts/dto:LayerSummary"
		detailID  = "example/contracts/dto:LayerDetail"
	)

	summaryType := &model.Type{
		Kind:     model.TypeKindStruct,
		TypeName: "LayerSummary",
		StructFields: []*model.StructField{
			{Name: "LayerID", TypeID: "string", Tags: map[string][]string{"json": {"layerId"}}},
			{Name: "Name", TypeID: "string", Tags: map[string][]string{"json": {"name"}}},
		},
	}
	detailType := &model.Type{
		Kind:     model.TypeKindStruct,
		TypeName: "LayerDetail",
		StructFields: []*model.StructField{
			{
				TypeID:    summaryID,
				Name:      "LayerSummary",
				Anonymous: true,
				Tags:      map[string][]string{"json": {"", "inline"}},
			},
			{Name: "NeedsRefresh", TypeID: "bool", Tags: map[string][]string{"json": {"needsRefresh"}}},
		},
	}

	gen := newGenerator(&model.Project{
		Types: map[string]*model.Type{
			"string":  {Kind: model.TypeKindString},
			summaryID: summaryType,
			detailID:  detailType,
		},
	})

	if schemaRef := gen.structTypeToSchema(detailType, nil); schemaRef == nil || schemaRef.Ref == "" {
		t.Fatal("expected schema ref")
	}

	schema := gen.schemas["dto.LayerDetail"]
	if _, ok := schema.Properties["layerId"]; !ok {
		t.Fatalf("expected layerId property, got %#v", schema.Properties)
	}
	if _, ok := schema.Properties["name"]; !ok {
		t.Fatalf("expected name property, got %#v", schema.Properties)
	}
	if _, ok := schema.Properties["needsRefresh"]; !ok {
		t.Fatalf("expected needsRefresh property, got %#v", schema.Properties)
	}
	if _, ok := schema.Properties["layerSummary"]; ok {
		t.Fatalf("embedded field must be flattened, got %#v", schema.Properties)
	}

	wantRequired := []string{"layerId", "name", "needsRefresh"}
	if !reflect.DeepEqual(schema.Required, wantRequired) {
		t.Fatalf("required = %v, want %v", schema.Required, wantRequired)
	}
}
