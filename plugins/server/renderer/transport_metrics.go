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
	for _, contract := range r.project.Contracts {
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
	srcFile.ImportName(PackagePrometheus, "prometheus")
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
		Id("PanicsTotal").Qual(PackagePrometheus, "Counter"),
		Id("ErrorResponsesTotal").Op("*").Qual(PackagePrometheus, "CounterVec"),
		Id("RequestsInFlight").Qual(PackagePrometheus, "Gauge"),
		Id("RequestDuration").Op("*").Qual(PackagePrometheus, "HistogramVec"),
		Id("BatchSize").Qual(PackagePrometheus, "Histogram"),
		Id("RequestCount").Op("*").Qual(PackagePrometheus, "CounterVec"),
		Id("RequestLatency").Op("*").Qual(PackagePrometheus, "HistogramVec"),
	)

	srcFile.Line().Add(r.newMetricsFunc())
	srcFile.Add(r.serveMetricsFunc())

	return srcFile.Save(metricsPath)
}

func (r *transportRenderer) newMetricsFunc() Code {

	return Func().Id("NewMetrics").
		Params().
		Params(Op("*").Id("Metrics")).
		BlockFunc(func(bg *Group) {
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
				Id("PanicsTotal"): Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewCounter").Call(
					Qual(PackagePrometheus, "CounterOpts").Values(Dict{
						Id("Help"):      Lit("Number of panics caught"),
						Id("Name"):      Lit("panics_total"),
						Id("Namespace"): Lit("service"),
					}),
				),
				Id("ErrorResponsesTotal"): Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewCounterVec").Call(
					Qual(PackagePrometheus, "CounterOpts").Values(Dict{
						Id("Help"):      Lit("Error responses sent to clients"),
						Id("Name"):      Lit("error_responses_total"),
						Id("Namespace"): Lit("service"),
					}),
					Index().String().Values(Lit("protocol"), Lit("code"), Lit("client_id")),
				),
				Id("RequestsInFlight"): Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewGauge").Call(
					Qual(PackagePrometheus, "GaugeOpts").Values(Dict{
						Id("Help"):      Lit("Current number of HTTP requests being processed"),
						Id("Name"):      Lit("requests_in_flight"),
						Id("Namespace"): Lit("service"),
					}),
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
				Id("BatchSize"): Qual("github.com/prometheus/client_golang/prometheus/promauto", "NewHistogram").Call(
					Qual(PackagePrometheus, "HistogramOpts").Values(Dict{
						Id("Help"):      Lit("Distribution of batch size"),
						Id("Name"):      Lit("batch_size"),
						Id("Namespace"): Lit("service"),
					}),
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
		Params(Id("log").Op("*").Qual(PackageSlog, "Logger"), Id("path").String(), Id("address").String()).
		Block(
			Id("srv").Dot("srvMetrics").Op("=").Qual(PackageFiber, "New").Call(Qual(PackageFiber, "Config").Values(Dict{Id("DisableStartupMessage"): True()})),
			Id("srv").Dot("srvMetrics").Dot("All").Call(Id("path"), Qual(PackageFiberAdaptor, "HTTPHandler").Call(Qual(PackagePrometheusHttp, "Handler").Call())),
			Go().Func().Params().Block(
				Err().Op(":=").Id("srv").Dot("srvMetrics").Dot("Listen").Call(Id("address")),
				Id("ExitOnError").Call(Id("log"), Err(), Lit("serve metrics on ").Op("+").Id("address")),
			).Call(),
		)
}
