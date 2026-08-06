// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive

	"tgp/internal/model"
)

func (r *transportRenderer) contractHasHTTPTransport(contract *model.Contract) (ok bool) {

	return model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) ||
		model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJSONRPC)
}

func (r *transportRenderer) contractHasObservableTransport(contract *model.Contract) (ok bool) {

	return r.contractHasHTTPTransport(contract) ||
		model.ContractHasWS(r.project, contract) ||
		model.ContractHasSSE(r.project, contract)
}

func (r *transportRenderer) fiberFunc() (c Code) {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("Fiber").
		Params().
		Params(Op("*").Qual(PackageFiber, "App")).
		Block(
			Return(Id("srv").Dot("srvHTTP")),
		)
}

func (r *transportRenderer) withLogFunc() (c Code) {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("WithLog").
		Params().
		Params(Op("*").Id("Server")).
		BlockFunc(func(bg *Group) {
			for _, contract := range r.contractsSorted() {
				if !model.IsAnnotationSet(r.project, contract, nil, nil, TagLogger) {
					continue
				}
				if !r.contractHasObservableTransport(contract) {
					continue
				}
				bg.If(Id("srv").Dot("http" + contract.Name).Op("!=").Nil()).Block(
					Id("srv").Dot("http" + contract.Name).Op("=").Id("srv").Dot("http" + contract.Name).Dot("WithLog").Call(),
				)
			}
			bg.Return(Id("srv"))
		})
}

func (r *transportRenderer) withTraceFunc() (c Code) {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("WithTrace").
		Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("appName").String(), Id("endpoint").String(), Id("attributes").Op("...").Qual(PackageAttributeOTEL, "KeyValue")).
		Params(Op("*").Id("Server")).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Id("srv").Dot("tracerProvider").Op("=").Qual(r.pkgPath(r.outDir)+"/tracer", "Init").Call(
				Id(VarNameCtx),
				Id("appName"),
				Id("endpoint"),
				Id("srv").Dot("traceSampler"),
				Id("srv").Dot("traceInsecure"),
				Id("attributes").Op("..."),
			)
			for _, contract := range r.contractsSorted() {
				if !model.IsAnnotationSet(r.project, contract, nil, nil, TagTrace) {
					continue
				}
				if !r.contractHasObservableTransport(contract) {
					continue
				}
				bg.If(Id("srv").Dot("http" + contract.Name).Op("!=").Nil()).Block(
					Id("srv").Dot("http" + contract.Name).Op("=").Id("srv").Dot("http" + contract.Name).Dot("WithTrace").Call(),
				)
			}
			bg.Return(Id("srv"))
		})
}

func (r *transportRenderer) withMetricsFunc() (c Code) {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("WithMetrics").
		Params().
		Params(Op("*").Id("Server")).
		BlockFunc(func(bg *Group) {
			bg.If(Id("srv").Dot("metrics").Op("==").Nil()).Block(
				Id("srv").Dot("metrics").Op("=").Id("newMetrics").Call(Id("srv").Dot("durationBuckets"), Id("srv").Dot("latencyBuckets")),
			)
			for _, contract := range r.contractsSorted() {
				if !model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
					continue
				}
				if !r.contractHasObservableTransport(contract) {
					continue
				}
				bg.If(Id("srv").Dot("http" + contract.Name).Op("!=").Nil()).Block(
					Id("srv").Dot("http" + contract.Name).Op("=").Id("srv").Dot("http" + contract.Name).Dot("WithMetrics").Call(Id("srv").Dot("metrics")),
				)
			}
			bg.Return(Id("srv"))
		})
}

func (r *transportRenderer) serverNewFunc() (c Code) {

	return Func().Id("New").
		Params(Id("log").Op("*").Qual(PackageSlog, "Logger"), Id("options").Op("...").Id("Option")).
		Params(Id("srv").Op("*").Id("Server")).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.List(Id("rootCtx"), Id("rootCancel")).Op(":=").Qual(PackageContext, "WithCancel").Call(Qual(PackageContext, "Background").Call())
			bg.Id("srv").Op("=").Op("&").Id("Server").Values(DictFunc(func(dict Dict) {
				dict[Id("log")] = Id("log")
				dict[Id("maxClientIDs")] = Id("defaultMaxClientIDs")
				dict[Id("clientIDs")] = Make(Map(Uint64()).Struct(), Id("defaultMaxClientIDs"))
				dict[Id("maxStreamBodySize")] = Id("defaultBodyLimit")
				dict[Id("rootCtx")] = Id("rootCtx")
				dict[Id("rootCancel")] = Id("rootCancel")
				dict[Id("streams")] = Make(Map(Uint64()).Id("streamCloser"))
				if r.hasJSONRPC() {
					dict[Id("maxBatchSize")] = Id("defaultMaxBatchSize")
					dict[Id("maxParallelBatch")] = Id("defaultMaxParallelBatch")
				}
				if r.hasSSE() {
					dict[Id("sseHeartbeat")] = Id("defaultSSEHeartbeat")
					dict[Id("maxSSEConnections")] = Id("defaultMaxSSEConnections")
				}
				if r.hasWS() || r.hasSSE() {
					dict[Id("streamWriteTimeout")] = Id("defaultWriteTimeout")
				}
				if r.hasWS() {
					dict[Id("maxOpenStreams")] = Id("defaultMaxOpenStreams")
					dict[Id("maxWSConnections")] = Id("defaultMaxWSConnections")
					dict[Id("maxWSMessageSize")] = Id("defaultMaxWSMessageSize")
				}
				dict[Id("headerHandlers")] = Make(Map(String()).Id("HeaderHandler"))
				dict[Id("config")] = Qual(PackageFiber, "Config").Values(Dict{
					Id("Immutable"):                    True(),
					Id("StreamRequestBody"):            True(),
					Id("DisableStartupMessage"):        True(),
					Id("DisablePreParseMultipartForm"): True(),
					Id("BodyLimit"):                    Id("defaultBodyLimit"),
					Id("Concurrency"):                  Id("defaultConcurrency"),
					Id("IdleTimeout"):                  Id("defaultIdleTimeout"),
					Id("ReadTimeout"):                  Id("defaultReadTimeout"),
					Id("WriteTimeout"):                 Id("defaultWriteTimeout"),
					Id("ReadBufferSize"):               Id("defaultReadBufferSize"),
					Id("WriteBufferSize"):              Id("defaultWriteBufferSize"),
				})
			},
			))
			bg.Line()
			// Config options mutate srv.config; service options no-op while srvHTTP is nil.
			bg.For(List(Id("_"), Id("option")).Op(":=").Range().Id("options")).Block(
				Id("option").Call(Id("srv")),
			)
			bg.Line()
			bg.Id("srv").Dot("srvHTTP").Op("=").Qual(PackageFiber, "New").Call(Id("srv").Dot("config"))
			bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Func().Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).Params(Error()).Block(
				Id(VarNameFtx).Dot("Locals").Call(Lit("server"), Id("srv")),
				Return(Id("recoverHandler").Call(Id(VarNameFtx))),
			))
			bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("srv").Dot("clientIDMiddleware"))
			if r.hasBearerSecurity() {
				bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("srv").Dot("bearerMiddleware"))
			}
			if r.hasMetrics() {
				bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("srv").Dot("inFlightMiddleware"))
				bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("srv").Dot("requestDurationMiddleware"))
			}
			if r.hasTrace() {
				bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Qual(r.pkgPath(r.outDir)+"/tracer", "Middleware").Call())
			}
			bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("srv").Dot("setLogger"))
			bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("srv").Dot("headersHandler"))
			// Second pass: service options register routes now that srvHTTP exists.
			bg.For(List(Id("_"), Id("option")).Op(":=").Range().Id("options")).Block(
				Id("option").Call(Id("srv")),
			)
			if r.hasBearerSecurity() {
				bg.If(Id("srv").Dot("auth").Op("!=").Nil()).Block(
					Id("srv").Dot("srvHTTP").Dot("Get").Call(Qual(r.pkgPath(r.outDir)+"/bearer", "WellKnownPath"), Id("srv").Dot("resourceMetadata")),
				)
			}
			bg.Line()
			if r.hasJSONRPC() {
				bg.Id("initJSONRPCMethodMaps").Call(Id("srv"))
				for _, mount := range model.JSONRPCBatchMounts(r.project) {
					bg.Id("srv").Dot("srvHTTP").Dot("Post").Call(Lit(mount), Id("srv").Dot("batchHandler").Call(Lit(mount)))
				}
			}
			bg.Return()
		})
}

func (r *transportRenderer) serveHealthFunc() (c Code) {

	jsonPkg := r.getPackageJSON()
	return Func().Id("ServeHealth").
		Params(
			Id("log").Op("*").Qual(PackageSlog, "Logger"),
			Id("path").String(),
			Id("address").String(),
			Id("opts").Op("...").Id("HealthOption"),
		).
		Params(Id("hs").Op("*").Id("HealthServer")).
		BlockFunc(func(bg *Group) {
			bg.Var().Err().Error()
			bg.Id("hs").Op("=").Op("&").Id("HealthServer").Values(Dict{
				Id("checkTimeout"): Id("defaultHealthCheckTimeout"),
			})
			bg.For(List(Id("_"), Id("opt")).Op(":=").Range().Id("opts")).Block(
				Id("opt").Call(Id("hs")),
			)
			bg.If(Id("hs").Dot("response").Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.List(Id("hs").Dot("responseBody"), Err()).Op("=").Qual(jsonPkg, "Marshal").Call(Id("hs").Dot("response"))
				ig.If(Err().Op("!=").Nil()).BlockFunc(func(eg *Group) {
					eg.Id("log").Dot("Error").Call(Lit("failed to marshal health response"), Qual(PackageSlog, "Any").Call(Lit("error"), Err()))
					eg.Id("hs").Dot("responseBody").Op("=").Op("[]").Byte().Call(Lit(`{"status":"error","message":"health check misconfigured"}`))
				})
			}).Else().Block(
				Id("hs").Dot("responseBody").Op("=").Op("[]").Byte().Call(Lit(`"ok"`)),
			)
			bg.Id("hs").Dot("srv").Op("=").Qual(PackageFiber, "New").Call(Qual(PackageFiber, "Config").Values(Dict{
				Id("DisableStartupMessage"):        True(),
				Id("DisablePreParseMultipartForm"): True(),
				Id("IdleTimeout"):                  Id("defaultIdleTimeout"),
				Id("ReadTimeout"):                  Id("defaultReadTimeout"),
				Id("WriteTimeout"):                 Id("defaultWriteTimeout"),
			}))
			bg.Id("hs").Dot("srv").Dot("Get").Call(Id("path"), Id("hs").Dot("handle"))
			bg.Go().Func().Params().Block(
				Err().Op("=").Id("hs").Dot("srv").Dot("Listen").Call(Id("address")),
				Id("ExitOnError").Call(Id("log"), Err(), Lit("serve health on ").Op("+").Id("address")),
			).Call()
			bg.Return()
		})
}

func (r *transportRenderer) shutdownDefaultFunc() (c Code) {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("Shutdown").
		Params().
		Params(Id("err").Error()).
		Block(
			Return(Id("srv").Dot("ShutdownWithTimeout").Call(Id("defaultShutdownTimeout"))),
		)
}

func (r *transportRenderer) shutdownFunc() (c Code) {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("ShutdownWithTimeout").
		Params(Id("timeout").Qual(PackageTime, "Duration")).
		Params(Id("err").Error()).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Id("srv").Dot("shutdownOnce").Dot("Do").Call(Func().Params().BlockFunc(func(sg *Group) {
				sg.Id("shutdownCtx").Op(",").Id("cancel").Op(":=").Qual(PackageContext, "WithTimeout").Call(Qual(PackageContext, "Background").Call(), Id("timeout"))
				sg.Defer().Id("cancel").Call()
				sg.If(Id("srv").Dot("rootCancel").Op("!=").Nil()).Block(
					Id("srv").Dot("rootCancel").Call(),
				)
				sg.Id("srv").Dot("streamsMu").Dot("Lock").Call()
				sg.Id("streams").Op(":=").Make(Index().Id("streamCloser"), Lit(0), Len(Id("srv").Dot("streams")))
				sg.For(List(Id("_"), Id("stream")).Op(":=").Range().Id("srv").Dot("streams")).Block(
					Id("streams").Op("=").Append(Id("streams"), Id("stream")),
				)
				sg.Clear(Id("srv").Dot("streams"))
				sg.Id("srv").Dot("streamsMu").Dot("Unlock").Call()
				sg.For(List(Id("_"), Id("stream")).Op(":=").Range().Id("streams")).Block(
					Id("stream").Dot("Close").Call(),
				)
				sg.Var().Defs(
					Id("httpErr").Error(),
					Id("metricsErr").Error(),
					Id("traceErr").Error(),
					Id("streamErr").Error(),
					Id("wg").Qual(PackageSync, "WaitGroup"),
				)
				sg.If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).Block(
					Id("wg").Dot("Go").Call(Func().Params().Block(
						Id("httpErr").Op("=").Id("srv").Dot("srvHTTP").Dot("ShutdownWithTimeout").Call(Id("timeout")),
					)),
				)
				sg.If(Id("srv").Dot("srvMetrics").Op("!=").Nil()).Block(
					Id("wg").Dot("Go").Call(Func().Params().Block(
						Id("metricsErr").Op("=").Id("srv").Dot("srvMetrics").Dot("ShutdownWithTimeout").Call(Id("timeout")),
					)),
				)
				sg.Id("wg").Dot("Wait").Call()
				if r.hasTrace() {
					sg.If(Id("srv").Dot("tracerProvider").Op("!=").Nil()).Block(
						Id("traceErr").Op("=").Id("srv").Dot("tracerProvider").Dot("Shutdown").Call(Id("shutdownCtx")),
					)
				}
				sg.Id("streamsDone").Op(":=").Make(Chan().Struct())
				sg.Go().Func().Params().Block(
					Id("srv").Dot("streamsWG").Dot("Wait").Call(),
					Close(Id("streamsDone")),
				).Call()
				sg.Select().Block(
					Case(Op("<-").Id("streamsDone")).Block(),
					Case(Op("<-").Id("shutdownCtx").Dot("Done").Call()).Block(
						Id("streamErr").Op("=").Id("shutdownCtx").Dot("Err").Call(),
					),
				)
				sg.Id("srv").Dot("shutdownErr").Op("=").Qual(PackageErrors, "Join").Call(Id("httpErr"), Id("metricsErr"), Id("traceErr"), Id("streamErr"))
			}))
			bg.Err().Op("=").Id("srv").Dot("shutdownErr")
			bg.Return()
		})
}

func (r *transportRenderer) sendResponseFunc() (c Code) {

	jsonPkg := r.getPackageJSON()
	return Func().Id("sendResponse").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx"), Id("resp").Any()).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Lit("application/json"))
			bg.If(Err().Op("=").Qual(jsonPkg, "NewEncoder").Call(Id(VarNameFtx).Dot("Response").Call().Dot("BodyWriter").Call()).Dot("Encode").Call(Id("resp")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.If(Id("logger").Op(":=").Qual(r.pkgPath(r.outDir)+"/srvctx", "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(Id(VarNameFtx).Dot("UserContext").Call()).Op(";").Id("logger").Op("!=").Nil()).Block(
					Id("logger").Dot("Error").Call(Lit("response marshal error"), Qual(PackageSlog, "Any").Call(Lit("error"), Err())),
				)
				ig.Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusInternalServerError"))
				ig.Return(Err())
			})
			bg.Return(Err())
		})
}

func (r *transportRenderer) sendHTTPErrorFunc() (c Code) {

	srvctxPkgPath := r.pkgPath(r.outDir) + "/srvctx"
	return Func().Id("sendHTTPError").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx"), Id("statusCode").Int(), Id("message").String()).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.Id("clientID").Op(":=").Qual(srvctxPkgPath, "GetClientID").Call(Id(VarNameFtx).Dot("UserContext").Call())
			bg.If(List(Id("server"), Id("ok")).Op(":=").Id(VarNameFtx).Dot("Locals").Call(Lit("server")).Assert(Op("*").Id("Server")).Op(";").Id("ok").Op("&&").Id("server").Dot("metrics").Op("!=").Nil()).Block(
				Id("server").Dot("metrics").Dot("ErrorResponsesTotal").Dot("WithLabelValues").Call(
					Lit("json-rpc"),
					Qual(PackageStrconv, "Itoa").Call(Id("statusCode")),
					Id("clientID"),
				).Dot("Inc").Call(),
			)
			bg.Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Lit("text/plain"))
			bg.Id(VarNameFtx).Dot("Status").Call(Id("statusCode"))
			bg.List(Id("_"), Err()).Op("=").Id(VarNameFtx).Dot("WriteString").Call(Id("message"))
			bg.Return(Err())
		})
}
