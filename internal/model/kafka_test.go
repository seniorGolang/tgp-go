// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"testing"
)

func TestContractIsKafka(t *testing.T) {

	project := &Project{}
	kafka := &Contract{Name: "Events", Annotations: map[string]string{"kafka": ""}}
	legacy := &Contract{Name: "Handlers", Annotations: map[string]string{"kafka-consumer": ""}}
	plain := &Contract{Name: "API", Annotations: map[string]string{"http-server": ""}}

	if !ContractIsKafka(project, kafka) {
		t.Fatal("expected @tg kafka")
	}
	if ContractIsKafka(project, legacy) {
		t.Fatal("legacy role must not be ContractIsKafka")
	}
	if !ContractHasLegacyKafkaRole(project, legacy) {
		t.Fatal("expected legacy role")
	}
	if ContractIsKafka(project, plain) {
		t.Fatal("plain http contract must not be kafka")
	}
}

func TestMethodKafkaTopicKeyAcksCodec(t *testing.T) {

	project := &Project{}
	contract := &Contract{
		Name:        "Events",
		Annotations: map[string]string{"kafka": "", "kafka-acks": "leaderAck", "kafka-codec": "msgpack"},
	}
	method := &Method{
		Name:        "Created",
		Annotations: map[string]string{"kafka-topic": " orders.created ", "kafka-key": "orderID", "kafka-acks": "noAck"},
	}
	if got := MethodKafkaTopic(project, contract, method); got != "orders.created" {
		t.Fatalf("topic: got %q", got)
	}
	if got := MethodKafkaKeyArg(project, contract, method); got != "orderID" {
		t.Fatalf("key: got %q", got)
	}
	if got := MethodKafkaAcks(project, contract, method); got != KafkaAcksNoAck {
		t.Fatalf("acks method override: got %q", got)
	}
	if got := MethodKafkaCodec(project, contract, method); got != KafkaCodecMsgpack {
		t.Fatalf("codec: got %q", got)
	}
	plain := &Method{Name: "Other", Annotations: map[string]string{"kafka-topic": "t"}}
	if got := MethodKafkaAcks(project, contract, plain); got != KafkaAcksLeader {
		t.Fatalf("acks from interface: got %q", got)
	}
	if got := MethodKafkaAcks(project, &Contract{Name: "X"}, plain); got != KafkaAcksAllISR {
		t.Fatalf("acks default: got %q", got)
	}
}

func TestMethodKafkaMessageArgExplicitAndHeuristic(t *testing.T) {

	project := &Project{}
	contract := &Contract{Name: "Events", Annotations: map[string]string{"kafka": "", "kafka-message": "event"}}
	method := &Method{
		Name:        "Publish",
		Annotations: map[string]string{"kafka-topic": "t", "kafka-key": "id", "kafka-headers": "traceID|x-trace-id"},
		Args: []*Variable{
			{Name: "ctx", TypeRef: TypeRef{TypeID: "context:Context"}},
			{Name: "id", TypeRef: TypeRef{TypeID: "string"}},
			{Name: "traceID", TypeRef: TypeRef{TypeID: "string"}},
			{Name: "event", TypeRef: TypeRef{TypeID: "Order"}},
			{Name: "extra", TypeRef: TypeRef{TypeID: "string"}},
		},
	}
	arg, ok := MethodKafkaMessageArg(project, contract, method)
	if !ok || arg.Name != "event" {
		t.Fatalf("explicit interface message: ok=%v arg=%v", ok, arg)
	}
	extras := MethodKafkaExtraArgs(project, contract, method)
	if len(extras) != 1 || extras[0].Name != "extra" {
		t.Fatalf("extras: %+v", extras)
	}

	heuristic := &Method{
		Name:        "Publish2",
		Annotations: map[string]string{"kafka-topic": "t2", "kafka-key": "id"},
		Args: []*Variable{
			{Name: "ctx", TypeRef: TypeRef{TypeID: "context:Context"}},
			{Name: "id", TypeRef: TypeRef{TypeID: "string"}},
			{Name: "payload", TypeRef: TypeRef{TypeID: "Order"}},
		},
	}
	arg, ok = MethodKafkaMessageArg(project, &Contract{Name: "E"}, heuristic)
	if !ok || arg.Name != "payload" {
		t.Fatalf("heuristic: ok=%v arg=%v", ok, arg)
	}
}

func TestTypeRefByteForms(t *testing.T) {

	if !TypeRefIsByteSlice(&TypeRef{TypeID: "byte", IsSlice: true}) {
		t.Fatal("[]byte")
	}
	if !TypeRefIsByteSliceSlice(&TypeRef{TypeID: "[]byte", IsSlice: true}) {
		t.Fatal("[][]byte")
	}
	if !TypeRefIsByteSliceEllipsis(&TypeRef{TypeID: "byte", IsEllipsis: true, IsSlice: true}) {
		t.Fatal("...[]byte")
	}
	if !TypeRefIsKafkaKeyOrHeader(&TypeRef{TypeID: "string"}) {
		t.Fatal("string key")
	}
	if !TypeRefIsKafkaKeyOrHeader(&TypeRef{TypeID: "string", IsSlice: true}) {
		t.Fatal("[]string key")
	}
}
