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

func TestResolveSchemaForMerge_oneOfPointerRef(t *testing.T) {

	const cfgJWKSTypeName = "dto.CfgJWKS"

	gen := newGenerator(&model.Project{Types: map[string]*model.Type{
		"string": {Kind: model.TypeKindString},
	}})
	gen.schemas[cfgJWKSTypeName] = types.Schema{
		Type: "object",
		Properties: types.Properties{
			"issuer":   {Type: "string"},
			"jwks_uri": {Type: "string"},
		},
		Required: []string{"issuer", "jwks_uri"},
	}

	source := &types.Schema{
		OneOf: []types.Schema{
			{Ref: componentsSchemasPrefix + cfgJWKSTypeName},
			{Nullable: true},
		},
	}

	resolved := gen.resolveSchemaForMerge(source)
	if len(resolved.Properties) != 2 {
		t.Fatalf("expected 2 properties, got %#v", resolved.Properties)
	}
	if _, ok := resolved.Properties["issuer"]; !ok {
		t.Fatalf("expected issuer property, got %#v", resolved.Properties)
	}
}

func TestResolveSchemaForMerge_allOfAliasRef(t *testing.T) {

	const aliasTypeName = "dto.JSONWebKeySet"
	const baseTypeName = "jose.JSONWebKeySet"

	gen := newGenerator(&model.Project{Types: map[string]*model.Type{}})
	gen.schemas[aliasTypeName] = types.Schema{
		AllOf: []types.Schema{{Ref: componentsSchemasPrefix + baseTypeName}},
	}
	gen.schemas[baseTypeName] = types.Schema{
		Type: "object",
		Properties: types.Properties{
			"keys": {Type: "array"},
		},
		Required: []string{"keys"},
	}

	source := &types.Schema{
		OneOf: []types.Schema{
			{Ref: componentsSchemasPrefix + aliasTypeName},
			{Nullable: true},
		},
	}

	resolved := gen.resolveSchemaForMerge(source)
	if len(resolved.Properties) != 1 {
		t.Fatalf("expected 1 property, got %#v", resolved.Properties)
	}
	if _, ok := resolved.Properties["keys"]; !ok {
		t.Fatalf("expected keys property, got %#v", resolved.Properties)
	}
}

func TestEffectiveResponseSchema_enableInlineSinglePointerStruct(t *testing.T) {

	const cfgTypeID = "example/contracts/dto:CfgJWKS"
	const cfgTypeName = "dto.CfgJWKS"

	project := &model.Project{
		ModulePath: "example",
		Types: map[string]*model.Type{
			"string": {Kind: model.TypeKindString},
			cfgTypeID: {
				Kind:     model.TypeKindStruct,
				TypeName: "CfgJWKS",
				StructFields: []*model.StructField{
					{Name: "Issuer", TypeRef: model.TypeRef{TypeID: "string"}, Tags: map[string][]string{"json": {"issuer"}}},
					{Name: "UriJWKS", TypeRef: model.TypeRef{TypeID: "string"}, Tags: map[string][]string{"json": {"jwks_uri"}}},
				},
			},
		},
	}
	contract := &model.Contract{
		Name:    "JWKS",
		PkgPath: "example/contracts",
		Annotations: tags.DocTags{
			model.TagHttpEnableInlineSingle: "",
		},
		Methods: []*model.Method{{
			Name: "Configuration",
			Results: []*model.Variable{
				{Name: "config", TypeRef: model.TypeRef{TypeID: cfgTypeID, NumberOfPointers: 1}},
				{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
			},
		}},
	}

	gen := newGenerator(project)
	method := contract.Methods[0]
	gen.registerStruct(gen.responseStructName(contract, method), contract.PkgPath, method, gen.resultsWithoutError(method), contentJSON, false)

	schema := gen.effectiveResponseSchema(contract, method, gen.responseStructName(contract, method), nil)
	if len(schema.Properties) != 2 {
		t.Fatalf("expected inline struct properties, got %#v", schema)
	}
	if _, ok := schema.Properties["issuer"]; !ok {
		t.Fatalf("expected issuer in response schema, got %#v", schema.Properties)
	}
	if _, found := gen.schemas[cfgTypeName]; !found {
		t.Fatalf("expected registered schema %q", cfgTypeName)
	}
}
