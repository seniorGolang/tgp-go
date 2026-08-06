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
	"tgp/internal/converter"
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
		grp.Comment("Вызывает HTTP метод " + method.Name)
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
				grp.Comment("  - " + typeName)
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
				paramStmt.ID(tsSafeName(arg.Name))
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
					vg.Add(tsg.NewStatement().ID(tsSafeName(arg.Name)).Colon().ID(tsSafeName(arg.Name)))
				}
			})
			mg.Add(paramsObj.Semicolon())
		}

		fullPath := model.MethodHTTPFullPath(r.project, contract, method)
		pathStmt := tsg.NewStatement()
		pathStmt.Var(tsLocalVar("path")).Op("=").Lit(fullPath).Semicolon()
		mg.Add(pathStmt)
		for _, segmentName := range r.httpPathParamNames(method, contract) {
			if arg := r.argByPathParamName(contract, method, segmentName); arg != nil {
				mg.Add(tsg.NewStatement().
					ID(tsLocalVar("path")).
					Op("=").
					ID(tsLocalVar("path")).
					Dot("replace").
					Call(tsg.NewStatement().Lit(":"+segmentName), tsg.NewStatement().ID("encodeURIComponent").Call(tsg.NewStatement().ID(tsLocalVar("params")).Dot(tsSafeName(arg.Name)))).
					Semicolon())
			}
		}
		urlStmt := tsg.NewStatement()
		urlStmt.Var(tsLocalVar("url")).Op("=").ID("this").Dot("baseClient").Dot("resolveURL").Call(tsg.NewStatement().ID(tsLocalVar("path"))).Semicolon()
		mg.Add(urlStmt)
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
			valueExpr := r.encodeTransportValueExpr(tsg.NewStatement().ID(tsLocalVar("params")).Dot(tsSafeName(qp.arg.Name)), qp.arg)
			urlAppend := tsg.NewStatement().
				ID(tsLocalVar("url")).
				Op("=").
				ID(tsLocalVar("url")).
				Op("+").
				Lit(sep + qp.queryKey + "=").
				Op("+").
				ID("encodeURIComponent").Call(valueExpr)
			if qp.arg.NumberOfPointers > 0 {
				mg.Add(tsg.NewStatement().If(
					tsg.NewStatement().ID(tsLocalVar("params")).Dot(tsSafeName(qp.arg.Name)).Op("!==").ID(typeUndefined),
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
			ID("Record").
			Generic(typeString, typeString).
			Op("=").
			Await(tsg.NewStatement().ID("this").Dot("baseClient").Dot("getHeaders").Call()).
			Semicolon()
		mg.Add(headersVar)

		headersStmt := tsg.NewStatement()
		headersStmt.Const(tsLocalVar("headers")).Op("=").ID("new Headers").Call().Semicolon()
		mg.Add(headersStmt)
		r.renderHTTPHeaders(mg, contract, method, requestMultipart, bodyStreamArg, responseMultipart, responseStreamResult, len(bodyArgs) > 0)

		headerMap := model.HTTPHeaderArgMapForRequest(r.project, contract, method)
		for _, arg := range r.argsForClient(contract, method) {
			if headerName, ok := headerMap[arg.Name]; ok {
				headerValue := r.encodeTransportValueExpr(tsg.NewStatement().ID(tsLocalVar("params")).Dot(tsSafeName(arg.Name)), arg)
				mg.Add(tsg.NewStatement().ID(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit(headerName), headerValue).Semicolon())
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
			mg.Add(tsg.NewStatement().Const(tsLocalVar("cookieParts")).Colon().ID(typeString).Array(nil).Op("=").Index(nil).Semicolon())
			for _, arg := range r.argsForClient(contract, method) {
				if cookieName, ok := cookieMap[arg.Name]; ok {
					cookieValue := r.encodeTransportValueExpr(tsg.NewStatement().ID(tsLocalVar("params")).Dot(tsSafeName(arg.Name)), arg)
					pushStmt := tsg.NewStatement().
						ID(tsLocalVar("cookieParts")).
						Dot("push").
						Call(tsg.NewStatement().Lit(cookieName + "=").Op("+").ID("encodeURIComponent").Call(cookieValue)).
						Semicolon()
					mg.Add(pushStmt)
				}
			}
			mg.Add(tsg.NewStatement().ID(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Cookie"), tsg.NewStatement().ID(tsLocalVar("cookieParts")).Dot("join").Call(tsg.NewStatement().Lit("; "))).Semicolon())
		}

		if requestMultipart && (httpMethod == methodPOST || httpMethod == methodPUT || httpMethod == methodPATCH) {
			mg.Add(tsg.NewStatement().Const(tsLocalVar("multipartReq")).Op("=").ID("new Request").Call(tsg.NewStatement().ID(tsLocalVar("url")), tsg.NewStatement().Values(func(vg *tsg.Group) {
				vg.Add(tsg.NewStatement().ID("method").Colon().Lit(httpMethod))
				vg.Add(tsg.NewStatement().ID("body").Colon().ID(tsLocalVar("body")))
			})).Semicolon())
			mg.Add(tsg.NewStatement().ID(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Content-Type"), tsg.NewStatement().ID(tsLocalVar("multipartReq")).Dot("headers").Dot("get").Call(tsg.NewStatement().Lit("Content-Type")).Op("!")).Semicolon())
		}

		hasBodyContent := httpMethod == methodPOST || httpMethod == methodPUT || httpMethod == methodPATCH || len(bodyArgs) > 0
		fetchOptions := tsg.NewStatement()
		fetchOptions.Values(func(fg *tsg.Group) {
			fg.Add(tsg.NewStatement().ID("method").Colon().Lit(httpMethod))
			fg.Add(tsg.NewStatement().ID("headers").Colon().ID(tsLocalVar("headers")))
			if hasBodyContent {
				if requestMultipart && (httpMethod == methodPOST || httpMethod == methodPUT || httpMethod == methodPATCH) {
					fg.Add(tsg.NewStatement().ID("body").Colon().ID(tsLocalVar("multipartReq")).Dot("body"))
					fg.Add(tsg.NewStatement().ID("duplex").Colon().Lit("half"))
				} else {
					fg.Add(tsg.NewStatement().ID("body").Colon().ID(tsLocalVar("body")))
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
		fetchStmt.Const(tsLocalVar("response")).Op("=").Await(tsg.NewStatement().ID("fetch").Call(tsg.NewStatement().ID(tsLocalVar("url")), fetchOptsArg))
		mg.Add(fetchStmt.Semicolon())

		successCode := 200
		if model.IsAnnotationSet(r.project, contract, method, nil, model.TagHTTPSuccess) {
			if code, err := strconv.Atoi(model.GetAnnotationValue(r.project, contract, method, nil, model.TagHTTPSuccess, "200")); err == nil {
				successCode = code
			}
		}

		mg.If(tsg.NewStatement().ID(tsLocalVar("response")).Dot("status").Op("!=").Lit(successCode), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Const(tsLocalVar("errorBody")).Op("=").Await(tsg.NewStatement().ID(tsLocalVar("response")).Dot("text").Call()).Semicolon())
			ig.Add(tsg.NewStatement().Const(tsLocalVar("contentType")).Op("=").ID(tsLocalVar("response")).Dot("headers").Dot("get").Call(tsg.NewStatement().Lit("content-type")).Op("??").Lit("").Semicolon())
			ig.Throw(tsg.NewStatement().ID("this").Dot("baseClient").Dot("decodeHTTPError").Call(
				tsg.NewStatement().ID(tsLocalVar("response")).Dot("status"),
				tsg.NewStatement().ID(tsLocalVar("contentType")),
				tsg.NewStatement().ID(tsLocalVar("errorBody")),
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
	for token := range strings.SplitSeq(pathStr, "/") {
		token = strings.TrimSpace(token)
		if after, ok := strings.CutPrefix(token, ":"); ok {
			names = append(names, after)
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

func formKindFromTypeKind(kind model.TypeKind) (parseKind string) {

	switch kind {
	case model.TypeKindBool:
		return typeBoolean
	case model.TypeKindInt, model.TypeKindInt8, model.TypeKindInt16, model.TypeKindInt32, model.TypeKindInt64,
		model.TypeKindUint, model.TypeKindUint8, model.TypeKindUint16, model.TypeKindUint32, model.TypeKindUint64,
		model.TypeKindFloat32, model.TypeKindFloat64, model.TypeKindByte, model.TypeKindRune:
		return typeNumber
	case model.TypeKindString:
		return typeString
	default:
		return ""
	}
}

func (r *ClientRenderer) formElementParseKind(variable *model.Variable) (kind string) {

	typeID := variable.TypeID
	if typ, ok := r.project.Types[typeID]; ok && typ != nil {
		if typ.Kind == model.TypeKindAlias && typ.AliasOf != "" {
			if aliasTyp, ok := r.project.Types[typ.AliasOf]; ok && aliasTyp != nil {
				typ = aliasTyp
			}
		}
		if kind = formKindFromTypeKind(typ.Kind); kind != "" {
			return
		}
		if typ.UnderlyingKind != "" {
			if kind = formKindFromTypeKind(typ.UnderlyingKind); kind != "" {
				return
			}
		}
		return "json"
	}
	if converter.IsBuiltinTypeID(typeID) {
		if kind = formKindFromTypeKind(model.TypeKind(typeID)); kind != "" {
			return
		}
		if typeID == string(model.TypeKindError) {
			return typeString
		}
		return "json"
	}
	return "json"
}

func (r *ClientRenderer) isTransportSlice(variable *model.Variable) (ok bool) {

	return variable.IsSlice || variable.ArrayLen > 0
}

func (r *ClientRenderer) parseFormValueExpr(source *tsg.Statement, kind string) (expr *tsg.Statement) {

	expr = tsg.NewStatement().ID("parseFormValue").Call(source, tsg.NewStatement().Lit(kind))
	switch kind {
	case typeString:
		expr = expr.Op("as").ID(typeString)
	case typeNumber:
		expr = expr.Op("as").ID(typeNumber)
	case typeBoolean:
		expr = expr.Op("as").ID(typeBoolean)
	}
	return expr
}

func (r *ClientRenderer) parseTransportValueExpr(source *tsg.Statement, variable *model.Variable) (expr *tsg.Statement) {

	kind := r.formElementParseKind(variable)
	if r.isTransportSlice(variable) {
		r.needParseCSVFormHelper = true
		expr = tsg.NewStatement().ID("parseCSVFormValue").Call(source, tsg.NewStatement().Lit(kind))
		switch kind {
		case typeString:
			expr = expr.Op("as").ID(typeString).Array(nil)
		case typeNumber:
			expr = expr.Op("as").ID(typeNumber).Array(nil)
		case typeBoolean:
			expr = expr.Op("as").ID(typeBoolean).Array(nil)
		}
		return
	}
	r.needParseFormValueHelper = true
	expr = r.parseFormValueExpr(source, kind)
	return
}

func (r *ClientRenderer) encodeTransportValueExpr(value *tsg.Statement, variable *model.Variable) (expr *tsg.Statement) {

	if r.isTransportSlice(variable) {
		if r.formElementParseKind(variable) == typeString {
			return value.Dot("join").Call(tsg.NewStatement().Lit(","))
		}
		return value.Dot("map").Call(tsg.NewStatement().ID("String")).Dot("join").Call(tsg.NewStatement().Lit(","))
	}
	if r.formElementParseKind(variable) == typeString {
		return value
	}
	return tsg.NewStatement().ID("String").Call(value)
}

func (r *ClientRenderer) renderHTTPBody(mg *tsg.Group, contract *model.Contract, method *model.Method, args []*model.Variable, httpMethod string, requestMultipart bool, bodyStreamArg *model.Variable, bodyStreamArgs []*model.Variable) {

	needBody := httpMethod == methodPOST || httpMethod == methodPUT || httpMethod == methodPATCH || len(args) > 0
	if !needBody {
		return
	}
	if requestMultipart && len(bodyStreamArgs) > 0 {
		mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").ID("new FormData").Call().Semicolon())
		for _, arg := range bodyStreamArgs {
			partName := r.streamPartName(contract, method, arg)
			mg.Add(tsg.NewStatement().ID(tsLocalVar("body")).Dot("append").Call(tsg.NewStatement().Lit(partName), tsg.NewStatement().ID(tsLocalVar("params")).Dot(tsSafeName(arg.Name))).Semicolon())
		}
		return
	}
	if bodyStreamArg != nil {
		mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").ID(tsLocalVar("params")).Dot(tsSafeName(bodyStreamArg.Name)).Semicolon())
		return
	}
	if len(args) > 0 {
		reqKind := content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagRequestContentType, "application/json"))
		if reqKind == content.KindForm {
			mg.Add(tsg.NewStatement().Const(tsLocalVar("bodyParams")).Op("=").ID("new URLSearchParams").Call().Semicolon())
			for _, arg := range args {
				formKey := r.formFieldName(method, arg)
				mg.Add(tsg.NewStatement().If(
					tsg.NewStatement().ID(tsLocalVar("params")).Dot(tsSafeName(arg.Name)).Op("!==").ID(typeUndefined),
					func(bg *tsg.Group) {
						formValue := r.encodeTransportValueExpr(tsg.NewStatement().ID(tsLocalVar("params")).Dot(tsSafeName(arg.Name)), arg)
						bg.Add(tsg.NewStatement().ID(tsLocalVar("bodyParams")).Dot("append").Call(tsg.NewStatement().Lit(formKey), formValue).Semicolon())
					}))
			}
			mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").ID(tsLocalVar("bodyParams")).Dot("toString").Call().Semicolon())
		} else {
			bodyObj := tsg.NewStatement()
			bodyObj.Values(func(bg *tsg.Group) {
				for _, arg := range args {
					bg.Add(tsg.NewStatement().ID(tsSafeName(arg.Name)).Colon().ID(tsLocalVar("params")).Dot(tsSafeName(arg.Name)))
				}
			})
			mg.Add(tsg.NewStatement().Const(tsLocalVar("bodyObj")).Op("=").Add(bodyObj).Semicolon())
			switch reqKind {
			case content.KindXML:
				xmlRoot := "request" + contract.Name + method.Name
				wrapped := tsg.NewStatement().Values(func(bg *tsg.Group) {
					bg.Add(tsg.NewStatement().ID(xmlRoot).Colon().ID(tsLocalVar("bodyObj")))
				})
				mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").ID("new XMLBuilder").Call().Dot("build").Call(wrapped).Semicolon())
			case content.KindMsgpack:
				mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").ID("new Blob").Call(tsg.NewStatement().ID("Msgpack").Dot("encode").Call(tsg.NewStatement().ID(tsLocalVar("bodyObj")))).Semicolon())
			case content.KindCBOR:
				mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").ID("new Blob").Call(tsg.NewStatement().ID("Cbor").Dot("encode").Call(tsg.NewStatement().ID(tsLocalVar("bodyObj")))).Semicolon())
			case content.KindYAML:
				mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").ID("YAML").Dot("stringify").Call(tsg.NewStatement().ID(tsLocalVar("bodyObj"))).Semicolon())
			default:
				mg.Add(tsg.NewStatement().Const(tsLocalVar("body")).Op("=").ID("JSON").Dot("stringify").Call(tsg.NewStatement().ID(tsLocalVar("bodyObj"))).Semicolon())
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
			mg.Add(tsg.NewStatement().ID(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit("multipart/form-data")).Semicolon())
		case responseStreamResult != nil:
			acceptType := model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/octet-stream")
			mg.Add(tsg.NewStatement().ID(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit(acceptType)).Semicolon())
		default:
			acceptType := model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json")
			mg.Add(tsg.NewStatement().ID(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit(acceptType)).Semicolon())
		}
	case bodyStreamArg != nil:
		requestContentType := model.GetAnnotationValue(r.project, contract, method, nil, model.TagRequestContentType, "application/octet-stream")
		mg.Add(tsg.NewStatement().ID(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Content-Type"), tsg.NewStatement().Lit(requestContentType)).Semicolon())
		switch {
		case responseMultipart:
			mg.Add(tsg.NewStatement().ID(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit("multipart/form-data")).Semicolon())
		case responseStreamResult != nil:
			acceptType := model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/octet-stream")
			mg.Add(tsg.NewStatement().ID(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit(acceptType)).Semicolon())
		default:
			acceptType := model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json")
			mg.Add(tsg.NewStatement().ID(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit(acceptType)).Semicolon())
		}
	default:
		if hasBody {
			reqCT := model.GetAnnotationValue(r.project, contract, method, nil, model.TagRequestContentType, "application/json")
			mg.Add(tsg.NewStatement().ID(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Content-Type"), tsg.NewStatement().Lit(reqCT)).Semicolon())
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
		mg.Add(tsg.NewStatement().ID(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().Lit("Accept"), tsg.NewStatement().Lit(acceptType)).Semicolon())
	}
	mg.Add(tsg.NewStatement().
		ForOf("["+tsLocalVar("key")+", "+tsLocalVar("value")+"]", "Object.entries("+tsLocalVar("clientHeaders")+")", func(fg *tsg.Group) {
			fg.Add(tsg.NewStatement().ID(tsLocalVar("headers")).Dot("set").Call(tsg.NewStatement().ID(tsLocalVar("key")), tsg.NewStatement().ID(tsLocalVar("value"))))
		}).
		Semicolon())
}

func (r *ClientRenderer) renderHTTPResponse(mg *tsg.Group, contract *model.Contract, method *model.Method, results []*model.Variable, responseTypeName string, responseMultipart bool, responseStreamResult *model.Variable) {

	if responseMultipart {
		streamResults := r.methodResponseBodyStreamResults(method)
		mg.Add(tsg.NewStatement().Const(tsLocalVar("formData")).Op("=").Await(tsg.NewStatement().ID(tsLocalVar("response")).Dot("formData").Call()).Semicolon())
		if len(streamResults) == 1 {
			partName := r.streamPartName(contract, method, streamResults[0])
			mg.Add(tsg.NewStatement().Return(tsg.NewStatement().ID(tsLocalVar("formData")).Dot("get").Call(tsg.NewStatement().Lit(partName))).Semicolon())
		} else {
			returnObj := tsg.NewStatement()
			returnObj.Values(func(rg *tsg.Group) {
				for _, res := range streamResults {
					partName := r.streamPartName(contract, method, res)
					rg.Add(tsg.NewStatement().ID(tsSafeName(res.Name)).Colon().ID(tsLocalVar("formData")).Dot("get").Call(tsg.NewStatement().Lit(partName)))
				}
			})
			mg.Return(returnObj)
		}
		return
	}
	if responseStreamResult != nil {
		mg.Add(tsg.NewStatement().Const(tsLocalVar("bodyData")).Op("=").Await(tsg.NewStatement().ID(tsLocalVar("response")).Dot("blob").Call()).Semicolon())
		if len(results) == 1 {
			mg.Return(tsg.NewStatement().ID(tsLocalVar("bodyData")))
			return
		}
		responseHeaderMap := model.HTTPResultHeaderMapForResponse(r.project, contract, method)
		returnObj := tsg.NewStatement()
		returnObj.Values(func(rg *tsg.Group) {
			for _, ret := range results {
				if ret.TypeID == TypeIDIOReadCloser {
					rg.Add(tsg.NewStatement().ID(tsSafeName(ret.Name)).Colon().ID(tsLocalVar("bodyData")))
				} else {
					headerName := responseHeaderMap[ret.Name]
					headerExpr := tsg.NewStatement().ID(tsLocalVar("response")).Dot("headers").Dot("get").Call(tsg.NewStatement().Lit(headerName))
					rg.Add(tsg.NewStatement().ID(tsSafeName(ret.Name)).Colon().Add(headerExpr).Op("??").Lit(""))
				}
			}
		})
		mg.Return(returnObj)
		return
	}

	bodyResults := model.HTTPResultsForExchangeBody(r.project, contract, method)
	if len(bodyResults) == 0 {
		mg.Add(tsg.NewStatement().Var(tsLocalVar("responseData")).Colon().ID(responseTypeName).Op("=").Values(nil).Op("as").ID(responseTypeName).Semicolon())
	} else {
		resKind := content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json"))
		switch resKind {
		case content.KindForm:
			mg.Add(tsg.NewStatement().Const(tsLocalVar("text")).Op("=").Await(tsg.NewStatement().ID(tsLocalVar("response")).Dot("text").Call()).Semicolon())
			mg.Add(tsg.NewStatement().Const(tsLocalVar("formParams")).Op("=").ID("new URLSearchParams").Call(tsg.NewStatement().ID(tsLocalVar("text"))).Semicolon())
			responseDataObj := tsg.NewStatement()
			responseDataObj.Values(func(rg *tsg.Group) {
				for _, ret := range bodyResults {
					formKey := r.formFieldName(method, ret)
					rg.Add(tsg.NewStatement().ID(tsSafeName(ret.Name)).Colon().Add(r.parseTransportValueExpr(tsg.NewStatement().ID(tsLocalVar("formParams")).Dot("get").Call(tsg.NewStatement().Lit(formKey)), ret)))
				}
			})
			mg.Add(tsg.NewStatement().Var(tsLocalVar("responseData")).Colon().ID(responseTypeName).Op("=").Add(responseDataObj).Semicolon())
		case content.KindXML:
			mg.Add(tsg.NewStatement().Const(tsLocalVar("text")).Op("=").Await(tsg.NewStatement().ID(tsLocalVar("response")).Dot("text").Call()).Semicolon())
			mg.Add(tsg.NewStatement().Const(tsLocalVar("parsed")).Op("=").ID("new XMLParser").Call().Dot("parse").Call(tsg.NewStatement().ID(tsLocalVar("text"))).Semicolon())
			mg.Add(tsg.NewStatement().Const(tsLocalVar("rootKeys")).Op("=").ID("Object").Dot("keys").Call(tsg.NewStatement().ID(tsLocalVar("parsed"))).Semicolon())
			mg.Add(tsg.NewStatement().Var(tsLocalVar("responseData")).Colon().ID(responseTypeName).Op("=").Op("(").ID(tsLocalVar("rootKeys")).Dot("length").Op("===").Lit(1).Op("?").ID(tsLocalVar("parsed")).Index(tsg.NewStatement().ID(tsLocalVar("rootKeys")).Index(tsg.NewStatement().Lit(0))).Op(":").ID(tsLocalVar("parsed")).Op(")").Op("as").ID(responseTypeName).Semicolon())
		case content.KindMsgpack:
			mg.Add(tsg.NewStatement().Const(tsLocalVar("buf")).Op("=").Await(tsg.NewStatement().ID(tsLocalVar("response")).Dot("arrayBuffer").Call()).Semicolon())
			mg.Add(tsg.NewStatement().Var(tsLocalVar("responseData")).Colon().ID(responseTypeName).Op("=").ID("Msgpack").Dot("decode").Call(tsg.NewStatement().ID("new Uint8Array").Call(tsg.NewStatement().ID(tsLocalVar("buf")))).Op("as").ID(responseTypeName).Semicolon())
		case content.KindCBOR:
			mg.Add(tsg.NewStatement().Const(tsLocalVar("buf")).Op("=").Await(tsg.NewStatement().ID(tsLocalVar("response")).Dot("arrayBuffer").Call()).Semicolon())
			mg.Add(tsg.NewStatement().Var(tsLocalVar("responseData")).Colon().ID(responseTypeName).Op("=").ID("Cbor").Dot("decode").Call(tsg.NewStatement().ID("new Uint8Array").Call(tsg.NewStatement().ID(tsLocalVar("buf")))).Op("as").ID(responseTypeName).Semicolon())
		case content.KindYAML:
			mg.Add(tsg.NewStatement().Const(tsLocalVar("text")).Op("=").Await(tsg.NewStatement().ID(tsLocalVar("response")).Dot("text").Call()).Semicolon())
			mg.Add(tsg.NewStatement().Var(tsLocalVar("responseData")).Colon().ID(responseTypeName).Op("=").ID("YAML").Dot("parse").Call(tsg.NewStatement().ID(tsLocalVar("text"))).Op("as").ID(responseTypeName).Semicolon())
		default:
			mg.Add(tsg.NewStatement().Var(tsLocalVar("responseData")).Colon().ID(responseTypeName).Op("=").Await(tsg.NewStatement().ID(tsLocalVar("response")).Dot("json").Call()).Op("as").ID(responseTypeName).Semicolon())
		}
	}
	r.renderHTTPResponseMergeHeadersAndCookies(mg, contract, method, results, responseTypeName)
	switch {
	case len(results) == 1 && (model.IsAnnotationSet(r.project, contract, method, nil, model.TagHTTPEnableInlineSingle) || r.resultHasJSONInline(method, results[0])):
		mg.Return(tsg.NewStatement().ID(tsLocalVar("responseData")))
	case len(results) == 1:
		res := results[0]
		dataRef := tsg.NewStatement().ID(tsLocalVar("responseData"))
		if res.NumberOfPointers > 0 {
			mg.Return(dataRef.OptionalChain(tsSafeName(res.Name)).Op("??").ID("null"))
		} else {
			mg.Return(dataRef.Dot(tsSafeName(res.Name)))
		}
	case r.responseHasAnyInline(method, results):
		mg.Return(tsg.NewStatement().ID(tsLocalVar("responseData")))
	default:
		returnObj := tsg.NewStatement()
		returnObj.Values(func(rg *tsg.Group) {
			for _, ret := range results {
				dataRef := tsg.NewStatement().ID(tsLocalVar("responseData"))
				var val *tsg.Statement
				if ret.NumberOfPointers > 0 {
					val = dataRef.OptionalChain(tsSafeName(ret.Name)).Op("??").ID("null")
				} else {
					val = dataRef.Dot(tsSafeName(ret.Name))
				}
				rg.Add(tsg.NewStatement().ID(tsSafeName(ret.Name)).Colon().Add(val))
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
			pg.Add(tsg.NewStatement().ID("res").Colon().ID("Response"))
			pg.Add(tsg.NewStatement().ID("name").Colon().ID(typeString))
		})
		getCookieBlock := func(bg *tsg.Group) {
			hStmt := tsg.NewStatement().Const("h").Op("=").ID("res").Dot("headers").Dot("get").Call(tsg.NewStatement().Lit("Set-Cookie")).Op("??").Lit("")
			bg.Add(hStmt.Semicolon())
			forOfStmt := tsg.NewStatement().ForOf("part", "h.split(';')", func(fg *tsg.Group) {
				trimmedStmt := tsg.NewStatement().Const("trimmed").Op("=").ID("part").Dot("trim").Call()
				fg.Add(trimmedStmt.Semicolon())
				eqStmt := tsg.NewStatement().Const("eq").Op("=").ID("trimmed").Dot("indexOf").Call(tsg.NewStatement().Lit("="))
				fg.Add(eqStmt.Semicolon())
				ifContinue := tsg.NewStatement().If(tsg.NewStatement().ID("eq").Op("<").Lit(0), func(ig *tsg.Group) {
					ig.Add(tsg.NewStatement().ID("continue").Semicolon())
				})
				fg.Add(ifContinue)
				kStmt := tsg.NewStatement().Const("k").Op("=").ID("trimmed").Dot("slice").Call(tsg.NewStatement().Lit(0), tsg.NewStatement().ID("eq"))
				fg.Add(kStmt.Semicolon())
				vStmt := tsg.NewStatement().Const("v").Op("=").ID("trimmed").Dot("slice").Call(tsg.NewStatement().ID("eq").Op("+").Lit(1))
				fg.Add(vStmt.Semicolon())
				ifReturn := tsg.NewStatement().If(tsg.NewStatement().ID("k").Op("===").ID("name"), func(ig *tsg.Group) {
					ig.Add(tsg.NewStatement().Return(tsg.NewStatement().ID("decodeURIComponent").Call(tsg.NewStatement().ID("v"))).Semicolon())
				})
				fg.Add(ifReturn)
			})
			bg.Add(forOfStmt.Semicolon())
			bg.Add(tsg.NewStatement().Return(tsg.NewStatement().ID("null")).Semicolon())
		}
		getCookieStmt := tsg.NewStatement().Const("getResponseCookie").Op("=").Add(paramStmt).Colon().ID(typeString).Op("|").ID("null").Op("=>").Block(getCookieBlock)
		mg.Add(getCookieStmt.Semicolon())
	}

	mg.Add(tsg.NewStatement().Const("mergedResult").Colon().ID(responseTypeName).Op("=").Values(nil).Op("as").ID(responseTypeName).Semicolon())
	for _, ret := range results {
		_, fromHeaderOrCookieOnly := excludeFromBody[ret.Name]
		headerName, hasHeader := headerMap[ret.Name]
		cookieName, hasCookie := cookieMap[ret.Name]

		if !fromHeaderOrCookieOnly {
			mg.Add(tsg.NewStatement().Assign(
				tsg.NewStatement().ID("mergedResult").Dot(tsSafeName(ret.Name)),
				tsg.NewStatement().ID(tsLocalVar("responseData")).OptionalChain(tsSafeName(ret.Name)),
			).Semicolon())
		}

		if hasHeader || hasCookie {
			var transportSrc *tsg.Statement
			if hasHeader {
				transportSrc = tsg.NewStatement().ID(tsLocalVar("response")).Dot("headers").Dot("get").Call(tsg.NewStatement().Lit(headerName))
			} else {
				transportSrc = tsg.NewStatement().ID("getResponseCookie").Call(tsg.NewStatement().ID(tsLocalVar("response")), tsg.NewStatement().Lit(cookieName))
			}
			// Header/cookie всегда перекрывает body, включая пустое значение (как Go Header.Get).
			mg.Add(tsg.NewStatement().Assign(
				tsg.NewStatement().ID("mergedResult").Dot(tsSafeName(ret.Name)),
				r.parseTransportValueExpr(transportSrc, ret),
			).Semicolon())
		}
	}
	mg.Add(tsg.NewStatement().Assign(tsg.NewStatement().ID(tsLocalVar("responseData")), tsg.NewStatement().ID("mergedResult")).Semicolon())
}

func (r *ClientRenderer) resultHasJSONInline(method *model.Method, v *model.Variable) (ok bool) {

	if method == nil {
		return false
	}
	return tags.HasJSONInline(method.Annotations, v.Name)
}

func (r *ClientRenderer) responseHasAnyInline(method *model.Method, results []*model.Variable) (ok bool) {

	for _, v := range results {
		if r.resultHasJSONInline(method, v) {
			return true
		}
	}
	return false
}

// tsRPCSingleResultExpr — доступ к единственному результату JSON-RPC после cast в response type.
func (r *ClientRenderer) tsRPCSingleResultExpr(contract *model.Contract, method *model.Method, ret *model.Variable) *tsg.Statement {

	if model.IsAnnotationSet(r.project, contract, method, nil, model.TagHTTPEnableInlineSingle) {
		return tsg.NewStatement().ID(nameResult)
	}
	if model.ResultFieldEmbedded(r.project, contract, method, ret) {
		return tsg.NewStatement().ID(nameResult)
	}

	return tsg.NewStatement().ID(nameResult).Dot(tsSafeName(ret.Name))
}
