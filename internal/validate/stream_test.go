// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package validate

import (
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func TestContractStreamAnnotations_ServerOK(t *testing.T) {

	project := &model.Project{Types: map[string]*model.Type{}}
	contract := &model.Contract{
		Name: "Live",
		Annotations: tags.DocTags{
			model.TagServerWS:  "",
			model.TagServerSSE: "",
		},
		Methods: []*model.Method{
			{
				Name:        "Subscribe",
				Annotations: tags.DocTags{model.TagStream: model.StreamModeServer},
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
					{Name: "symbol", TypeRef: model.TypeRef{TypeID: "string"}},
				},
				Results: []*model.Variable{
					{Name: "ticks", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2}},
					{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
				},
			},
		},
	}
	if err := contractStreamAnnotations(project, contract); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContractStreamAnnotations_ChanWithoutStream(t *testing.T) {

	project := &model.Project{Types: map[string]*model.Type{}}
	contract := &model.Contract{
		Name: "Live",
		Annotations: tags.DocTags{
			model.TagServerWS: "",
		},
		Methods: []*model.Method{
			{
				Name: "Bad",
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
				},
				Results: []*model.Variable{
					{Name: "ticks", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2}},
					{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
				},
			},
		},
	}
	if err := contractStreamAnnotations(project, contract); err == nil {
		t.Fatal("expected error for chan without stream")
	}
}

func TestContractStreamAnnotations_SSERequiresServer(t *testing.T) {

	project := &model.Project{Types: map[string]*model.Type{}}
	contract := &model.Contract{
		Name: "Live",
		Annotations: tags.DocTags{
			model.TagServerSSE: "",
			model.TagServerWS:  "",
		},
		Methods: []*model.Method{
			{
				Name:        "Subscribe",
				Annotations: tags.DocTags{model.TagStream: model.StreamModeServer},
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
				},
				Results: []*model.Variable{
					{Name: "ticks", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2}},
					{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
				},
			},
			{
				Name:        "Ingest",
				Annotations: tags.DocTags{model.TagStream: model.StreamModeClient},
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
					{Name: "samples", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2}},
				},
				Results: []*model.Variable{
					{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
				},
			},
		},
	}
	if err := contractStreamAnnotations(project, contract); err != nil {
		t.Fatalf("ws+sse contract may mix server and client streams: %v", err)
	}
}

func TestContractStreamAnnotations_SSEPathOnlyServer(t *testing.T) {

	project := &model.Project{Types: map[string]*model.Type{}}
	contract := &model.Contract{
		Name:        "Live",
		Annotations: tags.DocTags{model.TagServerWS: ""},
		Methods: []*model.Method{
			{
				Name: "Ingest",
				Annotations: tags.DocTags{
					model.TagStream:  model.StreamModeClient,
					model.TagSSEPath: "/sse/x",
				},
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
					{Name: "samples", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2}},
				},
				Results: []*model.Variable{
					{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
				},
			},
		},
	}
	if err := contractStreamAnnotations(project, contract); err == nil {
		t.Fatal("expected error: sse-path on client stream")
	}
}
