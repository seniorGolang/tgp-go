// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/generated"
)

func (r *ClientRenderer) RenderHTTP() (err error) {

	outDir := r.outDir
	jsonrpcPkg := fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir))

	srcFile := NewSrcFile(filepath.Base(outDir))
	srcFile.PackageComment(generated.ByToolGateway)
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(PackageFmt, "fmt")
	srcFile.ImportName(PackageIO, "io")
	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(PackageHttp, "http")
	srcFile.ImportName(PackageStrconv, "strconv")
	srcFile.ImportName(PackageTime, "time")
	srcFile.ImportName(jsonrpcPkg, "jsonrpc")

	srcFile.Line().Add(r.httpApplyHeadersFromCtxFunc())
	srcFile.Line().Add(r.httpDoRoundTripFunc(outDir))
	if r.HasMetrics() {
		srcFile.Line().Add(r.httpRecordHTTPMetricsFunc())
	}

	err = srcFile.Save(path.Join(outDir, "http.go"))
	return
}

func (r *ClientRenderer) httpApplyHeadersFromCtxFunc() (c Code) {

	return Func().Params(Id("cli").Op("*").Id("Client")).
		Id("applyHeadersFromCtx").Params(
		Id("ctx").Qual(PackageContext, "Context"),
		Id("req").Op("*").Qual(PackageHttp, "Request"),
	).
		Block(
			For(List(Id("_"), Id("header")).Op(":=").Range().Id("cli").Dot("headersFromCtx")).Block(
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

func (r *ClientRenderer) httpDoRoundTripFunc(outDir string) (c Code) {

	jsonrpcPkg := fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir))
	return Func().Params(Id("cli").Op("*").Id("Client")).
		Id("doRoundTrip").Params(
		Id("ctx").Qual(PackageContext, "Context"),
		Id("methodName").String(),
		Id("httpReq").Op("*").Qual(PackageHttp, "Request"),
		Id("successCode").Int(),
	).Params(Id("httpResp").Op("*").Qual(PackageHttp, "Response"), Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.Id("httpReq").Dot("Header").Dot("Set").Call(Lit("X-Client-Id"), Id("cli").Dot("name"))
			bg.Id("cli").Dot("applyHeadersFromCtx").Call(Id("ctx"), Id("httpReq"))
			bg.If(Id("cli").Dot("beforeRequest").Op("!=").Nil()).Block(
				Id("ctx").Op("=").Id("cli").Dot("beforeRequest").Call(Id("ctx"), Id("httpReq")),
			)
			bg.Var().Id("curlCmd").String()
			bg.If(Id("cli").Dot("logRequests").Op("||").Id("cli").Dot("logOnError")).Block(
				If(List(Id("cmd"), Id("cmdErr")).Op(":=").Qual(jsonrpcPkg, "ToCurl").Call(Id("httpReq")).Op(";").Id("cmdErr").Op("==").Nil()).Block(
					Id("curlCmd").Op("=").Id("cmd").Dot("String").Call(),
					If(Id("cli").Dot("logRequests")).Block(
						Qual(PackageSlog, "DebugContext").Call(Id("ctx"), Lit("HTTP request"), Qual(PackageSlog, "String").Call(Lit("method"), Id("httpReq").Dot("Method")), Qual(PackageSlog, "String").Call(Lit("curl"), Id("curlCmd"))),
					),
				),
			)
			bg.Defer().Func().Params().Block(
				If(Err().Op("!=").Nil().Op("&&").Id("cli").Dot("logOnError")).Block(
					Qual(PackageSlog, "ErrorContext").Call(Id("ctx"), Lit("HTTP request failed"), Qual(PackageSlog, "String").Call(Lit("method"), Id("httpReq").Dot("Method")), Qual(PackageSlog, "String").Call(Lit("curl"), Id("curlCmd")), Qual(PackageSlog, "Any").Call(Lit("error"), Err())),
				),
			).Call()
			bg.If(List(Id("httpResp"), Err()).Op("=").Id("cli").Dot("httpClient").Dot("Do").Call(Id("httpReq")).Op(";").Err().Op("!=").Nil()).Block(Return(Nil(), Err()))
			bg.If(Id("cli").Dot("afterRequest").Op("!=").Nil()).Block(
				If(Err().Op("=").Id("cli").Dot("afterRequest").Call(Id("ctx"), Id("httpResp")).Op(";").Err().Op("!=").Nil()).Block(
					Id("_").Op("=").Id("httpResp").Dot("Body").Dot("Close").Call(),
					Return(Nil(), Err()),
				),
			)
			bg.If(Id("httpResp").Dot("StatusCode").Op("!=").Id("successCode")).BlockFunc(func(bgErr *Group) {
				bgErr.Var().Id("respBodyBytes").Index().Byte()
				bgErr.If(List(Id("respBodyBytes"), Err()).Op("=").Qual(PackageIO, "ReadAll").Call(Id("httpResp").Dot("Body")).Op(";").Err().Op("!=").Nil()).Block(
					Err().Op("=").Qual(PackageFmt, "Errorf").Call(
						Lit("HTTP error: %d. URL: %s, Method: %s"),
						Id("httpResp").Dot("StatusCode"),
						Id("httpReq").Dot("URL").Dot("String").Call(),
						Id("httpReq").Dot("Method"),
					),
				).Else().Block(
					Err().Op("=").Id("cli").Dot("httpErrorDecoder").Call(
						Id("httpResp").Dot("StatusCode"),
						Id("httpResp").Dot("Header").Dot("Get").Call(Lit("Content-Type")),
						Id("respBodyBytes"),
					),
				)
				bgErr.Id("_").Op("=").Id("httpResp").Dot("Body").Dot("Close").Call()
				bgErr.Return(Nil(), Err())
			})
			bg.Return(Id("httpResp"), Nil())
		})
}

func (r *ClientRenderer) httpRecordHTTPMetricsFunc() (c Code) {

	return Func().Params(Id("cli").Op("*").Id("Client")).
		Id("recordHTTPMetrics").Params(
		Id("serviceLabel").String(),
		Id("method").String(),
		Id("_begin").Qual(PackageTime, "Time"),
		Id("err").Error(),
	).
		Block(
			If(Id("cli").Dot("metrics").Op("==").Nil()).Block(
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
			Id("cli").Dot("metrics").Dot("RequestCount").Dot("WithLabelValues").Call(
				Id("serviceLabel"),
				Id("method"),
				Id("successStr"),
				Id("errCodeStr"),
				Id("cli").Dot("name")).
				Dot("Add").Call(Lit(1)),
			Id("cli").Dot("metrics").Dot("RequestCountAll").Dot("WithLabelValues").Call(
				Id("serviceLabel"),
				Id("method"),
				Id("successStr"),
				Id("errCodeStr"),
				Id("cli").Dot("name")).
				Dot("Add").Call(Lit(1)),
			Id("cli").Dot("metrics").Dot("RequestLatency").Dot("WithLabelValues").Call(
				Id("serviceLabel"),
				Id("method"),
				Id("successStr"),
				Id("errCodeStr"),
				Id("cli").Dot("name")).
				Dot("Observe").Call(Qual(PackageTime, "Since").Call(Id("_begin")).Dot("Seconds").Call()),
		)
}
