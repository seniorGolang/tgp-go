// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func TestHasKafkaContracts(t *testing.T) {

	project := &model.Project{
		Contracts: []*model.Contract{
			{Name: "Http", Annotations: tags.DocTags{model.TagServerHTTP: ""}},
			{Name: "Orders", Annotations: tags.DocTags{model.TagKafka: ""}},
		},
	}
	if !hasKafkaContracts(project) {
		t.Fatal("expected kafka contract")
	}
	httpOnly := &model.Project{
		Contracts: []*model.Contract{
			{Name: "Http", Annotations: tags.DocTags{model.TagServerHTTP: ""}},
		},
	}
	if hasKafkaContracts(httpOnly) {
		t.Fatal("http-only project must not report kafka")
	}
}

func TestMethodKafkaExtraArgsIgnoredForHTTPFamily(t *testing.T) {

	project := &model.Project{}
	contract := &model.Contract{
		Name:        "Http",
		Annotations: tags.DocTags{model.TagServerHTTP: ""},
		Methods: []*model.Method{{
			Name: "Get",
			Args: []*model.Variable{
				{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
				{Name: "id", TypeRef: model.TypeRef{TypeID: "string"}},
				{Name: "q", TypeRef: model.TypeRef{TypeID: "string"}},
			},
		}},
	}
	if model.ContractIsKafka(project, contract) {
		t.Fatal("http contract must not be kafka")
	}
	// HTTP methods often have multiple args; warn loop must skip non-kafka contracts.
	extras := model.MethodKafkaExtraArgs(project, contract, contract.Methods[0])
	if len(extras) == 0 {
		t.Fatal("expected extras for multi-arg method under kafka heuristic")
	}
}
