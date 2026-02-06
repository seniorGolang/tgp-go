// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/common"
	"tgp/internal/model"
)

func (r *ClientRenderer) httpClientMethodFunc(ctx context.Context, contract *model.Contract, method *model.Method, outDir string) Code {

	c := Comment(fmt.Sprintf("%s performs the %s operation.", method.Name, method.Name))
	c.Line()
	c.Func().Params(Id("cli").Op("*").Id("Client" + contract.Name)).
		Id(method.Name).
		Params(r.funcDefinitionParams(ctx, method.Args)).Params(r.funcDefinitionParams(ctx, method.Results)).
		BlockFunc(func(bg *Group) {
			bg.Line()
			if r.HasMetrics() && model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
				bg.Add(r.httpMetricsDefer(contract, method))
			}
			httpMethod := model.GetHTTPMethod(r.project, contract, method)
			successStatusCode := model.GetAnnotationValueInt(r.project, contract, method, nil, TagHttpSuccess, http.StatusOK)
			methodPath := model.GetAnnotationValue(r.project, contract, method, nil, TagHttpPath, ToLowerCamel(method.Name))
			pathParams := r.argPathMap(contract, method)
			svcPrefix := model.GetAnnotationValue(r.project, contract, nil, nil, TagHttpPrefix, "")
			argsMappings := r.argParamMap(contract, method)
			cookieMappings := r.varCookieMap(contract, method)
			headerMappings := r.varHeaderMap(contract, method)

			fullURLPath := path.Join(svcPrefix, methodPath)
			// Формирование URL через net/url: ведущий "/" гарантирует корректный путь при пустом baseURL.Path (url.JoinPath-подобное поведение).
			bg.Var().Id("baseURL").Op("*").Qual(PackageURL, "URL")
			bg.If(List(Id("baseURL"), Err()).Op("=").Qual(PackageURL, "Parse").Call(Id("cli").Dot("endpoint")).Op(";").Err().Op("!=").Nil()).Block(Return())
			bg.Line()
			pathSegments := strings.Split(fullURLPath, "/")
			pathJoinArgs := make([]Code, 0, len(pathSegments)+2)
			pathJoinArgs = append(pathJoinArgs, Lit("/"), Id("baseURL").Dot("Path"))
			for _, seg := range pathSegments {
				if seg == "" {
					continue
				}
				if strings.HasPrefix(seg, ":") {
					paramName := strings.TrimPrefix(seg, ":")
					paramVar := r.argByName(method, paramName)
					pathJoinArgs = append(pathJoinArgs, Qual(PackageURL, "PathEscape").Call(r.varToString(ctx, paramVar)))
				} else {
					pathJoinArgs = append(pathJoinArgs, Lit(seg))
				}
			}
			bg.Id("baseURL").Dot("Path").Op("=").Qual(PackagePath, "Join").Call(pathJoinArgs...)
			bg.Line()

			hasQueryParams := false
			for paramName := range common.SortedPairs(argsMappings) {
				if _, inPath := pathParams[paramName]; inPath {
					continue
				}
				if _, inHeader := headerMappings[paramName]; inHeader {
					continue
				}
				if _, inCookie := cookieMappings[paramName]; inCookie {
					continue
				}
				if r.argByName(method, paramName) != nil {
					hasQueryParams = true
					break
				}
			}
			if hasQueryParams {
				bg.Id("queryValues").Op(":=").Qual(PackageURL, "Values").Values(Dict{})
				for paramName, argName := range common.SortedPairs(argsMappings) {
					if _, inPath := pathParams[paramName]; inPath {
						continue
					}
					if _, inHeader := headerMappings[paramName]; inHeader {
						continue
					}
					if _, inCookie := cookieMappings[paramName]; inCookie {
						continue
					}
					paramVar := r.argByName(method, paramName)
					if paramVar == nil {
						continue
					}
					if paramVar.NumberOfPointers > 0 {
						bg.If(Id(ToLowerCamel(paramName)).Op("!=").Nil()).Block(
							Id("queryValues").Dot("Set").Call(Lit(argName), r.varToString(ctx, paramVar)),
						)
					} else {
						bg.Id("queryValues").Dot("Set").Call(Lit(argName), r.varToString(ctx, paramVar))
					}
				}
				bg.Id("baseURL").Dot("RawQuery").Op("=").Id("queryValues").Dot("Encode").Call()
			}
			bg.Line()

			requestMultipart := r.methodRequestMultipart(contract, method)
			bodyStreamArg := r.methodRequestBodyStreamArg(method)
			hasBody := false
			argsWithoutCtx := r.argsWithoutContext(method)
			fieldsArg := r.fieldsArgument(method)
			for _, arg := range argsWithoutCtx {
				if arg.TypeID == TypeIDIOReader {
					continue
				}
				if _, exists := argsMappings[arg.Name]; exists {
					continue
				}
				if _, exists := cookieMappings[arg.Name]; exists {
					continue
				}
				if _, exists := headerMappings[arg.Name]; exists {
					continue
				}
				if _, exists := pathParams[arg.Name]; exists {
					continue
				}
				hasBody = true
				break
			}
			switch {
			case requestMultipart:
				bg.Add(r.httpMultipartRequestBody(contract, method))
			case bodyStreamArg != nil:
				bg.Var().Id("httpReq").Op("*").Qual(PackageHttp, "Request")
				bg.If(List(Id("httpReq"), Err()).Op("=").Qual(PackageHttp, "NewRequestWithContext").Call(Id(_ctx_), Lit(httpMethod), Id("baseURL").Dot("String").Call(), Qual(PackageIO, "NopCloser").Call(Id(ToLowerCamel(bodyStreamArg.Name)))).Op(";").Err().Op("!=").Nil()).Block(Return())
				requestContentType := model.GetAnnotationValue(r.project, contract, method, nil, TagRequestContentType, "application/octet-stream")
				bg.Id("httpReq").Dot("Header").Dot("Set").Call(Lit("Content-Type"), Lit(requestContentType))
				bg.Id("httpReq").Dot("Close").Op("=").True()
			case hasBody:
				bg.Var().Id("httpReq").Op("*").Qual(PackageHttp, "Request")
				bg.Id("_request").Op(":=").Id(r.requestStructName(contract, method)).Values(DictFunc(func(dict Dict) {
					for _, field := range fieldsArg {
						var arg *model.Variable
						for _, a := range argsWithoutCtx {
							if a.Name == field.name {
								arg = a
								break
							}
						}
						if arg == nil {
							continue
						}
						if _, exists := argsMappings[arg.Name]; exists {
							continue
						}
						if _, exists := cookieMappings[arg.Name]; exists {
							continue
						}
						if _, exists := headerMappings[arg.Name]; exists {
							continue
						}
						if _, exists := pathParams[arg.Name]; exists {
							continue
						}
						dict[Id(ToCamel(field.name))] = Id(ToLowerCamel(arg.Name))
					}
				}))
				jsonPkg := r.getPackageJSON(contract)
				bg.Var().Id("bodyBytes").Index().Byte()
				bg.If(List(Id("bodyBytes"), Err()).Op("=").Qual(jsonPkg, "Marshal").Call(Id("_request")).Op(";").Err().Op("!=").Nil()).Block(Return())
				bg.If(List(Id("httpReq"), Err()).Op("=").Qual(PackageHttp, "NewRequestWithContext").Call(Id(_ctx_), Lit(httpMethod), Id("baseURL").Dot("String").Call(), Qual(PackageBytes, "NewReader").Call(Id("bodyBytes"))).Op(";").Err().Op("!=").Nil()).Block(Return())
				bg.Id("httpReq").Dot("ContentLength").Op("=").Int64().Call(Id("len").Call(Id("bodyBytes")))
			}

			if !requestMultipart && bodyStreamArg == nil && !hasBody {
				bg.Var().Id("httpReq").Op("*").Qual(PackageHttp, "Request")
				bg.If(List(Id("httpReq"), Err()).Op("=").Qual(PackageHttp, "NewRequestWithContext").Call(Id(_ctx_), Lit(httpMethod), Id("baseURL").Dot("String").Call(), Nil()).Op(";").Err().Op("!=").Nil()).Block(Return())
			}

			if !requestMultipart && bodyStreamArg == nil {
				bg.Id("httpReq").Dot("Header").Dot("Set").Call(Lit("Accept"), Lit("application/json"))
				if hasBody {
					bg.Id("httpReq").Dot("Header").Dot("Set").Call(Lit("Content-Type"), Lit("application/json"))
				}
			}
			for paramName, headerName := range common.SortedPairs(headerMappings) {
				if paramVar := r.argByName(method, paramName); paramVar != nil {
					bg.Id("httpReq").Dot("Header").Dot("Set").Call(Lit(headerName), r.varToString(ctx, paramVar))
				}
			}
			for paramName, cookieName := range common.SortedPairs(cookieMappings) {
				if paramVar := r.argByName(method, paramName); paramVar != nil {
					bg.Id("httpReq").Dot("AddCookie").Call(Op("&").Qual(PackageHttp, "Cookie").Values(Dict{
						Id("Name"):  Lit(cookieName),
						Id("Value"): r.varToString(ctx, paramVar),
					}))
				}
			}

			responseStreamResult := r.methodResponseBodyStreamResult(method)
			responseMultipart := r.methodResponseMultipart(contract, method)
			bg.Var().Id("httpResp").Op("*").Qual(PackageHttp, "Response")
			bg.List(Id("httpResp"), Err()).Op("=").Id("cli").Dot("doRoundTrip").Call(
				Id(_ctx_),
				Lit(r.methodNameToLowerCamel(method)),
				Id("httpReq"),
				Lit(successStatusCode),
			)
			bg.If(Err().Op("!=").Nil()).Block(Return())
			if responseStreamResult == nil && !responseMultipart {
				bg.Add(r.httpDeferBodyClose())
			}

			resultsWithoutErr := r.resultsWithoutError(method)
			fieldsResult := r.fieldsResult(method)
			switch {
			case responseMultipart:
				bg.Add(r.httpMultipartResponseBody(contract, method))
			case responseStreamResult != nil:
				for _, ret := range resultsWithoutErr {
					switch ret.TypeID {
					case TypeIDIOReadCloser:
						bg.Id(ToLowerCamel(ret.Name)).Op("=").Id("httpResp").Dot("Body")
					case "string":
						bg.Id(ToLowerCamel(ret.Name)).Op("=").Id("httpResp").Dot("Header").Dot("Get").Call(Lit("Content-Type"))
					}
				}
				bg.Return()
			case len(resultsWithoutErr) == 1 && model.IsAnnotationSet(r.project, contract, method, nil, TagHttpEnableInlineSingle):
				bg.Var().Id("_response").Id(r.responseStructName(contract, method))
				jsonPkg := r.getPackageJSON(contract)
				bg.Var().Id("decoder").Op("=").Qual(jsonPkg, "NewDecoder").Call(Id("httpResp").Dot("Body"))
				bg.If(Err().Op("=").Id("decoder").Dot("Decode").Call(Op("&").Id("_response").Dot(ToCamel(resultsWithoutErr[0].Name))).Op(";").Err().Op("!=").Nil()).Block(
					Return(),
				)
				for i, ret := range resultsWithoutErr {
					if i >= len(fieldsResult) {
						bg.Id(ToLowerCamel(ret.Name)).Op("=").Add(Id("_response").Dot(ToCamel(ret.Name)))
						continue
					}
					field := fieldsResult[i]
					fieldValue := Id("_response").Dot(ToCamel(ret.Name))
					// Согласование указатель/значение между полем структуры ответа и возвращаемым типом.
					switch {
					case field.numberOfPointers > 0 && ret.NumberOfPointers == 0:
						bg.Id(ToLowerCamel(ret.Name)).Op("=").Op("*").Add(fieldValue)
					case field.numberOfPointers == 0 && ret.NumberOfPointers > 0:
						bg.Id(ToLowerCamel(ret.Name)).Op("=").Op("&").Add(fieldValue)
					default:
						bg.Id(ToLowerCamel(ret.Name)).Op("=").Add(fieldValue)
					}
				}
				bg.Return()
			default:
				bg.Var().Id("_response").Id(r.responseStructName(contract, method))
				jsonPkg := r.getPackageJSON(contract)
				bg.Var().Id("decoder").Op("=").Qual(jsonPkg, "NewDecoder").Call(Id("httpResp").Dot("Body"))
				bg.If(Err().Op("=").Id("decoder").Dot("Decode").Call(Op("&").Id("_response")).Op(";").Err().Op("!=").Nil()).Block(
					Return(),
				)
				for i, ret := range resultsWithoutErr {
					if i >= len(fieldsResult) {
						bg.Id(ToLowerCamel(ret.Name)).Op("=").Add(Id("_response").Dot(ToCamel(ret.Name)))
						continue
					}
					field := fieldsResult[i]
					fieldValue := Id("_response").Dot(ToCamel(ret.Name))
					// Согласование указатель/значение между полем структуры ответа и возвращаемым типом.
					switch {
					case field.numberOfPointers > 0 && ret.NumberOfPointers == 0:
						bg.Id(ToLowerCamel(ret.Name)).Op("=").Op("*").Add(fieldValue)
					case field.numberOfPointers == 0 && ret.NumberOfPointers > 0:
						bg.Id(ToLowerCamel(ret.Name)).Op("=").Op("&").Add(fieldValue)
					default:
						bg.Id(ToLowerCamel(ret.Name)).Op("=").Add(fieldValue)
					}
				}
				bg.Return()
			}
		})
	return c
}
