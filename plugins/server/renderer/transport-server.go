// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive

	"tgp/internal/model"
)

func (r *transportRenderer) RenderTransportServer() (err error) {

	serverPath := path.Join(r.outDir, "server.go")

	if err = r.pkgRenderTo("logger", r.outDir, newPkgTemplateData()); err != nil {
		return fmt.Errorf("render logger package: %w", err)
	}
	if r.hasBearerSecurity() {
		if err = r.pkgRenderTo("bearer", r.outDir, r.bearerTemplateData()); err != nil {
			return fmt.Errorf("render bearer package: %w", err)
		}
	}
	if r.hasTrace() {
		if err = r.pkgRenderTo("tracer", r.outDir, newPkgTemplateData()); err != nil {
			return fmt.Errorf("render tracer package: %w", err)
		}
	}

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	r.renderServerImports(&srcFile)
	r.renderServerTypes(&srcFile)
	r.renderServerConstants(&srcFile)
	r.renderServerFunctions(&srcFile)

	return srcFile.Save(serverPath)
}

func (r *transportRenderer) renderServerImports(srcFile *GoFile) {

	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(r.pkgPath(r.outDir)+"/srvctx", "srvctx")
	jsonPkg := r.getPackageJSON()
	srcFile.ImportName(jsonPkg, keyJSON)
	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(PackageTime, "time")

	// Локальные пакеты
	srcFile.ImportName(r.pkgPath(r.outDir)+"/logger", "logger")
	if r.hasBearerSecurity() {
		srcFile.ImportName(r.pkgPath(r.outDir)+"/bearer", "bearer")
	}
	if r.hasTrace() {
		srcFile.ImportName(r.pkgPath(r.outDir)+"/tracer", "tracer")
		srcFile.ImportAlias(PackageTraceSDK, "trace")
	}
	if r.hasWS() {
		srcFile.ImportName(r.pkgPath(r.outDir)+"/stream", "stream")
	}

	// Внешние пакеты
	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(PackagePrometheus, "prometheus")
	srcFile.ImportName(PackagePrometheusAuto, "promauto")
	srcFile.ImportName(PackagePrometheusHTTP, "promhttp")
	srcFile.ImportName(PackageStrconv, "strconv")
	srcFile.ImportName(PackageSync, "sync")
	srcFile.ImportName(PackageErrors, "errors")
	if r.hasBearerSecurity() {
		srcFile.ImportName(PackageStrings, "strings")
	}
	if r.hasWS() || r.hasSSE() {
		srcFile.ImportName(PackageSyncAtomic, "atomic")
	}

	for _, contract := range r.contractsSorted() {
		srcFile.ImportName(contract.PkgPath, filepath.Base(contract.PkgPath))
	}
}

func (r *transportRenderer) renderServerTypes(srcFile *GoFile) {

	srcFile.Line().Type().Id("streamCloser").Interface(
		Id("Close").Params(),
	)
	srcFile.Line().Add(r.transportServerType())
}

func (r *transportRenderer) renderServerConstants(srcFile *GoFile) {

	srcFile.Const().Id("defaultShutdownTimeout").Op("=").Lit(30).Op("*").Qual(PackageTime, "Second")
	srcFile.Line()
	srcFile.Const().Id("defaultBodyLimit").Op("=").Lit(8).Op("*").Lit(1024).Op("*").Lit(1024)
	srcFile.Const().Id("defaultReadBufferSize").Op("=").Lit(4096)
	srcFile.Const().Id("defaultWriteBufferSize").Op("=").Lit(4096)
	srcFile.Const().Id("defaultReadTimeout").Op("=").Lit(30).Op("*").Qual(PackageTime, "Second")
	srcFile.Const().Id("defaultWriteTimeout").Op("=").Lit(30).Op("*").Qual(PackageTime, "Second")
	srcFile.Const().Id("defaultSSEHeartbeat").Op("=").Lit(15).Op("*").Qual(PackageTime, "Second")
	srcFile.Const().Id("defaultIdleTimeout").Op("=").Lit(120).Op("*").Qual(PackageTime, "Second")
	srcFile.Const().Id("defaultHealthCheckTimeout").Op("=").Lit(5).Op("*").Qual(PackageTime, "Second")
	srcFile.Const().Id("defaultConcurrency").Op("=").Lit(256).Op("*").Lit(1024)
	srcFile.Const().Id("defaultMaxClientIDs").Op("=").Lit(64)
	srcFile.Const().Id("contentTypeJson").Op("=").Lit("application/json")
	if r.hasWS() {
		srcFile.Const().Id("defaultMaxOpenStreams").Op("=").Qual(r.pkgPath(r.outDir)+"/stream", "DefaultMaxOpenStreams")
		srcFile.Const().Id("defaultMaxWSConnections").Op("=").Lit(512)
		srcFile.Const().Id("defaultMaxWSMessageSize").Op("=").Id("defaultBodyLimit")
	}
	if r.hasSSE() {
		srcFile.Const().Id("defaultMaxSSEConnections").Op("=").Lit(512)
	}

	srcFile.Line().Add(r.healthServerType())
	srcFile.Line().Add(r.healthOptionType())
}

func (r *transportRenderer) renderServerFunctions(srcFile *GoFile) {

	srcFile.Line().Add(r.withResponseFunc())
	srcFile.Line().Add(r.withCheckFunc())
	srcFile.Line().Add(r.healthServerHandleMethod())
	srcFile.Line().Add(r.healthServerStopMethod())
	srcFile.Line().Add(r.serverNewFunc())
	srcFile.Line().Add(r.fiberFunc())
	srcFile.Line().Add(r.withLogFunc())
	srcFile.Line().Add(r.serveHealthFunc())
	if !r.hasJSONRPC() {
		srcFile.Line().Add(r.sendResponseFunc())
	}
	srcFile.Line().Add(r.sendHTTPErrorFunc())
	srcFile.Line().Add(r.cancelCloserType())
	srcFile.Line().Add(r.registerStreamFunc())
	srcFile.Line().Add(r.unregisterStreamFunc())
	srcFile.Line().Add(r.shutdownDefaultFunc())
	srcFile.Line().Add(r.shutdownFunc())
	if r.hasTrace() {
		srcFile.Line().Add(r.withTraceFunc())
	}
	srcFile.Line().Add(r.clientIDMiddleware())
	if r.hasBearerSecurity() {
		srcFile.Line().Add(r.bearerMiddleware())
	}
	if r.hasMetrics() {
		srcFile.Line().Add(r.inFlightMiddleware())
		srcFile.Line().Add(r.requestDurationMiddleware())
		srcFile.Line().Add(r.withMetricsFunc())
	}
	if r.hasWS() {
		srcFile.Line().Add(r.acquireWSFunc())
		srcFile.Line().Add(r.releaseWSFunc())
	}
	if r.hasSSE() {
		srcFile.Line().Add(r.acquireSSEFunc())
		srcFile.Line().Add(r.releaseSSEFunc())
		srcFile.Line().Add(r.effectiveSSEHeartbeatFunc())
	}
	for _, contract := range r.contractsSorted() {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) ||
			model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJSONRPC) ||
			model.ContractHasWS(r.project, contract) ||
			model.ContractHasSSE(r.project, contract) {
			srcFile.Line().Add(r.httpServiceAccessorFunc(contract))
		}
	}
}

func (r *transportRenderer) transportServerType() Code {

	return Type().Id("Server").StructFunc(func(bg *Group) {
		bg.Id("log").Op("*").Qual(PackageSlog, "Logger")
		bg.Line().Id("config").Qual(PackageFiber, "Config")
		bg.Line().Id("srvHTTP").Op("*").Qual(PackageFiber, "App")
		bg.Id("srvMetrics").Op("*").Qual(PackageFiber, "App")
		bg.Line().Id("maxClientIDs").Int()
		bg.Id("clientIDs").Map(Uint64()).Struct()
		bg.Id("clientIDsMu").Qual(PackageSync, "RWMutex")
		bg.Line().Id("maxStreamBodySize").Int64()
		if r.hasWS() || r.hasSSE() {
			bg.Id("streamWriteTimeout").Qual(PackageTime, "Duration")
		}
		bg.Id("rootCtx").Qual(PackageContext, "Context")
		bg.Id("rootCancel").Qual(PackageContext, "CancelFunc")
		bg.Id("streamsMu").Qual(PackageSync, "Mutex")
		bg.Id("streams").Map(Uint64()).Id("streamCloser")
		bg.Id("streamsSeq").Uint64()
		bg.Id("streamsWG").Qual(PackageSync, "WaitGroup")
		bg.Id("shutdownOnce").Qual(PackageSync, "Once")
		bg.Id("shutdownErr").Error()
		if r.hasTrace() {
			bg.Id("tracerProvider").Op("*").Qual(PackageTraceSDK, "TracerProvider")
			bg.Id("traceSampler").Qual(PackageTraceSDK, "Sampler")
			bg.Id("traceInsecure").Bool()
		}
		bg.Id("allowClientLogLevel").Bool()
		if r.hasMetrics() {
			bg.Line().Id("metrics").Op("*").Id("Metrics")
			bg.Id("durationBuckets").Index().Float64()
			bg.Id("latencyBuckets").Index().Float64()
		}
		if r.hasJSONRPC() {
			bg.Line().Id("maxBatchSize").Int()
			bg.Id("maxParallelBatch").Int()
			bg.Id("jsonRPCMethodMaps").Map(String()).Map(String()).Id("methodJsonRPC").Line()
		}
		if r.hasSSE() {
			bg.Line().Id("sseHeartbeat").Qual(PackageTime, "Duration")
			bg.Id("maxSSEConnections").Int()
			bg.Id("sseConnections").Qual(PackageSyncAtomic, "Int64")
		}
		if r.hasWS() {
			bg.Line().Id("maxOpenStreams").Int()
			bg.Id("maxWSConnections").Int()
			bg.Id("maxWSMessageSize").Int64()
			bg.Id("wsConnections").Qual(PackageSyncAtomic, "Int64")
		}
		for _, contract := range r.contractsSorted() {
			if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) ||
				model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJSONRPC) ||
				model.ContractHasWS(r.project, contract) ||
				model.ContractHasSSE(r.project, contract) {
				bg.Line()
				bg.Id("http" + contract.Name).Op("*").Id("http" + contract.Name)
			}
		}
		bg.Line()
		bg.Id("headerHandlers").Map(String()).Id("HeaderHandler")
		if r.hasBearerSecurity() {
			bg.Id("auth").Op("*").Id("authenticator")
			bg.Id("authChallenge").String()
			bg.Id("authMetadata").Index().Byte()
		}
	})
}

func (r *transportRenderer) healthServerType() Code {

	return Type().Id("HealthServer").StructFunc(func(bg *Group) {
		bg.Id("srv").Op("*").Qual(PackageFiber, "App")
		bg.Id("response").Any()
		bg.Id("responseBody").Index().Byte()
		bg.Id("check").Func().Params(Id("ctx").Qual(PackageContext, "Context")).Params(Id("err").Error())
		bg.Id("checkTimeout").Qual(PackageTime, "Duration")
	})
}

func (r *transportRenderer) healthOptionType() Code {

	return Type().Id("HealthOption").Func().Params(Id("hs").Op("*").Id("HealthServer"))
}

func (r *transportRenderer) withResponseFunc() Code {

	return Func().Id("WithResponse").
		Params(Id("response").Any()).
		Params(Id("opt").Id("HealthOption")).
		Block(
			Return(Func().Params(Id("hs").Op("*").Id("HealthServer")).Block(
				Id("hs").Dot("response").Op("=").Id("response"),
			)),
		)
}

func (r *transportRenderer) withCheckFunc() Code {

	return Func().Id("WithCheck").
		Params(Id("check").Func().Params(Id("ctx").Qual(PackageContext, "Context")).Params(Id("err").Error())).
		Params(Id("opt").Id("HealthOption")).
		Block(
			Return(Func().Params(Id("hs").Op("*").Id("HealthServer")).Block(
				Id("hs").Dot("check").Op("=").Id("check"),
			)),
		)
}

func (r *transportRenderer) healthServerHandleMethod() Code {

	return Func().Params(Id("hs").Op("*").Id("HealthServer")).
		Id("handle").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Id("err").Error()).
		BlockFunc(func(bg *Group) {
			bg.If(Id("hs").Dot("check").Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.List(Id("checkCtx"), Id("cancel")).Op(":=").Qual(PackageContext, "WithTimeout").Call(Id(VarNameFtx).Dot("UserContext").Call(), Id("hs").Dot("checkTimeout"))
				ig.Defer().Id("cancel").Call()
				ig.If(Err().Op("=").Id("hs").Dot("check").Call(Id("checkCtx")).Op(";").Err().Op("!=").Nil()).Block(
					Return(Id(VarNameFtx).Dot("SendStatus").Call(Qual(PackageFiber, "StatusServiceUnavailable"))),
				)
			})
			bg.Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Id("contentTypeJson"))
			bg.Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentLength").Call(Len(Id("hs").Dot("responseBody")))
			bg.List(Id("_"), Err()).Op("=").Id(VarNameFtx).Dot("Write").Call(Id("hs").Dot("responseBody"))
			bg.Return()
		})
}

func (r *transportRenderer) healthServerStopMethod() Code {

	return Func().Params(Id("hs").Op("*").Id("HealthServer")).
		Id("Stop").
		Params().
		Block(
			If(Id("hs").Dot("srv").Op("!=").Nil()).Block(
				Id("_").Op("=").Id("hs").Dot("srv").Dot("ShutdownWithTimeout").Call(Id("defaultShutdownTimeout")),
			),
		)
}

func (r *transportRenderer) httpServiceAccessorFunc(contract *model.Contract) Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id(contract.Name).
		Params().
		Params(Op("*").Id("http" + contract.Name)).
		Block(
			Return(Id("srv").Dot("http" + contract.Name)),
		)
}

func (r *transportRenderer) clientIDMiddleware() Code {

	srvctxPkgPath := r.pkgPath(r.outDir) + "/srvctx"
	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("clientIDMiddleware").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Error()).
		BlockFunc(func(bg *Group) {
			bg.Id("clientID").Op(":=").Id("srv").Dot("boundClientID").Call(Id("normalizeClientID").Call(Id(VarNameFtx).Dot("Get").Call(Lit("X-Client-Id"))))
			bg.Id("ctx").Op(":=").Id(VarNameFtx).Dot("UserContext").Call()
			bg.Id(VarNameFtx).Dot("SetUserContext").Call(
				Qual(srvctxPkgPath, "WithCtx").Call(Id("ctx"), Qual(srvctxPkgPath, "ClientID").Call(Id("clientID"))),
			)
			bg.Return(Id(VarNameFtx).Dot("Next").Call())
		})
}

func (r *transportRenderer) bearerMiddleware() Code {

	srvctxPkgPath := r.pkgPath(r.outDir) + "/srvctx"
	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("bearerMiddleware").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.If(Id("srv").Dot("auth").Op("==").Nil()).Block(
				Return(Id(VarNameFtx).Dot("Next").Call()),
			)
			bg.If(Qual(PackageStrings, "HasPrefix").Call(Id(VarNameFtx).Dot("Path").Call(), Lit("/.well-known/"))).Block(
				Return(Id(VarNameFtx).Dot("Next").Call()),
			)
			bg.Id("header").Op(":=").Id(VarNameFtx).Dot("Get").Call(Lit("Authorization"))
			bg.Const().Id("prefix").Op("=").Lit("Bearer ")
			bg.If(Len(Id("header")).Op("<").Len(Id("prefix")).Op("||").Op("!").Qual(PackageStrings, "EqualFold").Call(Id("header").Index(Empty(), Len(Id("prefix"))), Id("prefix"))).Block(
				Return(Id("srv").Dot("denyBearer").Call(Id(VarNameFtx), Lit("missing"), Lit(""))),
			)
			bg.Id("token").Op(":=").Qual(PackageStrings, "TrimSpace").Call(Id("header").Index(Len(Id("prefix")), Empty()))
			bg.Var().Id("subject").String()
			bg.Var().Id("out").Qual(PackageContext, "Context")
			bg.If(List(Id("out"), Id("subject"), Err()).Op("=").Id("srv").Dot("auth").Dot("Authenticate").Call(Id(VarNameFtx).Dot("UserContext").Call(), Id("token")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Id("reason").Op(":=").Lit("invalid")
				ig.If(Qual(PackageErrors, "Is").Call(Err(), Id("errBearerForbidden"))).Block(
					Id("reason").Op("=").Lit("forbidden"),
				)
				ig.Return(Id("srv").Dot("denyBearer").Call(Id(VarNameFtx), Id("reason"), Id("subject")))
			})
			bg.Id(VarNameFtx).Dot("SetUserContext").Call(
				Qual(srvctxPkgPath, "WithCtx").Call(Id("out"), Qual(srvctxPkgPath, "Subject").Call(Id("subject"))),
			)
			bg.Return(Id(VarNameFtx).Dot("Next").Call())
		}).Line().
		Func().Params(Id("srv").Op("*").Id("Server")).
		Id("denyBearer").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx"), Id("reason").String(), Id("subject").String()).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.If(Id("srv").Dot("log").Op("!=").Nil()).BlockFunc(func(lg *Group) {
				lg.Id("attrs").Op(":=").Index().Any().Values(Lit("reason"), Id("reason"))
				lg.If(Id("subject").Op("!=").Lit("")).Block(
					Id("attrs").Op("=").Append(Id("attrs"), Lit("subject"), Id("subject")),
				)
				lg.Id("srv").Dot("log").Dot("Warn").Call(Lit("auth denied"), Id("attrs").Op("..."))
			})
			if r.hasMetrics() {
				bg.If(Id("srv").Dot("metrics").Op("!=").Nil()).Block(
					Id("srv").Dot("metrics").Dot("AuthDenied").Dot("WithLabelValues").Call(Id("reason")).Dot("Inc").Call(),
				)
			}
			bg.If(Id("reason").Op("==").Lit("forbidden")).Block(
				Return(Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusForbidden")).Dot("SendString").Call(Lit("forbidden"))),
			)
			bg.If(Id("srv").Dot("authChallenge").Op("!=").Lit("")).Block(
				Id(VarNameFtx).Dot("Set").Call(Lit("WWW-Authenticate"), Id("srv").Dot("authChallenge")),
			)
			bg.Return(Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusUnauthorized")).Dot("SendString").Call(Lit("unauthorized")))
		}).Line().
		Func().Params(Id("srv").Op("*").Id("Server")).
		Id("resourceMetadata").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.If(Len(Id("srv").Dot("authMetadata")).Op("==").Lit(0)).Block(
				Return(Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusInternalServerError")).Dot("SendString").Call(Lit("resource metadata is not configured"))),
			)
			bg.Id(VarNameFtx).Dot("Set").Call(Lit("Content-Type"), Lit("application/json"))
			bg.Id(VarNameFtx).Dot("Set").Call(Lit("Cache-Control"), Lit("public, max-age=300"))
			bg.Return(Id(VarNameFtx).Dot("Send").Call(Id("srv").Dot("authMetadata")))
		})
}

func (r *transportRenderer) inFlightMiddleware() Code {

	srvctxPkgPath := r.pkgPath(r.outDir) + "/srvctx"
	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("inFlightMiddleware").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Error()).
		BlockFunc(func(bg *Group) {
			bg.If(Id("srv").Dot("metrics").Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Id("path").Op(":=").Id("metricRoutePath").Call(Id(VarNameFtx))
				ig.Id("clientID").Op(":=").Qual(srvctxPkgPath, "GetClientID").Call(Id(VarNameFtx).Dot("UserContext").Call())
				ig.Id("srv").Dot("metrics").Dot("RequestsInFlight").Dot("WithLabelValues").Call(Id("path"), Id("clientID")).Dot("Inc").Call()
				ig.Defer().Id("srv").Dot("metrics").Dot("RequestsInFlight").Dot("WithLabelValues").Call(Id("path"), Id("clientID")).Dot("Dec").Call()
			})
			bg.Return(Id(VarNameFtx).Dot("Next").Call())
		})
}

func (r *transportRenderer) requestDurationMiddleware() Code {

	srvctxPkgPath := r.pkgPath(r.outDir) + "/srvctx"
	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("requestDurationMiddleware").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Error()).
		BlockFunc(func(bg *Group) {
			bg.If(Id("srv").Dot("metrics").Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Id("start").Op(":=").Qual(PackageTime, "Now").Call()
				ig.Defer().Func().Params().Block(
					Id("clientID").Op(":=").Qual(srvctxPkgPath, "GetClientID").Call(Id(VarNameFtx).Dot("UserContext").Call()),
					Id("srv").Dot("metrics").Dot("RequestDuration").Dot("WithLabelValues").Call(Id("clientID")).Dot("Observe").Call(
						Qual(PackageTime, "Since").Call(Id("start")).Dot("Seconds").Call(),
					),
				).Call()
			})
			bg.Return(Id(VarNameFtx).Dot("Next").Call())
		})
}
