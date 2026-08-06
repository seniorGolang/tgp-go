// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive
)

func (r *transportRenderer) RenderTransportMetrics() (err error) {

	metricsPath := path.Join(r.outDir, "metrics.go")

	if !r.hasMetrics() {
		return
	}

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.ImportName("os", "os")
	srcFile.ImportName(PackageSync, "sync")
	srcFile.ImportName(PackagePrometheus, "prometheus")
	srcFile.ImportName(PackagePrometheusCollectors, "collectors")
	srcFile.ImportName("github.com/prometheus/client_golang/prometheus/promauto", "promauto")
	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(PackageFiberAdaptor, "adaptor")
	srcFile.ImportName(PackagePrometheusHTTP, "promhttp")

	srcFile.Line().Const().Defs(
		Id("metricSuccessTrue").Op("=").Lit("true"),
		Id("metricSuccessFalse").Op("=").Lit("false"),
		Id("metricErrCodeSuccess").Op("=").Lit("0"),
	)

	srcFile.Line().Type().Id("Metrics").StructFunc(func(sg *Group) {
		sg.Id("VersionGauge").Op("*").Qual(PackagePrometheus, "GaugeVec")
		sg.Id("EntryRequestsTotal").Op("*").Qual(PackagePrometheus, "CounterVec")
		sg.Id("PanicsTotal").Op("*").Qual(PackagePrometheus, "CounterVec")
		sg.Id("ErrorResponsesTotal").Op("*").Qual(PackagePrometheus, "CounterVec")
		sg.Id("RequestsInFlight").Op("*").Qual(PackagePrometheus, "GaugeVec")
		sg.Id("RequestDuration").Op("*").Qual(PackagePrometheus, "HistogramVec")
		sg.Id("BatchSize").Op("*").Qual(PackagePrometheus, "HistogramVec")
		sg.Id("RequestCount").Op("*").Qual(PackagePrometheus, "CounterVec")
		sg.Id("RequestLatency").Op("*").Qual(PackagePrometheus, "HistogramVec")
		if r.hasBearerSecurity() {
			sg.Id("AuthDenied").Op("*").Qual(PackagePrometheus, "CounterVec")
		}
	})

	srcFile.Line().Var().Id("registerGoCollectorOnce").Qual(PackageSync, "Once")

	srcFile.Line().Var().Id("defaultSecondsBuckets").Op("=").Add(secondsBucketsLit())

	srcFile.Line().Add(r.validBucketsFunc())
	srcFile.Line().Add(r.newMetricsFunc())
	srcFile.Line().Add(r.newMetricsWithBucketsFunc())
	srcFile.Add(r.serveMetricsFunc())

	err = srcFile.Save(metricsPath)
	return
}

func secondsBucketsLit() Code {

	vals := make([]Code, 0, len(defaultSecondsBucketValues))
	for _, bucket := range defaultSecondsBucketValues {
		vals = append(vals, Lit(bucket))
	}
	return Index().Float64().Values(vals...)
}

var defaultSecondsBucketValues = []float64{
	0.001, 0.0025, 0.005, 0.0075, 0.01, 0.015, 0.02, 0.025, 0.035, 0.05,
	0.075, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

func (r *transportRenderer) validBucketsFunc() (c Code) {

	return Func().Id("validBuckets").
		Params(Id("buckets").Index().Float64()).
		Params(Id("ok").Bool()).
		BlockFunc(func(bg *Group) {
			bg.If(Len(Id("buckets")).Op("==").Lit(0)).Block(Return())
			bg.For(List(Id("i"), Id("bucket")).Op(":=").Range().Id("buckets")).Block(
				If(Id("bucket").Op("<=").Lit(0).Op("||").Parens(Id("i").Op(">").Lit(0).Op("&&").Id("bucket").Op("<=").Id("buckets").Index(Id("i").Op("-").Lit(1)))).Block(
					Return(),
				),
			)
			bg.Id("ok").Op("=").True()
			bg.Return()
		})
}

func (r *transportRenderer) newMetricsFunc() (c Code) {

	return Func().Id("NewMetrics").
		Params().
		Params(Id("m").Op("*").Id("Metrics")).
		Block(
			Id("m").Op("=").Id("newMetrics").Call(Nil(), Nil()),
			Return(),
		)
}

func (r *transportRenderer) newMetricsWithBucketsFunc() (c Code) {

	return Func().Id("newMetrics").
		Params(Id("durationBuckets").Index().Float64(), Id("latencyBuckets").Index().Float64()).
		Params(Id("m").Op("*").Id("Metrics")).
		BlockFunc(func(bg *Group) {
			bg.Line().Id("registerGoCollectorOnce").Dot("Do").Call(Func().Params().Block(
				Op("_").Op("=").Qual(PackagePrometheus, "Register").Call(Qual(PackagePrometheusCollectors, "NewGoCollector").Call()),
			))
			bg.List(Id("hostname"), Id("_")).Op(":=").Qual("os", "Hostname").Call()
			bg.If(Len(Id("durationBuckets")).Op("==").Lit(0)).Block(
				Id("durationBuckets").Op("=").Id("defaultSecondsBuckets"),
			)
			bg.If(Len(Id("latencyBuckets")).Op("==").Lit(0)).Block(
				Id("latencyBuckets").Op("=").Id("defaultSecondsBuckets"),
			)
			bg.Id("m").Op("=").Op("&").Id("Metrics").Values(DictFunc(func(d Dict) {
				d[Id("EntryRequestsTotal")] = Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewCounterVec").Call(
					Qual(PackagePrometheus, "CounterOpts").Values(Dict{
						Id("Help"):      Lit("Incoming HTTP requests to all endpoints"),
						Id("Name"):      Lit("entry_requests_total"),
						Id("Namespace"): Lit("service"),
					}),
					Index().String().Values(Lit("protocol"), Lit("result"), Lit("client_id")),
				)
				d[Id("PanicsTotal")] = Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewCounterVec").Call(
					Qual(PackagePrometheus, "CounterOpts").Values(Dict{
						Id("Help"):      Lit("Number of panics caught"),
						Id("Name"):      Lit("panics_total"),
						Id("Namespace"): Lit("service"),
					}),
					Index().String().Values(Lit("path"), Lit("method"), Lit("client_id")),
				)
				d[Id("ErrorResponsesTotal")] = Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewCounterVec").Call(
					Qual(PackagePrometheus, "CounterOpts").Values(Dict{
						Id("Help"):      Lit("Error responses sent to clients"),
						Id("Name"):      Lit("error_responses_total"),
						Id("Namespace"): Lit("service"),
					}),
					Index().String().Values(Lit("protocol"), Lit("code"), Lit("client_id")),
				)
				d[Id("RequestsInFlight")] = Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewGaugeVec").Call(
					Qual(PackagePrometheus, "GaugeOpts").Values(Dict{
						Id("Help"):      Lit("Current number of HTTP requests being processed"),
						Id("Name"):      Lit("requests_in_flight"),
						Id("Namespace"): Lit("service"),
					}),
					Index().String().Values(Lit("path"), Lit("client_id")),
				)
				d[Id("RequestDuration")] = Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewHistogramVec").Call(
					Qual(PackagePrometheus, "HistogramOpts").Values(Dict{
						Id("Help"):      Lit("Full HTTP request duration in seconds"),
						Id("Name"):      Lit("request_duration_seconds"),
						Id("Namespace"): Lit("service"),
						Id("Buckets"):   Id("durationBuckets"),
					}),
					Index().String().Values(Lit("client_id")),
				)
				d[Id("BatchSize")] = Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewHistogramVec").Call(
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
				)
				d[Id("RequestCount")] = Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewCounterVec").Call(
					Qual(PackagePrometheus, "CounterOpts").Values(Dict{
						Id("Help"):      Lit("Number of method calls processed"),
						Id("Name"):      Lit("requests_count"),
						Id("Namespace"): Lit("service"),
					}),
					Index().String().Values(Lit("service"), Lit("method"), Lit("success"), Lit("errCode"), Lit("client_id")),
				)
				d[Id("RequestLatency")] = Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewHistogramVec").Call(
					Qual(PackagePrometheus, "HistogramOpts").Values(Dict{
						Id("Help"):      Lit("Method execution latency in seconds"),
						Id("Name"):      Lit("requests_latency_seconds"),
						Id("Namespace"): Lit("service"),
						Id("Buckets"):   Id("latencyBuckets"),
					}),
					Index().String().Values(Lit("service"), Lit("method"), Lit("success"), Lit("errCode"), Lit("client_id")),
				)
				d[Id("VersionGauge")] = Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewGaugeVec").Call(
					Qual(PackagePrometheus, "GaugeOpts").Values(Dict{
						Id("Help"):      Lit("Versions of service parts"),
						Id("Name"):      Lit("count"),
						Id("Namespace"): Lit("service"),
						Id("Subsystem"): Lit("versions"),
					}),
					Index().String().Values(Lit("part"), Lit("version"), Lit("hostname")),
				)
				if r.hasBearerSecurity() {
					d[Id("AuthDenied")] = Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewCounterVec").Call(
						Qual(PackagePrometheus, "CounterOpts").Values(Dict{
							Id("Help"):      Lit("Bearer authentication denials"),
							Id("Name"):      Lit("auth_denied_total"),
							Id("Namespace"): Lit("service"),
						}),
						Index().String().Values(Lit("reason")),
					)
				}
			}))
			bg.Id("m").Dot("VersionGauge").Dot("WithLabelValues").Call(Lit("astg"), Id("VersionASTg"), Id("hostname")).Dot("Set").Call(Lit(1))
			bg.Return()
		})
}

func (r *transportRenderer) serveMetricsFunc() (c Code) {

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
			g.Id("handler").Op(":=").Qual(PackagePrometheusHTTP, "HandlerFor").Call(
				Qual(PackagePrometheus, "Gatherers").Call(Id("gatherers")),
				Qual(PackagePrometheusHTTP, "HandlerOpts").Values(Dict{}),
			)
			g.Id("srv").Dot("srvMetrics").Dot("All").Call(Id("path"), Qual(PackageFiberAdaptor, "HTTPHandler").Call(Id("handler")))
			g.Go().Func().Params().Block(
				Err().Op(":=").Id("srv").Dot("srvMetrics").Dot("Listen").Call(Id("address")),
				Id("ExitOnError").Call(Id("log"), Err(), Lit("serve metrics on ").Op("+").Id("address")),
			).Call()
		})
}
