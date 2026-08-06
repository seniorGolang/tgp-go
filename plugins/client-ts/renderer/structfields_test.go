// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
	"tgp/plugins/client-ts/tsg"
)

func TestCollectStructReadmeFieldsFlattensInlineEmbed(t *testing.T) {

	const summaryID = "example/dto:LayerSummary"

	project := &model.Project{
		Types: map[string]*model.Type{
			summaryID: {
				Kind: model.TypeKindStruct,
				StructFields: []*model.StructField{
					{Name: "LayerID", TypeID: "string"},
					{Name: "Name", TypeID: "string"},
				},
			},
		},
	}

	structType := &model.Type{
		Kind: model.TypeKindStruct,
		StructFields: []*model.StructField{
			{
				TypeID:    summaryID,
				Name:      "LayerSummary",
				Anonymous: true,
				Tags:      map[string][]string{"json": {"", "inline"}},
			},
			{Name: "NeedsRefresh", TypeID: "bool"},
		},
	}

	renderer := NewClientRenderer(project, t.TempDir(), false, "", false)
	fields := renderer.collectStructReadmeFields(structType)
	if len(fields) != 3 {
		t.Fatalf("expected 3 flattened fields, got %d", len(fields))
	}
}

func TestAppendStructFieldTSInterfaceFlattensInlineEmbed(t *testing.T) {

	const summaryID = "example/dto:LayerSummary"

	project := &model.Project{
		Types: map[string]*model.Type{
			"string": {Kind: model.TypeKindString},
			"bool":   {Kind: model.TypeKindBool},
			summaryID: {
				Kind:          model.TypeKindStruct,
				TypeName:      "LayerSummary",
				ImportPkgPath: "example/dto",
				StructFields: []*model.StructField{
					{Name: "LayerID", TypeID: "string", Tags: map[string][]string{"json": {"layerId"}}},
					{Name: "Name", TypeID: "string", Tags: map[string][]string{"json": {"name"}}},
				},
			},
		},
	}

	detailField := &model.StructField{
		TypeID:    summaryID,
		Name:      "LayerSummary",
		Anonymous: true,
		Tags:      map[string][]string{"json": {"", "inline"}},
	}

	renderer := NewClientRenderer(project, t.TempDir(), false, "", false)
	renderer.knownTypes = make(map[string]int)
	renderer.typeDefTS = make(map[string]typeDefTS)

	stmt := tsg.NewStatement()
	stmt.Export().Interface("ResponseExample", func(grp *tsg.Group) {
		opts := tsInterfaceFieldOptions{seen: make(map[string]bool), isArgument: false}
		renderer.appendStructFieldTSInterface(grp, detailField, "example/dto", opts)
		field := &model.StructField{
			Name:   "NeedsRefresh",
			TypeID: "bool",
			Tags:   map[string][]string{"json": {"needsRefresh"}},
		}
		renderer.appendStructFieldTSInterface(grp, field, "example/dto", opts)
	})

	source := stmt.String()
	if !strings.Contains(source, "layerId") || !strings.Contains(source, "name") {
		t.Fatalf("expected flattened inline fields in TS interface, got %s", source)
	}
	if strings.Contains(source, "LayerSummary") {
		t.Fatalf("embedded type name must not appear as property, got %s", source)
	}
}

func TestJsonNameInlineVariants(t *testing.T) {

	renderer := NewClientRenderer(&model.Project{}, t.TempDir(), false, "", false)

	tests := []struct {
		name       string
		field      *model.StructField
		wantName   string
		wantInline bool
	}{
		{
			name: "comma inline",
			field: &model.StructField{
				Name: "LayerSummary",
				Tags: map[string][]string{"json": {"", "inline"}},
			},
			wantInline: true,
		},
		{
			name: "named inline",
			field: &model.StructField{
				Name: "Body",
				Tags: map[string][]string{"json": {"body", "inline"}},
			},
			wantName:   "body",
			wantInline: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotInline := renderer.jsonName(tt.field)
			if gotName != tt.wantName || gotInline != tt.wantInline {
				t.Fatalf("jsonName() = (%q, %v), want (%q, %v)", gotName, gotInline, tt.wantName, tt.wantInline)
			}
		})
	}

	if _, inline := tags.StructFieldJSONName(map[string][]string{"json": {"inline"}}, "X"); !inline {
		t.Fatal("single inline token must be detected")
	}
}
