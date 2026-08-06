// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive

	"tgp/internal/model"
)

func (r *transportRenderer) RenderTransportOptions() (err error) {

	optionsPath := path.Join(r.outDir, "options.go")

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(PackageUUID, "uuid")
	srcFile.ImportName(PackageTime, "time")
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(PackageStrings, "strings")
	if r.hasTrace() {
		srcFile.ImportName(PackageTraceSDK, "trace")
	}

	r.renderOptionsTypes(&srcFile)
	r.renderOptionsService(&srcFile)
	r.renderOptionsForContracts(&srcFile)
	r.renderOptionsConfig(&srcFile)
	r.renderOptionsTimeouts(&srcFile)
	r.renderOptionsHeaders(&srcFile)
	r.renderOptionsUse(&srcFile)

	return srcFile.Save(optionsPath)
}

func (r *transportRenderer) renderOptionsTypes(srcFile *GoFile) {

	srcFile.Line().Type().Id("ServiceRoute").Interface(
		Id("SetRoutes").Params(Id("route").Op("*").Qual(PackageFiber, "App")),
	)

	srcFile.Line().Type().Id("Option").Func().Params(Id("srv").Op("*").Id("Server"))
	srcFile.Type().Id("Handler").Op("=").Qual(PackageFiber, "Handler")
	srcFile.Type().Id("ErrorHandler").Func().Params(Err().Error()).Params(Error())
}

func (r *transportRenderer) renderOptionsService(srcFile *GoFile) {

	srcFile.Line().Func().Id("Service").
		Params(Id("svc").Id("ServiceRoute")).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).Block(
					Id("svc").Dot("SetRoutes").Call(Id("srv").Dot("Fiber").Call()),
				),
			)),
		)
}

func (r *transportRenderer) renderOptionsForContracts(srcFile *GoFile) {

	for _, contract := range r.contractsSorted() {
		hasHTTP := model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP)
		hasJSONRPC := model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJSONRPC)
		hasStream := model.ContractHasWS(r.project, contract) || model.ContractHasSSE(r.project, contract)
		if !hasHTTP && !hasJSONRPC && !hasStream {
			continue
		}
		if hasHTTP && !hasJSONRPC {
			srcFile.ImportName(contract.PkgPath, filepath.Base(contract.PkgPath))
			srcFile.Line().Func().Id(contract.Name).
				Params(Id("svc").Qual(contract.PkgPath, contract.Name)).
				Id("Option").
				Block(
					Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
						If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).BlockFunc(func(gr *Group) {
							gr.Id("httpSvc").Op(":=").Id("new" + contract.Name).Call(Id("svc"))
							gr.Id("srv").Dot("http" + contract.Name).Op("=").Id("httpSvc")
							if model.ContractHasSSE(r.project, contract) || model.ContractHasWS(r.project, contract) {
								gr.Id("httpSvc").Dot("srv").Op("=").Id("srv")
							}
							gr.Id("httpSvc").Dot("SetRoutes").Call(Id("srv").Dot("Fiber").Call())
						}),
					)),
				)
			continue
		}
		if hasJSONRPC {
			srcFile.ImportName(contract.PkgPath, filepath.Base(contract.PkgPath))
			srcFile.Line().Func().Id(contract.Name).
				Params(Id("svc").Qual(contract.PkgPath, contract.Name)).
				Id("Option").
				Block(
					Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
						If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).BlockFunc(func(gr *Group) {
							gr.Id("httpSvc").Op(":=").Id("new" + contract.Name).Call(Id("svc"))
							gr.Id("srv").Dot("http" + contract.Name).Op("=").Id("httpSvc")
							gr.Id("httpSvc").Dot("srv").Op("=").Id("srv")
							gr.Id("httpSvc").Dot("SetRoutes").Call(Id("srv").Dot("Fiber").Call())
						}),
					)),
				)
			continue
		}
		srcFile.ImportName(contract.PkgPath, filepath.Base(contract.PkgPath))
		srcFile.Line().Func().Id(contract.Name).
			Params(Id("svc").Qual(contract.PkgPath, contract.Name)).
			Id("Option").
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).BlockFunc(func(gr *Group) {
						gr.Id("httpSvc").Op(":=").Id("new" + contract.Name).Call(Id("svc"))
						gr.Id("srv").Dot("http" + contract.Name).Op("=").Id("httpSvc")
						if model.ContractHasSSE(r.project, contract) || model.ContractHasWS(r.project, contract) {
							gr.Id("httpSvc").Dot("srv").Op("=").Id("srv")
						}
						gr.Id("httpSvc").Dot("SetRoutes").Call(Id("srv").Dot("Fiber").Call())
					}),
				)),
			)
	}
}

func (r *transportRenderer) renderOptionsConfig(srcFile *GoFile) {

	srcFile.Line().Func().Id("SetFiberCfg").
		Params(Id("cfg").Qual(PackageFiber, "Config")).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("config").Op("=").Id("cfg"),
				Id("srv").Dot("config").Dot("Immutable").Op("=").True(),
				Id("srv").Dot("config").Dot("StreamRequestBody").Op("=").True(),
				Id("srv").Dot("config").Dot("DisableStartupMessage").Op("=").True(),
				Id("srv").Dot("config").Dot("DisablePreParseMultipartForm").Op("=").True(),
			)),
		)
	srcFile.Line().Func().Id("SetReadBufferSize").
		Params(Id("size").Int()).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("config").Dot("ReadBufferSize").Op("=").Id("size"),
			)),
		)
	srcFile.Line().Func().Id("SetWriteBufferSize").
		Params(Id("size").Int()).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("config").Dot("WriteBufferSize").Op("=").Id("size"),
			)),
		)
	srcFile.Line().Func().Id("MaxBodySize").
		Params(Id("size").Int()).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("config").Dot("BodyLimit").Op("=").Id("size"),
			)),
		)
	srcFile.Line().Func().Id("MaxStreamBodySize").
		Params(Id("size").Int64()).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				If(Id("size").Op("<").Lit(0)).Block(
					Return(),
				),
				Id("srv").Dot("maxStreamBodySize").Op("=").Id("size"),
			)),
		)
	srcFile.Line().Func().Id("MaxClientIDs").
		Params(Id("n").Int()).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				If(Id("n").Op("<").Lit(0)).Block(
					Return(),
				),
				Id("srv").Dot("maxClientIDs").Op("=").Id("n"),
			)),
		)
	srcFile.Line().Func().Id("AllowClientLogLevel").
		Params().
		Params(Id("opt").Id("Option")).
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("allowClientLogLevel").Op("=").True(),
			)),
		)
	if r.hasMetrics() {
		srcFile.Line().Func().Id("DurationBuckets").
			Params(Id("buckets").Op("...").Float64()).
			Params(Id("opt").Id("Option")).
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					If(Op("!").Id("validBuckets").Call(Id("buckets"))).Block(
						Return(),
					),
					Id("srv").Dot("durationBuckets").Op("=").Append(Index().Float64().Values(), Id("buckets").Op("...")),
				)),
			)
		srcFile.Line().Func().Id("LatencyBuckets").
			Params(Id("buckets").Op("...").Float64()).
			Params(Id("opt").Id("Option")).
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					If(Op("!").Id("validBuckets").Call(Id("buckets"))).Block(
						Return(),
					),
					Id("srv").Dot("latencyBuckets").Op("=").Append(Index().Float64().Values(), Id("buckets").Op("...")),
				)),
			)
	}
	if r.hasJSONRPC() {
		srcFile.Line().Func().Id("MaxBatchSize").
			Params(Id("size").Int()).
			Id("Option").
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					If(Id("size").Op("<=").Lit(0)).Block(
						Return(),
					),
					Id("srv").Dot("maxBatchSize").Op("=").Id("size"),
				)),
			)
		srcFile.Line().Func().Id("MaxBatchWorkers").
			Params(Id("size").Int()).
			Id("Option").
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					If(Id("size").Op("<=").Lit(0)).Block(
						Return(),
					),
					Id("srv").Dot("maxParallelBatch").Op("=").Id("size"),
				)),
			)
	}
	if r.hasWS() {
		srcFile.Line().Func().Id("MaxOpenStreams").
			Params(Id("size").Int()).
			Id("Option").
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					If(Id("size").Op(">").Lit(0)).Block(
						Id("srv").Dot("maxOpenStreams").Op("=").Id("size"),
					),
				)),
			)
		srcFile.Line().Func().Id("MaxWSConnections").
			Params(Id("n").Int()).
			Id("Option").
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					If(Id("n").Op("<").Lit(0)).Block(
						Return(),
					),
					Id("srv").Dot("maxWSConnections").Op("=").Id("n"),
				)),
			)
		srcFile.Line().Func().Id("MaxWSMessageSize").
			Params(Id("n").Int64()).
			Id("Option").
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					If(Id("n").Op("<").Lit(0)).Block(
						Return(),
					),
					Id("srv").Dot("maxWSMessageSize").Op("=").Id("n"),
				)),
			)
	}
	if r.hasSSE() {
		srcFile.Line().Func().Id("MaxSSEConnections").
			Params(Id("n").Int()).
			Id("Option").
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					If(Id("n").Op("<").Lit(0)).Block(
						Return(),
					),
					Id("srv").Dot("maxSSEConnections").Op("=").Id("n"),
				)),
			)
	}
}

func (r *transportRenderer) renderOptionsTimeouts(srcFile *GoFile) {

	srcFile.Line().Func().Id("ReadTimeout").
		Params(Id("timeout").Qual(PackageTime, "Duration")).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("config").Dot("ReadTimeout").Op("=").Id("timeout"),
			)),
		)
	srcFile.Line().Func().Id("WriteTimeout").
		Params(Id("timeout").Qual(PackageTime, "Duration")).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("config").Dot("WriteTimeout").Op("=").Id("timeout"),
			)),
		)
	if r.hasWS() || r.hasSSE() {
		srcFile.Line().Func().Id("StreamWriteTimeout").
			Params(Id("timeout").Qual(PackageTime, "Duration")).
			Id("Option").
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					If(Id("timeout").Op("<").Lit(0)).Block(
						Return(),
					),
					Id("srv").Dot("streamWriteTimeout").Op("=").Id("timeout"),
				)),
			)
	}
	if r.hasSSE() {
		srcFile.Line().Func().Id("SetSSEHeartbeat").
			Params(Id("interval").Qual(PackageTime, "Duration")).
			Id("Option").
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					Id("srv").Dot("sseHeartbeat").Op("=").Id("interval"),
				)),
			)
	}
	if r.hasTrace() {
		srcFile.Line().Func().Id("TraceSampler").
			Params(Id("sampler").Qual(PackageTraceSDK, "Sampler")).
			Id("Option").
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					Id("srv").Dot("traceSampler").Op("=").Id("sampler"),
				)),
			)
		srcFile.Line().Func().Id("TraceInsecure").
			Params().
			Params(Id("opt").Id("Option")).
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					Id("srv").Dot("traceInsecure").Op("=").True(),
				)),
			)
	}
}

func (r *transportRenderer) renderOptionsHeaders(srcFile *GoFile) {

	srcFile.Line().Func().Id("WithRequestID").
		Params(Id("headerName").String()).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("headerHandlers").Op("[").Id("headerName").Op("]").Op("=").
					Func().Params(Id("value").String()).Params(Id("Header")).Block(
					If(Id("value").Op("==").Lit("")).Block(
						Id("value").Op("=").Qual(PackageUUID, "New").Call().Dot("String").Call(),
					),
					Return(Id("Header").Block(Dict{
						Id("SpanKey"):       Lit("requestID"),
						Id("SpanValue"):     Id("value"),
						Id("ResponseKey"):   Id("headerName"),
						Id("ResponseValue"): Id("value"),
						Id("LogKey"):        Lit("requestID"),
						Id("LogValue"):      Id("value"),
					})),
				),
			)),
		)
	srcFile.Line().Func().Id("WithHeader").
		Params(Id("headerName").String(), Id("handler").Id("HeaderHandler")).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("headerHandlers").Op("[").Id("headerName").Op("]").Op("=").Id("handler"),
			)),
		)
}

func (r *transportRenderer) renderOptionsUse(srcFile *GoFile) {

	srcFile.Line().Func().Id("Use").
		Params(Id("args").Op("...").Any()).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).Block(
					Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("args").Op("...")),
				),
			)),
		)
}
