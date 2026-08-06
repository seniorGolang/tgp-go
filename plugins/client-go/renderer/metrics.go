// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive
)

func (r *ClientRenderer) RenderClientMetrics() (err error) {

	outDir := r.outDir
	srcFile := NewSrcFile(filepath.Base(outDir))
	srcFile.ImportName(PackagePrometheus, "prometheus")
	srcFile.ImportName(PackagePrometheusAuto, "promauto")

	srcFile.Line().Var().Id("defaultSecondsBuckets").Op("=").Add(clientSecondsBucketsLit())

	srcFile.Line().Type().Id("Metrics").Struct(
		Id("VersionGauge").Op("*").Qual(PackagePrometheus, "GaugeVec"),
		Id("RequestCount").Op("*").Qual(PackagePrometheus, "CounterVec"),
		Id("RequestCountAll").Op("*").Qual(PackagePrometheus, "CounterVec"),
		Id("RequestLatency").Op("*").Qual(PackagePrometheus, "HistogramVec"),
	).Line()

	srcFile.Line().Add(r.clientValidBucketsFunc())

	srcFile.Line().Func().Params(Id("cli").Op("*").Id("Client")).Id("newMetrics").Params().Params(Id("m").Op("*").Id("Metrics"), Id("reg").Op("*").Qual(PackagePrometheus, "Registry")).BlockFunc(func(bg *Group) {
		bg.Id("buckets").Op(":=").Id("cli").Dot("latencyBuckets")
		bg.If(Len(Id("buckets")).Op("==").Lit(0)).Block(
			Id("buckets").Op("=").Id("defaultSecondsBuckets"),
		)
		bg.Id("reg").Op("=").Qual(PackagePrometheus, "NewRegistry").Call()
		bg.Id("m").Op("=").Op("&").Id("Metrics").Values(Dict{
			Id("VersionGauge"): Qual(PackagePrometheusAuto, "With").Call(Id("reg")).Dot("NewGaugeVec").Call(Qual(PackagePrometheus, "GaugeOpts").Values(
				DictFunc(func(d Dict) {
					d[Id("Name")] = Lit("count")
					d[Id("Namespace")] = Lit(pkgClient)
					d[Id("Subsystem")] = Lit("versions")
					d[Id("Help")] = Lit("Versions of client parts")
				}),
			), Index().String().Values(Lit("part"), Lit("version"), Lit("hostname"))),
			Id("RequestCount"): Qual(PackagePrometheusAuto, "With").Call(Id("reg")).Dot("NewCounterVec").Call(Qual(PackagePrometheus, "CounterOpts").Values(
				DictFunc(func(d Dict) {
					d[Id("Name")] = Lit("count")
					d[Id("Namespace")] = Lit(pkgClient)
					d[Id("Subsystem")] = Lit("requests")
					d[Id("Help")] = Lit("Number of requests sent")
				}),
			), Index().String().Values(Lit("service"), Lit("method"), Lit("success"), Lit("errCode"), Lit("client_id"))),
			Id("RequestCountAll"): Qual(PackagePrometheusAuto, "With").Call(Id("reg")).Dot("NewCounterVec").Call(Qual(PackagePrometheus, "CounterOpts").Values(
				DictFunc(func(d Dict) {
					d[Id("Name")] = Lit("all_count")
					d[Id("Namespace")] = Lit(pkgClient)
					d[Id("Subsystem")] = Lit("requests")
					d[Id("Help")] = Lit("Number of all requests sent")
				}),
			), Index().String().Values(Lit("service"), Lit("method"), Lit("success"), Lit("errCode"), Lit("client_id"))),
			Id("RequestLatency"): Qual(PackagePrometheusAuto, "With").Call(Id("reg")).Dot("NewHistogramVec").Call(Qual(PackagePrometheus, "HistogramOpts").Values(
				DictFunc(func(d Dict) {
					d[Id("Name")] = Lit("latency_seconds")
					d[Id("Namespace")] = Lit(pkgClient)
					d[Id("Subsystem")] = Lit("requests")
					d[Id("Help")] = Lit("Total duration of requests in seconds")
					d[Id("Buckets")] = Id("buckets")
				}),
			), Index().String().Values(Lit("service"), Lit("method"), Lit("success"), Lit("errCode"), Lit("client_id"))),
		})
		bg.Id("m").Dot("VersionGauge").Dot("WithLabelValues").Call(Lit("astg"), Id("VersionASTg"), Id("cli").Dot("name")).Dot("Set").Call(Lit(1))
		bg.Return()
	})

	return srcFile.Save(path.Join(outDir, "metrics.go"))
}

func clientSecondsBucketsLit() Code {

	vals := make([]Code, 0, len(clientDefaultSecondsBucketValues))
	for _, bucket := range clientDefaultSecondsBucketValues {
		vals = append(vals, Lit(bucket))
	}
	return Index().Float64().Values(vals...)
}

var clientDefaultSecondsBucketValues = []float64{
	0.001, 0.0025, 0.005, 0.0075, 0.01, 0.015, 0.02, 0.025, 0.035, 0.05,
	0.075, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

func (r *ClientRenderer) clientValidBucketsFunc() (c Code) {

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
