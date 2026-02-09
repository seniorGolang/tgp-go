// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (r *ClientRenderer) RenderClientMetrics() error {

	outDir := r.outDir
	srcFile := NewSrcFile(filepath.Base(outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackagePrometheus, "prometheus")
	srcFile.ImportName(PackagePrometheusAuto, "promauto")

	srcFile.Line().Type().Id("Metrics").Struct(
		Id("VersionGauge").Op("*").Qual(PackagePrometheus, "GaugeVec"),
		Id("RequestCount").Op("*").Qual(PackagePrometheus, "CounterVec"),
		Id("RequestCountAll").Op("*").Qual(PackagePrometheus, "CounterVec"),
		Id("RequestLatency").Op("*").Qual(PackagePrometheus, "HistogramVec"),
	).Line()

	srcFile.Line().Func().Params(Id("cli").Op("*").Id("Client")).Id("newMetrics").Params().Params(Id("m").Op("*").Id("Metrics"), Id("reg").Op("*").Qual(PackagePrometheus, "Registry")).BlockFunc(func(bg *Group) {
		bg.Id("reg").Op("=").Qual(PackagePrometheus, "NewRegistry").Call()
		bg.Id("m").Op("=").Op("&").Id("Metrics").Values(Dict{
			Id("VersionGauge"): Qual(PackagePrometheusAuto, "With").Call(Id("reg")).Dot("NewGaugeVec").Call(Qual(PackagePrometheus, "GaugeOpts").Values(
				DictFunc(func(d Dict) {
					d[Id("Name")] = Lit("count")
					d[Id("Namespace")] = Lit("client")
					d[Id("Subsystem")] = Lit("versions")
					d[Id("Help")] = Lit("Versions of client parts")
				}),
			), Index().String().Values(Lit("part"), Lit("version"), Lit("hostname"))),
			Id("RequestCount"): Qual(PackagePrometheusAuto, "With").Call(Id("reg")).Dot("NewCounterVec").Call(Qual(PackagePrometheus, "CounterOpts").Values(
				DictFunc(func(d Dict) {
					d[Id("Name")] = Lit("count")
					d[Id("Namespace")] = Lit("client")
					d[Id("Subsystem")] = Lit("requests")
					d[Id("Help")] = Lit("Number of requests sent")
				}),
			), Index().String().Values(Lit("service"), Lit("method"), Lit("success"), Lit("errCode"), Lit("client_id"))),
			Id("RequestCountAll"): Qual(PackagePrometheusAuto, "With").Call(Id("reg")).Dot("NewCounterVec").Call(Qual(PackagePrometheus, "CounterOpts").Values(
				DictFunc(func(d Dict) {
					d[Id("Name")] = Lit("all_count")
					d[Id("Namespace")] = Lit("client")
					d[Id("Subsystem")] = Lit("requests")
					d[Id("Help")] = Lit("Number of all requests sent")
				}),
			), Index().String().Values(Lit("service"), Lit("method"), Lit("success"), Lit("errCode"), Lit("client_id"))),
			Id("RequestLatency"): Qual(PackagePrometheusAuto, "With").Call(Id("reg")).Dot("NewHistogramVec").Call(Qual(PackagePrometheus, "HistogramOpts").Values(
				DictFunc(func(d Dict) {
					d[Id("Name")] = Lit("latency_seconds")
					d[Id("Namespace")] = Lit("client")
					d[Id("Subsystem")] = Lit("requests")
					d[Id("Help")] = Lit("Total duration of requests in seconds")
				}),
			), Index().String().Values(Lit("service"), Lit("method"), Lit("success"), Lit("errCode"), Lit("client_id"))),
		})
		bg.Id("m").Dot("VersionGauge").Dot("WithLabelValues").Call(Lit("astg"), Id("VersionASTg"), Id("cli").Dot("name")).Dot("Set").Call(Lit(1))
		bg.Return()
	})

	return srcFile.Save(path.Join(outDir, "metrics.go"))
}
