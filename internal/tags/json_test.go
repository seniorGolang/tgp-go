// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package tags

import "testing"

func TestStructFieldJSONTag(t *testing.T) {

	tests := []struct {
		name string
		tags map[string][]string
		want string
	}{
		{
			name: "inline",
			tags: map[string][]string{"json": {"", "inline"}},
			want: ",inline",
		},
		{
			name: "named omitempty",
			tags: map[string][]string{"json": {"layerId", "omitempty"}},
			want: "layerId,omitempty",
		},
		{
			name: "empty",
			tags: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StructFieldJSONTag(tt.tags); got != tt.want {
				t.Fatalf("StructFieldJSONTag() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStructFieldHasJSONInline(t *testing.T) {

	tests := []struct {
		name string
		tags map[string][]string
		want bool
	}{
		{name: "comma inline", tags: map[string][]string{"json": {"", "inline"}}, want: true},
		{name: "named inline", tags: map[string][]string{"json": {"body", "inline"}}, want: true},
		{name: "inline only token", tags: map[string][]string{"json": {"inline"}}, want: true},
		{name: "plain name", tags: map[string][]string{"json": {"layerId"}}, want: false},
		{name: "dash", tags: map[string][]string{"json": {"-"}}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StructFieldHasJSONInline(tt.tags); got != tt.want {
				t.Fatalf("StructFieldHasJSONInline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStructFieldJSONName(t *testing.T) {

	tests := []struct {
		name       string
		tags       map[string][]string
		fieldName  string
		wantName   string
		wantInline bool
	}{
		{
			name:       "inline empty name",
			tags:       map[string][]string{"json": {"", "inline"}},
			fieldName:  "LayerSummary",
			wantName:   "",
			wantInline: true,
		},
		{
			name:       "explicit name",
			tags:       map[string][]string{"json": {"layerId", "omitempty"}},
			fieldName:  "LayerID",
			wantName:   "layerId",
			wantInline: false,
		},
		{
			name:       "dash",
			tags:       map[string][]string{"json": {"-"}},
			fieldName:  "Hidden",
			wantName:   "-",
			wantInline: false,
		},
		{
			name:       "unexported without tag",
			tags:       nil,
			fieldName:  "hidden",
			wantName:   "",
			wantInline: false,
		},
		{
			name:       "exported without tag",
			tags:       nil,
			fieldName:  "LayerID",
			wantName:   "LayerID",
			wantInline: false,
		},
		{
			name:       "named inline",
			tags:       map[string][]string{"json": {"body", "inline"}},
			fieldName:  "Body",
			wantName:   "body",
			wantInline: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotInline := StructFieldJSONName(tt.tags, tt.fieldName)
			if gotName != tt.wantName || gotInline != tt.wantInline {
				t.Fatalf("StructFieldJSONName() = (%q, %v), want (%q, %v)", gotName, gotInline, tt.wantName, tt.wantInline)
			}
		})
	}
}
