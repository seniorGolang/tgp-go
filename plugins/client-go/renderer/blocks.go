// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

// Логика метрик вынесена в хелпер, чтобы не дублировать код в каждом HTTP-методе.
func (r *ClientRenderer) httpRecordHTTPMetricsHelper(contract *model.Contract) Code {

	serviceLabel := r.contractNameToLowerCamel(contract)
	return Line().
		Func().Params(Id("cli").Op("*").Id("Client"+contract.Name)).
		Id("recordHTTPMetrics").Params(
		Id("method").String(),
		Id("_begin").Qual(PackageTime, "Time"),
		Id("err").Error(),
	).
		Block(
			If(Id("cli").Dot("Client").Dot("metrics").Op("==").Nil()).Block(
				Return(),
			),
			Var().Defs(
				Id("success").Op("=").True(),
				Id("errCode").Op("=").Qual(PackageHttp, "StatusInternalServerError"),
			),
			If(Id("err").Op("!=").Nil()).Block(
				Id("success").Op("=").False(),
				List(Id("ec"), Id("ok")).Op(":=").Id("err").Assert(Id("withErrorCode")),
				If(Id("ok")).Block(
					Id("errCode").Op("=").Id("ec").Dot("Code").Call(),
				),
			),
			Var().Id("successStr").String(),
			Var().Id("errCodeStr").String(),
			If(Id("success")).Block(
				List(Id("successStr"), Id("errCodeStr")).Op("=").List(Lit("true"), Lit("0")),
			).Else().Block(
				List(Id("successStr"), Id("errCodeStr")).Op("=").List(Lit("false"), Qual(PackageStrconv, "Itoa").Call(Id("errCode"))),
			),
			Id("cli").Dot("Client").Dot("metrics").Dot("RequestCount").Dot("WithLabelValues").Call(
				Lit(serviceLabel),
				Id("method"),
				Id("successStr"),
				Id("errCodeStr"),
				Id("cli").Dot("Client").Dot("name")).
				Dot("Add").Call(Lit(1)),
			Id("cli").Dot("Client").Dot("metrics").Dot("RequestCountAll").Dot("WithLabelValues").Call(
				Lit(serviceLabel),
				Id("method"),
				Id("successStr"),
				Id("errCodeStr"),
				Id("cli").Dot("Client").Dot("name")).
				Dot("Add").Call(Lit(1)),
			Id("cli").Dot("Client").Dot("metrics").Dot("RequestLatency").Dot("WithLabelValues").Call(
				Lit(serviceLabel),
				Id("method"),
				Id("successStr"),
				Id("errCodeStr"),
				Id("cli").Dot("Client").Dot("name")).
				Dot("Observe").Call(Qual(PackageTime, "Since").Call(Id("_begin")).Dot("Seconds").Call()),
		)
}

func (r *ClientRenderer) httpMetricsDefer(contract *model.Contract, method *model.Method) Code {

	return Defer().Func().Params(Id("_begin").Qual(PackageTime, "Time")).Block(
		Id("cli").Dot("recordHTTPMetrics").Call(
			Lit(r.methodNameToLowerCamel(method)),
			Id("_begin"),
			Err(),
		),
	).Call(Qual(PackageTime, "Now").Call()).Line()
}

func (r *ClientRenderer) rpcRecordMetricsHelper(contract *model.Contract) Code {

	serviceLabel := r.contractNameToLowerCamel(contract)
	return Line().
		Func().Params(Id("cli").Op("*").Id("Client"+contract.Name)).
		Id("recordRPCMetrics").Params(
		Id("method").String(),
		Id("_begin").Qual(PackageTime, "Time"),
		Err().Error(),
	).
		Block(
			If(Id("cli").Dot("metrics").Op("==").Nil()).Block(
				Return(),
			),
			Var().Defs(
				Id("success").Op("=").True(),
				Id("errCode").Op("=").Id("internalError"),
			),
			If(Err().Op("!=").Nil()).Block(
				Id("success").Op("=").False(),
				List(Id("ec"), Id("ok")).Op(":=").Err().Assert(Id("withErrorCode")),
				If(Id("ok")).Block(
					Id("errCode").Op("=").Id("ec").Dot("Code").Call(),
				),
			),
			Var().Id("successStr").String(),
			Var().Id("errCodeStr").String(),
			If(Id("success")).Block(
				List(Id("successStr"), Id("errCodeStr")).Op("=").List(Lit("true"), Lit("0")),
			).Else().Block(
				List(Id("successStr"), Id("errCodeStr")).Op("=").List(Lit("false"), Qual(PackageStrconv, "Itoa").Call(Id("errCode"))),
			),
			Id("cli").Dot("metrics").Dot("RequestCount").Dot("WithLabelValues").Call(
				Lit(serviceLabel),
				Id("method"),
				Id("successStr"),
				Id("errCodeStr"),
				Id("cli").Dot("Client").Dot("name")).
				Dot("Add").Call(Lit(1)),
			Id("cli").Dot("metrics").Dot("RequestCountAll").Dot("WithLabelValues").Call(
				Lit(serviceLabel),
				Id("method"),
				Id("successStr"),
				Id("errCodeStr"),
				Id("cli").Dot("Client").Dot("name")).
				Dot("Add").Call(Lit(1)),
			Id("cli").Dot("metrics").Dot("RequestLatency").Dot("WithLabelValues").Call(
				Lit(serviceLabel),
				Id("method"),
				Id("successStr"),
				Id("errCodeStr"),
				Id("cli").Dot("Client").Dot("name")).
				Dot("Observe").Call(Qual(PackageTime, "Since").Call(Id("_begin")).Dot("Seconds").Call()),
		)
}

func (r *ClientRenderer) rpcMetricsDefer(contract *model.Contract, method *model.Method) Code {

	return Defer().Func().Params(Id("_begin").Qual(PackageTime, "Time")).Block(
		Id("cli").Dot("recordRPCMetrics").Call(
			Lit(r.methodNameToLowerCamel(method)),
			Id("_begin"),
			Err(),
		),
	).Call(Qual(PackageTime, "Now").Call())
}

func (r *ClientRenderer) httpApplyHeadersFromCtxHelper(contract *model.Contract) Code {

	return Line().
		Func().Params(Id("cli").Op("*").Id("Client"+contract.Name)).
		Id("applyHeadersFromCtx").Params(
		Id("ctx").Qual(PackageContext, "Context"),
		Id("req").Op("*").Qual(PackageHttp, "Request"),
	).
		Block(
			For(List(Id("_"), Id("header")).Op(":=").Range().Id("cli").Dot("Client").Dot("headersFromCtx")).Block(
				If(Id("value").Op(":=").Id("ctx").Dot("Value").Call(Id("header")).Op(";").Id("value").Op("!=").Nil()).Block(
					Var().Id("k").String(),
					Var().Id("v").String(),
					If(List(Id("h"), Id("ok")).Op(":=").Id("header").Assert(String()).Op(";").Id("ok")).Block(
						Id("k").Op("=").Id("h"),
					).Else().If(List(Id("h"), Id("ok")).Op(":=").Id("header").Assert(Qual(PackageFmt, "Stringer")).Op(";").Id("ok")).Block(
						Id("k").Op("=").Id("h").Dot("String").Call(),
					).Else().Block(
						Id("k").Op("=").Qual(PackageFmt, "Sprint").Call(Id("header")),
					),
					If(List(Id("val"), Id("ok")).Op(":=").Id("value").Assert(String()).Op(";").Id("ok")).Block(
						Id("v").Op("=").Id("val"),
					).Else().If(List(Id("val"), Id("ok")).Op(":=").Id("value").Assert(Qual(PackageFmt, "Stringer")).Op(";").Id("ok")).Block(
						Id("v").Op("=").Id("val").Dot("String").Call(),
					).Else().Block(
						Id("v").Op("=").Qual(PackageFmt, "Sprint").Call(Id("value")),
					),
					If(Id("k").Op("!=").Lit("").Op("&&").Id("v").Op("!=").Lit("")).Block(
						Id("req").Dot("Header").Dot("Set").Call(Id("k"), Id("v")),
					),
				),
			),
		)
}

// Вызывается из каждого HTTP-метода вместо дублирования Do/defer log/afterRequest/checkStatusCode.
func (r *ClientRenderer) httpDoRoundTripHelper(contract *model.Contract, outDir string) Code {

	jsonrpcPkg := fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir))
	return Line().
		Func().Params(Id("cli").Op("*").Id("Client"+contract.Name)).
		Id("doRoundTrip").Params(
		Id("ctx").Qual(PackageContext, "Context"),
		Id("methodName").String(),
		Id("httpReq").Op("*").Qual(PackageHttp, "Request"),
		Id("successCode").Int(),
	).Params(Id("httpResp").Op("*").Qual(PackageHttp, "Response"), Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.Id("httpReq").Dot("Header").Dot("Set").Call(Lit("X-Client-Id"), Id("cli").Dot("Client").Dot("name"))
			bg.Id("cli").Dot("applyHeadersFromCtx").Call(Id("ctx"), Id("httpReq"))
			bg.If(Id("cli").Dot("Client").Dot("beforeRequest").Op("!=").Nil()).Block(
				Id("ctx").Op("=").Id("cli").Dot("Client").Dot("beforeRequest").Call(Id("ctx"), Id("httpReq")),
			)
			bg.If(Id("cli").Dot("Client").Dot("logRequests")).Block(
				If(List(Id("cmd"), Id("cmdErr")).Op(":=").Qual(jsonrpcPkg, "ToCurl").Call(Id("httpReq")).Op(";").Id("cmdErr").Op("==").Nil()).Block(
					Qual(PackageSlog, "DebugContext").Call(Id("ctx"), Lit("HTTP request"), Qual(PackageSlog, "String").Call(Lit("method"), Id("httpReq").Dot("Method")), Qual(PackageSlog, "String").Call(Lit("curl"), Id("cmd").Dot("String").Call())),
				),
			)
			bg.List(Id("httpResp"), Err()).Op("=").Id("cli").Dot("httpClient").Dot("Do").Call(Id("httpReq"))
			bg.Defer().Func().Params().Block(
				If(Err().Op("!=").Nil().Op("&&").Id("cli").Dot("Client").Dot("logOnError").Op("&&").Id("httpReq").Op("!=").Nil()).Block(
					If(List(Id("cmd"), Id("cmdErr")).Op(":=").Qual(jsonrpcPkg, "ToCurl").Call(Id("httpReq")).Op(";").Id("cmdErr").Op("==").Nil()).Block(
						Qual(PackageSlog, "ErrorContext").Call(Id("ctx"), Lit("HTTP request failed"), Qual(PackageSlog, "String").Call(Lit("method"), Id("httpReq").Dot("Method")), Qual(PackageSlog, "String").Call(Lit("curl"), Id("cmd").Dot("String").Call()), Qual(PackageSlog, "Any").Call(Lit("error"), Err())),
					),
				),
			).Call()
			bg.If(Err().Op("!=").Nil()).Block(Return(Nil(), Err()))
			bg.If(Id("cli").Dot("Client").Dot("afterRequest").Op("!=").Nil()).Block(
				If(Err().Op("=").Id("cli").Dot("Client").Dot("afterRequest").Call(Id("ctx"), Id("httpResp")).Op(";").Err().Op("!=").Nil()).Block(
					Id("_").Op("=").Id("httpResp").Dot("Body").Dot("Close").Call(),
					Return(Nil(), Err()),
				),
			)
			bg.If(Id("httpResp").Dot("StatusCode").Op("!=").Id("successCode")).BlockFunc(func(bgErr *Group) {
				bgErr.Var().Id("respBodyBytes").Index().Byte()
				bgErr.List(Id("respBodyBytes"), Err()).Op("=").Qual(PackageIO, "ReadAll").Call(Id("httpResp").Dot("Body"))
				bgErr.Id("httpResp").Dot("Body").Dot("Close").Call()
				bgErr.If(Err().Op("!=").Nil()).Block(
					Err().Op("=").Qual(PackageFmt, "Errorf").Call(
						Lit("HTTP error: %d. URL: %s, Method: %s"),
						Id("httpResp").Dot("StatusCode"),
						Id("httpReq").Dot("URL").Dot("String").Call(),
						Id("httpReq").Dot("Method"),
					),
				).Else().Block(
					Err().Op("=").Id("cli").Dot("errorDecoder").Call(Id("respBodyBytes")),
				)
				bgErr.Return(Nil(), Err())
			})
			bg.Return(Id("httpResp"), Nil())
		})
}

func (r *ClientRenderer) httpDeferBodyClose() Code {

	return Defer().Id("httpResp").Dot("Body").Dot("Close").Call()
}
