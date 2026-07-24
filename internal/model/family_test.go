// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"testing"

	"tgp/internal/tags"
)

func TestContractMarks(t *testing.T) {

	project := &Project{}

	httpFamily, kafka := ContractMarks(project, nil)
	if httpFamily || kafka {
		t.Fatal("nil contract")
	}

	httpContract := &Contract{Name: "Http", Annotations: tags.DocTags{TagServerHTTP: ""}}
	httpFamily, kafka = ContractMarks(project, httpContract)
	if !httpFamily || kafka {
		t.Fatalf("http: httpFamily=%v kafka=%v", httpFamily, kafka)
	}

	rpcContract := &Contract{Name: "Rpc", Annotations: tags.DocTags{TagServerJsonRPC: ""}}
	if !ContractIsHTTPFamily(project, rpcContract) {
		t.Fatal("jsonRPC-server is HTTP family")
	}

	wsContract := &Contract{Name: "Live", Annotations: tags.DocTags{TagServerWS: ""}}
	if !ContractIsHTTPFamily(project, wsContract) {
		t.Fatal("ws-server is HTTP family")
	}

	sseContract := &Contract{Name: "LiveSSE", Annotations: tags.DocTags{TagServerSSE: ""}}
	if !ContractIsHTTPFamily(project, sseContract) {
		t.Fatal("sse-server is HTTP family")
	}

	kafkaContract := &Contract{Name: "Orders", Annotations: tags.DocTags{TagKafka: ""}}
	httpFamily, kafka = ContractMarks(project, kafkaContract)
	if httpFamily || !kafka {
		t.Fatalf("kafka: httpFamily=%v kafka=%v", httpFamily, kafka)
	}

	plain := &Contract{Name: "Plain"}
	httpFamily, kafka = ContractMarks(project, plain)
	if httpFamily || kafka {
		t.Fatal("plain contract has no family")
	}
}

func TestContractIsKafkaIgnoresLegacy(t *testing.T) {

	project := &Project{}
	legacy := &Contract{Name: "Legacy", Annotations: tags.DocTags{TagKafkaConsumer: ""}}
	if ContractIsKafka(project, legacy) {
		t.Fatal("legacy role must not be ContractIsKafka")
	}
	if !ContractHasLegacyKafkaRole(project, legacy) {
		t.Fatal("legacy role expected")
	}
}
