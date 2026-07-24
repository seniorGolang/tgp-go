// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"testing"

	"tgp/internal/tags"
)

func TestMethodTransportClassification(t *testing.T) {

	project := &Project{}
	contract := &Contract{
		Name: "Live",
		Annotations: tags.DocTags{
			TagServerWS:  "",
			TagServerSSE: "",
		},
	}
	serverMethod := &Method{
		Name:        "Subscribe",
		Annotations: tags.DocTags{TagStream: StreamModeServer},
	}
	if !MethodIsWS(project, contract, serverMethod) {
		t.Fatal("expected WS")
	}
	if !MethodIsSSE(project, contract, serverMethod) {
		t.Fatal("expected SSE")
	}
	if MethodIsJSONRPC(project, contract, serverMethod) {
		t.Fatal("stream must not be jsonrpc")
	}
	if MethodIsHTTP(project, contract, serverMethod) {
		t.Fatal("stream must not be http")
	}
}

func TestContractWSPathDefault(t *testing.T) {

	project := &Project{}
	contract := &Contract{
		Name:        "Live",
		Annotations: tags.DocTags{TagHttpPrefix: "api/v1", TagServerWS: ""},
	}
	path := ContractWSPath(project, contract)
	if path != "/api/v1/ws/live" {
		t.Fatalf("path=%q", path)
	}
}
