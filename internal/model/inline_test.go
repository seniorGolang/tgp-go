// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import "testing"

func TestStructFieldEmbedded(t *testing.T) {

	const summaryID = "example/dto:LayerSummary"

	project := &Project{
		Types: map[string]*Type{
			summaryID: {Kind: TypeKindStruct, TypeName: "LayerSummary"},
			"string":  {Kind: TypeKindString},
		},
	}

	tests := []struct {
		name  string
		field *StructField
		want  bool
	}{
		{
			name: "anonymous inline struct",
			field: &StructField{
				TypeID:    summaryID,
				Name:      "LayerSummary",
				Anonymous: true,
				Tags:      map[string][]string{"json": {"", "inline"}},
			},
			want: true,
		},
		{
			name: "inline primitive",
			field: &StructField{
				TypeID: "string",
				Tags:   map[string][]string{"json": {"", "inline"}},
			},
			want: false,
		},
		{
			name: "anonymous without inline",
			field: &StructField{
				TypeID:    summaryID,
				Anonymous: true,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StructFieldEmbedded(project, tt.field); got != tt.want {
				t.Fatalf("StructFieldEmbedded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStructFieldIsAnonymousEmbed(t *testing.T) {

	if !StructFieldIsAnonymousEmbed(&StructField{Anonymous: true}) {
		t.Fatal("expected anonymous embed")
	}
	if StructFieldIsAnonymousEmbed(&StructField{Name: "LayerSummary"}) {
		t.Fatal("expected non-anonymous embed")
	}
	if StructFieldIsAnonymousEmbed(nil) {
		t.Fatal("nil field must not be anonymous embed")
	}
}
