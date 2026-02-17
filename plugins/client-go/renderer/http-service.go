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
	"tgp/internal/content"
	"tgp/internal/model"
)

func (r *ClientRenderer) httpClientMethodFunc(ctx context.Context, contract *model.Contract, method *model.Method, outDir string) Code {

	c := Comment(fmt.Sprintf("%s performs the %s operation.", method.Name, method.Name))
	c.Line()
	c.Func().Params(Id("cli").Op("*").Id("Client" + contract.Name)).
		Id(method.Name).
		Params(r.clientMethodParams(ctx, contract, method)).Params(r.funcDefinitionParams(ctx, method.Results)).
		BlockFunc(func(bg *Group) {
			bg.Line()
			if r.HasMetrics() && model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
				bg.Add(r.httpMetricsDefer(contract, method))
			}
			httpMethod := model.GetHTTPMethod(r.project, contract, method)
			successStatusCode := model.GetAnnotationValueInt(r.project, contract, method, nil, model.TagHttpSuccess, http.StatusOK)
			methodPath := model.GetAnnotationValue(r.project, contract, method, nil, model.TagHttpPath, r.defaultMethodHTTPPath(contract, method))
			pathParams := r.argPathMap(contract, method)
			svcPrefix := model.GetAnnotationValue(r.project, contract, nil, nil, model.TagHttpPrefix, "")
			argsMappings := model.HTTPArgQueryMapForRequest(r.project, contract, method)
			cookieMappings := model.HTTPCookieArgMapForRequest(r.project, contract, method)
			headerMappings := model.HTTPHeaderArgMapForRequest(r.project, contract, method)

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
					paramVar := r.argByPathParamName(contract, method, paramName)
					pathJoinArgs = append(pathJoinArgs, Qual(PackageURL, "PathEscape").Call(r.varToString(ctx, paramVar)))
				} else {
					pathJoinArgs = append(pathJoinArgs, Lit(seg))
				}
			}
			bg.Id("baseURL").Dot("Path").Op("=").Qual(PackagePath, "Join").Call(pathJoinArgs...)
			bg.Line()

			argsForClientSet := make(map[string]struct{})
			for _, a := range r.argsForClient(contract, method) {
				argsForClientSet[a.Name] = struct{}{}
			}
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
				if _, inArgs := argsForClientSet[paramName]; inArgs {
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
					if _, inArgs := argsForClientSet[paramName]; !inArgs {
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
			argsForBody := r.argsForRequestBody(contract, method)
			hasBody := len(argsForBody) > 0
			switch {
			case requestMultipart:
				bg.Add(r.httpMultipartRequestBody(contract, method))
			case bodyStreamArg != nil:
				bg.Var().Id("httpReq").Op("*").Qual(PackageHttp, "Request")
				bg.If(List(Id("httpReq"), Err()).Op("=").Qual(PackageHttp, "NewRequestWithContext").Call(Id(_ctx_), Lit(httpMethod), Id("baseURL").Dot("String").Call(), Qual(PackageIO, "NopCloser").Call(Id(ToLowerCamel(bodyStreamArg.Name)))).Op(";").Err().Op("!=").Nil()).Block(Return())
				requestContentType := model.GetAnnotationValue(r.project, contract, method, nil, model.TagRequestContentType, "application/octet-stream")
				bg.Id("httpReq").Dot("Header").Dot("Set").Call(Lit("Content-Type"), Lit(requestContentType))
				bg.Id("httpReq").Dot("Close").Op("=").True()
			case hasBody:
				bg.Var().Id("httpReq").Op("*").Qual(PackageHttp, "Request")
				bg.Id("_request_").Op(":=").Id(r.requestBodyStructName(contract, method)).Values(DictFunc(func(dict Dict) {
					for _, arg := range argsForBody {
						dict[Id(ToCamel(arg.Name))] = Id(ToLowerCamel(arg.Name))
					}
				}))
				jsonPkg := r.getPackageJSON(contract)
				reqKind := content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagRequestContentType, "application/json"))
				r.httpRequestBodyEncode(bg, contract, method, r.requestBodyStructName(contract, method), "_request_", jsonPkg, reqKind)
			}

			if !requestMultipart && bodyStreamArg == nil && !hasBody {
				bg.Var().Id("httpReq").Op("*").Qual(PackageHttp, "Request")
				bg.If(List(Id("httpReq"), Err()).Op("=").Qual(PackageHttp, "NewRequestWithContext").Call(Id(_ctx_), Lit(httpMethod), Id("baseURL").Dot("String").Call(), Nil()).Op(";").Err().Op("!=").Nil()).Block(Return())
			}

			if !requestMultipart && bodyStreamArg == nil {
				acceptType := model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json")
				bg.Id("httpReq").Dot("Header").Dot("Set").Call(Lit("Accept"), Lit(acceptType))
				if hasBody {
					reqCT := model.GetAnnotationValue(r.project, contract, method, nil, model.TagRequestContentType, "application/json")
					bg.Id("httpReq").Dot("Header").Dot("Set").Call(Lit("Content-Type"), Lit(reqCT))
				}
			}
			for _, arg := range r.argsForClient(contract, method) {
				if headerName, ok := headerMappings[arg.Name]; ok {
					bg.Id("httpReq").Dot("Header").Dot("Set").Call(Lit(headerName), r.varToString(ctx, arg))
				}
			}
			for _, arg := range r.argsForClient(contract, method) {
				if cookieName, ok := cookieMappings[arg.Name]; ok {
					bg.Id("httpReq").Dot("AddCookie").Call(Op("&").Qual(PackageHttp, "Cookie").Values(Dict{
						Id("Name"):  Lit(cookieName),
						Id("Value"): r.varToString(ctx, arg),
					}))
				}
			}

			responseStreamResult := r.methodResponseBodyStreamResult(method)
			responseMultipart := r.methodResponseMultipart(contract, method)
			bg.Var().Id("httpResp").Op("*").Qual(PackageHttp, "Response")
			bg.If(List(Id("httpResp"), Err()).Op("=").Id("cli").Dot("doRoundTrip").Call(
				Id(_ctx_),
				Lit(r.methodNameToLowerCamel(method)),
				Id("httpReq"),
				Lit(successStatusCode),
			).Op(";").Err().Op("!=").Nil()).Block(Return())
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
					if ret.TypeID == TypeIDIOReadCloser {
						bg.Id(ToLowerCamel(ret.Name)).Op("=").Id("httpResp").Dot("Body")
					}
				}
				r.httpResponseMergeHeadersAndCookies(bg, ctx, contract, method, true)
				bg.Return()
			case len(resultsWithoutErr) == 1 && model.IsAnnotationSet(r.project, contract, method, nil, model.TagHttpEnableInlineSingle):
				excludeSingle := r.resultNamesExcludeFromBody(contract, method)
				if len(excludeSingle) > 0 {
					bg.Var().Id("_response_").Id(r.responseBodyStructName(contract, method))
					jsonPkg := r.getPackageJSON(contract)
					resKind := content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json"))
					r.httpResponseDecode(bg, contract, method, jsonPkg, resKind, "_response_")
					r.httpResponseMergeHeadersAndCookies(bg, ctx, contract, method, true)
					fieldsResultBodySingle := r.fieldsResultBody(contract, method)
					for _, ret := range resultsWithoutErr {
						if _, excluded := excludeSingle[ret.Name]; excluded {
							continue
						}
						fieldValue := Id("_response_").Dot(ToCamel(ret.Name))
						var field exchangeField
						for i := range fieldsResultBodySingle {
							if fieldsResultBodySingle[i].name == ret.Name {
								field = fieldsResultBodySingle[i]
								break
							}
						}
						switch {
						case field.numberOfPointers > 0 && ret.NumberOfPointers == 0:
							bg.Id(ToLowerCamel(ret.Name)).Op("=").Op("*").Add(fieldValue)
						case field.numberOfPointers == 0 && ret.NumberOfPointers > 0:
							bg.Id(ToLowerCamel(ret.Name)).Op("=").Op("&").Add(fieldValue)
						default:
							bg.Id(ToLowerCamel(ret.Name)).Op("=").Add(fieldValue)
						}
					}
				} else {
					bg.Var().Id("_response_").Id(r.responseStructName(contract, method))
					jsonPkg := r.getPackageJSON(contract)
					resKind := content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json"))
					r.httpResponseDecode(bg, contract, method, jsonPkg, resKind, "_response_")
					r.httpResponseMergeHeadersAndCookies(bg, ctx, contract, method, false)
					for i, ret := range resultsWithoutErr {
						if i >= len(fieldsResult) {
							bg.Id(ToLowerCamel(ret.Name)).Op("=").Add(Id("_response_").Dot(ToCamel(ret.Name)))
							continue
						}
						field := fieldsResult[i]
						fieldValue := Id("_response_").Dot(ToCamel(ret.Name))
						switch {
						case field.numberOfPointers > 0 && ret.NumberOfPointers == 0:
							bg.Id(ToLowerCamel(ret.Name)).Op("=").Op("*").Add(fieldValue)
						case field.numberOfPointers == 0 && ret.NumberOfPointers > 0:
							bg.Id(ToLowerCamel(ret.Name)).Op("=").Op("&").Add(fieldValue)
						default:
							bg.Id(ToLowerCamel(ret.Name)).Op("=").Add(fieldValue)
						}
					}
				}
				bg.Return()
			default:
				excludeDefault := r.resultNamesExcludeFromBody(contract, method)
				if len(excludeDefault) > 0 {
					bg.Var().Id("_response_").Id(r.responseBodyStructName(contract, method))
					jsonPkg := r.getPackageJSON(contract)
					resKind := content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json"))
					r.httpResponseDecode(bg, contract, method, jsonPkg, resKind, "_response_")
					r.httpResponseMergeHeadersAndCookies(bg, ctx, contract, method, true)
					fieldsResultBody := r.fieldsResultBody(contract, method)
					for _, ret := range resultsWithoutErr {
						if _, excluded := excludeDefault[ret.Name]; excluded {
							continue
						}
						fieldValue := Id("_response_").Dot(ToCamel(ret.Name))
						var field exchangeField
						for i := range fieldsResultBody {
							if fieldsResultBody[i].name == ret.Name {
								field = fieldsResultBody[i]
								break
							}
						}
						switch {
						case field.numberOfPointers > 0 && ret.NumberOfPointers == 0:
							bg.Id(ToLowerCamel(ret.Name)).Op("=").Op("*").Add(fieldValue)
						case field.numberOfPointers == 0 && ret.NumberOfPointers > 0:
							bg.Id(ToLowerCamel(ret.Name)).Op("=").Op("&").Add(fieldValue)
						default:
							bg.Id(ToLowerCamel(ret.Name)).Op("=").Add(fieldValue)
						}
					}
				} else {
					bg.Var().Id("_response_").Id(r.responseStructName(contract, method))
					jsonPkg := r.getPackageJSON(contract)
					resKind := content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json"))
					r.httpResponseDecode(bg, contract, method, jsonPkg, resKind, "_response_")
					r.httpResponseMergeHeadersAndCookies(bg, ctx, contract, method, false)
					for i, ret := range resultsWithoutErr {
						if i >= len(fieldsResult) {
							bg.Id(ToLowerCamel(ret.Name)).Op("=").Add(Id("_response_").Dot(ToCamel(ret.Name)))
							continue
						}
						field := fieldsResult[i]
						fieldValue := Id("_response_").Dot(ToCamel(ret.Name))
						switch {
						case field.numberOfPointers > 0 && ret.NumberOfPointers == 0:
							bg.Id(ToLowerCamel(ret.Name)).Op("=").Op("*").Add(fieldValue)
						case field.numberOfPointers == 0 && ret.NumberOfPointers > 0:
							bg.Id(ToLowerCamel(ret.Name)).Op("=").Op("&").Add(fieldValue)
						default:
							bg.Id(ToLowerCamel(ret.Name)).Op("=").Add(fieldValue)
						}
					}
				}
				bg.Return()
			}
		})
	return c
}
