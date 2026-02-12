// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"context"
	"fmt"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

func (r *ClientRenderer) jsonrpcClientMethodFunc(ctx context.Context, contract *model.Contract, method *model.Method, outDir string) Code {

	return Func().
		Params(Id("cli").Op("*").Id("Client" + contract.Name)).
		Id(method.Name).
		Params(r.funcDefinitionParams(ctx, method.Args)).Params(r.funcDefinitionParams(ctx, method.Results)).BlockFunc(func(bg *Group) {

		if r.HasMetrics() && model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
			bg.Line().Add(r.rpcMetricsDefer(contract, method))
		}

		bg.Line()
		bg.Id("_request").Op(":=").Id(r.requestStructName(contract, method)).Values(DictFunc(func(dict Dict) {
			argsWithoutCtx := r.argsWithoutContext(method)
			fieldsArg := r.fieldsArgument(method)
			for idx, arg := range fieldsArg {
				if idx < len(argsWithoutCtx) {
					dict[Id(ToCamel(arg.name))] = Id(ToLowerCamel(argsWithoutCtx[idx].Name))
				}
			}
		}))
		bg.Var().Id("_response").Id(r.responseStructName(contract, method))
		bg.Var().Id("rpcResponse").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "ResponseRPC")
		bg.If(List(Id("rpcResponse"), Err()).Op("=").Id("cli").Dot("rpc").Dot("Call").Call(Id(_ctx_), Lit(r.contractNameToLower(contract)+"."+r.methodNameToLower(method)), Id("_request")).Op(";").Err().Op("!=").Nil().Op("||").Id("rpcResponse").Op("==").Nil()).Block(
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
		resp := Id("_response")
		resultsWithoutErr := r.resultsWithoutError(method)
		if len(resultsWithoutErr) == 1 && model.IsAnnotationSet(r.project, contract, method, nil, model.TagHttpEnableInlineSingle) {
			resp = Id("_response").Dot(ToCamel(resultsWithoutErr[0].Name))
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
					// Если поле не найдено, используем значение по умолчанию
					rg.Add(Id("_response").Dot(ToCamel(ret.Name)))
					continue
				}
				field := fieldsResult[i]
				fieldValue := Id("_response").Dot(ToCamel(ret.Name))
				// Если поле в структуре - указатель, а возвращаемое значение - не указатель, разыменовываем
				switch {
				case field.numberOfPointers > 0 && ret.NumberOfPointers == 0:
					rg.Op("*").Add(fieldValue)
				case field.numberOfPointers == 0 && ret.NumberOfPointers > 0:
					// Если поле в структуре - не указатель, а возвращаемое значение - указатель, берем адрес
					rg.Op("&").Add(fieldValue)
				default:
					rg.Add(fieldValue)
				}
			}
			rg.Err()
		})
	})
}

func (r *ClientRenderer) jsonrpcClientRequestFunc(ctx context.Context, contract *model.Contract, method *model.Method, outDir string) Code {

	return Func().Params(Id("cli").Op("*").Id("Client"+contract.Name)).
		Id("Req"+method.Name).
		Params(Id("callback").Id("ret"+contract.Name+method.Name), r.funcDefinitionParams(ctx, r.argsWithoutContext(method))).
		Params(Id("_request").Id("RequestRPC")).BlockFunc(func(bg *Group) {

		bg.Line()
		bg.Id("_request").Op("=").Id("RequestRPC").Values(Dict{
			Id("rpcRequest"): Op("&").Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "RequestRPC").Values(Dict{
				Id("ID"):      Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "NewID").Call(),
				Id("JSONRPC"): Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "Version"),
				Id("Method"):  Lit(r.contractNameToLower(contract) + "." + r.methodNameToLower(method)),
				Id("Params"): Id(r.requestStructName(contract, method)).Values(DictFunc(func(dg Dict) {
					argsWithoutCtx := r.argsWithoutContext(method)
					fieldsArg := r.fieldsArgument(method)
					for idx, arg := range fieldsArg {
						if idx < len(argsWithoutCtx) {
							dg[Id(ToCamel(arg.name))] = Id(ToLowerCamel(argsWithoutCtx[idx].Name))
						}
					}
				})),
			}),
		})
		resp := Id("_response")
		resultsWithoutErr := r.resultsWithoutError(method)
		if len(resultsWithoutErr) == 1 && model.IsAnnotationSet(r.project, contract, method, nil, model.TagHttpEnableInlineSingle) {
			resp = Id("_response").Dot(ToCamel(resultsWithoutErr[0].Name))
		}
		jsonPkg := r.getPackageJSON(contract)
		bg.If(Id("callback").Op("!=").Nil()).Block(
			Var().Id("_response").Id(r.responseStructName(contract, method)),
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
							// Если возвращаемое значение не найдено, используем значение по умолчанию
							cg.Add(Id("_response").Dot(ToCamel(field.name)))
							continue
						}
						ret := resultsWithoutErr[i]
						fieldValue := Id("_response").Dot(ToCamel(field.name))
						// Если поле в структуре - указатель, а возвращаемое значение - не указатель, разыменовываем
						switch {
						case field.numberOfPointers > 0 && ret.NumberOfPointers == 0:
							cg.Op("*").Add(fieldValue)
						case field.numberOfPointers == 0 && ret.NumberOfPointers > 0:
							// Если поле в структуре - не указатель, а возвращаемое значение - указатель, берем адрес
							cg.Op("&").Add(fieldValue)
						default:
							cg.Add(fieldValue)
						}
					}
					cg.Err()
				})
			}),
		)
		bg.Return()
	})
}
