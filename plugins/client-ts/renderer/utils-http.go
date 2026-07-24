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
	"tgp/internal/tags"
	"tgp/plugins/client-ts/tsg"
)

func (r *ClientRenderer) isHTTP(method *model.Method, contract *model.Contract) (ok bool) {

	return model.MethodIsHTTP(r.project, contract, method)
}

func (r *ClientRenderer) renderHTTPMethod(grp *tsg.Group, method *model.Method, contract *model.Contract) {

	args := r.argsForClient(contract, method)
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
				if arg.NumberOfPointers > 0 || model.IsAnnotationSet(r.project, contract, method, nil, "nullable") {
					paramStmt.Optional()
				}
				paramStmt.Colon()
				paramStmt.Add(tsg.TypeFromString(typeStr))
				pg.Add(paramStmt)
			}
		}
	})

	returnType := r.resultToTypeStatement(method, results)

	var responseTypeName string
	if len(results) > 0 {
		responseTypeName = r.responseTypeName(contract, method)
	}

	methodStmt := tsg.NewStatement()
	methodStmt.Public()
	methodStmt.AsyncMethodWithParams(r.lcName(method.Name), methodParams, returnType, func(mg *tsg.Group) {
		if len(args) > 0 {
			paramsObj := tsg.NewStatement()
			paramsObj.Const(tsLocalVar("params")).Op("=")
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

		fullPath := strings.TrimPrefix(model.MethodHTTPFullPath(r.project, contract, method), "/")
		urlStmt2 := tsg.NewStatement()
		urlStmt2.Var(tsLocalVar("url")).Op("=").Id(tsLocalVar("baseURL"))
		ternaryExpr := tsg.NewStatement()
		ternaryExpr.Op("(").Id(tsLocalVar("baseURL")).Dot("endsWith").Call(tsg.NewStatement().Lit("/")).Op("?").Lit("").Op(":").Lit("/").Op(")")
		urlStmt2.Op("+").Add(ternaryExpr)
		urlStmt2.Op("+").Lit(fullPath)
		mg.Add(urlStmt2.Semicolon())
		for _, segmentName := range r.httpPathParamNames(method, contract) {
			if arg := r.argByPathParamName(contract, method, segmentName); arg != nil {
				mg.Add(tsg.NewStatement().
					Id(tsLocalVar("url")).
					Op("=").
					Id(tsLocalVar("url")).
					Dot("replace").
					Call(tsg.NewStatement().Lit(":"+segmentName), tsg.NewStatement().Id("encodeURIComponent").Call(tsg.NewStatement().Id(tsLocalVar("params")).Dot(tsSafeName(arg.Name)))).
					Semicolon())
			}
		}
		pathParamSet := make(map[string]bool)
		for _, segmentName := range r.httpPathParamNames(method, contract) {
			if arg := r.argByPathParamName(contract, method, segmentName); arg != nil {
				pathParamSet[arg.Name] = true
			}
		}
		argParamMap := model.HTTPArgQueryMapForRequest(r.project, contract, method)
		var queryParams []struct {
			arg      *model.Variable
			queryKey string
		}
		for _, arg := range args {
			if pathParamSet[arg.Name] {
				continue
			}
			if queryKey, ok := argParamMap[arg.Name]; ok {
				queryParams = append(queryParams, struct {
					arg      *model.Variable
					queryKey string
				}{arg, queryKey})
			}
		}
		sort.Slice(queryParams, func(i, j int) bool { return queryParams[i].queryKey < queryParams[j].queryKey })
		for i, qp := range queryParams {
			sep := "?"
			if i > 0 {
				sep = "&"
			}
			urlAppend := tsg.NewStatement().
				Id(tsLocalVar("url")).
				Op("=").
				Id(tsLocalVar("url")).
				Op("+").
				Lit(sep + qp.queryKey + "=").
				Op("+").
				Id("encodeURIComponent").Call(tsg.NewStatement().Id(tsLocalVar("params")).Dot(tsSafeName(qp.arg.Name)))
			if qp.arg.NumberOfPointers > 0 {
				mg.Add(tsg.NewStatement().If(
					tsg.NewStatement().Id(tsLocalVar("params")).Dot(tsSafeName(qp.arg.Name)).Op("!==").Id("undefined"),
					func(bg *tsg.Group) {
						bg.Add(urlAppend.Semicolon())
					}))
				continue
			}
			mg.Add(urlAppend.Semicolon())
		}

		requestMultipart := r.methodRequestMultipart(contract, method)
		bodyStreamArg := r.methodRequestBodyStreamArg(method)
		bodyStreamArgs := r.methodRequestBodyStreamArgs(method)
		responseMultipart := r.methodResponseMultipart(contract, method)
		responseStreamResult := r.methodResponseBodyStreamResult(method)
		bodyArgs := r.argsForRequestBody(contract, method)

		r.renderHTTPBody(mg, contract, method, bodyArgs, httpMethod, requestMultipart, bodyStreamArg, bodyStreamArgs)

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
		r.renderHTTPHeaders(mg, contract, method, requestMultipart, bodyStreamArg, responseMultipart, responseStreamResult, len(bodyArgs) > 0)

		headerMap := model.HTTPHeaderArgMapForRequest(r.project, contract, method)
		for _, arg := range r.argsForClient(contract, method) {
			if headerName, ok := headerMap[arg.Name]; ok {
				mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit(headerName), tsg.NewStatement().Id(tsLocalVar("params")).Dot(tsSafeName(arg.Name))).Semicolon())
			}
		}

		cookieMap := model.HTTPCookieArgMapForRequest(r.project, contract, method)
		hasCookieParams := false
		for _, arg := range r.argsForClient(contract, method) {
			if _, ok := cookieMap[arg.Name]; ok {
				hasCookieParams = true
				break
			}
		}
		if hasCookieParams {
			mg.Add(tsg.NewStatement().Const(tsLocalVar("cookieParts")).Colon().Id("string").Array(nil).Op("=").Index(nil).Semicolon())
			for _, arg := range r.argsForClient(contract, method) {
				if cookieName, ok := cookieMap[arg.Name]; ok {
					pushStmt := tsg.NewStatement().
						Id(tsLocalVar("cookieParts")).
						Dot("push").
						Call(tsg.NewStatement().Lit(cookieName + "=").Op("+").Id("encodeURIComponent").Call(tsg.NewStatement().Id(tsLocalVar("params")).Dot(tsSafeName(arg.Name)))).
						Semicolon()
					mg.Add(pushStmt)
				}
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

		hasBodyContent := httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH" || len(bodyArgs) > 0
		fetchOptions := tsg.NewStatement()
		fetchOptions.Values(func(fg *tsg.Group) {
			fg.Add(tsg.NewStatement().Id("method").Colon().Lit(httpMethod))
			fg.Add(tsg.NewStatement().Id("headers").Colon().Id(tsLocalVar("headers")))
			if hasBodyContent {
				if requestMultipart && (httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH") {
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
			ig.Add(tsg.NewStatement().Const(tsLocalVar("contentType")).Op("=").Id(tsLocalVar("response")).Dot("headers").Dot("get").Call(tsg.NewStatement().Lit("content-type")).Op("??").Lit("").Semicolon())
			ig.Throw(tsg.NewStatement().Id("this").Dot("baseClient").Dot("decodeHTTPError").Call(
				tsg.NewStatement().Id(tsLocalVar("response")).Dot("status"),
				tsg.NewStatement().Id(tsLocalVar("contentType")),
				tsg.NewStatement().Id(tsLocalVar("errorBody")),
			))
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

func (r *ClientRenderer) httpPath(method *model.Method, contract *model.Contract) (s string) {

	return model.MethodHTTPPathValue(r.project, contract, method)
}

func (r *ClientRenderer) httpPathParamNames(method *model.Method, contract *model.Contract) (out []string) {

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

func (r *ClientRenderer) argByPathParamName(_ *model.Contract, method *model.Method, pathSegmentName string) (v *model.Variable) {

	return model.ArgByPathSegment(method, pathSegmentName)
}

func (r *ClientRenderer) argsForClient(contract *model.Contract, method *model.Method) (out []*model.Variable) {

	mappings := model.BuildHTTPArgMappings(r.project, contract, method)
	implicit := model.HTTPImplicitArgSet(mappings)
	var list []*model.Variable
	for _, arg := range r.argsWithoutContext(method) {
		if _, ok := implicit[arg.Name]; !ok {
			list = append(list, arg)
		}
	}
	return list
}

func (r *ClientRenderer) argsForExchangeRequest(contract *model.Contract, method *model.Method) (out []*model.Variable) {

	omit := model.HTTPOmitFromRequestJSON(r.project, contract, method)
	var list []*model.Variable
	for _, arg := range r.argsWithoutContext(method) {
		if model.TypeRefIsChan(r.project, &arg.TypeRef) {
			continue
		}
		if _, ok := omit[arg.Name]; !ok {
			list = append(list, arg)
		}
	}
	return list
}

func (r *ClientRenderer) argsForRequestBody(contract *model.Contract, method *model.Method) (out []*model.Variable) {

	return model.HTTPArgsFromRequestBody(r.project, contract, method)
}

func (r *ClientRenderer) formFieldName(method *model.Method, variable *model.Variable) (key string) {

	if method == nil || method.Annotations == nil {
		return toLowerCamel(variable.Name)
	}
	if name, ok := tags.FormFieldName(method.Annotations, variable.Name); ok {
		return name
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
	stmt.Block(func(bg *tsg.Group) {
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

func (r *ClientRenderer) parseFormValueExpr(source *tsg.Statement, kind string) (expr *tsg.Statement) {

	expr = tsg.NewStatement().Id("parseFormValue").Call(source, tsg.NewStatement().Lit(kind))
	switch kind {
	case "string":
		expr = expr.Op("as").Id("string")
	case "number":
		expr = expr.Op("as").Id("number")
	case "boolean":
		expr = expr.Op("as").Id("boolean")
	}
	return expr
}

func (r *ClientRenderer) renderHTTPBody(mg *tsg.Group, contract *model.Contract, method *model.Method, args []*model.Variable, httpMethod string, requestMultipart bool, bodyStreamArg *model.Variable, bodyStreamArgs []*model.Variable) {

	needBody := httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH" || len(args) > 0
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
				xmlRoot := "request" + contract.Name + method.Name
				wrapped := tsg.NewStatement().Values(func(bg *tsg.Group) {
					bg.Add(tsg.NewStatement().Id(xmlRoot).Colon().Id(tsLocalVar("bodyObj")))
				})
				mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").Id("new XMLBuilder").Call().Dot("build").Call(wrapped).Semicolon())
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

func (r *ClientRenderer) renderHTTPHeaders(mg *tsg.Group, contract *model.Contract, method *model.Method, requestMultipart bool, bodyStreamArg *model.Variable, responseMultipart bool, responseStreamResult *model.Variable, hasBody bool) {

	switch {
	case requestMultipart:
		switch {
		case responseMultipart:
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit("multipart/form-data")).Semicolon())
		case responseStreamResult != nil:
			acceptType := model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/octet-stream")
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit(acceptType)).Semicolon())
		default:
			acceptType := model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json")
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit(acceptType)).Semicolon())
		}
	case bodyStreamArg != nil:
		requestContentType := model.GetAnnotationValue(r.project, contract, method, nil, model.TagRequestContentType, "application/octet-stream")
		mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Content-Type"), tsg.NewStatement().Lit(requestContentType)).Semicolon())
		switch {
		case responseMultipart:
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit("multipart/form-data")).Semicolon())
		case responseStreamResult != nil:
			acceptType := model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/octet-stream")
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit(acceptType)).Semicolon())
		default:
			acceptType := model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json")
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit(acceptType)).Semicolon())
		}
	default:
		if hasBody {
			reqCT := model.GetAnnotationValue(r.project, contract, method, nil, model.TagRequestContentType, "application/json")
			mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Content-Type"), tsg.NewStatement().Lit(reqCT)).Semicolon())
		}
		var defaultAccept string
		switch {
		case responseMultipart:
			defaultAccept = "multipart/form-data"
		case responseStreamResult != nil:
			defaultAccept = "application/octet-stream"
		default:
			defaultAccept = "application/json"
		}
		acceptType := model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, defaultAccept)
		mg.Add(tsg.NewStatement().Id(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit(acceptType)).Semicolon())
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
		responseHeaderMap := model.HTTPResultHeaderMapForResponse(r.project, contract, method)
		returnObj := tsg.NewStatement()
		returnObj.Values(func(rg *tsg.Group) {
			for _, ret := range results {
				if ret.TypeID == TypeIDIOReadCloser {
					rg.Add(tsg.NewStatement().Id(tsSafeName(ret.Name)).Colon().Id(tsLocalVar("bodyData")))
				} else {
					headerName := responseHeaderMap[ret.Name]
					headerExpr := tsg.NewStatement().Id(tsLocalVar("response")).Dot("headers").Dot("get").Call(tsg.NewStatement().Lit(headerName))
					rg.Add(tsg.NewStatement().Id(tsSafeName(ret.Name)).Colon().Add(headerExpr).Op("??").Lit(""))
				}
			}
		})
		mg.Return(returnObj)
		return
	}

	bodyResults := model.HTTPResultsForExchangeBody(r.project, contract, method)
	if len(bodyResults) == 0 {
		mg.Add(tsg.NewStatement().Var(tsLocalVar("responseData")).Colon().Id(responseTypeName).Op("=").Values(nil).Op("as").Id(responseTypeName).Semicolon())
	} else {
		resKind := content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json"))
		switch resKind {
		case content.KindForm:
			r.needParseFormValueHelper = true
			mg.Add(tsg.NewStatement().Const(tsLocalVar("text")).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("text").Call()).Semicolon())
			mg.Add(tsg.NewStatement().Const(tsLocalVar("formParams")).Op("=").Id("new URLSearchParams").Call(tsg.NewStatement().Id(tsLocalVar("text"))).Semicolon())
			responseDataObj := tsg.NewStatement()
			responseDataObj.Values(func(rg *tsg.Group) {
				for _, ret := range bodyResults {
					formKey := r.formFieldName(method, ret)
					kind := r.formParseKind(ret)
					rg.Add(tsg.NewStatement().Id(tsSafeName(ret.Name)).Colon().Add(r.parseFormValueExpr(tsg.NewStatement().Id(tsLocalVar("formParams")).Dot("get").Call(tsg.NewStatement().Lit(formKey)), kind)))
				}
			})
			mg.Add(tsg.NewStatement().Var(tsLocalVar("responseData")).Colon().Id(responseTypeName).Op("=").Add(responseDataObj).Semicolon())
		case content.KindXML:
			mg.Add(tsg.NewStatement().Const(tsLocalVar("text")).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("text").Call()).Semicolon())
			mg.Add(tsg.NewStatement().Const(tsLocalVar("parsed")).Op("=").Id("new XMLParser").Call().Dot("parse").Call(tsg.NewStatement().Id(tsLocalVar("text"))).Semicolon())
			mg.Add(tsg.NewStatement().Const(tsLocalVar("rootKeys")).Op("=").Id("Object").Dot("keys").Call(tsg.NewStatement().Id(tsLocalVar("parsed"))).Semicolon())
			mg.Add(tsg.NewStatement().Var(tsLocalVar("responseData")).Colon().Id(responseTypeName).Op("=").Op("(").Id(tsLocalVar("rootKeys")).Dot("length").Op("===").Lit(1).Op("?").Id(tsLocalVar("parsed")).Index(tsg.NewStatement().Id(tsLocalVar("rootKeys")).Index(tsg.NewStatement().Lit(0))).Op(":").Id(tsLocalVar("parsed")).Op(")").Op("as").Id(responseTypeName).Semicolon())
		case content.KindMsgpack:
			mg.Add(tsg.NewStatement().Const(tsLocalVar("buf")).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("arrayBuffer").Call()).Semicolon())
			mg.Add(tsg.NewStatement().Var(tsLocalVar("responseData")).Colon().Id(responseTypeName).Op("=").Id("Msgpack").Dot("decode").Call(tsg.NewStatement().Id("new Uint8Array").Call(tsg.NewStatement().Id(tsLocalVar("buf")))).Op("as").Id(responseTypeName).Semicolon())
		case content.KindCBOR:
			mg.Add(tsg.NewStatement().Const(tsLocalVar("buf")).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("arrayBuffer").Call()).Semicolon())
			mg.Add(tsg.NewStatement().Var(tsLocalVar("responseData")).Colon().Id(responseTypeName).Op("=").Id("Cbor").Dot("decode").Call(tsg.NewStatement().Id("new Uint8Array").Call(tsg.NewStatement().Id(tsLocalVar("buf")))).Op("as").Id(responseTypeName).Semicolon())
		case content.KindYAML:
			mg.Add(tsg.NewStatement().Const(tsLocalVar("text")).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("text").Call()).Semicolon())
			mg.Add(tsg.NewStatement().Var(tsLocalVar("responseData")).Colon().Id(responseTypeName).Op("=").Id("YAML").Dot("parse").Call(tsg.NewStatement().Id(tsLocalVar("text"))).Op("as").Id(responseTypeName).Semicolon())
		default:
			mg.Add(tsg.NewStatement().Var(tsLocalVar("responseData")).Colon().Id(responseTypeName).Op("=").Await(tsg.NewStatement().Id(tsLocalVar("response")).Dot("json").Call()).Op("as").Id(responseTypeName).Semicolon())
		}
	}
	r.renderHTTPResponseMergeHeadersAndCookies(mg, contract, method, results, responseTypeName)
	switch {
	case len(results) == 1 && (model.IsAnnotationSet(r.project, contract, method, nil, model.TagHttpEnableInlineSingle) || r.resultHasJsonInline(method, results[0])):
		mg.Return(tsg.NewStatement().Id(tsLocalVar("responseData")))
	case len(results) == 1:
		res := results[0]
		dataRef := tsg.NewStatement().Id(tsLocalVar("responseData"))
		if res.NumberOfPointers > 0 {
			mg.Return(dataRef.OptionalChain(tsSafeName(res.Name)).Op("??").Id("null"))
		} else {
			mg.Return(dataRef.Dot(tsSafeName(res.Name)))
		}
	case r.responseHasAnyInline(method, results):
		mg.Return(tsg.NewStatement().Id(tsLocalVar("responseData")))
	default:
		returnObj := tsg.NewStatement()
		returnObj.Values(func(rg *tsg.Group) {
			for _, ret := range results {
				dataRef := tsg.NewStatement().Id(tsLocalVar("responseData"))
				var val *tsg.Statement
				if ret.NumberOfPointers > 0 {
					val = dataRef.OptionalChain(tsSafeName(ret.Name)).Op("??").Id("null")
				} else {
					val = dataRef.Dot(tsSafeName(ret.Name))
				}
				rg.Add(tsg.NewStatement().Id(tsSafeName(ret.Name)).Colon().Add(val))
			}
		})
		mg.Return(returnObj)
	}
}

func (r *ClientRenderer) renderHTTPResponseMergeHeadersAndCookies(mg *tsg.Group, contract *model.Contract, method *model.Method, results []*model.Variable, responseTypeName string) {

	headerMap := model.HTTPResultHeaderMapForResponse(r.project, contract, method)
	cookieMap := model.HTTPResultCookieMapForResponse(r.project, contract, method)
	if len(headerMap) == 0 && len(cookieMap) == 0 {
		return
	}

	excludeFromBody := model.HTTPResultNamesExcludeFromBody(r.project, contract, method)

	needGetResponseCookie := false
	for _, ret := range results {
		if cookieMap[ret.Name] != "" {
			needGetResponseCookie = true
			break
		}
	}
	if needGetResponseCookie {
		paramStmt := tsg.NewStatement().Params(func(pg *tsg.Group) {
			pg.Add(tsg.NewStatement().Id("res").Colon().Id("Response"))
			pg.Add(tsg.NewStatement().Id("name").Colon().Id("string"))
		})
		getCookieBlock := func(bg *tsg.Group) {
			hStmt := tsg.NewStatement().Const("h").Op("=").Id("res").Dot("headers").Dot("get").Call(tsg.NewStatement().Lit("Set-Cookie")).Op("??").Lit("")
			bg.Add(hStmt.Semicolon())
			forOfStmt := tsg.NewStatement().ForOf("part", "h.split(';')", func(fg *tsg.Group) {
				trimmedStmt := tsg.NewStatement().Const("trimmed").Op("=").Id("part").Dot("trim").Call()
				fg.Add(trimmedStmt.Semicolon())
				eqStmt := tsg.NewStatement().Const("eq").Op("=").Id("trimmed").Dot("indexOf").Call(tsg.NewStatement().Lit("="))
				fg.Add(eqStmt.Semicolon())
				ifContinue := tsg.NewStatement().If(tsg.NewStatement().Id("eq").Op("<").Lit(0), func(ig *tsg.Group) {
					ig.Add(tsg.NewStatement().Id("continue").Semicolon())
				})
				fg.Add(ifContinue)
				kStmt := tsg.NewStatement().Const("k").Op("=").Id("trimmed").Dot("slice").Call(tsg.NewStatement().Lit(0), tsg.NewStatement().Id("eq"))
				fg.Add(kStmt.Semicolon())
				vStmt := tsg.NewStatement().Const("v").Op("=").Id("trimmed").Dot("slice").Call(tsg.NewStatement().Id("eq").Op("+").Lit(1))
				fg.Add(vStmt.Semicolon())
				ifReturn := tsg.NewStatement().If(tsg.NewStatement().Id("k").Op("===").Id("name"), func(ig *tsg.Group) {
					ig.Add(tsg.NewStatement().Return(tsg.NewStatement().Id("decodeURIComponent").Call(tsg.NewStatement().Id("v"))).Semicolon())
				})
				fg.Add(ifReturn)
			})
			bg.Add(forOfStmt.Semicolon())
			bg.Add(tsg.NewStatement().Return(tsg.NewStatement().Id("null")).Semicolon())
		}
		getCookieStmt := tsg.NewStatement().Const("getResponseCookie").Op("=").Add(paramStmt).Colon().Id("string").Op("|").Id("null").Op("=>").Block(getCookieBlock)
		mg.Add(getCookieStmt.Semicolon())
	}

	mg.Add(tsg.NewStatement().Const("mergedResult").Colon().Id(responseTypeName).Op("=").Values(nil).Op("as").Id(responseTypeName).Semicolon())
	for _, ret := range results {
		_, fromHeaderOrCookieOnly := excludeFromBody[ret.Name]
		headerName, hasHeader := headerMap[ret.Name]
		cookieName, hasCookie := cookieMap[ret.Name]

		if !fromHeaderOrCookieOnly {
			mg.Add(tsg.NewStatement().Assign(
				tsg.NewStatement().Id("mergedResult").Dot(tsSafeName(ret.Name)),
				tsg.NewStatement().Id(tsLocalVar("responseData")).OptionalChain(tsSafeName(ret.Name)),
			).Semicolon())
		}

		if hasHeader || hasCookie {
			r.needParseFormValueHelper = true
			kind := r.formParseKind(ret)
			var transportSrc *tsg.Statement
			if hasHeader {
				transportSrc = tsg.NewStatement().Id(tsLocalVar("response")).Dot("headers").Dot("get").Call(tsg.NewStatement().Lit(headerName))
			} else {
				transportSrc = tsg.NewStatement().Id("getResponseCookie").Call(tsg.NewStatement().Id(tsLocalVar("response")), tsg.NewStatement().Lit(cookieName))
			}
			// Header/cookie всегда перекрывает body, включая пустое значение (как Go Header.Get).
			mg.Add(tsg.NewStatement().Assign(
				tsg.NewStatement().Id("mergedResult").Dot(tsSafeName(ret.Name)),
				r.parseFormValueExpr(transportSrc, kind),
			).Semicolon())
		}
	}
	mg.Add(tsg.NewStatement().Assign(tsg.NewStatement().Id(tsLocalVar("responseData")), tsg.NewStatement().Id("mergedResult")).Semicolon())
}

func (r *ClientRenderer) resultHasJsonInline(method *model.Method, v *model.Variable) (ok bool) {

	if method == nil {
		return false
	}
	return tags.HasJSONInline(method.Annotations, v.Name)
}

func (r *ClientRenderer) responseHasAnyInline(method *model.Method, results []*model.Variable) (ok bool) {

	for _, v := range results {
		if r.resultHasJsonInline(method, v) {
			return true
		}
	}
	return false
}

// tsRPCSingleResultExpr — доступ к единственному результату JSON-RPC после cast в response type.
func (r *ClientRenderer) tsRPCSingleResultExpr(contract *model.Contract, method *model.Method, ret *model.Variable) *tsg.Statement {

	if model.IsAnnotationSet(r.project, contract, method, nil, model.TagHttpEnableInlineSingle) {
		return tsg.NewStatement().Id("result")
	}
	if model.ResultFieldEmbedded(r.project, contract, method, ret) {
		return tsg.NewStatement().Id("result")
	}

	return tsg.NewStatement().Id("result").Dot(tsSafeName(ret.Name))
}
