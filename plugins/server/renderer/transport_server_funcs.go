// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

func (r *transportRenderer) fiberFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("Fiber").
		Params().
		Params(Op("*").Qual(PackageFiber, "App")).
		Block(
			Return(Id("srv").Dot("srvHTTP")),
		)
}

func (r *transportRenderer) withLogFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("WithLog").
		Params().
		Params(Op("*").Id("Server")).
		BlockFunc(func(bg *Group) {
			for _, contract := range r.project.Contracts {
				if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerHTTP) {
					bg.If(Id("srv").Dot("http" + contract.Name).Op("!=").Nil()).Block(
						Id("srv").Dot("http" + contract.Name).Op("=").Id("srv").Dot(contract.Name).Call().Dot("WithLog").Call(),
					)
				}
				if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerJsonRPC) {
					bg.If(Id("srv").Dot("http" + contract.Name).Op("!=").Nil()).Block(
						Id("srv").Dot("http" + contract.Name).Op("=").Id("srv").Dot("http" + contract.Name).Dot("WithLog").Call(),
					)
				}
			}
			bg.Return(Id("srv"))
		})
}

func (r *transportRenderer) withTraceFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("WithTrace").
		Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("appName").String(), Id("endpoint").String(), Id("attributes").Op("...").Qual(PackageAttributeOTEL, "KeyValue")).
		Params(Op("*").Id("Server")).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Qual(fmt.Sprintf("%s/tracer", r.pkgPath(r.outDir)), "Init").Call(Id(VarNameCtx), Id("appName"), Id("endpoint"), Id("attributes").Op("..."))
			for _, contract := range r.project.Contracts {
				if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerHTTP) {
					bg.If(Id("srv").Dot("http" + contract.Name).Op("!=").Nil()).Block(
						Id("srv").Dot("http" + contract.Name).Op("=").Id("srv").Dot(contract.Name).Call().Dot("WithTrace").Call(),
					)
				}
				if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerJsonRPC) {
					bg.If(Id("srv").Dot("http" + contract.Name).Op("!=").Nil()).Block(
						Id("srv").Dot("http" + contract.Name).Op("=").Id("srv").Dot("http" + contract.Name).Dot("WithTrace").Call(),
					)
				}
			}
			bg.Return(Id("srv"))
		})
}

func (r *transportRenderer) withMetricsFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("WithMetrics").
		Params().
		Params(Op("*").Id("Server")).
		BlockFunc(func(bg *Group) {
			bg.If(Id("srv").Dot("metrics").Op("==").Nil()).Block(
				Id("srv").Dot("metrics").Op("=").Id("NewMetrics").Call(),
			)
			for _, contract := range r.project.Contracts {
				if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerHTTP) {
					bg.If(Id("srv").Dot("http" + contract.Name).Op("!=").Nil()).Block(
						Id("srv").Dot("http" + contract.Name).Op("=").Id("srv").Dot(contract.Name).Call().Dot("WithMetrics").Call(Id("srv").Dot("metrics")),
					)
				}
				if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerJsonRPC) {
					bg.If(Id("srv").Dot("http" + contract.Name).Op("!=").Nil()).Block(
						Id("srv").Dot("http" + contract.Name).Op("=").Id("srv").Dot("http" + contract.Name).Dot("WithMetrics").Call(Id("srv").Dot("metrics")),
					)
				}
			}
			bg.Return(Id("srv"))
		})
}

func (r *transportRenderer) serverNewFunc() Code {

	return Func().Id("New").
		Params(Id("log").Op("*").Qual(PackageSlog, "Logger"), Id("options").Op("...").Id("Option")).
		Params(Id("srv").Op("*").Id("Server")).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Id("srv").Op("=").Op("&").Id("Server").Values(DictFunc(func(dict Dict) {
				dict[Id("log")] = Id("log")
				if r.hasJsonRPC() {
					dict[Id("maxBatchSize")] = Id("defaultMaxBatchSize")
					dict[Id("maxParallelBatch")] = Id("defaultMaxParallelBatch")
				}
				dict[Id("headerHandlers")] = Make(Map(String()).Id("HeaderHandler"))
				dict[Id("config")] = Qual(PackageFiber, "Config").Values(Dict{
					Id("StreamRequestBody"):     True(),
					Id("DisableStartupMessage"): True(),
					Id("BodyLimit"):             Id("defaultBodyLimit"),
					Id("Concurrency"):           Id("defaultConcurrency"),
					Id("IdleTimeout"):           Id("defaultIdleTimeout"),
					Id("ReadTimeout"):           Id("defaultReadTimeout"),
					Id("WriteTimeout"):          Id("defaultWriteTimeout"),
					Id("ReadBufferSize"):        Id("defaultReadBufferSize"),
					Id("WriteBufferSize"):       Id("defaultWriteBufferSize"),
				})
			},
			))
			bg.Line()
			bg.Var().Id("configOptions").Index().Id("Option")
			bg.Var().Id("serviceOptions").Index().Id("Option")
			bg.Line()
			bg.For(List(Id("_"), Id("option")).Op(":=").Range().Id("options")).Block(
				If(Id("doesNotRequireHTTP").Call(Id("option"))).Block(
					Id("serviceOptions").Op("=").Append(Id("serviceOptions"), Id("option")),
				).Else().Block(
					Id("configOptions").Op("=").Append(Id("configOptions"), Id("option")),
				),
			)
			bg.Line()
			bg.For(List(Id("_"), Id("option")).Op(":=").Range().Id("configOptions")).Block(
				Id("option").Call(Id("srv")),
			)
			bg.Line()
			bg.Id("srv").Dot("srvHTTP").Op("=").Qual(PackageFiber, "New").Call(Id("srv").Dot("config"))
			bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Func().Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).Params(Error()).Block(
				Id(VarNameFtx).Dot("Locals").Call(Lit("server"), Id("srv")),
				Return(Id("recoverHandler").Call(Id(VarNameFtx))),
			))
			if r.hasMetrics() {
				bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("srv").Dot("clientIDMiddleware"))
				bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("srv").Dot("inFlightMiddleware"))
				bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("srv").Dot("requestDurationMiddleware"))
			}
			if r.hasJsonRPC() {
				bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("requestOverlayMiddleware"))
			}
			if r.hasTrace() {
				bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Qual(fmt.Sprintf("%s/tracer", r.pkgPath(r.outDir)), "Middleware").Call())
			}
			bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("srv").Dot("setLogger"))
			bg.Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("srv").Dot("headersHandler"))
			bg.For(List(Id("_"), Id("option")).Op(":=").Range().Id("serviceOptions")).Block(
				Id("option").Call(Id("srv")),
			)
			if r.hasJsonRPC() {
				bg.Id("srv").Dot("srvHTTP").Dot("Post").Call(Lit("/"), Id("srv").Dot("serveBatch"))
			}
			if r.hasJsonRPC() {
				bg.Id("initJsonRPCMethodMap").Call(Id("srv"))
			}
			bg.Return()
		})
}

func (r *transportRenderer) serveHealthFunc() Code {

	jsonPkg := r.getPackageJSON()
	return Func().Id("ServeHealth").
		Params(
			Id("log").Op("*").Qual(PackageSlog, "Logger"),
			Id("path").String(),
			Id("address").String(),
			Id("response").Any(),
		).
		Params(Op("*").Id("HealthServer")).
		BlockFunc(func(bg *Group) {
			bg.Var().Err().Error()
			bg.Var().Id("responseBody").Index().Byte()
			bg.If(Id("response").Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.List(Id("responseBody"), Err()).Op("=").Qual(jsonPkg, "Marshal").Call(Id("response"))
				ig.If(Err().Op("!=").Nil()).BlockFunc(func(eg *Group) {
					eg.Id("log").Dot("Error").Call(Lit("failed to marshal health response"), Qual(PackageSlog, "Any").Call(Lit("error"), Err()))
					eg.Id("responseBody").Op("=").Op("[]").Byte().Call(Lit(`{"status":"error","message":"health check misconfigured"}`))
				})
			}).Else().Block(
				Id("responseBody").Op("=").Op("[]").Byte().Call(Lit(`"ok"`)),
			)
			bg.Id("srv").Op(":=").Qual(PackageFiber, "New").Call(Qual(PackageFiber, "Config").Values(Dict{
				Id("DisableStartupMessage"): True(),
				Id("IdleTimeout"):           Id("defaultIdleTimeout"),
				Id("ReadTimeout"):           Id("defaultReadTimeout"),
				Id("WriteTimeout"):          Id("defaultWriteTimeout"),
			}))
			bg.Id("srv").Dot("Get").Call(Id("path"),
				Func().Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).Params(Id("err").Error()).BlockFunc(func(hg *Group) {
					hg.Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Id("contentTypeJson"))
					hg.Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentLength").Call(Len(Id("responseBody")))
					hg.List(Id("_"), Id("err")).Op("=").Id(VarNameFtx).Dot("Write").Call(Id("responseBody"))
					hg.Return()
				}))
			bg.Go().Func().Params().Block(
				Err().Op("=").Id("srv").Dot("Listen").Call(Id("address")),
				Id("ExitOnError").Call(Id("log"), Err(), Lit("serve health on ").Op("+").Id("address")),
			).Call()
			bg.Return(Op("&").Id("HealthServer").Values(Dict{
				Id("responseBody"): Id("responseBody"),
				Id("srv"):          Id("srv"),
			}))
		})
}

func (r *transportRenderer) shutdownFunc() Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id("Shutdown").
		Params().
		Params(Id("err").Error()).
		BlockFunc(func(bg *Group) {
			bg.If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).Block(
				If(Err().Op("=").Id("srv").Dot("srvHTTP").Dot("ShutdownWithTimeout").Call(Id("defaultShutdownTimeout")).Op(";").Err().Op("!=").Nil()).Block(
					Return(Err()),
				),
			)
			if r.hasMetrics() {
				bg.If(Id("srv").Dot("srvMetrics").Op("!=").Nil()).Block(
					If(Err().Op("=").Id("srv").Dot("srvMetrics").Dot("ShutdownWithTimeout").Call(Id("defaultShutdownTimeout")).Op(";").Err().Op("!=").Nil()).Block(
						Return(Err()),
					),
				)
			}
			bg.Return()
		})
}

func (r *transportRenderer) sendResponseFunc() Code {

	jsonPkg := r.getPackageJSON()
	return Func().Id("sendResponse").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx"), Id("resp").Any()).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.Id(VarNameFtx).Dot("Response").Call().Dot("Header").Dot("SetContentType").Call(Lit("application/json"))
			bg.If(Err().Op("=").Qual(jsonPkg, "NewEncoder").Call(Id(VarNameFtx).Dot("Response").Call().Dot("BodyWriter").Call()).Dot("Encode").Call(Id("resp")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.If(Id("logger").Op(":=").Qual(fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir)), "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(Id(VarNameFtx).Dot("UserContext").Call()).Op(";").Id("logger").Op("!=").Nil()).Block(
					Id("logger").Dot("Error").Call(Lit("response marshal error"), Qual(PackageSlog, "Any").Call(Lit("error"), Err())),
				)
				ig.Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusInternalServerError"))
				ig.Return(Err())
			})
			bg.Return(Err())
		})
}

func (r *transportRenderer) sendHTTPErrorFunc() Code {

	srvctxPkgPath := fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir))
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

func (r *transportRenderer) doesNotRequireHTTPFunc() Code {

	return Func().Id("doesNotRequireHTTP").
		Params(Id("option").Id("Option")).
		Params(Bool()).
		BlockFunc(func(bg *Group) {
			bg.Id("testSrv").Op(":=").Op("&").Id("Server").Do(func(s *Statement) {
				s.Add(Op("{"))
				s.Line()
				s.Id("headerHandlers").Op(":").Make(Map(String()).Id("HeaderHandler")).Op(",")
				s.Line()
				s.Add(Op("}"))
			})
			bg.Id("option").Call(Id("testSrv"))
			bg.Id("noHTTPService").Op(":=").Id("testSrv").Dot("srvHTTP").Op("==").Nil()
			for _, contract := range r.project.Contracts {
				if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerHTTP) ||
					model.IsAnnotationSet(r.project, contract, nil, nil, TagServerJsonRPC) {
					bg.Id("noHTTPService").Op("=").Id("noHTTPService").Op("&&").Id("testSrv").Dot("http" + contract.Name).Op("==").Nil()
				}
			}
			bg.Return(Id("noHTTPService"))
		})
}
