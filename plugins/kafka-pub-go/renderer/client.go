// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path/filepath"
	"sort"

	"github.com/dave/jennifer/jen"

	"tgp/internal/model"
)

func (r *Renderer) renderClient() (err error) {

	source := newSrcFile(filepath.Base(r.outDir))
	acks := r.acks()
	source.Type().Id("Client").StructFunc(func(group *jen.Group) {
		group.Id("log").Op("*").Qual("log/slog", "Logger")
		group.Id("codecs").Map(jen.String()).Id("codec")
		group.Id("mu").Qual("sync", "RWMutex")
		group.Id("closed").Bool()
		for _, ack := range acks {
			group.Id(ackField(ack)).Op("*").Qual(kgoPath, "Client")
		}
		if r.hasAnnotation(model.TagMetrics) {
			group.Id("metrics").Op("*").Id("metrics")
		}
		if r.hasAnnotation(model.TagTrace) {
			group.Id("tracer").Qual(tracePath, "Tracer")
		}
	})
	source.Line()
	r.addNewClient(source, acks)
	r.addClose(source, acks)
	r.addKafkaClientGetter(source, acks)
	r.addKafkaClientConstructor(source)
	r.addRequiredCodecs(source)
	return source.Save(filepath.Join(r.outDir, "client.go"))
}

func (r *Renderer) addNewClient(source *GoFile, acks []string) {

	body := []jen.Code{
		jen.If(jen.Id("log").Op("==").Nil()).Block(jen.Return(jen.Nil(), jen.Qual("errors", "New").Call(jen.Lit("kafka publisher logger is required")))),
		jen.Id("setup").Op(":=").Id("defaultSetup").Call(),
		jen.For(jen.List(jen.Id("_"), jen.Id("option")).Op(":=").Range().Id("options")).Block(
			jen.If(jen.Id("option").Op("==").Nil()).Block(jen.Continue()),
			jen.If(jen.Id("err").Op("=").Id("option").Call(jen.Op("&").Id("setup")), jen.Id("err").Op("!=").Nil()).Block(jen.Return(jen.Nil(), jen.Id("err"))),
		),
		jen.If(jen.Len(jen.Id("setup").Dot("brokers")).Op("==").Lit(0)).Block(jen.Return(jen.Nil(), jen.Qual("errors", "New").Call(jen.Lit("kafka publisher brokers are required")))),
		jen.If(jen.Id("err").Op("=").Id("validateSecurity").Call(jen.Id("setup")), jen.Id("err").Op("!=").Nil()).Block(jen.Return(jen.Nil(), jen.Id("err"))),
		jen.If(jen.Len(jen.Id("setup").Dot("compression")).Op("==").Lit(0)).Block(jen.Return(jen.Nil(), jen.Qual("errors", "New").Call(jen.Lit("kafka publisher compression requires at least one codec")))),
		jen.Id("codecs").Op(":=").Id("defaultCodecs").Call(),
		jen.For(jen.List(jen.Id("name"), jen.Id("value")).Op(":=").Range().Id("setup").Dot("codecs")).Block(
			jen.If(jen.Id("value").Op("==").Nil()).Block(jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("kafka codec %q is nil"), jen.Id("name")))),
			jen.Id("codecs").Index(jen.Id("name")).Op("=").Id("value"),
		),
		jen.For(jen.List(jen.Id("_"), jen.Id("name")).Op(":=").Range().Id("requiredCodecs").Call()).Block(
			jen.If(jen.Id("codecs").Index(jen.Id("name")).Op("==").Nil()).Block(jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("kafka codec %q is not registered"), jen.Id("name")))),
		),
		jen.Id("client").Op("=").Op("&").Id("Client").Values(jen.Dict{jen.Id("log"): jen.Id("log"), jen.Id("codecs"): jen.Id("codecs")}),
	}
	if r.hasAnnotation(model.TagMetrics) {
		body = append(body, jen.If(jen.Id("setup").Dot("metrics").Op("!=").Nil()).Block(
			jen.If(jen.List(jen.Id("client").Dot("metrics"), jen.Id("err")).Op("=").Id("newMetrics").Call(jen.Id("setup").Dot("metrics")), jen.Id("err").Op("!=").Nil()).Block(jen.Return(jen.Nil(), jen.Id("err"))),
		))
	}
	if r.hasAnnotation(model.TagTrace) {
		body = append(body, jen.Id("client").Dot("tracer").Op("=").Id("setup").Dot("tracer"))
	}
	body = append(body, jen.Var().Id("kafkaClient").Op("*").Qual(kgoPath, "Client"))
	for _, ack := range acks {
		body = append(body,
			jen.If(jen.List(jen.Id("kafkaClient"), jen.Id("err")).Op("=").Id("newKafkaClient").Call(jen.Id("setup"), ackExpression(ack), jen.Lit(ack != model.KafkaAcksAllISR)), jen.Id("err").Op("!=").Nil()).Block(jen.Return(jen.Nil(), jen.Id("err"))),
			jen.Id("client").Dot(ackField(ack)).Op("=").Id("kafkaClient"),
		)
	}
	body = append(body, jen.Return(jen.Id("client"), jen.Nil()))
	source.Func().Id("New").Params(jen.Id("log").Op("*").Qual("log/slog", "Logger"), jen.Id("options").Op("...").Id("Option")).Params(jen.Id("client").Op("*").Id("Client"), jen.Id("err").Error()).Block(body...)
	source.Line()
}

func (r *Renderer) addClose(source *GoFile, acks []string) {

	clients := make([]jen.Code, 0, len(acks))
	for _, ack := range acks {
		clients = append(clients, jen.Id("client").Dot(ackField(ack)))
	}
	source.Func().Params(jen.Id("client").Op("*").Id("Client")).Id("Close").Params().Block(
		jen.If(jen.Id("client").Op("==").Nil()).Block(jen.Return()),
		jen.Id("client").Dot("mu").Dot("Lock").Call(),
		jen.If(jen.Id("client").Dot("closed")).Block(jen.Id("client").Dot("mu").Dot("Unlock").Call(), jen.Return()),
		jen.Id("client").Dot("closed").Op("=").True(),
		jen.Id("clients").Op(":=").Index().Op("*").Qual(kgoPath, "Client").Values(clients...),
		jen.Id("client").Dot("mu").Dot("Unlock").Call(),
		jen.For(jen.List(jen.Id("_"), jen.Id("kafkaClient")).Op(":=").Range().Id("clients")).Block(
			jen.If(jen.Id("kafkaClient").Op("==").Nil()).Block(jen.Continue()),
			jen.Id("_").Op("=").Id("kafkaClient").Dot("Flush").Call(jen.Qual("context", "Background").Call()),
			jen.Id("kafkaClient").Dot("Close").Call(),
		),
	)
	source.Line()
}

func (r *Renderer) addKafkaClientGetter(source *GoFile, acks []string) {

	cases := make([]jen.Code, 0, len(acks)+1)
	for _, ack := range acks {
		cases = append(cases, jen.Case(jen.Lit(ack)).Block(jen.Return(jen.Id("client").Dot(ackField(ack)), jen.Nil())))
	}
	cases = append(cases, jen.Default().Block(jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("unknown kafka acks %q"), jen.Id("acks")))))
	source.Func().Params(jen.Id("client").Op("*").Id("Client")).Id("kafkaClient").Params(jen.Id("acks").String()).Params(jen.Id("result").Op("*").Qual(kgoPath, "Client"), jen.Id("err").Error()).Block(
		jen.Id("client").Dot("mu").Dot("RLock").Call(),
		jen.Defer().Id("client").Dot("mu").Dot("RUnlock").Call(),
		jen.If(jen.Id("client").Dot("closed")).Block(jen.Return(jen.Nil(), jen.Qual("errors", "New").Call(jen.Lit("kafka publisher is closed")))),
		jen.Switch(jen.Id("acks")).Block(cases...),
	)
	source.Line()
}

func (r *Renderer) addKafkaClientConstructor(source *GoFile) {

	source.Func().Id("newKafkaClient").Params(jen.Id("setup").Id("setup"), jen.Id("acks").Qual(kgoPath, "Acks"), jen.Id("disableIdempotence").Bool()).Params(jen.Id("client").Op("*").Qual(kgoPath, "Client"), jen.Id("err").Error()).Block(
		jen.Id("options").Op(":=").Index().Qual(kgoPath, "Opt").Values(
			jen.Qual(kgoPath, "SeedBrokers").Call(jen.Id("setup").Dot("brokers").Op("...")),
			jen.Qual(kgoPath, "RequiredAcks").Call(jen.Id("acks")),
			jen.Qual(kgoPath, "ProducerLinger").Call(jen.Id("setup").Dot("batchMaxLinger")),
			jen.Qual(kgoPath, "ProducerBatchMaxBytes").Call(jen.Id("setup").Dot("batchMaxBytes")),
			jen.Qual(kgoPath, "MaxBufferedRecords").Call(jen.Id("setup").Dot("maxBufferedRecords")),
			jen.Qual(kgoPath, "ProducerBatchCompression").Call(jen.Id("setup").Dot("compression").Op("...")),
		),
		jen.If(jen.Id("disableIdempotence")).Block(jen.Id("options").Op("=").Append(jen.Id("options"), jen.Qual(kgoPath, "DisableIdempotentWrite").Call())),
		jen.If(jen.Id("setup").Dot("tlsConfig").Op("!=").Nil()).Block(
			jen.Id("options").Op("=").Append(jen.Id("options"), jen.Qual(kgoPath, "DialTLSConfig").Call(jen.Id("setup").Dot("tlsConfig"))),
		),
		jen.If(jen.Id("setup").Dot("saslName").Op("!=").Lit("")).Block(
			jen.Var().Id("mechanism").Qual("github.com/twmb/franz-go/pkg/sasl", "Mechanism"),
			jen.If(jen.List(jen.Id("mechanism"), jen.Id("err")).Op("=").Id("saslMechanism").Call(jen.Id("setup").Dot("saslName"), jen.Id("setup").Dot("authUser"), jen.Id("setup").Dot("authPassword")), jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Id("err")),
			),
			jen.Id("options").Op("=").Append(jen.Id("options"), jen.Qual(kgoPath, "SASL").Call(jen.Id("mechanism"))),
		),
		jen.Id("options").Op("=").Append(jen.Id("options"), jen.Id("setup").Dot("clientOptions").Op("...")),
		jen.Return(jen.Qual(kgoPath, "NewClient").Call(jen.Id("options").Op("..."))),
	)
	source.Line()
}

func (r *Renderer) addRequiredCodecs(source *GoFile) {

	names := make([]string, 0)
	seen := make(map[string]struct{})
	for _, contract := range r.contracts {
		for _, method := range contract.Methods {
			name := model.MethodKafkaCodec(r.project, contract, method)
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}
	sort.Strings(names)
	values := make([]jen.Code, 0, len(names))
	for _, name := range names {
		values = append(values, jen.Lit(name))
	}
	source.Func().Id("requiredCodecs").Params().Params(jen.Id("names").Index().String()).Block(jen.Return(jen.Index().String().Values(values...)))
}

func ackExpression(acks string) (value jen.Code) {

	switch acks {
	case model.KafkaAcksNoAck:
		return jen.Qual(kgoPath, "NoAck").Call()
	case model.KafkaAcksLeader:
		return jen.Qual(kgoPath, "LeaderAck").Call()
	default:
		return jen.Qual(kgoPath, "AllISRAcks").Call()
	}
}
