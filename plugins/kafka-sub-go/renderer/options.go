// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (r *Renderer) renderOptions() (err error) {

	file := NewSrcFile(r.pkgName)
	hasMetrics := r.hasMetrics()
	hasTrace := r.hasTrace()

	file.Type().Id("registeredHandler").Struct(
		Id("kind").String(),
		Id("handler").Any(),
	)
	file.Line().Type().Id("setup").StructFunc(func(group *Group) {
		group.Id("brokers").Index().String()
		group.Id("group").String()
		group.Id("maxPollRecords").Int()
		group.Id("fetchMinBytes").Int32()
		group.Id("fetchMaxWait").Qual("time", "Duration")
		group.Id("resetPosition").Id("ResetPosition")
		group.Id("commitAfter").Bool()
		group.Id("commitAuto").Bool()
		group.Id("commitAfterSet").Bool()
		group.Id("commitAutoSet").Bool()
		group.Id("handlerConflict").String()
		group.Id("codecs").Map(String()).Id("codec")
		group.Id("tlsConfig").Op("*").Qual("crypto/tls", "Config")
		group.Id("authUser").String()
		group.Id("authPassword").String()
		group.Id("saslName").String()
		group.Id("clientOptions").Index().Qual("github.com/twmb/franz-go/pkg/kgo", "Opt")
		group.Id("handlers").Map(String()).Id("registeredHandler")
		group.Id("err").Error()
		if hasMetrics {
			group.Id("metrics").Qual("github.com/prometheus/client_golang/prometheus", "Registerer")
			group.Id("lagInterval").Qual("time", "Duration")
		}
		if hasTrace {
			group.Id("tracerProvider").Qual("go.opentelemetry.io/otel/trace", "TracerProvider")
		}
	})
	file.Line().Comment("Option настраивает подписчик Kafka.")
	file.Type().Id("Option").Func().Params(Id("setup").Op("*").Id("setup"))

	r.writeOptions(file, hasMetrics, hasTrace)
	return file.Save(filepath.Join(r.outDir, "options.go"))
}

func (r *Renderer) writeOptions(file GoFile, hasMetrics bool, hasTrace bool) {

	file.Line().Comment("Brokers задаёт адреса брокеров Kafka (host:port). Хотя бы один обязателен.")
	r.writeOption(file, "Brokers", Id("brokers").Op("...").String(), Id("setup").Dot("brokers").Op("=").Append(Id("setup").Dot("brokers"), Id("brokers").Op("...")))
	file.Line().Comment("Group задаёт consumer group. Обязателен для подписчика.")
	r.writeOption(file, "Group", Id("id").String(), Id("setup").Dot("group").Op("=").Id("id"))
	file.Line().Comment("MaxPollRecords — максимум записей за один PollRecords.")
	r.writeOption(file, "MaxPollRecords", Id("n").Int(), Id("setup").Dot("maxPollRecords").Op("=").Id("n"))
	file.Line().Comment("FetchMinBytes — минимальный объём данных перед fetch.")
	r.writeOption(file, "FetchMinBytes", Id("n").Int32(), Id("setup").Dot("fetchMinBytes").Op("=").Id("n"))
	file.Line().Comment("FetchMaxWait — максимум ожидания брокера при fetch.")
	r.writeOption(file, "FetchMaxWait", Id("duration").Qual("time", "Duration"), Id("setup").Dot("fetchMaxWait").Op("=").Id("duration"))
	file.Line().Comment("ResetOffset задаёт позицию чтения без committed offset.")
	r.writeOption(file, "ResetOffset", Id("position").Id("ResetPosition"), Id("setup").Dot("resetPosition").Op("=").Id("position"))
	file.Line().Comment("CommitAfterBatch выключает автокоммит offset.")
	r.writeOption(file, "CommitAfterBatch", Null(), Id("setup").Dot("commitAfter").Op("=").True(), Id("setup").Dot("commitAfterSet").Op("=").True())
	file.Line().Comment("CommitAuto включает автокоммит offset franz-go.")
	r.writeOption(file, "CommitAuto", Null(), Id("setup").Dot("commitAuto").Op("=").True(), Id("setup").Dot("commitAutoSet").Op("=").True())
	file.Line().Comment("Codec регистрирует или переопределяет кодек тела.")
	file.Func().Id("Codec").Params(Id("name").String(), Id("c").Id("codec")).Id("Option").Block(
		Return(Func().Params(Id("setup").Op("*").Id("setup")).Block(
			If(Id("name").Op("==").Lit("")).Block(
				Id("setup").Dot("err").Op("=").Qual("fmt", "Errorf").Call(Lit("kafka codec name is required")),
				Return(),
			),
			If(Id("c").Op("==").Nil()).Block(
				Id("setup").Dot("err").Op("=").Qual("fmt", "Errorf").Call(Lit("kafka codec %q is nil"), Id("name")),
				Return(),
			),
			Id("setup").Dot("codecs").Index(Id("name")).Op("=").Id("c"),
		)),
	)
	file.Line().Comment("TLS включает TLS к брокерам Kafka.")
	file.Func().Id("TLS").Params(Id("config").Op("*").Qual("crypto/tls", "Config")).Id("Option").Block(
		Return(Func().Params(Id("setup").Op("*").Id("setup")).Block(
			If(Id("config").Op("==").Nil()).Block(
				Id("setup").Dot("err").Op("=").Qual("fmt", "Errorf").Call(Lit("kafka TLS config is nil")),
				Return(),
			),
			Id("setup").Dot("tlsConfig").Op("=").Id("config"),
		)),
	)
	file.Line().Comment("Auth задаёт учётные данные SASL. Используется вместе с SASL.")
	file.Func().Id("Auth").Params(Id("user").String(), Id("password").String()).Id("Option").Block(
		Return(Func().Params(Id("setup").Op("*").Id("setup")).Block(
			If(Id("user").Op("==").Lit("")).Block(
				Id("setup").Dot("err").Op("=").Qual("fmt", "Errorf").Call(Lit("kafka Auth user is required")),
				Return(),
			),
			If(Id("password").Op("==").Lit("")).Block(
				Id("setup").Dot("err").Op("=").Qual("fmt", "Errorf").Call(Lit("kafka Auth password is required")),
				Return(),
			),
			Id("setup").Dot("authUser").Op("=").Id("user"),
			Id("setup").Dot("authPassword").Op("=").Id("password"),
		)),
	)
	file.Line().Comment("SASL задаёт механизм аутентификации: PLAIN, SCRAM-SHA-256 или SCRAM-SHA-512.")
	file.Func().Id("SASL").Params(Id("mechanism").String()).Id("Option").Block(
		Return(Func().Params(Id("setup").Op("*").Id("setup")).Block(
			Id("name").Op(":=").Qual("strings", "ToUpper").Call(Qual("strings", "TrimSpace").Call(Id("mechanism"))),
			Switch(Id("name")).Block(
				Case(Lit("PLAIN"), Lit("SCRAM-SHA-256"), Lit("SCRAM-SHA-512")).Block(
					Id("setup").Dot("saslName").Op("=").Id("name"),
				),
				Default().Block(
					Id("setup").Dot("err").Op("=").Qual("fmt", "Errorf").Call(Lit("kafka sasl mechanism %q is not supported"), Id("mechanism")),
				),
			),
		)),
	)
	file.Line().Comment("ClientOpt расширяет настройку franz-go. Не для TLS/SASL/Auth.")
	r.writeOption(file, "ClientOpt", Id("option").Qual("github.com/twmb/franz-go/pkg/kgo", "Opt"), Id("setup").Dot("clientOptions").Op("=").Append(Id("setup").Dot("clientOptions"), Id("option")))
	if hasMetrics {
		file.Line().Comment("Metrics включает Prometheus-метрики.")
		r.writeOption(file, "Metrics", Id("registerer").Qual("github.com/prometheus/client_golang/prometheus", "Registerer"),
			If(Id("registerer").Op("==").Nil()).Block(
				Id("setup").Dot("err").Op("=").Qual("fmt", "Errorf").Call(Lit("kafka metrics registerer is nil")),
				Return(),
			),
			Id("setup").Dot("metrics").Op("=").Id("registerer"),
		)
		file.Line().Comment("LagInterval задаёт период обновления лага чтения.")
		r.writeOption(file, "LagInterval", Id("duration").Qual("time", "Duration"), Id("setup").Dot("lagInterval").Op("=").Id("duration"))
	}
	if hasTrace {
		file.Line().Comment("Trace включает OpenTelemetry spans на обработчик чтения.")
		r.writeOption(file, "Trace", Id("provider").Qual("go.opentelemetry.io/otel/trace", "TracerProvider"),
			If(Id("provider").Op("==").Nil()).Block(
				Id("setup").Dot("err").Op("=").Qual("fmt", "Errorf").Call(Lit("kafka tracer provider is nil")),
				Return(),
			),
			Id("setup").Dot("tracerProvider").Op("=").Id("provider"),
		)
	}

	file.Line().Func().Id("defaultSetup").Params().Params(Id("result").Op("*").Id("setup")).BlockFunc(func(group *Group) {
		values := Dict{
			Id("maxPollRecords"): Lit(500),
			Id("fetchMinBytes"):  Lit(1),
			Id("fetchMaxWait"):   Lit(5).Op("*").Qual("time", "Second"),
			Id("resetPosition"):  Id("AtStart"),
			Id("commitAfter"):    True(),
			Id("codecs"):         Id("defaultCodecs").Call(),
			Id("handlers"):       Make(Map(String()).Id("registeredHandler")),
		}
		if hasMetrics {
			values[Id("lagInterval")] = Lit(15).Op("*").Qual("time", "Second")
		}
		group.Return(Op("&").Id("setup").Values(values))
	})
	file.Line().Func().Id("validateSetup").Params(Id("setup").Op("*").Id("setup")).Params(Id("err").Error()).Block(
		If(Id("setup").Dot("err").Op("!=").Nil()).Block(Return(Id("setup").Dot("err"))),
		If(Id("setup").Dot("commitAfterSet").Op("&&").Id("setup").Dot("commitAutoSet")).Block(
			Return(Qual("fmt", "Errorf").Call(
				Lit("kafka subscriber: CommitAfterBatch and CommitAuto cannot be combined"),
			)),
		),
		If(Id("setup").Dot("commitAutoSet")).Block(
			Id("setup").Dot("commitAfter").Op("=").False(),
			Id("setup").Dot("commitAuto").Op("=").True(),
		),
		If(Id("setup").Dot("handlerConflict").Op("!=").Lit("")).Block(
			Return(Qual("fmt", "Errorf").Call(
				Lit("kafka subscriber: multiple handler forms for contract %s"),
				Id("setup").Dot("handlerConflict"),
			)),
		),
		Id("hasAuth").Op(":=").Id("setup").Dot("authUser").Op("!=").Lit("").Op("||").Id("setup").Dot("authPassword").Op("!=").Lit(""),
		Id("hasSASL").Op(":=").Id("setup").Dot("saslName").Op("!=").Lit(""),
		If(Id("hasAuth").Op("!=").Id("hasSASL")).Block(
			Return(Qual("fmt", "Errorf").Call(Lit("kafka Auth and SASL must be set together"))),
		),
		Return(Nil()),
	)
}

func (r *Renderer) writeOption(file GoFile, name string, parameter Code, body ...Code) {

	file.Func().Id(name).Params(parameter).Id("Option").Block(
		Return(Func().Params(Id("setup").Op("*").Id("setup")).Block(body...)),
	)
}
