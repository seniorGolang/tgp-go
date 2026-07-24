// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

func (r *Renderer) writeTopicHandler(group *Group, contract *model.Contract, method *model.Method, metrics bool, log bool, trace bool) {

	topic := model.MethodKafkaTopic(r.project, contract, method)
	codec := model.MethodKafkaCodec(r.project, contract, method)
	group.BlockFunc(func(block *Group) {
		block.Id("registered").Op(":=").Id("setup").Dot("handlers").Index(Lit(contract.Name))
		block.Id("codec").Op(":=").Id("setup").Dot("codecs").Index(Lit(codec))
		block.Id("client").Dot("handlers").Index(Lit(topic)).Op("=").Func().
			Params(Id("ctx").Qual("context", "Context"), Id("records").Index().Op("*").Qual("github.com/twmb/franz-go/pkg/kgo", "Record")).
			Params(Id("err").Error()).BlockFunc(func(handler *Group) {
			if trace {
				handler.Var().Id("finish").Func().Params(Error())
				handler.List(Id("ctx"), Id("finish")).Op("=").Id("client").Dot("startConsumeSpan").Call(Id("ctx"), Lit(contract.Name), Lit(method.Name), Lit(topic), Len(Id("records")))
				handler.Defer().Func().Params().Block(Id("finish").Call(Err())).Call()
			}
			handler.Switch(Id("registered").Dot("kind")).BlockFunc(func(switchGroup *Group) {
				switchGroup.Case(Lit("")).BlockFunc(func(caseGroup *Group) {
					caseGroup.Id("handler").Op(":=").Id("registered").Dot("handler").Assert(Id(contract.Name + "Handler"))
					caseGroup.For(List(Id("_"), Id("record")).Op(":=").Range().Id("records")).BlockFunc(func(loop *Group) {
						r.writeDecodeRecord(loop, metrics, log, contract, method, topic)
						r.writeHandlerPlain(loop, metrics, log, contract, method, topic)
						loop.If(Err().Op("!=").Nil()).Block(Return(Err()))
					})
				})
				switchGroup.Case(Lit("Meta")).BlockFunc(func(caseGroup *Group) {
					caseGroup.Id("handler").Op(":=").Id("registered").Dot("handler").Assert(Id(contract.Name + "MetaHandler"))
					caseGroup.For(List(Id("_"), Id("record")).Op(":=").Range().Id("records")).BlockFunc(func(loop *Group) {
						r.writeDecodeRecord(loop, metrics, log, contract, method, topic)
						r.writeHandlerMeta(loop, metrics, log, contract, method, topic)
						loop.If(Err().Op("!=").Nil()).Block(Return(Err()))
					})
				})
				switchGroup.Case(Lit("Slice")).BlockFunc(func(caseGroup *Group) {
					eventType := r.eventType(contract, method)
					caseGroup.Id("handler").Op(":=").Id("registered").Dot("handler").Assert(Id(contract.Name + "SliceHandler"))
					caseGroup.Id("events").Op(":=").Make(Index().Add(eventType), Lit(0), Len(Id("records")))
					caseGroup.Var().Id("totalBytes").Int()
					caseGroup.For(List(Id("_"), Id("record")).Op(":=").Range().Id("records")).BlockFunc(func(loop *Group) {
						r.writeDecodeRecord(loop, metrics, log, contract, method, topic)
						loop.Id("events").Op("=").Append(Id("events"), Id("event"))
						loop.Id("totalBytes").Op("+=").Len(Id("record").Dot("Value"))
					})
					caseGroup.If(Len(Id("events")).Op("!=").Lit(0)).BlockFunc(func(ifGroup *Group) {
						r.writeHandlerSlice(ifGroup, metrics, log, contract, method, topic)
						ifGroup.If(Err().Op("!=").Nil()).Block(Return(Err()))
					})
				})
				switchGroup.Case(Lit("Batch")).BlockFunc(func(caseGroup *Group) {
					eventType := r.eventType(contract, method)
					caseGroup.Id("handler").Op(":=").Id("registered").Dot("handler").Assert(Id(contract.Name + "BatchHandler"))
					caseGroup.Id("batch").Op(":=").Id("Batch").Types(eventType).Values(Dict{
						Id("Records"): Make(Index().Id("Record").Types(eventType), Lit(0), Len(Id("records"))),
					})
					caseGroup.Var().Id("totalBytes").Int()
					caseGroup.For(List(Id("_"), Id("record")).Op(":=").Range().Id("records")).BlockFunc(func(loop *Group) {
						r.writeDecodeRecord(loop, metrics, log, contract, method, topic)
						loop.Id("batch").Dot("Records").Op("=").Append(Id("batch").Dot("Records"), Id("Record").Types(eventType).Values(Dict{
							Id("Value"): Id("event"),
							Id("Meta"):  Id("metaFromRecord").Call(Id("record")),
						}))
						loop.Id("totalBytes").Op("+=").Len(Id("record").Dot("Value"))
					})
					caseGroup.If(Len(Id("batch").Dot("Records")).Op("!=").Lit(0)).BlockFunc(func(ifGroup *Group) {
						r.writeHandlerBatch(ifGroup, metrics, log, contract, method, topic)
						ifGroup.If(Err().Op("!=").Nil()).Block(Return(Err()))
					})
				})
				switchGroup.Default().Block(Return(Qual("fmt", "Errorf").Call(Lit("kafka subscriber: unsupported handler form"))))
			})
			handler.Return(Err())
		})
	})
}

func (r *Renderer) writeDecodeRecord(group *Group, metrics bool, log bool, contract *model.Contract, method *model.Method, topic string) {

	group.Var().Id("event").Add(r.eventType(contract, method))
	r.writeDecode(group, metrics, log, contract, method, topic)
	group.If(Err().Op("!=").Nil()).Block(Return(Qual("fmt", "Errorf").Call(Lit("decode "+contract.Name+"."+method.Name+": %w"), Err())))
}

func (r *Renderer) writeDecode(group *Group, metrics bool, log bool, contract *model.Contract, method *model.Method, topic string) {

	if !metrics && !log {
		group.Err().Op("=").Id("codec").Dot("Unmarshal").Call(Id("record").Dot("Value"), Op("&").Id("event"))
		return
	}
	group.Id("started").Op(":=").Qual("time", "Now").Call()
	group.Err().Op("=").Id("codec").Dot("Unmarshal").Call(Id("record").Dot("Value"), Op("&").Id("event"))
	if metrics {
		group.Id("client").Dot("observeDecode").Call(Lit(contract.Name), Lit(method.Name), Lit(topic), Len(Id("record").Dot("Value")), Qual("time", "Since").Call(Id("started")), Err())
	}
	if log {
		group.If(Err().Op("!=").Nil()).Block(Id("client").Dot("log").Dot("Error").Call(Lit("kafka decode failed"), Lit("tgp.contract"), Lit(contract.Name), Lit("tgp.method"), Lit(method.Name), Lit("messaging.destination"), Lit(topic), Lit("tgp.records"), Lit(1), Lit("tgp.duration"), Qual("time", "Since").Call(Id("started")), Lit("error"), Err()))
	}
}

func (r *Renderer) writeHandlerPlain(group *Group, metrics bool, log bool, contract *model.Contract, method *model.Method, topic string) {

	r.writeHandlerCall(group, metrics, log, contract, method, topic, Id("handler").Dot(method.Name).Call(Id("ctx"), Id("event")), Lit(1), Len(Id("record").Dot("Value")), true)
}

func (r *Renderer) writeHandlerMeta(group *Group, metrics bool, log bool, contract *model.Contract, method *model.Method, topic string) {

	r.writeHandlerCall(group, metrics, log, contract, method, topic, Id("handler").Dot(method.Name).Call(Id("ctx"), Id("event"), Id("metaFromRecord").Call(Id("record"))), Lit(1), Len(Id("record").Dot("Value")), true)
}

func (r *Renderer) writeHandlerSlice(group *Group, metrics bool, log bool, contract *model.Contract, method *model.Method, topic string) {

	r.writeHandlerCall(group, metrics, log, contract, method, topic, Id("handler").Dot(method.Name).Call(Id("ctx"), Id("events")), Len(Id("events")), Id("totalBytes"), false)
}

func (r *Renderer) writeHandlerBatch(group *Group, metrics bool, log bool, contract *model.Contract, method *model.Method, topic string) {

	r.writeHandlerCall(group, metrics, log, contract, method, topic, Id("handler").Dot(method.Name).Call(Id("ctx"), Id("batch")), Len(Id("batch").Dot("Records")), Id("totalBytes"), false)
}

func (r *Renderer) writeHandlerCall(group *Group, metrics bool, log bool, contract *model.Contract, method *model.Method, topic string, call Code, records Code, bytes Code, reuseStarted bool) {

	if !metrics && !log {
		group.Err().Op("=").Add(call)
		return
	}
	if reuseStarted {
		group.Id("started").Op("=").Qual("time", "Now").Call()
	} else {
		group.Id("started").Op(":=").Qual("time", "Now").Call()
	}
	group.Err().Op("=").Add(call)
	if metrics {
		group.Id("client").Dot("observeHandler").Call(Lit(contract.Name), Lit(method.Name), Lit(topic), records, bytes, Qual("time", "Since").Call(Id("started")), Err())
	}
	if log {
		group.If(Err().Op("!=").Nil()).Block(
			Id("client").Dot("log").Dot("Error").Call(Lit("kafka handler failed"), Lit("tgp.contract"), Lit(contract.Name), Lit("tgp.method"), Lit(method.Name), Lit("messaging.destination"), Lit(topic), Lit("tgp.records"), records, Lit("tgp.duration"), Qual("time", "Since").Call(Id("started")), Lit("error"), Err()),
		).Else().Block(
			Id("client").Dot("log").Dot("Info").Call(Lit("kafka handler completed"), Lit("tgp.contract"), Lit(contract.Name), Lit("tgp.method"), Lit(method.Name), Lit("messaging.destination"), Lit(topic), Lit("tgp.records"), records, Lit("tgp.duration"), Qual("time", "Since").Call(Id("started"))),
		)
	}
}
