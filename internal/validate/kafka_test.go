// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package validate

import (
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func kafkaProject() (project *model.Project) {

	return &model.Project{
		ModulePath: "example.com/app",
		Types: map[string]*model.Type{
			"example:Order": {TypeName: "Order", Kind: model.TypeKindStruct, PkgName: "example"},
		},
	}
}

func kafkaMethod(name string, topic string, args ...*model.Variable) (method *model.Method) {

	return &model.Method{
		Name:        name,
		Annotations: tags.DocTags{model.TagKafkaTopic: topic},
		Args:        args,
		Results:     []*model.Variable{{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}}},
	}
}

func ctxArg() (arg *model.Variable) {

	return &model.Variable{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}}
}

func eventArg() (arg *model.Variable) {

	return &model.Variable{Name: "event", TypeRef: model.TypeRef{TypeID: "example:Order"}}
}

func TestContractKafkaOK(t *testing.T) {

	project := kafkaProject()
	contract := &model.Contract{
		Name:        "Events",
		Annotations: tags.DocTags{model.TagKafka: "", model.TagKafkaCodec: "json"},
		Methods: []*model.Method{
			{
				Name: "OrderCreated",
				Annotations: tags.DocTags{
					model.TagKafkaTopic:   "orders",
					model.TagKafkaKey:     "orderID",
					model.TagKafkaHeaders: "traceID|x-trace-id",
					model.TagKafkaMessage: "event",
				},
				Args: []*model.Variable{
					ctxArg(),
					{Name: "orderID", TypeRef: model.TypeRef{TypeID: "string"}},
					{Name: "traceID", TypeRef: model.TypeRef{TypeID: "string"}},
					eventArg(),
				},
				Results: []*model.Variable{{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}}},
			},
		},
	}
	if err := Contract(contract, project); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContractKafkaLegacyHardError(t *testing.T) {

	project := kafkaProject()
	for _, tag := range []string{model.TagKafkaConsumer, model.TagKafkaPublisher} {
		contract := &model.Contract{
			Name:        "Legacy",
			Annotations: tags.DocTags{tag: ""},
			Methods:     []*model.Method{kafkaMethod("X", "t", ctxArg(), eventArg())},
		}
		err := Contract(contract, project)
		if err == nil || !strings.Contains(err.Error(), "removed") {
			t.Fatalf("tag %s: expected migration error, got %v", tag, err)
		}
	}
}

func TestContractKafkaRejectsHTTPFamily(t *testing.T) {

	project := kafkaProject()
	contract := &model.Contract{
		Name:        "Bad",
		Annotations: tags.DocTags{model.TagKafka: "", model.TagServerHTTP: ""},
		Methods:     []*model.Method{kafkaMethod("X", "t", ctxArg(), eventArg())},
	}
	if err := Contract(contract, project); err == nil {
		t.Fatal("expected error for http+kafka")
	}
}

func TestContractKafkaRequiresTopic(t *testing.T) {

	project := kafkaProject()
	method := kafkaMethod("OnOrder", "", ctxArg(), eventArg())
	method.Annotations = tags.DocTags{}
	contract := &model.Contract{
		Name:        "Events",
		Annotations: tags.DocTags{model.TagKafka: ""},
		Methods:     []*model.Method{method},
	}
	if err := Contract(contract, project); err == nil {
		t.Fatal("expected missing topic error")
	}
}

func TestContractKafkaUnknownKey(t *testing.T) {

	project := kafkaProject()
	method := kafkaMethod("OrderCreated", "orders", ctxArg(), eventArg())
	method.Annotations[model.TagKafkaKey] = "missing"
	contract := &model.Contract{
		Name:        "Events",
		Annotations: tags.DocTags{model.TagKafka: ""},
		Methods:     []*model.Method{method},
	}
	if err := Contract(contract, project); err == nil {
		t.Fatal("expected unknown key error")
	}
}

func TestContractKafkaDuplicateTopicInContract(t *testing.T) {

	project := kafkaProject()
	contract := &model.Contract{
		Name:        "Events",
		Annotations: tags.DocTags{model.TagKafka: ""},
		Methods: []*model.Method{
			kafkaMethod("OnA", "orders", ctxArg(), eventArg()),
			kafkaMethod("OnB", "orders", ctxArg(), eventArg()),
		},
	}
	if err := Contract(contract, project); err == nil {
		t.Fatal("expected duplicate topic error")
	}
}

func TestKafkaProjectDuplicateTopicAcrossContracts(t *testing.T) {

	project := kafkaProject()
	project.Contracts = []*model.Contract{
		{
			Name:        "A",
			Annotations: tags.DocTags{model.TagKafka: ""},
			Methods:     []*model.Method{kafkaMethod("M1", "shared", ctxArg(), eventArg())},
		},
		{
			Name:        "B",
			Annotations: tags.DocTags{model.TagKafka: ""},
			Methods:     []*model.Method{kafkaMethod("M2", "shared", ctxArg(), eventArg())},
		},
	}
	if err := KafkaProject(project); err == nil {
		t.Fatal("expected cross-contract topic error")
	}
}

func TestContractKafkaRejectsPointerMessage(t *testing.T) {

	project := kafkaProject()
	method := kafkaMethod("X", "t", ctxArg(), &model.Variable{
		Name:    "event",
		TypeRef: model.TypeRef{TypeID: "example:Order", NumberOfPointers: 1},
	})
	contract := &model.Contract{
		Name:        "Events",
		Annotations: tags.DocTags{model.TagKafka: ""},
		Methods:     []*model.Method{method},
	}
	if err := Contract(contract, project); err == nil {
		t.Fatal("expected pointer message error")
	}
}

func TestContractKafkaBytesCodec(t *testing.T) {

	project := kafkaProject()
	method := kafkaMethod("X", "t", ctxArg(), &model.Variable{
		Name:    "body",
		TypeRef: model.TypeRef{TypeID: "byte", IsSlice: true},
	})
	method.Annotations[model.TagKafkaCodec] = model.KafkaCodecBytes
	contract := &model.Contract{
		Name:        "Events",
		Annotations: tags.DocTags{model.TagKafka: ""},
		Methods:     []*model.Method{method},
	}
	if err := Contract(contract, project); err != nil {
		t.Fatalf("[]byte ok: %v", err)
	}

	bad := kafkaMethod("Y", "t2", ctxArg(), eventArg())
	bad.Annotations[model.TagKafkaCodec] = model.KafkaCodecBytes
	contract.Methods = []*model.Method{bad}
	if err := Contract(contract, project); err == nil {
		t.Fatal("expected bytes codec type error")
	}
}

func TestContractKafkaInvalidAcks(t *testing.T) {

	project := kafkaProject()
	method := kafkaMethod("X", "t", ctxArg(), eventArg())
	method.Annotations[model.TagKafkaAcks] = "maybe"
	contract := &model.Contract{
		Name:        "Events",
		Annotations: tags.DocTags{model.TagKafka: ""},
		Methods:     []*model.Method{method},
	}
	if err := Contract(contract, project); err == nil {
		t.Fatal("expected invalid acks error")
	}
}

func TestContractKafkaContextMustBeFirst(t *testing.T) {

	project := kafkaProject()
	method := kafkaMethod("X", "t", eventArg(), ctxArg())
	contract := &model.Contract{
		Name:        "Events",
		Annotations: tags.DocTags{model.TagKafka: ""},
		Methods:     []*model.Method{method},
	}
	if err := Contract(contract, project); err == nil {
		t.Fatal("expected ctx first error")
	}
}

func TestContractKafkaEmptyHeaderName(t *testing.T) {

	project := kafkaProject()
	method := kafkaMethod("X", "t", ctxArg(),
		&model.Variable{Name: "traceID", TypeRef: model.TypeRef{TypeID: "string"}},
		eventArg(),
	)
	method.Annotations[model.TagKafkaHeaders] = "traceID|   "
	contract := &model.Contract{
		Name:        "Events",
		Annotations: tags.DocTags{model.TagKafka: ""},
		Methods:     []*model.Method{method},
	}
	if err := Contract(contract, project); err == nil {
		t.Fatal("expected empty header name error")
	}
}
