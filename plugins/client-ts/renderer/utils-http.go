// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tgp/internal/common"
	"tgp/internal/content"
	"tgp/internal/model"
	"tgp/plugins/client-ts/tsg"
)

func (r *ClientRenderer) isHTTP(method *model.Method, contract *model.Contract) bool {

	if method == nil || contract == nil {
		return false
	}
	contractHasHTTP := model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP)
	if !contractHasHTTP {
		return false
	}
	contractHasJsonRPC := model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC)
	methodHasExplicitHTTP := model.IsAnnotationSet(r.project, contract, method, nil, model.TagHTTPMethod)
	return !contractHasJsonRPC || methodHasExplicitHTTP
}

func (r *ClientRenderer) renderHTTPMethod(grp *tsg.Group, method *model.Method, contract *model.Contract) {
	args := r.argsWithoutContext(method)
	results := r.resultsWithoutError(method)

	filteredDocs := r.filterDocsComments(method.Docs)
	if len(filteredDocs) > 0 {
		grp.Comment(strings.Join(filteredDocs, "\n"))
	} else {
		grp.Comment(fmt.Sprintf("Вызывает HTTP метод %s", method.Name))
	}

	methodErrors := r.collectMethodErrors(method, contract)
	if len(methodErrors) > 0 {
		grp.Comment("@throws {Error} - Possible errors:")
		errorsList := make([]errorInfo, 0, len(methodErrors))
		for _, errInfo := range common.SortedPairs(methodErrors) {
			errorsList = append(errorsList, errInfo)
		}
		sort.Slice(errorsList, func(i, j int) bool {
			if errorsList[i].code == 0 && errorsList[j].code != 0 {
				return false
			}
			if errorsList[i].code != 0 && errorsList[j].code == 0 {
				return true
			}
			if errorsList[i].code != errorsList[j].code {
				return errorsList[i].code < errorsList[j].code
			}
			// При равных кодах — по ключу map для детерминированности вывода.
			return errorsList[i].pkgPath+":"+errorsList[i].typeName < errorsList[j].pkgPath+":"+errorsList[j].typeName
		})
		for _, errInfo := range errorsList {
			typeName := fmt.Sprintf("%s%s", strings.Split(errInfo.pkgPath, "/")[len(strings.Split(errInfo.pkgPath, "/"))-1], errInfo.typeName)
			if errInfo.code != 0 {
				grp.Comment(fmt.Sprintf("  - %s (HTTP %d: %s)", typeName, errInfo.code, errInfo.codeText))
			} else {
				grp.Comment(fmt.Sprintf("  - %s", typeName))
			}
		}
	}

	httpMethod := strings.ToUpper(model.GetHTTPMethod(r.project, contract, method))

	methodParams := tsg.NewStatement()
	methodParams.Params(func(pg *tsg.Group) {
		if len(args) > 0 {
			for _, arg := range args {
				typeStr := r.walkVariable(arg.Name, contract.PkgPath, arg, method.Annotations, true).typeLink()
				paramStmt := tsg.NewStatement()
				paramStmt.Id(tsSafeName(arg.Name))
				if model.IsAnnotationSet(r.project, contract, method, nil, "nullable") {
					paramStmt.Optional()
				}
				paramStmt.Colon()
				paramStmt.Add(tsg.TypeFromString(typeStr))
				pg.Add(paramStmt)
			}
		}
	})

	returnType := r.resultToTypeStatement(method, results)

	var requestTypeName string
	if len(args) > 0 {
		requestTypeName = r.requestTypeName(contract, method)
	}
	var responseTypeName string
	if len(results) > 0 {
		responseTypeName = r.responseTypeName(contract, method)
	}

	methodStmt := tsg.NewStatement()
	methodStmt.Public()
	methodStmt.AsyncMethodWithParams(r.lcName(method.Name), methodParams, returnType, func(mg *tsg.Group) {
		if len(args) > 0 {
			paramsObj := tsg.NewStatement()
			paramsObj.Const(tsLocalVar("params")).Colon().Id(requestTypeName).Op("=")
			paramsObj.Values(func(vg *tsg.Group) {
				for _, arg := range args {
					vg.Add(tsg.NewStatement().Id(tsSafeName(arg.Name)).Colon().Id(tsSafeName(arg.Name)))
				}
			})
			mg.Add(paramsObj.Semicolon())
		}

		urlStmt := tsg.NewStatement()
		urlStmt.Const(tsLocalVar("baseURL")).Op("=").Id("this").Dot("baseClient").Dot("getEndpoint").Call().Semicolon()
		mg.Add(urlStmt)

		pathPrefix := model.GetAnnotationValue(r.project, contract, nil, nil, model.TagHttpPrefix, "")
		path := r.httpPath(method, contract)
		fullPath := strings.TrimPrefix(path, "/")
		if pathPrefix != "" {
			fullPath = pathPrefix + "/" + fullPath
		}
		urlStmt2 := tsg.NewStatement()
		urlStmt2.Var(tsLocalVar("url")).Op("=").Id(tsLocalVar("baseURL"))
		ternaryExpr := tsg.NewStatement()
		ternaryExpr.Op("(").Id(tsLocalVar("baseURL")).Dot("endsWith").Call(tsg.NewStatement().Lit("/")).Op("?").Lit("").Op(":").Lit("/").Op(")")
		urlStmt2.Op("+").Add(ternaryExpr)
		urlStmt2.Op("+").Lit(fullPath)
		mg.Add(urlStmt2.Semicolon())
		for _, paramName := range r.httpPathParamNames(method, contract) {
			mg.Add(tsg.NewStatement().
				Id(tsLocalVar("url")).
				Op("=").
				Id(tsLocalVar("url")).
				Dot("replace").
				Call(tsg.NewStatement().Lit(":"+paramName), tsg.NewStatement().Id("encodeURIComponent").Call(tsg.NewStatement().Id(tsLocalVar("params")).Dot(tsSafeName(paramName)))).
				Semicolon())
		}
		pathParamSet := make(map[string]bool)
		for _, n := range r.httpPathParamNames(method, contract) {
			pathParamSet[n] = true
		}
		argParamMap := r.httpArgParamMap(contract, method)
		var queryParams []struct{ argName, queryKey string }
		for _, arg := range args {
			if pathParamSet[arg.Name] {
				continue
			}
			if queryKey, ok := argParamMap[arg.Name]; ok {
				queryParams = append(queryParams, struct{ argName, queryKey string }{arg.Name, queryKey})
			}
		}
		sort.Slice(queryParams, func(i, j int) bool { return queryParams[i].queryKey < queryParams[j].queryKey })
		for i, qp := range queryParams {
			sep := "?"
			if i > 0 {
				sep = "&"
			}
			mg.Add(tsg.NewStatement().
				Id(tsLocalVar("url")).
				Op("=").
				Id(tsLocalVar("url")).
				Op("+").
				Lit(sep + qp.queryKey + "=").
				Op("+").
				Id("encodeURIComponent").Call(tsg.NewStatement().Id(tsLocalVar("params")).Dot(tsSafeName(qp.argName))).
				Semicolon())
		}

		requestMultipart := r.methodRequestMultipart(contract, method)
		bodyStreamArg := r.methodRequestBodyStreamArg(method)
		bodyStreamArgs := r.methodRequestBodyStreamArgs(method)
		responseMultipart := r.methodResponseMultipart(contract, method)
		responseStreamResult := r.methodResponseBodyStreamResult(method)

		r.renderHTTPBody(mg, contract, method, args, httpMethod, requestMultipart, bodyStreamArg, bodyStreamArgs)

		headersVar := tsg.NewStatement().
			Const(tsLocalVar("clientHeaders")).
			Colon().
			Id("Record").
			Generic("string", "string").
			Op("=").
			Await(tsg.NewStatement().Id("this").Dot("baseClient").Dot("getHeaders").Call()).
			Semicolon()
		mg.Add(headersVar)

		headersStmt := tsg.NewStatement()
		headersStmt.Const(tsLocalVar("headers")).Op("=").Id("new Headers").Call().Semicolon()
		mg.Add(headersStmt)
		r.renderHTTPHeaders(mg, contract, method, requestMultipart, bodyStreamArg, responseMultipart, responseStreamResult)

		mg.Add(tsg.NewStatement().
			ForOf("["+tsLocalVar("key")+", "+tsLocalVar("value")+"]", "Object.entries("+tsLocalVar("clientHeaders")+")", func(fg *tsg.Group) {
				fg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Id(tsLocalVar("key")), tsg.NewStatement().Id(tsLocalVar("value"))))
			}).
			Semicolon())

		for argName, headerName := range common.SortedPairs(r.httpHeaderParamMap(contract, method)) {
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit(headerName), tsg.NewStatement().Id(tsLocalVar("params")).Dot(tsSafeName(argName))).Semicolon())
		}

		cookieMap := r.httpCookieParamMap(contract, method)
		if len(cookieMap) > 0 {
			mg.Add(tsg.NewStatement().Const(tsLocalVar("cookieParts")).Colon().Id("string").Array(nil).Op("=").Index(nil).Semicolon())
			for argName, cookieName := range common.SortedPairs(cookieMap) {
				pushStmt := tsg.NewStatement().
					Id(tsLocalVar("cookieParts")).
					Dot("push").
					Call(tsg.NewStatement().Lit(cookieName + "=").Op("+").Id("encodeURIComponent").Call(tsg.NewStatement().Id(tsLocalVar("params")).Dot(tsSafeName(argName)))).
					Semicolon()
				mg.Add(pushStmt)
			}
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Cookie"), tsg.NewStatement().Id(tsLocalVar("cookieParts")).Dot("join").Call(tsg.NewStatement().Lit("; "))).Semicolon())
		}

		if requestMultipart && (httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH") {
			mg.Add(tsg.NewStatement().Const(tsLocalVar("multipartReq")).Op("=").Id("new Request").Call(tsg.NewStatement().Id(tsLocalVar("url")), tsg.NewStatement().Values(func(vg *tsg.Group) {
				vg.Add(tsg.NewStatement().Id("method").Colon().Lit(httpMethod))
				vg.Add(tsg.NewStatement().Id("body").Colon().Id(tsLocalVar("body")))
			})).Semicolon())
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Content-Type"), tsg.NewStatement().Id(tsLocalVar("multipartReq")).Dot("headers").Dot("get").Call(tsg.NewStatement().Lit("Content-Type")).Op("!")).Semicolon())
		}

		// Multipart: body как stream, запрос с chunked; Content-Type с boundary уже в headers.
		fetchOptions := tsg.NewStatement()
		fetchOptions.Values(func(fg *tsg.Group) {
			fg.Add(tsg.NewStatement().Id("method").Colon().Lit(httpMethod))
			fg.Add(tsg.NewStatement().Id("headers").Colon().Id(tsLocalVar("headers")))
			if httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH" {
				if requestMultipart {
					fg.Add(tsg.NewStatement().Id("body").Colon().Id(tsLocalVar("multipartReq")).Dot("body"))
					fg.Add(tsg.NewStatement().Id("duplex").Colon().Lit("half"))
				} else {
					fg.Add(tsg.NewStatement().Id("body").Colon().Id(tsLocalVar("body")))
				}
			}
		})
		// duplex не входит в RequestInit; приведение типа устраняет ошибку линтера.
		var fetchOptsArg *tsg.Statement
		if requestMultipart {
			fetchOptsArg = tsg.NewStatement().Add(fetchOptions).Op("as").Add(tsg.TypeFromString("RequestInit & { duplex?: 'half' }"))
		} else {
			fetchOptsArg = fetchOptions
		}
		fetchStmt := tsg.NewStatement()
		fetchStmt.Const(tsLocalVar("response")).Op("=").Await(tsg.NewStatement().Id("fetch").Call(tsg.NewStatement().Id(tsLocalVar("url")), fetchOptsArg))
		mg.Add(fetchStmt.Semicolon())

		successCode := 200
		if model.IsAnnotationSet(r.project, contract, method, nil, model.TagHttpSuccess) {
			if code, err := strconv.Atoi(model.GetAnnotationValue(r.project, contract, method, nil, model.TagHttpSuccess, "200")); err == nil {
				successCode = code
			}
		}

		mg.If(tsg.NewStatement().Id(tsLocalVar("response")).Dot("status").Op("!=").Lit(successCode), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Const(tsLocalVar("errorBody")).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("text").Call()).Semicolon())
			if len(methodErrors) > 0 {
				unionTypeName := fmt.Sprintf("%sError", method.Name)
				ig.Try(
					func(tg *tsg.Group) {
						tg.Add(tsg.NewStatement().Const(tsLocalVar("errorData")).Op("=").Id("JSON.parse").Call(tsg.NewStatement().Id(tsLocalVar("errorBody"))).Semicolon())
						tg.Add(tsg.NewStatement().Const(tsLocalVar("error")).Colon().Id(unionTypeName).Op("=").Id(tsLocalVar("errorData")).Op("as").Id(unionTypeName).Semicolon())
						tg.Throw(tsg.NewStatement().Id(tsLocalVar("error")))
					},
					func(cg *tsg.Group) {
						cg.If(
							tsg.NewStatement().Id(tsLocalVar("e")).Op("&&").Typeof(tsg.NewStatement().Id(tsLocalVar("e")), "object").Op("&&").In("message", tsg.NewStatement().Id(tsLocalVar("e"))),
							func(ig *tsg.Group) {
								ig.Throw(tsg.NewStatement().Id(tsLocalVar("e")))
							},
						)
						cg.Throw(tsg.NewStatement().New("Error").Call(tsg.NewStatement().TemplateString(
							[]string{fmt.Sprintf("HTTP error: %d. ", successCode), ""},
							[]*tsg.Statement{tsg.NewStatement().Id(tsLocalVar("errorBody"))},
						)))
					},
				)
			} else {
				ig.Throw(tsg.NewStatement().New("Error").Call(tsg.NewStatement().Lit(fmt.Sprintf("HTTP error: %d. ", successCode)).Op("+").Id(tsLocalVar("errorBody"))))
			}
		})

		if len(results) == 0 {
			mg.Return()
		} else {
			r.renderHTTPResponse(mg, contract, method, results, responseTypeName, responseMultipart, responseStreamResult)
		}
	})
	grp.Add(methodStmt)
	grp.Line()
}

func (r *ClientRenderer) httpPath(method *model.Method, contract *model.Contract) string {

	return model.GetAnnotationValue(r.project, contract, method, nil, model.TagHttpPath, "/"+r.lcName(method.Name))
}

func (r *ClientRenderer) httpPathParamNames(method *model.Method, contract *model.Contract) []string {

	pathStr := r.httpPath(method, contract)
	var names []string
	for _, token := range strings.Split(pathStr, "/") {
		token = strings.TrimSpace(token)
		if strings.HasPrefix(token, ":") {
			names = append(names, strings.TrimPrefix(token, ":"))
		}
	}
	return names
}

func (r *ClientRenderer) httpArgParamMap(contract *model.Contract, method *model.Method) map[string]string {

	out := make(map[string]string)
	if v := model.GetAnnotationValue(r.project, contract, method, nil, model.TagHttpArg, ""); v != "" {
		for _, pair := range strings.Split(v, ",") {
			parts := strings.SplitN(strings.TrimSpace(pair), "|", 2)
			if len(parts) == 2 {
				arg, param := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
				if arg != "" && param != "" {
					out[arg] = param
				}
			}
		}
	}
	return out
}

func (r *ClientRenderer) httpHeaderParamMap(contract *model.Contract, method *model.Method) map[string]string {

	out := make(map[string]string)
	if v := model.GetAnnotationValue(r.project, contract, method, nil, model.TagHttpHeader, ""); v != "" {
		for _, pair := range strings.Split(v, ",") {
			parts := strings.SplitN(strings.TrimSpace(pair), "|", 2)
			if len(parts) == 2 {
				arg, header := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
				if arg != "" && header != "" {
					out[arg] = header
				}
			}
		}
	}
	return out
}

func (r *ClientRenderer) httpCookieParamMap(contract *model.Contract, method *model.Method) map[string]string {

	out := make(map[string]string)
	if v := model.GetAnnotationValue(r.project, contract, method, nil, model.TagHttpCookies, ""); v != "" {
		for _, pair := range strings.Split(v, ",") {
			parts := strings.SplitN(strings.TrimSpace(pair), "|", 2)
			if len(parts) == 2 {
				arg, cookie := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
				if arg != "" && cookie != "" {
					out[arg] = cookie
				}
			}
		}
	}
	return out
}

func (r *ClientRenderer) formFieldName(method *model.Method, variable *model.Variable) (key string) {

	if method == nil || method.Annotations == nil {
		return toLowerCamel(variable.Name)
	}
	sub := method.Annotations.Sub(variable.Name)
	paramTags := sub.Value(model.TagParamTags, "")
	for _, item := range strings.Split(paramTags, "|") {
		tokens := strings.SplitN(strings.TrimSpace(item), ":", 2)
		if len(tokens) == 2 && strings.TrimSpace(tokens[0]) == "form" {
			return strings.TrimSpace(tokens[1])
		}
	}
	return toLowerCamel(variable.Name)
}

func (r *ClientRenderer) renderFormParseHelper() (stmt *tsg.Statement) {

	stmt = tsg.NewStatement()
	stmt.Func("parseFormValue")
	stmt.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().Id("val").Colon().Id("string").Op("|").Id("null"))
		pg.Add(tsg.NewStatement().Id("kind").Colon().Lit("string").Op("|").Lit("number").Op("|").Lit("boolean").Op("|").Lit("json"))
	})
	stmt.Colon().Id("string").Op("|").Id("number").Op("|").Id("boolean").Op("|").Id("unknown").Op("|").Id("undefined")
	stmt.BlockFunc(func(bg *tsg.Group) {
		bg.Add(tsg.NewStatement().If(tsg.NewStatement().Id("val").Op("==").Id("null"), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Return(tsg.NewStatement().Id("undefined")).Semicolon())
		}))
		bg.Add(tsg.NewStatement().If(tsg.NewStatement().Id("kind").Op("===").Lit("string"), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Return(tsg.NewStatement().Id("val")).Semicolon())
		}))
		bg.Add(tsg.NewStatement().If(tsg.NewStatement().Id("kind").Op("===").Lit("number"), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Const(tsLocalVar("n")).Op("=").Id("Number").Call(tsg.NewStatement().Id("val")).Semicolon())
			ig.Add(tsg.NewStatement().Return(tsg.NewStatement().Id("Number").Dot("isNaN").Call(tsg.NewStatement().Id(tsLocalVar("n"))).Op("?").Id("undefined").Op(":").Id(tsLocalVar("n"))).Semicolon())
		}))
		bg.Add(tsg.NewStatement().If(tsg.NewStatement().Id("kind").Op("===").Lit("boolean"), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Return(tsg.NewStatement().Id("val").Op("===").Lit("true")).Semicolon())
		}))
		bg.Add(tsg.NewStatement().If(tsg.NewStatement().Id("kind").Op("===").Lit("json"), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Try(func(tg *tsg.Group) {
				tg.Add(tsg.NewStatement().Return(tsg.NewStatement().Id("JSON").Dot("parse").Call(tsg.NewStatement().Id("val"))).Semicolon())
			}, func(cg *tsg.Group) {
				cg.Add(tsg.NewStatement().Return(tsg.NewStatement().Id("undefined")).Semicolon())
			}))
		}))
		bg.Add(tsg.NewStatement().Return(tsg.NewStatement().Id("val")).Semicolon())
	})
	stmt.Line()
	return stmt
}

func (r *ClientRenderer) formParseKind(variable *model.Variable) (kind string) {

	typ, ok := r.project.Types[variable.TypeID]
	if !ok {
		return "string"
	}
	if typ.Kind == model.TypeKindAlias && typ.AliasOf != "" {
		if aliasTyp, ok := r.project.Types[typ.AliasOf]; ok {
			typ = aliasTyp
		}
	}
	switch typ.Kind {
	case model.TypeKindBool:
		return "boolean"
	case model.TypeKindInt, model.TypeKindInt8, model.TypeKindInt16, model.TypeKindInt32, model.TypeKindInt64,
		model.TypeKindUint, model.TypeKindUint8, model.TypeKindUint16, model.TypeKindUint32, model.TypeKindUint64,
		model.TypeKindFloat32, model.TypeKindFloat64, model.TypeKindByte, model.TypeKindRune:
		return "number"
	case model.TypeKindString:
		return "string"
	default:
		return "json"
	}
}

func (r *ClientRenderer) renderHTTPBody(mg *tsg.Group, contract *model.Contract, method *model.Method, args []*model.Variable, httpMethod string, requestMultipart bool, bodyStreamArg *model.Variable, bodyStreamArgs []*model.Variable) {

	// Генерируем body только для POST/PUT/PATCH, чтобы избежать noUnusedLocals в сгенерированном коде.
	needBody := httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH"
	if !needBody {
		return
	}
	if requestMultipart && len(bodyStreamArgs) > 0 {
		mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").Id("new FormData").Call().Semicolon())
		for _, arg := range bodyStreamArgs {
			partName := r.streamPartName(contract, method, arg)
			mg.Add(tsg.NewStatement().Id(tsLocalVar("body")).Dot("append").Call(tsg.NewStatement().Lit(partName), tsg.NewStatement().Id(tsLocalVar("params")).Dot(tsSafeName(arg.Name))).Semicolon())
		}
		return
	}
	if bodyStreamArg != nil {
		mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").Id(tsLocalVar("params")).Dot(tsSafeName(bodyStreamArg.Name)).Semicolon())
		return
	}
	if len(args) > 0 {
		reqKind := content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagRequestContentType, "application/json"))
		if reqKind == content.KindForm {
			mg.Add(tsg.NewStatement().Const(tsLocalVar("bodyParams")).Op("=").Id("new URLSearchParams").Call().Semicolon())
			for _, arg := range args {
				formKey := r.formFieldName(method, arg)
				mg.Add(tsg.NewStatement().If(
					tsg.NewStatement().Id(tsLocalVar("params")).Dot(tsSafeName(arg.Name)).Op("!==").Id("undefined"),
					func(bg *tsg.Group) {
						bg.Add(tsg.NewStatement().Id(tsLocalVar("bodyParams")).Dot("append").Call(tsg.NewStatement().Lit(formKey), tsg.NewStatement().Id("String").Call(tsg.NewStatement().Id(tsLocalVar("params")).Dot(tsSafeName(arg.Name)))).Semicolon())
					}))
			}
			mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").Id(tsLocalVar("bodyParams")).Dot("toString").Call().Semicolon())
		} else {
			bodyObj := tsg.NewStatement()
			bodyObj.Values(func(bg *tsg.Group) {
				for _, arg := range args {
					bg.Add(tsg.NewStatement().Id(tsSafeName(arg.Name)).Colon().Id(tsLocalVar("params")).Dot(tsSafeName(arg.Name)))
				}
			})
			mg.Add(tsg.NewStatement().Const(tsLocalVar("bodyObj")).Op("=").Add(bodyObj).Semicolon())
			switch reqKind {
			case content.KindXML:
				mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").Id("new XMLBuilder").Call().Dot("build").Call(tsg.NewStatement().Id(tsLocalVar("bodyObj"))).Semicolon())
			case content.KindMsgpack:
				mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").Id("new Blob").Call(tsg.NewStatement().Id("Msgpack").Dot("encode").Call(tsg.NewStatement().Id(tsLocalVar("bodyObj")))).Semicolon())
			case content.KindCBOR:
				mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").Id("new Blob").Call(tsg.NewStatement().Id("Cbor").Dot("encode").Call(tsg.NewStatement().Id(tsLocalVar("bodyObj")))).Semicolon())
			case content.KindYAML:
				mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").Id("YAML").Dot("stringify").Call(tsg.NewStatement().Id(tsLocalVar("bodyObj"))).Semicolon())
			default:
				mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").Id("JSON").Dot("stringify").Call(tsg.NewStatement().Id(tsLocalVar("bodyObj"))).Semicolon())
			}
		}
	} else {
		mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").Lit("null").Semicolon())
	}
}

func (r *ClientRenderer) renderHTTPHeaders(mg *tsg.Group, contract *model.Contract, method *model.Method, requestMultipart bool, bodyStreamArg *model.Variable, responseMultipart bool, responseStreamResult *model.Variable) {

	switch {
	case requestMultipart:
		switch {
		case responseMultipart:
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit("multipart/form-data")).Semicolon())
		case responseStreamResult != nil:
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit("*/*")).Semicolon())
		default:
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit("application/json")).Semicolon())
		}
	case bodyStreamArg != nil:
		requestContentType := model.GetAnnotationValue(r.project, contract, method, nil, model.TagRequestContentType, "application/octet-stream")
		mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Content-Type"), tsg.NewStatement().Lit(requestContentType)).Semicolon())
		switch {
		case responseMultipart:
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit("multipart/form-data")).Semicolon())
		case responseStreamResult != nil:
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit("*/*")).Semicolon())
		default:
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit("application/json")).Semicolon())
		}
	default:
		reqCT := model.GetAnnotationValue(r.project, contract, method, nil, model.TagRequestContentType, "application/json")
		mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Content-Type"), tsg.NewStatement().Lit(reqCT)).Semicolon())
		resKind := content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json"))
		mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit(content.CanonicalMIME(resKind))).Semicolon())
	}
	mg.Add(tsg.NewStatement().
		ForOf("["+tsLocalVar("key")+", "+tsLocalVar("value")+"]", "Object.entries("+tsLocalVar("clientHeaders")+")", func(fg *tsg.Group) {
			fg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Id(tsLocalVar("key")), tsg.NewStatement().Id(tsLocalVar("value"))))
		}).
		Semicolon())
}

func (r *ClientRenderer) renderHTTPResponse(mg *tsg.Group, contract *model.Contract, method *model.Method, results []*model.Variable, responseTypeName string, responseMultipart bool, responseStreamResult *model.Variable) {

	if responseMultipart {
		streamResults := r.methodResponseBodyStreamResults(method)
		mg.Add(tsg.NewStatement().Const(tsLocalVar("formData")).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("formData").Call()).Semicolon())
		if len(streamResults) == 1 {
			partName := r.streamPartName(contract, method, streamResults[0])
			mg.Add(tsg.NewStatement().Return(tsg.NewStatement().Id(tsLocalVar("formData")).Dot("get").Call(tsg.NewStatement().Lit(partName))).Semicolon())
		} else {
			returnObj := tsg.NewStatement()
			returnObj.Values(func(rg *tsg.Group) {
				for _, res := range streamResults {
					partName := r.streamPartName(contract, method, res)
					rg.Add(tsg.NewStatement().Id(tsSafeName(res.Name)).Colon().Id(tsLocalVar("formData")).Dot("get").Call(tsg.NewStatement().Lit(partName)))
				}
			})
			mg.Return(returnObj)
		}
		return
	}
	if responseStreamResult != nil {
		mg.Add(tsg.NewStatement().Const(tsLocalVar("bodyData")).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("blob").Call()).Semicolon())
		if len(results) == 1 {
			mg.Return(tsg.NewStatement().Id(tsLocalVar("bodyData")))
			return
		}
		returnObj := tsg.NewStatement()
		returnObj.Values(func(rg *tsg.Group) {
			for _, ret := range results {
				if ret.TypeID == TypeIDIOReadCloser {
					rg.Add(tsg.NewStatement().Id(tsSafeName(ret.Name)).Colon().Id(tsLocalVar("bodyData")))
				} else {
					rg.Add(tsg.NewStatement().Id(tsSafeName(ret.Name)).Colon().Id(tsLocalVar("response")).Dot("headers").Dot("get").Call(tsg.NewStatement().Lit("Content-Type")))
				}
			}
		})
		mg.Return(returnObj)
		return
	}
	resKind := content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json"))
	switch resKind {
	case content.KindForm:
		mg.Add(tsg.NewStatement().Const(tsLocalVar("text")).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("text").Call()).Semicolon())
		mg.Add(tsg.NewStatement().Const(tsLocalVar("formParams")).Op("=").Id("new URLSearchParams").Call(tsg.NewStatement().Id(tsLocalVar("text"))).Semicolon())
		responseDataObj := tsg.NewStatement()
		responseDataObj.Values(func(rg *tsg.Group) {
			for _, ret := range results {
				formKey := r.formFieldName(method, ret)
				kind := r.formParseKind(ret)
				rg.Add(tsg.NewStatement().Id(tsSafeName(ret.Name)).Colon().Id("parseFormValue").Call(tsg.NewStatement().Id(tsLocalVar("formParams")).Dot("get").Call(tsg.NewStatement().Lit(formKey)), tsg.NewStatement().Lit(kind)))
			}
		})
		mg.Add(tsg.NewStatement().Const(tsLocalVar("responseData")).Colon().Id(responseTypeName).Op("=").Add(responseDataObj).Semicolon())
	case content.KindXML:
		mg.Add(tsg.NewStatement().Const(tsLocalVar("text")).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("text").Call()).Semicolon())
		mg.Add(tsg.NewStatement().Const(tsLocalVar("responseData")).Colon().Id(responseTypeName).Op("=").Id("new XMLParser").Call().Dot("parse").Call(tsg.NewStatement().Id(tsLocalVar("text"))).Op("as").Id(responseTypeName).Semicolon())
	case content.KindMsgpack:
		mg.Add(tsg.NewStatement().Const(tsLocalVar("buf")).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("arrayBuffer").Call()).Semicolon())
		mg.Add(tsg.NewStatement().Const(tsLocalVar("responseData")).Colon().Id(responseTypeName).Op("=").Id("Msgpack").Dot("decode").Call(tsg.NewStatement().Id("new Uint8Array").Call(tsg.NewStatement().Id(tsLocalVar("buf")))).Op("as").Id(responseTypeName).Semicolon())
	case content.KindCBOR:
		mg.Add(tsg.NewStatement().Const(tsLocalVar("buf")).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("arrayBuffer").Call()).Semicolon())
		mg.Add(tsg.NewStatement().Const(tsLocalVar("responseData")).Colon().Id(responseTypeName).Op("=").Id("Cbor").Dot("decode").Call(tsg.NewStatement().Id("new Uint8Array").Call(tsg.NewStatement().Id(tsLocalVar("buf")))).Op("as").Id(responseTypeName).Semicolon())
	case content.KindYAML:
		mg.Add(tsg.NewStatement().Const(tsLocalVar("text")).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("text").Call()).Semicolon())
		mg.Add(tsg.NewStatement().Const(tsLocalVar("responseData")).Colon().Id(responseTypeName).Op("=").Id("YAML").Dot("parse").Call(tsg.NewStatement().Id(tsLocalVar("text"))).Op("as").Id(responseTypeName).Semicolon())
	default:
		mg.Add(tsg.NewStatement().Const(tsLocalVar("responseData")).Colon().Id(responseTypeName).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("json").Call()).Op("as").Id(responseTypeName).Semicolon())
	}
	switch {
	case len(results) == 1 && (model.IsAnnotationSet(r.project, contract, method, nil, model.TagHttpEnableInlineSingle) || r.resultHasJsonInline(method, results[0])):
		mg.Return(tsg.NewStatement().Id(tsLocalVar("responseData")))
	case len(results) == 1:
		mg.Return(tsg.NewStatement().Id(tsLocalVar("responseData")).Dot(tsSafeName(results[0].Name)))
	case r.responseHasAnyInline(method, results):
		mg.Return(tsg.NewStatement().Id(tsLocalVar("responseData")))
	default:
		returnObj := tsg.NewStatement()
		returnObj.Values(func(rg *tsg.Group) {
			for _, ret := range results {
				rg.Add(tsg.NewStatement().Id(tsSafeName(ret.Name)).Colon().Id(tsLocalVar("responseData")).Dot(tsSafeName(ret.Name)))
			}
		})
		mg.Return(returnObj)
	}
}

func (r *ClientRenderer) resultHasJsonInline(method *model.Method, v *model.Variable) bool {

	sub := method.Annotations.Sub(v.Name)
	for key, value := range sub {
		if key != model.TagParamTags {
			continue
		}
		for _, item := range strings.Split(value, "|") {
			tokens := strings.SplitN(strings.TrimSpace(item), ":", 2)
			if len(tokens) < 2 {
				continue
			}
			tagName := strings.TrimSpace(tokens[0])
			tagValue := strings.TrimSpace(tokens[1])
			if tagName == "json" && (tagValue == "inline" || strings.Contains(tagValue, ",inline")) {
				return true
			}
		}
	}
	return false
}

func (r *ClientRenderer) responseHasAnyInline(method *model.Method, results []*model.Variable) bool {

	for _, v := range results {
		if r.resultHasJsonInline(method, v) {
			return true
		}
	}
	return false
}
