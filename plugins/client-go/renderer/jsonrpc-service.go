// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"context"
	"fmt"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

func (r *ClientRenderer) jsonrpcClientMethodFunc(ctx context.Context, contract *model.Contract, method *model.Method, outDir string) (c Code) {

	return Func().
		Params(Id("cli").Op("*").Id("Client" + contract.Name)).
		Id(method.Name).
		Params(r.clientMethodParamsJsonRPC(ctx, contract, method)).Params(r.funcDefinitionParams(ctx, method.Results)).BlockFunc(func(bg *Group) {

		if r.HasMetrics() && model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
			bg.Line().Add(r.rpcMetricsDefer(contract, method))
		}

		bg.Line()
		bg.Id("_request").Op(":=").Id(r.requestStructName(contract, method)).Values(DictFunc(func(dict Dict) {
			for _, arg := range r.argsForExchangeRequest(contract, method) {
				dict[Id(ToCamel(arg.Name))] = Id(ToLowerCamel(arg.Name))
			}
		}))
		bg.Var().Id("_response_").Id(r.responseStructName(contract, method))
		bg.Var().Id("rpcResponse").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "ResponseRPC")
		bg.If(List(Id("rpcResponse"), Err()).Op("=").Id("cli").Dot("rpc").Dot("Call").Call(Id(_ctx_), Lit(r.jsonRPCWireMethod(contract, method)), Id("_request")).Op(";").Err().Op("!=").Nil().Op("||").Id("rpcResponse").Op("==").Nil()).Block(
			Return(),
		)
		bg.If(Id("rpcResponse").Dot("Error").Op("!=").Nil()).Block(
			If(Id("cli").Dot("errorDecoder").Op("!=").Nil()).Block(
				Err().Op("=").Id("cli").Dot("errorDecoder").Call(Id("rpcResponse").Dot("Error").Dot("Raw").Call()),
			).Else().Block(
				Err().Op("=").Qual(PackageFmt, "Errorf").Call(Lit("%s"), Id("rpcResponse").Dot("Error").Dot("Message")),
			),
			Return(),
		)
		resp := Id("_response_")
		resultsWithoutErr := r.resultsWithoutError(method)
		if len(resultsWithoutErr) == 1 && model.IsAnnotationSet(r.project, contract, method, nil, model.TagHttpEnableInlineSingle) {
			resp = Id("_response_").Dot(ToCamel(resultsWithoutErr[0].Name))
		}
		jsonPkg := r.getPackageJSON(contract)
		bg.If(Err().Op("=").Qual(jsonPkg, "Unmarshal").Call(Id("rpcResponse").Dot("Result"), Op("&").Add(resp)).Op(";").Err().Op("!=").Nil()).Block(
			Return(),
		)
		bg.ReturnFunc(func(rg *Group) {
			fieldsResult := r.fieldsResult(method)
			// fieldsResult и resultsWithoutErr имеют одинаковый порядок и количество элементов
			for i, ret := range resultsWithoutErr {
				if i >= len(fieldsResult) {
					rg.Add(r.clientRPCResultValue(contract, method, ret, exchangeField{name: ret.Name, typeID: ret.TypeID, numberOfPointers: ret.NumberOfPointers}, "_response_"))
					continue
				}
				rg.Add(r.clientRPCResultValue(contract, method, ret, fieldsResult[i], "_response_"))
			}
			rg.Err()
		})
	})
}

func (r *ClientRenderer) jsonrpcClientRequestFunc(ctx context.Context, contract *model.Contract, method *model.Method, outDir string) (c Code) {

	return Func().Params(Id("cli").Op("*").Id("Client" + contract.Name)).
		Id("Req" + method.Name).
		Params(r.clientRequestMethodParamsJsonRPC(ctx, contract, method)).
		Params(Id("_request").Id("RequestRPC")).BlockFunc(func(bg *Group) {

		bg.Line()
		bg.Id("_request").Op("=").Id("RequestRPC").Values(Dict{
			Id("rpcRequest"): Op("&").Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "RequestRPC").Values(Dict{
				Id("ID"):      Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "NewID").Call(),
				Id("JSONRPC"): Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "Version"),
				Id("Method"):  Lit(r.jsonRPCWireMethod(contract, method)),
				Id("Params"): Id(r.requestStructName(contract, method)).Values(DictFunc(func(dg Dict) {
					for _, arg := range r.argsForExchangeRequest(contract, method) {
						dg[Id(ToCamel(arg.Name))] = Id(ToLowerCamel(arg.Name))
					}
				})),
			}),
		})
		resp := Id("_response_")
		resultsWithoutErr := r.resultsWithoutError(method)
		if len(resultsWithoutErr) == 1 && model.IsAnnotationSet(r.project, contract, method, nil, model.TagHttpEnableInlineSingle) {
			resp = Id("_response_").Dot(ToCamel(resultsWithoutErr[0].Name))
		}
		jsonPkg := r.getPackageJSON(contract)
		bg.If(Id("callback").Op("!=").Nil()).Block(
			Var().Id("_response_").Id(r.responseStructName(contract, method)),
			Id("_request").Dot("retHandler").Op("=").Func().Params(
				Err().Error(),
				Id("rpcResponse").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "ResponseRPC"),
			).BlockFunc(func(bg *Group) {
				bg.If(Err().Op("==").Nil().Op("&&").Id("rpcResponse").Op("!=").Nil()).Block(
					If(Id("rpcResponse").Dot("Error").Op("!=").Nil()).Block(
						If(Id("cli").Dot("errorDecoder").Op("!=").Nil()).Block(
							Err().Op("=").Id("cli").Dot("errorDecoder").Call(Id("rpcResponse").Dot("Error").Dot("Raw").Call()),
						).Else().Block(
							Err().Op("=").Qual(PackageFmt, "Errorf").Call(Lit("%s"), Id("rpcResponse").Dot("Error").Dot("Message")),
						),
					).Else().Block(
						Err().Op("=").Qual(jsonPkg, "Unmarshal").Call(Id("rpcResponse").Dot("Result"), Op("&").Add(resp)),
					),
				)
				bg.Id("callback").CallFunc(func(cg *Group) {
					fieldsResult := r.fieldsResult(method)
					resultsWithoutErr := r.resultsWithoutError(method)
					// fieldsResult и resultsWithoutErr имеют одинаковый порядок и количество элементов
					for i, field := range fieldsResult {
						if i >= len(resultsWithoutErr) {
							cg.Add(r.clientRPCResultValue(contract, method, &model.Variable{Name: field.name, TypeRef: model.TypeRef{TypeID: field.typeID, NumberOfPointers: field.numberOfPointers}}, field, "_response_"))
							continue
						}
						cg.Add(r.clientRPCResultValue(contract, method, resultsWithoutErr[i], field, "_response_"))
					}
					cg.Err()
				})
			}),
		)
		bg.Return()
	})
}
