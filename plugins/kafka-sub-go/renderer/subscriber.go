// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path/filepath"
	"sort"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

func (r *Renderer) renderSubscriber() (err error) {

	file := NewSrcFile(r.pkgName)
	hasMetrics := r.hasMetrics()
	hasTrace := r.hasTrace()
	hasLog := r.hasLog()

	file.Comment("Client управляет единой consumer group Kafka.")
	file.Type().Id("Client").StructFunc(func(group *Group) {
		group.Id("log").Op("*").Qual("log/slog", "Logger")
		group.Id("client").Op("*").Qual("github.com/twmb/franz-go/pkg/kgo", "Client")
		group.Id("handlers").Map(String()).Id("TopicHandler")
		group.Id("maxPoll").Int()
		group.Id("commitAfter").Bool()
		group.Id("stop").Qual("context", "CancelFunc")
		group.Id("mu").Qual("sync", "Mutex")
		group.Id("running").Bool()
		group.Id("closed").Bool()
		if hasMetrics {
			group.Id("metrics").Op("*").Id("metrics")
			group.Id("lagStop").Qual("context", "CancelFunc")
			group.Id("topics").Index().String()
		}
		if hasTrace {
			group.Id("tracer").Qual("go.opentelemetry.io/otel/trace", "Tracer")
		}
	})
	r.writeNew(file, hasMetrics, hasTrace, hasLog)
	r.writeClose(file, hasMetrics)
	r.writeRun(file, hasMetrics)
	if hasMetrics {
		r.writeLag(file)
	}
	return file.Save(filepath.Join(r.outDir, "subscriber.go"))
}

func (r *Renderer) writeNew(file GoFile, hasMetrics bool, hasTrace bool, hasLog bool) {

	file.Line().Comment("New создаёт подписчик Kafka.")
	file.Func().Id("New").Params(Id("log").Op("*").Qual("log/slog", "Logger"), Id("options").Op("...").Id("Option")).Params(Id("client").Op("*").Id("Client"), Id("err").Error()).BlockFunc(func(group *Group) {
		group.If(Id("log").Op("==").Nil()).Block(Return(Nil(), Qual("fmt", "Errorf").Call(Lit("kafka subscriber: log is required"))))
		group.Id("setup").Op(":=").Id("defaultSetup").Call()
		group.For(List(Id("_"), Id("option")).Op(":=").Range().Id("options")).Block(
			If(Id("option").Op("!=").Nil()).Block(Id("option").Call(Id("setup"))),
		)
		group.If(Err().Op("=").Id("validateSetup").Call(Id("setup")), Err().Op("!=").Nil()).Block(Return(Nil(), Err()))
		group.If(Len(Id("setup").Dot("brokers")).Op("==").Lit(0)).Block(Return(Nil(), Qual("fmt", "Errorf").Call(Lit("kafka subscriber: brokers are required"))))
		group.If(Id("setup").Dot("group").Op("==").Lit("")).Block(Return(Nil(), Qual("fmt", "Errorf").Call(Lit("kafka subscriber: group is required"))))
		requiredCodecs := make([]string, 0)
		seenCodec := make(map[string]struct{})
		for _, contract := range r.contracts() {
			group.If(List(Id("registered"), Id("ok")).Op(":=").Id("setup").Dot("handlers").Index(Lit(contract.Name)), Op("!").Id("ok").Op("||").Id("registered").Dot("handler").Op("==").Nil()).Block(
				Return(Nil(), Qual("fmt", "Errorf").Call(Lit("kafka subscriber: "+contract.Name+" handler is required"))),
			)
			for _, method := range contract.Methods {
				codec := model.MethodKafkaCodec(r.project, contract, method)
				if _, exists := seenCodec[codec]; exists {
					continue
				}
				seenCodec[codec] = struct{}{}
				requiredCodecs = append(requiredCodecs, codec)
			}
		}
		sort.Strings(requiredCodecs)
		for _, codec := range requiredCodecs {
			group.If(Id("setup").Dot("codecs").Index(Lit(codec)).Op("==").Nil()).Block(Return(Nil(), Qual("fmt", "Errorf").Call(Lit("kafka subscriber: codec "+codec+" is required"))))
		}
		options := []Code{
			Qual("github.com/twmb/franz-go/pkg/kgo", "SeedBrokers").Call(Id("setup").Dot("brokers").Op("...")),
			Qual("github.com/twmb/franz-go/pkg/kgo", "ConsumerGroup").Call(Id("setup").Dot("group")),
			Qual("github.com/twmb/franz-go/pkg/kgo", "ConsumeTopics").Call(r.topicLiterals()...),
			Qual("github.com/twmb/franz-go/pkg/kgo", "BlockRebalanceOnPoll").Call(),
			Qual("github.com/twmb/franz-go/pkg/kgo", "FetchMinBytes").Call(Id("setup").Dot("fetchMinBytes")),
			Qual("github.com/twmb/franz-go/pkg/kgo", "FetchMaxWait").Call(Id("setup").Dot("fetchMaxWait")),
		}
		group.Id("clientOptions").Op(":=").Index().Qual("github.com/twmb/franz-go/pkg/kgo", "Opt").Values(options...)
		group.If(Id("setup").Dot("resetPosition").Op("==").Id("AtEnd")).Block(
			Id("clientOptions").Op("=").Append(Id("clientOptions"), Qual("github.com/twmb/franz-go/pkg/kgo", "ConsumeResetOffset").Call(Qual("github.com/twmb/franz-go/pkg/kgo", "NewOffset").Call().Dot("AtEnd").Call())),
		).Else().Block(
			Id("clientOptions").Op("=").Append(Id("clientOptions"), Qual("github.com/twmb/franz-go/pkg/kgo", "ConsumeResetOffset").Call(Qual("github.com/twmb/franz-go/pkg/kgo", "NewOffset").Call().Dot("AtStart").Call())),
		)
		group.If(Id("setup").Dot("commitAfter")).Block(Id("clientOptions").Op("=").Append(Id("clientOptions"), Qual("github.com/twmb/franz-go/pkg/kgo", "DisableAutoCommit").Call()))
		group.If(Id("setup").Dot("tlsConfig").Op("!=").Nil()).Block(
			Id("clientOptions").Op("=").Append(Id("clientOptions"), Qual("github.com/twmb/franz-go/pkg/kgo", "DialTLSConfig").Call(Id("setup").Dot("tlsConfig"))),
		)
		group.If(Id("setup").Dot("saslName").Op("!=").Lit("")).Block(
			Var().Id("mechanism").Qual("github.com/twmb/franz-go/pkg/sasl", "Mechanism"),
			If(List(Id("mechanism"), Err()).Op("=").Id("saslMechanism").Call(Id("setup").Dot("saslName"), Id("setup").Dot("authUser"), Id("setup").Dot("authPassword")), Err().Op("!=").Nil()).Block(
				Return(Nil(), Err()),
			),
			Id("clientOptions").Op("=").Append(Id("clientOptions"), Qual("github.com/twmb/franz-go/pkg/kgo", "SASL").Call(Id("mechanism"))),
		)
		group.Id("clientOptions").Op("=").Append(Id("clientOptions"), Id("setup").Dot("clientOptions").Op("..."))
		group.Var().Id("kafkaClient").Op("*").Qual("github.com/twmb/franz-go/pkg/kgo", "Client")
		group.If(List(Id("kafkaClient"), Err()).Op("=").Qual("github.com/twmb/franz-go/pkg/kgo", "NewClient").Call(Id("clientOptions").Op("...")), Err().Op("!=").Nil()).Block(Return(Nil(), Err()))
		values := Dict{
			Id("log"):         Id("log"),
			Id("client"):      Id("kafkaClient"),
			Id("commitAfter"): Id("setup").Dot("commitAfter"),
			Id("maxPoll"):     Id("setup").Dot("maxPollRecords"),
			Id("handlers"):    Make(Map(String()).Id("TopicHandler")),
		}
		if hasMetrics {
			values[Id("topics")] = Index().String().Values(r.topicLiterals()...)
		}
		group.Id("client").Op("=").Op("&").Id("Client").Values(values)
		if hasMetrics {
			group.If(Id("setup").Dot("metrics").Op("!=").Nil()).Block(
				If(List(Id("client").Dot("metrics"), Err()).Op("=").Id("newMetrics").Call(Id("setup").Dot("metrics")), Err().Op("!=").Nil()).Block(
					Id("kafkaClient").Dot("Close").Call(),
					Return(Nil(), Err()),
				),
				If(Id("setup").Dot("lagInterval").Op(">").Lit(0)).Block(Id("client").Dot("startLagLoop").Call(Id("setup").Dot("lagInterval"))),
			)
		}
		if hasTrace {
			group.If(Id("setup").Dot("tracerProvider").Op("!=").Nil()).Block(Id("client").Dot("tracer").Op("=").Id("setup").Dot("tracerProvider").Dot("Tracer").Call(Lit("tgp.kafka.subscriber")))
		}
		for _, contract := range r.contracts() {
			for _, method := range contract.Methods {
				r.writeTopicHandler(group, contract, method, hasMetrics, hasLog, hasTrace)
			}
		}
		group.Return(Id("client"), Nil())
	})
}

func (r *Renderer) writeClose(file GoFile, hasMetrics bool) {

	file.Line().Comment("Close останавливает Run и закрывает клиент Kafka.")
	file.Func().Params(Id("client").Op("*").Id("Client")).Id("Close").Params().BlockFunc(func(group *Group) {
		group.If(Id("client").Op("==").Nil()).Block(Return())
		group.Id("client").Dot("mu").Dot("Lock").Call()
		group.If(Id("client").Dot("closed")).Block(Id("client").Dot("mu").Dot("Unlock").Call(), Return())
		group.Id("client").Dot("closed").Op("=").True()
		group.Id("stop").Op(":=").Id("client").Dot("stop")
		group.Id("client").Dot("mu").Dot("Unlock").Call()
		group.If(Id("stop").Op("!=").Nil()).Block(Id("stop").Call())
		if hasMetrics {
			group.If(Id("client").Dot("lagStop").Op("!=").Nil()).Block(Id("client").Dot("lagStop").Call())
		}
		group.If(Id("client").Dot("client").Op("!=").Nil()).Block(Id("client").Dot("client").Dot("Close").Call())
	})
}

func (r *Renderer) writeRun(file GoFile, hasMetrics bool) {

	file.Line().Comment("Run запускает цикл чтения до отмены контекста или ошибки.")
	file.Func().Params(Id("client").Op("*").Id("Client")).Id("Run").Params(Id("ctx").Qual("context", "Context")).Params(Id("err").Error()).BlockFunc(func(group *Group) {
		group.If(Id("client").Op("==").Nil().Op("||").Id("client").Dot("client").Op("==").Nil()).Block(Return(Qual("fmt", "Errorf").Call(Lit("kafka subscriber: client is nil"))))
		group.Id("client").Dot("mu").Dot("Lock").Call()
		group.If(Id("client").Dot("closed")).Block(Id("client").Dot("mu").Dot("Unlock").Call(), Return(Qual("fmt", "Errorf").Call(Lit("kafka subscriber: client is closed"))))
		group.If(Id("client").Dot("running")).Block(Id("client").Dot("mu").Dot("Unlock").Call(), Return(Qual("fmt", "Errorf").Call(Lit("kafka subscriber: Run is already active"))))
		group.List(Id("runContext"), Id("stop")).Op(":=").Qual("context", "WithCancel").Call(Id("ctx"))
		group.Id("client").Dot("running").Op("=").True()
		group.Id("client").Dot("stop").Op("=").Id("stop")
		group.Id("client").Dot("mu").Dot("Unlock").Call()
		group.Defer().Func().Params().Block(
			Id("stop").Call(),
			Id("client").Dot("mu").Dot("Lock").Call(),
			Id("client").Dot("running").Op("=").False(),
			Id("client").Dot("stop").Op("=").Nil(),
			Id("client").Dot("mu").Dot("Unlock").Call(),
		).Call()
		group.Id("maxPoll").Op(":=").Id("client").Dot("maxPoll")
		group.If(Id("maxPoll").Op("<=").Lit(0)).Block(Id("maxPoll").Op("=").Lit(500))
		group.For().BlockFunc(func(loop *Group) {
			loop.If(Err().Op("=").Id("runContext").Dot("Err").Call(), Err().Op("!=").Nil()).Block(Return(Err()))
			if hasMetrics {
				loop.Id("pollStarted").Op(":=").Qual("time", "Now").Call()
				loop.If(Id("client").Dot("metrics").Op("!=").Nil()).Block(Id("client").Dot("metrics").Dot("pollActive").Dot("Set").Call(Lit(1)))
			}
			loop.Id("fetches").Op(":=").Id("client").Dot("client").Dot("PollRecords").Call(Id("runContext"), Id("maxPoll"))
			loop.If(Id("fetches").Dot("IsClientClosed").Call()).Block(
				r.pollFinished(hasMetrics, "ok"),
				Return(Nil()),
			)
			loop.If(List(Id("fetchErrs")).Op(":=").Id("fetches").Dot("Errors").Call(), Len(Id("fetchErrs")).Op(">").Lit(0)).BlockFunc(func(errors *Group) {
				errors.For(List(Id("_"), Id("fetchErr")).Op(":=").Range().Id("fetchErrs")).Block(
					If(Id("fetchErr").Dot("Err").Op("==").Id("runContext").Dot("Err").Call()).Block(Id("client").Dot("client").Dot("AllowRebalance").Call(), Return(Id("runContext").Dot("Err").Call())),
					Id("client").Dot("log").Dot("Error").Call(Lit("kafka fetch error"), Qual("log/slog", "String").Call(Lit("topic"), Id("fetchErr").Dot("Topic")), Qual("log/slog", "Int").Call(Lit("partition"), Int().Call(Id("fetchErr").Dot("Partition"))), Qual("log/slog", "Any").Call(Lit("error"), Id("fetchErr").Dot("Err"))),
				)
			})
			loop.Id("byTopic").Op(":=").Id("groupRecordsByTopic").Call(Id("fetches"))
			loop.If(Err().Op("=").Id("dispatchTopics").Call(Id("runContext"), Id("byTopic"), Id("client").Dot("handlers")), Err().Op("!=").Nil()).Block(
				r.pollFinished(hasMetrics, "error"),
				Id("client").Dot("client").Dot("AllowRebalance").Call(),
				Return(Err()),
			)
			loop.If(Id("client").Dot("commitAfter").Op("&&").Len(Id("byTopic")).Op(">").Lit(0)).Block(
				If(Err().Op("=").Id("client").Dot("client").Dot("CommitUncommittedOffsets").Call(Id("runContext")), Err().Op("!=").Nil()).Block(
					r.pollFinished(hasMetrics, "error"),
					Id("client").Dot("client").Dot("AllowRebalance").Call(),
					Return(Err()),
				),
			)
			loop.Add(r.pollFinished(hasMetrics, "ok"))
			loop.Id("client").Dot("client").Dot("AllowRebalance").Call()
		})
	})
}

func (r *Renderer) pollFinished(hasMetrics bool, result string) Code {

	if !hasMetrics {
		return Null()
	}
	return If(Id("client").Dot("metrics").Op("!=").Nil()).Block(
		Id("client").Dot("metrics").Dot("polls").Dot("WithLabelValues").Call(Lit(result), Lit("none")).Dot("Inc").Call(),
		Id("client").Dot("metrics").Dot("pollDuration").Dot("WithLabelValues").Call(Lit(result)).Dot("Observe").Call(Qual("time", "Since").Call(Id("pollStarted")).Dot("Seconds").Call()),
		Id("client").Dot("metrics").Dot("pollActive").Dot("Set").Call(Lit(0)),
	)
}

func (r *Renderer) topicLiterals() (topics []Code) {

	for _, topic := range r.topics() {
		topics = append(topics, Lit(topic))
	}
	return
}

func (r *Renderer) writeLag(file GoFile) {

	file.Line().Func().Params(Id("client").Op("*").Id("Client")).Id("startLagLoop").Params(Id("interval").Qual("time", "Duration")).Block(
		List(Id("ctx"), Id("stop")).Op(":=").Qual("context", "WithCancel").Call(Qual("context", "Background").Call()),
		Id("client").Dot("lagStop").Op("=").Id("stop"),
		Go().Id("client").Dot("runLagLoop").Call(Id("ctx"), Id("interval")),
	)
	file.Line().Func().Params(Id("client").Op("*").Id("Client")).Id("runLagLoop").Params(Id("ctx").Qual("context", "Context"), Id("interval").Qual("time", "Duration")).Block(
		Id("client").Dot("scrapeLag").Call(Id("ctx")),
		Id("ticker").Op(":=").Qual("time", "NewTicker").Call(Id("interval")),
		Defer().Id("ticker").Dot("Stop").Call(),
		For().Block(
			Select().Block(
				Case(Op("<-").Id("ctx").Dot("Done").Call()).Block(Return()),
				Case(Op("<-").Id("ticker").Dot("C")).Block(Id("client").Dot("scrapeLag").Call(Id("ctx"))),
			),
		),
	)
	file.Line().Func().Params(Id("client").Op("*").Id("Client")).Id("scrapeLag").Params(Id("ctx").Qual("context", "Context")).Block(
		Defer().Func().Params().Block(
			If(Id("recover").Call().Op("!=").Nil().Op("&&").Id("client").Dot("metrics").Op("!=").Nil()).Block(
				Id("client").Dot("metrics").Dot("lagScrapeErrors").Dot("WithLabelValues").Call(Lit("rebalance")).Dot("Inc").Call(),
			),
		).Call(),
		If(Id("client").Dot("metrics").Op("==").Nil().Op("||").Id("client").Dot("client").Op("==").Nil()).Block(Return()),
		Id("admin").Op(":=").Qual("github.com/twmb/franz-go/pkg/kadm", "NewClient").Call(Id("client").Dot("client")),
		List(Id("endOffsets"), Id("err")).Op(":=").Id("admin").Dot("ListEndOffsets").Call(Id("ctx"), Id("client").Dot("topics").Op("...")),
		If(Id("err").Op("!=").Nil()).Block(
			Id("client").Dot("metrics").Dot("lagScrapeErrors").Dot("WithLabelValues").Call(Lit("list_offsets")).Dot("Inc").Call(),
			Return(),
		),
		Id("positions").Op(":=").Id("client").Dot("client").Dot("UncommittedOffsets").Call(),
		If(Len(Id("positions")).Op("==").Lit(0)).Block(
			Id("positions").Op("=").Id("client").Dot("client").Dot("CommittedOffsets").Call(),
		),
		Id("client").Dot("metrics").Dot("lag").Dot("Reset").Call(),
		Id("endOffsets").Dot("Each").Call(Func().Params(Id("offset").Qual("github.com/twmb/franz-go/pkg/kadm", "ListedOffset")).Block(
			List(Id("position"), Id("ok")).Op(":=").Id("positions").Index(Id("offset").Dot("Topic")).Index(Id("offset").Dot("Partition")),
			If(Op("!").Id("ok")).Block(Return()),
			Id("client").Dot("metrics").Dot("lag").Dot("WithLabelValues").Call(Id("offset").Dot("Topic"), Qual("strconv", "Itoa").Call(Int().Call(Id("offset").Dot("Partition")))).Dot("Set").Call(Float64().Call(Id("offset").Dot("Offset").Op("-").Id("position").Dot("Offset"))),
		)),
	)
}
