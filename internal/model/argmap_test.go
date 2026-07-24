// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"testing"

	"tgp/internal/tags"
)

func testStringVar(name string) (variable *Variable) {

	return &Variable{Name: name, TypeRef: TypeRef{TypeID: "string"}}
}

func TestArgByPathSegment(t *testing.T) {

	method := &Method{
		Name: "Get",
		Args: []*Variable{
			{Name: "ctx", TypeRef: TypeRef{TypeID: typeIDContext}},
			testStringVar("UserId"),
		},
	}

	if got := ArgByPathSegment(method, "userId"); got == nil || got.Name != "UserId" {
		t.Fatalf("LowerCamel match: got %#v", got)
	}
	if got := ArgByPathSegment(method, "UserId"); got == nil || got.Name != "UserId" {
		t.Fatalf("exact match: got %#v", got)
	}
	if got := ArgByPathSegment(method, "missing"); got != nil {
		t.Fatalf("missing segment: got %#v", got)
	}
	if got := ArgByPathSegment(nil, "userId"); got != nil {
		t.Fatalf("nil method: got %#v", got)
	}
}

func TestHTTPPathParamArgMapAndSet(t *testing.T) {

	contract := &Contract{Name: "Svc"}
	method := &Method{
		Name: "Get",
		Args: []*Variable{
			testStringVar("UserId"),
			testStringVar("extra"),
		},
		Annotations: tags.DocTags{
			TagHttpPath: "/users/:userId",
		},
	}
	project := &Project{}

	pathMap := HTTPPathParamArgMap(project, contract, method)
	if pathMap["UserId"] != "userId" {
		t.Fatalf("pathMap = %#v, want UserId→userId", pathMap)
	}
	if _, ok := pathMap["extra"]; ok {
		t.Fatalf("extra must not be in pathMap: %#v", pathMap)
	}

	pathSet := HTTPPathParamArgSet(project, contract, method)
	if _, ok := pathSet["UserId"]; !ok {
		t.Fatalf("pathSet missing UserId: %#v", pathSet)
	}

	bodyArgs := HTTPArgsFromRequestBody(project, contract, method)
	if len(bodyArgs) != 1 || bodyArgs[0].Name != "extra" {
		t.Fatalf("bodyArgs = %#v, want only extra", bodyArgs)
	}
}

func TestStreamPathParamArgMapAndOmit(t *testing.T) {

	contract := &Contract{
		Name: "Live",
		Annotations: tags.DocTags{
			TagServerWS:   "",
			TagServerSSE:  "",
			TagHttpPrefix: "api/v1",
			TagWSPath:     "/ws/:room",
		},
	}
	method := &Method{
		Name: "Subscribe",
		Args: []*Variable{
			testStringVar("room"),
			testStringVar("symbol"),
			testStringVar("token"),
		},
		Results: []*Variable{
			{Name: "ticks", TypeRef: TypeRef{ChanOf: &TypeRef{TypeID: "string"}, ChanDirection: 2}},
			{Name: "err", TypeRef: TypeRef{TypeID: "error"}},
		},
		Annotations: tags.DocTags{
			TagStream:     StreamModeServer,
			TagSSEPath:    "/sse/:room/subscribe",
			TagHttpHeader: "token|X-Token|explicit",
		},
	}
	project := &Project{Contracts: []*Contract{contract}}

	wsMap := StreamPathParamArgMap(project, contract, method)
	if wsMap["room"] != "room" {
		t.Fatalf("ws path map = %#v", wsMap)
	}

	methodSSE := *method
	methodSSE.Annotations = tags.DocTags{
		TagStream:  StreamModeServer,
		TagSSEPath: "/sse/:room/subscribe",
	}
	// MethodIsSSE requires sse-server on contract — already set.
	sseMap := PathParamArgMap(&methodSSE, MethodSSEPath(project, contract, &methodSSE))
	if sseMap["room"] != "room" {
		t.Fatalf("sse path map = %#v", sseMap)
	}

	omit := HTTPOmitFromRequestJSON(project, contract, method)
	if _, ok := omit["token"]; !ok {
		t.Fatalf("token must be omitted from JSON: %#v", omit)
	}
	if _, ok := omit["room"]; !ok {
		t.Fatalf("room path param must be omitted from JSON: %#v", omit)
	}
	if _, ok := omit["symbol"]; ok {
		t.Fatalf("symbol must stay in JSON body: %#v", omit)
	}
}

func TestHTTPResultNamesBodyModeStaysInBody(t *testing.T) {

	contract := &Contract{Name: "Svc"}
	method := &Method{
		Name: "ResultHeader",
		Results: []*Variable{
			testStringVar("correlationId"),
			{Name: "err", TypeRef: TypeRef{TypeID: "error"}},
		},
		Annotations: tags.DocTags{
			TagHttpHeader: "correlationId|X-Correlation-Id",
		},
	}
	project := &Project{}

	if _, ok := HTTPResultNamesExcludeFromBody(project, contract, method)["correlationId"]; ok {
		t.Fatal("body-mode result must stay in JSON body for clients")
	}
	if _, ok := HTTPResultNamesOmitFromExchangeBody(project, contract, method)["correlationId"]; ok {
		t.Fatal("body-mode result must stay in JSON body for server/swagger")
	}
	exchangeBody := HTTPResultsForExchangeBody(project, contract, method)
	if len(exchangeBody) != 1 || exchangeBody[0].Name != "correlationId" {
		t.Fatalf("exchange body = %#v", exchangeBody)
	}
}

func TestHTTPResultNamesExplicitNotInBody(t *testing.T) {

	contract := &Contract{Name: "Svc"}
	method := &Method{
		Name: "ResultHeader",
		Results: []*Variable{
			testStringVar("correlationId"),
			{Name: "err", TypeRef: TypeRef{TypeID: "error"}},
		},
		Annotations: tags.DocTags{
			TagHttpHeader: "correlationId|X-Correlation-Id|explicit",
		},
	}
	project := &Project{}

	if _, ok := HTTPResultNamesExcludeFromBody(project, contract, method)["correlationId"]; !ok {
		t.Fatal("explicit result must not be in client JSON body")
	}
	if _, ok := HTTPResultNamesOmitFromExchangeBody(project, contract, method)["correlationId"]; !ok {
		t.Fatal("explicit result must not be in server JSON body")
	}
	if len(HTTPResultsForExchangeBody(project, contract, method)) != 0 {
		t.Fatal("exchange body must be empty for explicit-only result")
	}
}

func TestHTTPResultNamesImplicitNotInBody(t *testing.T) {

	contract := &Contract{Name: "Svc"}
	method := &Method{
		Name: "ResultHeader",
		Results: []*Variable{
			testStringVar("locale"),
			{Name: "err", TypeRef: TypeRef{TypeID: "error"}},
		},
		Annotations: tags.DocTags{
			TagHttpHeader: "locale|X-Locale|implicit",
		},
	}
	project := &Project{}

	if _, ok := HTTPResultNamesExcludeFromBody(project, contract, method)["locale"]; !ok {
		t.Fatal("implicit result must not be in JSON body")
	}
}

func TestHTTPOmitFromRequestJSON_IncludesPath(t *testing.T) {

	contract := &Contract{Name: "Svc"}
	method := &Method{
		Name: "Put",
		Args: []*Variable{
			testStringVar("id"),
			testStringVar("body"),
		},
		Annotations: tags.DocTags{
			TagHttpPath: "/item/:id",
		},
	}
	project := &Project{}

	omit := HTTPOmitFromRequestJSON(project, contract, method)
	if _, ok := omit["id"]; !ok {
		t.Fatalf("path arg id must be omitted from JSON: %#v", omit)
	}
	if _, ok := omit["body"]; ok {
		t.Fatalf("body must stay in JSON: %#v", omit)
	}
}

func TestArgByNameNoFalsePositive(t *testing.T) {

	method := &Method{
		Args: []*Variable{
			testStringVar("a"),
			testStringVar("b"),
		},
	}
	if got := argByName(method, "missing"); got != nil {
		t.Fatalf("argByName missing returned %#v", got)
	}
	if got := resultByName(method, "missing"); got != nil {
		t.Fatalf("resultByName missing returned %#v", got)
	}
}
