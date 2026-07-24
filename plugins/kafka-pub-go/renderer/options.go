// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path/filepath"

	"github.com/dave/jennifer/jen"

	"tgp/internal/model"
)

const (
	kgoPath        = "github.com/twmb/franz-go/pkg/kgo"
	prometheusPath = "github.com/prometheus/client_golang/prometheus"
	tracePath      = "go.opentelemetry.io/otel/trace"
)

func (r *Renderer) renderOptions() (err error) {

	source := newSrcFile(filepath.Base(r.outDir))
	source.Type().Id("setup").StructFunc(func(group *jen.Group) {
		group.Id("brokers").Index().String()
		group.Id("batchMaxLinger").Qual("time", "Duration")
		group.Id("batchMaxBytes").Int32()
		group.Id("maxBufferedRecords").Int()
		group.Id("compression").Index().Qual(kgoPath, "CompressionCodec")
		group.Id("codecs").Map(jen.String()).Id("codec")
		group.Id("tlsConfig").Op("*").Qual("crypto/tls", "Config")
		group.Id("authUser").String()
		group.Id("authPassword").String()
		group.Id("saslName").String()
		group.Id("clientOptions").Index().Qual(kgoPath, "Opt")
		if r.hasAnnotation(model.TagMetrics) {
			group.Id("metrics").Qual(prometheusPath, "Registerer")
		}
		if r.hasAnnotation(model.TagTrace) {
			group.Id("tracer").Qual(tracePath, "Tracer")
		}
	})
	source.Line()
	source.Type().Id("Option").Func().Params(jen.Id("setup").Op("*").Id("setup")).Params(jen.Id("err").Error())
	source.Line()
	source.Func().Id("defaultSetup").Params().Params(jen.Id("result").Id("setup")).Block(
		jen.Return(jen.Id("setup").Values(jen.Dict{
			jen.Id("batchMaxLinger"):     jen.Lit(10).Op("*").Qual("time", "Millisecond"),
			jen.Id("batchMaxBytes"):      jen.Lit(1000012),
			jen.Id("maxBufferedRecords"): jen.Lit(10000),
			jen.Id("compression"):        jen.Index().Qual(kgoPath, "CompressionCodec").Values(jen.Qual(kgoPath, "NoCompression").Call()),
			jen.Id("codecs"):             jen.Make(jen.Map(jen.String()).Id("codec")),
		})),
	)
	source.Line()
	r.addOption(source, "Brokers", "задаёт адреса брокеров Kafka (host:port). Хотя бы один обязателен.", jen.Id("brokers").Op("...").String(), jen.Id("setup").Dot("brokers").Op("=").Append(jen.Id("setup").Dot("brokers"), jen.Id("brokers").Op("...")))
	r.addOption(source, "BatchMaxLinger", "задаёт максимальное время накопления записей перед отправкой.", jen.Id("duration").Qual("time", "Duration"), jen.Id("setup").Dot("batchMaxLinger").Op("=").Id("duration"))
	r.addOption(source, "BatchMaxBytes", "задаёт максимальный размер пакета на partition.", jen.Id("bytes").Int32(), jen.Id("setup").Dot("batchMaxBytes").Op("=").Id("bytes"))
	r.addOption(source, "MaxBufferedRecords", "задаёт максимум записей в буфере клиента.", jen.Id("records").Int(), jen.Id("setup").Dot("maxBufferedRecords").Op("=").Id("records"))
	source.Comment("Compression задаёт алгоритмы сжатия пакетов Kafka.")
	source.Func().Id("Compression").Params(jen.Id("codecs").Op("...").Qual(kgoPath, "CompressionCodec")).Params(jen.Id("option").Id("Option")).Block(
		jen.Return(jen.Func().Params(jen.Id("setup").Op("*").Id("setup")).Params(jen.Id("err").Error()).Block(
			jen.If(jen.Len(jen.Id("codecs")).Op("==").Lit(0)).Block(jen.Return(jen.Qual("errors", "New").Call(jen.Lit("kafka publisher compression requires at least one codec")))),
			jen.Id("setup").Dot("compression").Op("=").Id("codecs"),
			jen.Return(jen.Nil()),
		)),
	)
	source.Line()
	source.Comment("Codec регистрирует или переопределяет кодек тела по имени.")
	source.Func().Id("Codec").Params(jen.Id("name").String(), jen.Id("value").Id("codec")).Params(jen.Id("option").Id("Option")).Block(
		jen.Return(jen.Func().Params(jen.Id("setup").Op("*").Id("setup")).Params(jen.Id("err").Error()).Block(
			jen.If(jen.Id("name").Op("==").Lit("")).Block(jen.Return(jen.Qual("errors", "New").Call(jen.Lit("kafka codec name is required")))),
			jen.Id("setup").Dot("codecs").Index(jen.Id("name")).Op("=").Id("value"),
			jen.Return(jen.Nil()),
		)),
	)
	source.Line()
	source.Comment("TLS включает TLS к брокерам Kafka.")
	source.Func().Id("TLS").Params(jen.Id("config").Op("*").Qual("crypto/tls", "Config")).Params(jen.Id("option").Id("Option")).Block(
		jen.Return(jen.Func().Params(jen.Id("setup").Op("*").Id("setup")).Params(jen.Id("err").Error()).Block(
			jen.If(jen.Id("config").Op("==").Nil()).Block(jen.Return(jen.Qual("errors", "New").Call(jen.Lit("kafka TLS config is nil")))),
			jen.Id("setup").Dot("tlsConfig").Op("=").Id("config"),
			jen.Return(jen.Nil()),
		)),
	)
	source.Line()
	source.Comment("Auth задаёт учётные данные SASL. Используется вместе с SASL.")
	source.Func().Id("Auth").Params(jen.Id("user").String(), jen.Id("password").String()).Params(jen.Id("option").Id("Option")).Block(
		jen.Return(jen.Func().Params(jen.Id("setup").Op("*").Id("setup")).Params(jen.Id("err").Error()).Block(
			jen.If(jen.Id("user").Op("==").Lit("")).Block(jen.Return(jen.Qual("errors", "New").Call(jen.Lit("kafka Auth user is required")))),
			jen.If(jen.Id("password").Op("==").Lit("")).Block(jen.Return(jen.Qual("errors", "New").Call(jen.Lit("kafka Auth password is required")))),
			jen.Id("setup").Dot("authUser").Op("=").Id("user"),
			jen.Id("setup").Dot("authPassword").Op("=").Id("password"),
			jen.Return(jen.Nil()),
		)),
	)
	source.Line()
	source.Comment("SASL задаёт механизм аутентификации: PLAIN, SCRAM-SHA-256 или SCRAM-SHA-512.")
	source.Func().Id("SASL").Params(jen.Id("mechanism").String()).Params(jen.Id("option").Id("Option")).Block(
		jen.Return(jen.Func().Params(jen.Id("setup").Op("*").Id("setup")).Params(jen.Id("err").Error()).Block(
			jen.Id("name").Op(":=").Qual("strings", "ToUpper").Call(jen.Qual("strings", "TrimSpace").Call(jen.Id("mechanism"))),
			jen.Switch(jen.Id("name")).Block(
				jen.Case(jen.Lit("PLAIN"), jen.Lit("SCRAM-SHA-256"), jen.Lit("SCRAM-SHA-512")).Block(
					jen.Id("setup").Dot("saslName").Op("=").Id("name"),
					jen.Return(jen.Nil()),
				),
				jen.Default().Block(
					jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("kafka sasl mechanism %q is not supported"), jen.Id("mechanism"))),
				),
			),
		)),
	)
	source.Line()
	r.addOption(source, "ClientOpt", "передаёт дополнительную franz-go опцию всем клиентам отправки. Не для TLS/SASL/Auth.", jen.Id("clientOption").Qual(kgoPath, "Opt"), jen.Id("setup").Dot("clientOptions").Op("=").Append(jen.Id("setup").Dot("clientOptions"), jen.Id("clientOption")))
	if r.hasAnnotation(model.TagMetrics) {
		r.addRequiredOption(source, "Metrics", "включает сбор Prometheus-метрик в заданном реестре.", "registerer", jen.Id("registerer").Qual(prometheusPath, "Registerer"), "kafka metrics registerer is nil", jen.Id("setup").Dot("metrics").Op("=").Id("registerer"))
	}
	if r.hasAnnotation(model.TagTrace) {
		r.addRequiredOption(source, "Trace", "включает OpenTelemetry-трейсинг операций публикации.", "provider", jen.Id("provider").Qual(tracePath, "TracerProvider"), "kafka tracer provider is nil", jen.Id("setup").Dot("tracer").Op("=").Id("provider").Dot("Tracer").Call(jen.Lit("tgp.kafka.publisher")))
	}
	source.Func().Id("validateSecurity").Params(jen.Id("setup").Id("setup")).Params(jen.Id("err").Error()).Block(
		jen.Id("hasAuth").Op(":=").Id("setup").Dot("authUser").Op("!=").Lit("").Op("||").Id("setup").Dot("authPassword").Op("!=").Lit(""),
		jen.Id("hasSASL").Op(":=").Id("setup").Dot("saslName").Op("!=").Lit(""),
		jen.If(jen.Id("hasAuth").Op("!=").Id("hasSASL")).Block(
			jen.Return(jen.Qual("errors", "New").Call(jen.Lit("kafka Auth and SASL must be set together"))),
		),
		jen.Return(jen.Nil()),
	)
	source.Line()
	return source.Save(filepath.Join(r.outDir, "options.go"))
}

func (r *Renderer) addOption(source *GoFile, name string, comment string, parameter jen.Code, assignment jen.Code) {

	source.Comment(name + " " + comment)
	source.Func().Id(name).Params(parameter).Params(jen.Id("option").Id("Option")).Block(
		jen.Return(jen.Func().Params(jen.Id("setup").Op("*").Id("setup")).Params(jen.Id("err").Error()).Block(assignment, jen.Return(jen.Nil()))),
	)
	source.Line()
}

func (r *Renderer) addRequiredOption(source *GoFile, name string, comment string, parameterName string, parameter jen.Code, errorText string, assignment jen.Code) {

	source.Comment(name + " " + comment)
	source.Func().Id(name).Params(parameter).Params(jen.Id("option").Id("Option")).Block(
		jen.Return(jen.Func().Params(jen.Id("setup").Op("*").Id("setup")).Params(jen.Id("err").Error()).Block(
			jen.If(jen.Id(parameterName).Op("==").Nil()).Block(jen.Return(jen.Qual("errors", "New").Call(jen.Lit(errorText)))),
			assignment,
			jen.Return(jen.Nil()),
		)),
	)
	source.Line()
}
