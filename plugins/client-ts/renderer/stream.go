// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"tgp/internal/model"
	"tgp/plugins/client-ts/tsg"
)

func (r *ClientRenderer) renderStreamSupport(grp *tsg.Group, contract *model.Contract) {

	needWSOut, needWSResult, needSSE := r.streamHelperNeeds(contract)
	if !needWSOut && !needWSResult && !needSSE {
		return
	}
	if needWSOut || needWSResult {
		grp.Add(renderStreamHelperWebSocketURL())
		grp.Line()
		grp.Add(renderStreamHelperOpenWebSocket())
		grp.Line()
	}
	if needWSOut {
		grp.Add(renderStreamHelperWebSocket())
		grp.Line()
	}
	if needWSResult {
		grp.Add(renderStreamHelperWebSocketResult())
		grp.Line()
	}
	if needSSE {
		grp.Add(renderStreamHelperSSE())
		grp.Line()
	}
}

func (r *ClientRenderer) streamHelperNeeds(contract *model.Contract) (needWSOut bool, needWSResult bool, needSSE bool) {

	for _, method := range contract.Methods {
		if model.MethodIsSSE(r.project, contract, method) {
			needSSE = true
		}
		if !model.MethodIsWS(r.project, contract, method) {
			continue
		}
		switch model.MethodStreamMode(r.project, contract, method) {
		case model.StreamModeClient:
			needWSResult = true
		case model.StreamModeServer, model.StreamModeBidi:
			needWSOut = true
		}
	}
	return
}

// appendStreamExchangeTypes добавляет Response* exchange-типы, на которые ссылается stream-клиент.
func (r *ClientRenderer) appendStreamExchangeTypes(contract *model.Contract, method *model.Method, exchangeTypes *[]string, seenTypes map[string]bool) {

	if !model.MethodIsStream(r.project, contract, method) {
		return
	}
	if model.MethodStreamMode(r.project, contract, method) != model.StreamModeClient {
		return
	}
	if len(r.resultsWithoutChannel(method)) <= 1 {
		return
	}
	responseType := r.responseTypeName(contract, method)
	if seenTypes[responseType] {
		return
	}
	*exchangeTypes = append(*exchangeTypes, responseType)
	seenTypes[responseType] = true
}

func (r *ClientRenderer) renderStreamMethods(grp *tsg.Group, contract *model.Contract, method *model.Method) {

	if model.MethodIsWS(r.project, contract, method) {
		grp.Add(r.renderStreamMethod(contract, method, method.Name, true))
		grp.Line()
	}
	if model.MethodIsSSE(r.project, contract, method) {
		name := method.Name
		if model.MethodIsWS(r.project, contract, method) {
			name += "SSE"
		}
		grp.Add(r.renderStreamMethod(contract, method, name, false))
		grp.Line()
	}
}

func (r *ClientRenderer) renderStreamMethod(contract *model.Contract, method *model.Method, name string, websocket bool) (stmt *tsg.Statement) {

	mode := model.MethodStreamMode(r.project, contract, method)
	clientArgs := r.streamClientArgs(contract, method)
	bodyArgs := r.argsForExchangeRequest(contract, method)
	params := r.streamTSParams(contract, method, clientArgs, mode)
	paramsObject := r.streamTSParamsObject(bodyArgs)
	methodName := model.JSONRPCWireMethod(contract.Name, method.Name)
	rawPath := model.MethodSSEPath(r.project, contract, method)
	if websocket {
		rawPath = model.ContractWSPath(r.project, contract)
	}
	pathExpr := r.streamTSPathExpr(contract, method, rawPath)
	queryExpr := r.streamTSQueryExpr(contract, method, websocket)
	headersExpr := r.streamTSHeadersExpr(contract, method)
	methodNameTS := r.lcName(name)

	stmt = tsg.NewStatement()
	stmt.Public()
	if mode == model.StreamModeClient {
		in, _, _ := model.MethodStreamInChan(r.project, method)
		resultType := r.streamTSResultType(contract, method)
		returnType := tsg.NewStatement().ID("Promise").Generic(resultType)
		stmt.AsyncMethod(methodNameTS, params, returnType, func(mg *tsg.Group) {
			addStreamMethodLocals(mg, paramsObject, pathExpr, queryExpr, nil)
			mg.Return(tsg.NewStatement().This().Dot("streamWebSocketResult").Generic(resultType).Call(
				tsg.NewStatement().ID("streamPath"),
				tsg.NewStatement().Lit(methodName),
				tsg.NewStatement().ID("params"),
				tsg.NewStatement().ID(tsSafeName(in.Name)),
				tsg.NewStatement().ID("query"),
				tsg.NewStatement().ID("signal"),
				tsg.NewStatement().ID("onReady"),
			))
		})
		return stmt
	}

	_, element, _ := model.MethodStreamOutChan(r.project, method)
	outType := r.streamTSType(contract, method, element, false)
	inputArg := typeUndefined
	if mode == model.StreamModeBidi {
		in, _, _ := model.MethodStreamInChan(r.project, method)
		inputArg = tsSafeName(in.Name)
	}
	returnType := tsg.NewStatement().ID("AsyncGenerator").Generic(outType)
	stmt.AsyncGeneratorMethod(methodNameTS, params, returnType, func(mg *tsg.Group) {
		if websocket {
			addStreamMethodLocals(mg, paramsObject, pathExpr, queryExpr, nil)
			mg.ForAwait("item", tsg.NewStatement().This().Dot("streamWebSocket").Generic(outType).Call(
				tsg.NewStatement().ID("streamPath"),
				tsg.NewStatement().Lit(methodName),
				tsg.NewStatement().ID("params"),
				tsg.NewStatement().ID(inputArg),
				tsg.NewStatement().ID("query"),
				tsg.NewStatement().ID("signal"),
				tsg.NewStatement().ID("onReady"),
			), func(fg *tsg.Group) {
				fg.Yield(tsg.NewStatement().ID("item"))
			})
			return
		}
		addStreamMethodLocals(mg, paramsObject, pathExpr, queryExpr, headersExpr)
		mg.ForAwait("item", tsg.NewStatement().This().Dot("streamSSE").Generic(outType).Call(
			tsg.NewStatement().ID("streamPath"),
			tsg.NewStatement().Lit(methodName),
			tsg.NewStatement().ID("params"),
			tsg.NewStatement().ID("query"),
			tsg.NewStatement().ID("extraHeaders"),
			tsg.NewStatement().ID("signal"),
			tsg.NewStatement().ID("onReady"),
		), func(fg *tsg.Group) {
			fg.Yield(tsg.NewStatement().ID("item"))
		})
	})
	return stmt
}

func addStreamMethodLocals(mg *tsg.Group, paramsObject *tsg.Statement, pathExpr *tsg.Statement, queryExpr *tsg.Statement, headersExpr *tsg.Statement) {

	mg.Add(tsg.NewStatement().Const("params").Op("=").Add(paramsObject).Semicolon())
	mg.Add(tsg.NewStatement().Const("streamPath").Op("=").Add(pathExpr).Semicolon())
	mg.Add(tsg.NewStatement().Const("query").Op("=").Add(queryExpr).Semicolon())
	if headersExpr == nil {
		return
	}
	mg.Add(tsg.NewStatement().Const("extraHeaders").Op("=").Add(headersExpr).Semicolon())
}

func (r *ClientRenderer) streamClientArgs(contract *model.Contract, method *model.Method) (out []*model.Variable) {

	for _, arg := range r.argsForClient(contract, method) {
		if model.TypeRefIsChan(r.project, &arg.TypeRef) {
			continue
		}
		out = append(out, arg)
	}
	return out
}

func (r *ClientRenderer) streamTSPathExpr(contract *model.Contract, method *model.Method, rawPath string) (expr *tsg.Statement) {

	expr = tsg.NewStatement().Lit(rawPath)
	for argName, segment := range model.StreamPathParamArgMap(r.project, contract, method) {
		expr = tsg.NewStatement().Paren(expr).Dot("split").Call(tsg.NewStatement().Lit(":" + segment)).Dot("join").Call(
			tsg.NewStatement().ID("encodeURIComponent").Call(
				tsg.NewStatement().ID("String").Call(tsg.NewStatement().ID(tsSafeName(argName))),
			),
		)
	}
	return expr
}

func (r *ClientRenderer) streamTSQueryExpr(contract *model.Contract, method *model.Method, websocket bool) (expr *tsg.Statement) {

	implicit := model.HTTPImplicitArgSet(model.BuildHTTPArgMappings(r.project, contract, method))
	fields := make([]*tsg.Statement, 0)
	for argName, key := range model.HTTPArgQueryMapForRequest(r.project, contract, method) {
		if _, skip := implicit[argName]; skip {
			continue
		}
		if r.clientArgByName(contract, method, argName) == nil {
			continue
		}
		fields = append(fields, streamStringField(key, argName))
	}
	if websocket {
		// Browser WebSocket cannot set custom headers/cookies; mirror them as query for WS upgrade fallback.
		for argName, headerName := range model.HTTPHeaderArgMapForRequest(r.project, contract, method) {
			if _, skip := implicit[argName]; skip {
				continue
			}
			if r.clientArgByName(contract, method, argName) == nil {
				continue
			}
			fields = append(fields, streamStringField(headerName, argName))
		}
		for argName, cookieName := range model.HTTPCookieArgMapForRequest(r.project, contract, method) {
			if _, skip := implicit[argName]; skip {
				continue
			}
			if r.clientArgByName(contract, method, argName) == nil {
				continue
			}
			fields = append(fields, streamStringField(cookieName, argName))
		}
	}
	if len(fields) == 0 {
		return tsg.NewStatement().ID(typeUndefined)
	}
	return streamObjectLit(fields...)
}

func (r *ClientRenderer) streamTSHeadersExpr(contract *model.Contract, method *model.Method) (expr *tsg.Statement) {

	implicit := model.HTTPImplicitArgSet(model.BuildHTTPArgMappings(r.project, contract, method))
	fields := make([]*tsg.Statement, 0)
	for argName, headerName := range model.HTTPHeaderArgMapForRequest(r.project, contract, method) {
		if _, skip := implicit[argName]; skip {
			continue
		}
		if r.clientArgByName(contract, method, argName) == nil {
			continue
		}
		fields = append(fields, streamStringField(headerName, argName))
	}
	cookieParts := make([]*tsg.Statement, 0)
	for argName, cookieName := range model.HTTPCookieArgMapForRequest(r.project, contract, method) {
		if _, skip := implicit[argName]; skip {
			continue
		}
		if r.clientArgByName(contract, method, argName) == nil {
			continue
		}
		cookieParts = append(cookieParts, tsg.NewStatement().Lit(cookieName).Op("+").Lit("=").Op("+").ID("encodeURIComponent").Call(
			tsg.NewStatement().ID("String").Call(tsg.NewStatement().ID(tsSafeName(argName))),
		))
	}
	if len(cookieParts) > 0 {
		fields = append(fields, tsg.NewStatement().ObjectField("Cookie", tsg.NewStatement().ArrayLit(cookieParts...).Dot("join").Call(tsg.NewStatement().Lit("; "))))
	}
	if len(fields) == 0 {
		return tsg.NewStatement().ID(typeUndefined)
	}
	return streamObjectLit(fields...)
}

func streamStringField(key string, argName string) (field *tsg.Statement) {

	return tsg.NewStatement().ObjectField(key, tsg.NewStatement().ID("String").Call(tsg.NewStatement().ID(tsSafeName(argName))))
}

func streamObjectLit(fields ...*tsg.Statement) (expr *tsg.Statement) {

	expr = tsg.NewStatement()
	expr.Values(func(vg *tsg.Group) {
		for _, field := range fields {
			vg.Add(field)
		}
	})
	return expr
}

func (r *ClientRenderer) streamTSParams(contract *model.Contract, method *model.Method, args []*model.Variable, mode string) (params *tsg.Statement) {

	params = tsg.NewStatement()
	params.Params(func(pg *tsg.Group) {
		for _, arg := range args {
			pg.Add(tsg.NewStatement().ID(tsSafeName(arg.Name)).Add(tsg.TypeFromString(": " + r.streamTSType(contract, method, &arg.TypeRef, true))))
		}
		if mode == model.StreamModeClient || mode == model.StreamModeBidi {
			in, element, _ := model.MethodStreamInChan(r.project, method)
			pg.Add(tsg.NewStatement().ID(tsSafeName(in.Name)).Add(tsg.TypeFromString(": AsyncIterable<" + r.streamTSType(contract, method, element, true) + ">")))
		}
		pg.Add(tsg.NewStatement().ID("signal").Optional().Add(tsg.TypeFromString(": AbortSignal")))
		pg.Add(tsg.NewStatement().ID("onReady").Optional().Add(tsg.TypeFromString(": () => void")))
	})
	return params
}

func (r *ClientRenderer) streamTSParamsObject(args []*model.Variable) (object *tsg.Statement) {

	if len(args) == 0 {
		return tsg.NewStatement().ObjectLiteral(nil)
	}
	object = tsg.NewStatement()
	object.Values(func(vg *tsg.Group) {
		for _, arg := range args {
			name := tsSafeName(arg.Name)
			vg.Add(tsg.NewStatement().ID(name).Colon().ID(name))
		}
	})
	return object
}

func (r *ClientRenderer) streamTSType(contract *model.Contract, method *model.Method, typeRef *model.TypeRef, input bool) (typeName string) {

	if typeRef == nil {
		return "unknown"
	}
	variable := &model.Variable{Name: "item", TypeRef: *typeRef}
	return r.walkVariable(variable.Name, contract.PkgPath, variable, method.Annotations, input).typeLink()
}

func (r *ClientRenderer) streamTSResultType(contract *model.Contract, method *model.Method) (typeName string) {

	results := r.resultsWithoutChannel(method)
	if len(results) == 0 {
		return "void"
	}
	if len(results) == 1 {
		return r.streamTSType(contract, method, &results[0].TypeRef, false)
	}
	return r.responseTypeName(contract, method)
}

func (r *ClientRenderer) resultsWithoutChannel(method *model.Method) (out []*model.Variable) {

	for _, result := range r.resultsWithoutError(method) {
		if !model.TypeRefIsChan(r.project, &result.TypeRef) {
			out = append(out, result)
		}
	}
	return
}

func (r *ClientRenderer) clientArgByName(contract *model.Contract, method *model.Method, argName string) (variable *model.Variable) {

	for _, arg := range r.argsForClient(contract, method) {
		if arg.Name == argName {
			return arg
		}
	}
	return nil
}
