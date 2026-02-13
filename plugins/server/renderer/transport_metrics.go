// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

func (r *transportRenderer) RenderTransportMetrics() error {

	metricsPath := path.Join(r.outDir, "metrics.go")

	hasMetrics := false
	for _, contract := range r.contractsSorted() {
		if model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
			hasMetrics = true
			break
		}
	}
	if !hasMetrics {
		return nil
	}

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName("os", "os")
	srcFile.ImportName(PackageSync, "sync")
	srcFile.ImportName(PackagePrometheus, "prometheus")
	srcFile.ImportName(PackagePrometheusCollectors, "collectors")
	srcFile.ImportName("github.com/prometheus/client_golang/prometheus/promauto", "promauto")
	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(PackageFiberAdaptor, "adaptor")
	srcFile.ImportName(PackagePrometheusHttp, "promhttp")

	srcFile.Line().Const().Defs(
		Id("metricSuccessTrue").Op("=").Lit("true"),
		Id("metricSuccessFalse").Op("=").Lit("false"),
		Id("metricErrCodeSuccess").Op("=").Lit("0"),
	)

	srcFile.Line().Type().Id("Metrics").Struct(
		Id("VersionGauge").Op("*").Qual(PackagePrometheus, "GaugeVec"),
		Id("EntryRequestsTotal").Op("*").Qual(PackagePrometheus, "CounterVec"),
		Id("PanicsTotal").Op("*").Qual(PackagePrometheus, "CounterVec"),
		Id("ErrorResponsesTotal").Op("*").Qual(PackagePrometheus, "CounterVec"),
		Id("RequestsInFlight").Op("*").Qual(PackagePrometheus, "GaugeVec"),
		Id("RequestDuration").Op("*").Qual(PackagePrometheus, "HistogramVec"),
		Id("BatchSize").Op("*").Qual(PackagePrometheus, "HistogramVec"),
		Id("RequestCount").Op("*").Qual(PackagePrometheus, "CounterVec"),
		Id("RequestLatency").Op("*").Qual(PackagePrometheus, "HistogramVec"),
	)

	srcFile.Line().Var().Id("registerGoCollectorOnce").Qual(PackageSync, "Once")

	srcFile.Line().Add(r.newMetricsFunc())
	srcFile.Add(r.serveMetricsFunc())

	return srcFile.Save(metricsPath)
}

func (r *transportRenderer) newMetricsFunc() Code {

	return Func().Id("NewMetrics").
		Params().
		Params(Op("*").Id("Metrics")).
		BlockFunc(func(bg *Group) {
			bg.Line().Id("registerGoCollectorOnce").Dot("Do").Call(Func().Params().Block(
				Op("_").Op("=").Qual(PackagePrometheus, "Register").Call(Qual(PackagePrometheusCollectors, "NewGoCollector").Call()),
			))
			bg.List(Id("hostname"), Id("_")).Op(":=").Qual("os", "Hostname").Call()
			buckets := Index().Float64().Values(
				Lit(0.001), Lit(0.005), Lit(0.01), Lit(0.025), Lit(0.05), Lit(0.1),
				Lit(0.25), Lit(0.5), Lit(1), Lit(2.5), Lit(5), Lit(10),
			)
			bg.Id("m").Op(":=").Op("&").Id("Metrics").Values(Dict{
				Id("EntryRequestsTotal"): Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewCounterVec").Call(
					Qual(PackagePrometheus, "CounterOpts").Values(Dict{
						Id("Help"):      Lit("Incoming HTTP requests to all endpoints"),
						Id("Name"):      Lit("entry_requests_total"),
						Id("Namespace"): Lit("service"),
					}),
					Index().String().Values(Lit("protocol"), Lit("result"), Lit("client_id")),
				),
				Id("PanicsTotal"): Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewCounterVec").Call(
					Qual(PackagePrometheus, "CounterOpts").Values(Dict{
						Id("Help"):      Lit("Number of panics caught"),
						Id("Name"):      Lit("panics_total"),
						Id("Namespace"): Lit("service"),
					}),
					Index().String().Values(Lit("path"), Lit("method"), Lit("client_id")),
				),
				Id("ErrorResponsesTotal"): Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewCounterVec").Call(
					Qual(PackagePrometheus, "CounterOpts").Values(Dict{
						Id("Help"):      Lit("Error responses sent to clients"),
						Id("Name"):      Lit("error_responses_total"),
						Id("Namespace"): Lit("service"),
					}),
					Index().String().Values(Lit("protocol"), Lit("code"), Lit("client_id")),
				),
				Id("RequestsInFlight"): Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewGaugeVec").Call(
					Qual(PackagePrometheus, "GaugeOpts").Values(Dict{
						Id("Help"):      Lit("Current number of HTTP requests being processed"),
						Id("Name"):      Lit("requests_in_flight"),
						Id("Namespace"): Lit("service"),
					}),
					Index().String().Values(Lit("path"), Lit("client_id")),
				),
				Id("RequestDuration"): Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewHistogramVec").Call(
					Qual(PackagePrometheus, "HistogramOpts").Values(Dict{
						Id("Help"):      Lit("Full HTTP request duration in seconds"),
						Id("Name"):      Lit("request_duration_seconds"),
						Id("Namespace"): Lit("service"),
						Id("Buckets"):   buckets,
					}),
					Index().String().Values(Lit("client_id")),
				),
				Id("BatchSize"): Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewHistogramVec").Call(
					Qual(PackagePrometheus, "HistogramOpts").Values(Dict{
						Id("Help"):      Lit("Distribution of batch size"),
						Id("Name"):      Lit("batch_size"),
						Id("Namespace"): Lit("service"),
						Id("Buckets"): Index().Float64().Values(
							Lit(1), Lit(5), Lit(10), Lit(25), Lit(50), Lit(100),
							Lit(200), Lit(400), Lit(600), Lit(800), Lit(1000),
						),
					}),
					Index().String().Values(Lit("protocol"), Lit("endpoint"), Lit("client_id")),
				),
				Id("RequestCount"): Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewCounterVec").Call(
					Qual(PackagePrometheus, "CounterOpts").Values(Dict{
						Id("Help"):      Lit("Number of method calls processed"),
						Id("Name"):      Lit("requests_count"),
						Id("Namespace"): Lit("service"),
					}),
					Index().String().Values(Lit("service"), Lit("method"), Lit("success"), Lit("errCode"), Lit("client_id")),
				),
				Id("RequestLatency"): Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewHistogramVec").Call(
					Qual(PackagePrometheus, "HistogramOpts").Values(Dict{
						Id("Help"):      Lit("Method execution latency in seconds"),
						Id("Name"):      Lit("requests_latency_seconds"),
						Id("Namespace"): Lit("service"),
						Id("Buckets"):   buckets,
					}),
					Index().String().Values(Lit("service"), Lit("method"), Lit("success"), Lit("errCode"), Lit("client_id")),
				),
				Id("VersionGauge"): Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewGaugeVec").Call(
					Qual(PackagePrometheus, "GaugeOpts").Values(Dict{
						Id("Help"):      Lit("Versions of service parts"),
						Id("Name"):      Lit("count"),
						Id("Namespace"): Lit("service"),
						Id("Subsystem"): Lit("versions"),
					}),
					Index().String().Values(Lit("part"), Lit("version"), Lit("hostname")),
				),
			})
			bg.Id("m").Dot("VersionGauge").Dot("WithLabelValues").Call(Lit("astg"), Id("VersionASTg"), Id("hostname")).Dot("Set").Call(Lit(1))
			bg.Return(Id("m"))
		})
}

func (r *transportRenderer) serveMetricsFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("ServeMetrics").
		Params(
			Id("log").Op("*").Qual(PackageSlog, "Logger"),
			Id("path").String(),
			Id("address").String(),
			Id("regs").Op("...").Op("*").Qual(PackagePrometheus, "Registry"),
		).
		BlockFunc(func(g *Group) {
			g.Id("srv").Dot("srvMetrics").Op("=").Qual(PackageFiber, "New").Call(Qual(PackageFiber, "Config").Values(Dict{Id("DisableStartupMessage"): True()}))
			g.Id("gatherers").Op(":=").Index().Qual(PackagePrometheus, "Gatherer").Values(Qual(PackagePrometheus, "DefaultGatherer"))
			g.For(List(Id("_"), Id("r")).Op(":=").Range().Id("regs")).Block(
				If(Id("r").Op("!=").Nil()).Block(
					Id("gatherers").Op("=").Append(Id("gatherers"), Id("r")),
				),
			)
			g.Id("handler").Op(":=").Qual(PackagePrometheusHttp, "HandlerFor").Call(
				Qual(PackagePrometheus, "Gatherers").Call(Id("gatherers")),
				Qual(PackagePrometheusHttp, "HandlerOpts").Values(Dict{}),
			)
			g.Id("srv").Dot("srvMetrics").Dot("All").Call(Id("path"), Qual(PackageFiberAdaptor, "HTTPHandler").Call(Id("handler")))
			g.Go().Func().Params().Block(
				Err().Op(":=").Id("srv").Dot("srvMetrics").Dot("Listen").Call(Id("address")),
				Id("ExitOnError").Call(Id("log"), Err(), Lit("serve metrics on ").Op("+").Id("address")),
			).Call()
		})
}
