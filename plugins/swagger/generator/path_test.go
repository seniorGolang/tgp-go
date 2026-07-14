// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
	"tgp/plugins/swagger/types"
)

func TestAddResponseHeaders_BodyModeIncluded(t *testing.T) {

	project := &model.Project{
		Types: map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Annotations",
				PkgPath: "example/contracts",
				Methods: []*model.Method{
					{
						Name: "ResultHeader",
						Results: []*model.Variable{
							{Name: "correlationId", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
						Annotations: tags.DocTags{
							model.TagHttpHeader: "correlationId|X-Correlation-Id",
						},
					},
				},
			},
		},
	}
	g := &generator{project: project}
	contract := project.Contracts[0]
	method := contract.Methods[0]
	operation := &types.Operation{
		Responses: types.Responses{
			"200": types.Response{Description: "OK"},
		},
	}
	g.addResponseHeaders(operation, contract, method, 200)
	headers := operation.Responses["200"].Headers
	if headers == nil {
		t.Fatal("expected response headers")
	}
	if _, ok := headers["X-Correlation-Id"]; !ok {
		t.Fatalf("body-mode result header missing: %#v", headers)
	}
}
