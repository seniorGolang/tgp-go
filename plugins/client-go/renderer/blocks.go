// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

// Логика метрик вынесена в хелпер, чтобы не дублировать код в каждом HTTP-методе.
func (r *ClientRenderer) httpMetricsDefer(contract *model.Contract, method *model.Method) (c Code) {

	return Defer().Func().Params(Id("_begin").Qual(PackageTime, "Time")).Block(
		Id("cli").Dot("recordHTTPMetrics").Call(
			Lit(r.contractNameToLowerCamel(contract)),
			Lit(r.methodNameToLowerCamel(method)),
			Id("_begin"),
			Err(),
		),
	).Call(Qual(PackageTime, "Now").Call()).Line()
}

func (r *ClientRenderer) rpcRecordMetricsHelper(contract *model.Contract) (c Code) {

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

func (r *ClientRenderer) rpcMetricsDefer(contract *model.Contract, method *model.Method) (c Code) {

	return Defer().Func().Params(Id("_begin").Qual(PackageTime, "Time")).Block(
		Id("cli").Dot("recordRPCMetrics").Call(
			Lit(r.methodNameToLowerCamel(method)),
			Id("_begin"),
			Err(),
		),
	).Call(Qual(PackageTime, "Now").Call())
}

func (r *ClientRenderer) httpDeferBodyClose() (c Code) {

	return Defer().Func().Params().Block(
		Id("_").Op("=").Id("httpResp").Dot("Body").Dot("Close").Call(),
	).Call()
}
