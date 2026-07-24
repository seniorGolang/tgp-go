// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"context"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/common"
	"tgp/internal/generated"
	"tgp/internal/model"
)

const packageWebsocket = "github.com/gorilla/websocket"

func (r *ClientRenderer) RenderStreamHelpers() (err error) {

	if !r.HasWS() && !r.HasSSE() {
		return
	}
	outDir := r.outDir
	srcFile := NewSrcFile(filepath.Base(outDir))
	srcFile.PackageComment(generated.ByToolGateway)
	srcFile.ImportName(PackageURL, "url")
	srcFile.ImportName(r.getPackageJSON(nil), "json")
	r.renderStreamHelpers(&srcFile)
	return srcFile.Save(path.Join(outDir, "stream.go"))
}

func (r *ClientRenderer) renderStreamHelpers(srcFile *GoFile) {

	jsonPkg := r.getPackageJSON(nil)
	srcFile.Type().Id("rpcError").Struct(
		Id("Code").Int().Tag(map[string]string{"json": "code"}),
		Id("Message").String().Tag(map[string]string{"json": "message"}),
		Id("Data").Any().Tag(map[string]string{"json": "data,omitempty"}),
	)
	srcFile.Type().Id("rpcMessage").Struct(
		Id("ID").Qual(jsonPkg, "RawMessage").Tag(map[string]string{"json": "id,omitempty"}),
		Id("Version").String().Tag(map[string]string{"json": "jsonrpc"}),
		Id("Method").String().Tag(map[string]string{"json": "method,omitempty"}),
		Id("Error").Op("*").Id("rpcError").Tag(map[string]string{"json": "error,omitempty"}),
		Id("Params").Qual(jsonPkg, "RawMessage").Tag(map[string]string{"json": "params,omitempty"}),
		Id("Result").Qual(jsonPkg, "RawMessage").Tag(map[string]string{"json": "result,omitempty"}),
	)
	srcFile.Type().Id("rpcStreamChunk").Struct(
		Id("ID").Qual(jsonPkg, "RawMessage").Tag(map[string]string{"json": "id"}),
		Id("Seq").Int64().Tag(map[string]string{"json": "seq"}),
		Id("Item").Qual(jsonPkg, "RawMessage").Tag(map[string]string{"json": "item"}),
	)
	srcFile.Type().Id("streamChunkParams").Struct(
		Id("ID").Qual(jsonPkg, "RawMessage").Tag(map[string]string{"json": "id"}),
		Id("Item").Any().Tag(map[string]string{"json": "item"}),
	)
	srcFile.Type().Id("streamEndParams").Struct(
		Id("ID").Qual(jsonPkg, "RawMessage").Tag(map[string]string{"json": "id"}),
	)
	srcFile.Func().Params(Id("cli").Op("*").Id("Client")).Id("wsURL").
		Params(Id("streamPath").String(), Id("query").Qual(PackageURL, "Values")).
		Params(Id("streamURL").String()).Block(
		Id("streamURL").Op("=").Id("cli").Dot("endpoint"),
		List(Id("parsed"), Id("parseErr")).Op(":=").Qual(PackageURL, "Parse").Call(Id("cli").Dot("endpoint")),
		If(Id("parseErr").Op("!=").Nil()).Block(Return()),
		Switch(Id("parsed").Dot("Scheme")).Block(
			Case(Lit("https")).Block(Id("parsed").Dot("Scheme").Op("=").Lit("wss")),
			Default().Block(Id("parsed").Dot("Scheme").Op("=").Lit("ws")),
		),
		Id("parsed").Dot("Path").Op("=").Id("streamPath"),
		If(Id("query").Op("!=").Nil()).Block(
			Id("parsed").Dot("RawQuery").Op("=").Id("query").Dot("Encode").Call(),
		).Else().Block(
			Id("parsed").Dot("RawQuery").Op("=").Lit(""),
		),
		Return(Id("parsed").Dot("String").Call()),
	)
}

func (r *ClientRenderer) streamClientMethod(ctx context.Context, contract *model.Contract, method *model.Method) (c Code) {

	mode := model.MethodStreamMode(r.project, contract, method)
	if model.MethodIsWS(r.project, contract, method) {
		return r.wsClientMethod(ctx, contract, method, method.Name, mode)
	}
	return r.sseClientMethod(ctx, contract, method, method.Name)
}

func (r *ClientRenderer) streamSSEAlternateMethod(ctx context.Context, contract *model.Contract, method *model.Method) (c Code) {

	return r.sseClientMethod(ctx, contract, method, method.Name+"SSE")
}

func (r *ClientRenderer) wsClientMethod(ctx context.Context, contract *model.Contract, method *model.Method, name, mode string) (c Code) {

	if mode == model.StreamModeClient {
		return r.wsClientInputMethod(ctx, contract, method, name)
	}
	return r.wsOutputMethod(ctx, contract, method, name, true)
}

func (r *ClientRenderer) emitStreamPathAndQuery(bg *Group, ctx context.Context, contract *model.Contract, method *model.Method, rawPath string) {

	bg.Id("streamPath").Op(":=").Lit(rawPath)
	for argName, segment := range common.SortedPairs(model.StreamPathParamArgMap(r.project, contract, method)) {
		bg.Id("streamPath").Op("=").Qual(PackageStrings, "ReplaceAll").Call(
			Id("streamPath"),
			Lit(":"+segment),
			Qual(PackageURL, "PathEscape").Call(r.varToString(ctx, r.argByName(method, argName))),
		)
	}
	bg.Id("query").Op(":=").Qual(PackageURL, "Values").Values()
	implicit := model.HTTPImplicitArgSet(model.BuildHTTPArgMappings(r.project, contract, method))
	for argName, key := range common.SortedPairs(model.HTTPArgQueryMapForRequest(r.project, contract, method)) {
		if _, skip := implicit[argName]; skip {
			continue
		}
		if r.clientArgByName(contract, method, argName) == nil {
			continue
		}
		bg.Id("query").Dot("Set").Call(Lit(key), r.varToString(ctx, r.argByName(method, argName)))
	}
}

func (r *ClientRenderer) emitStreamDialHeader(bg *Group, ctx context.Context, contract *model.Contract, method *model.Method) {

	bg.Id("dialHeader").Op(":=").Qual(PackageHttp, "Header").Values()
	implicit := model.HTTPImplicitArgSet(model.BuildHTTPArgMappings(r.project, contract, method))
	for argName, headerName := range common.SortedPairs(model.HTTPHeaderArgMapForRequest(r.project, contract, method)) {
		if _, skip := implicit[argName]; skip {
			continue
		}
		if r.clientArgByName(contract, method, argName) == nil {
			continue
		}
		bg.Id("dialHeader").Dot("Set").Call(Lit(headerName), r.varToString(ctx, r.argByName(method, argName)))
	}
	for argName, cookieName := range common.SortedPairs(model.HTTPCookieArgMapForRequest(r.project, contract, method)) {
		if _, skip := implicit[argName]; skip {
			continue
		}
		if r.clientArgByName(contract, method, argName) == nil {
			continue
		}
		bg.Id("dialHeader").Dot("Add").Call(Lit("Cookie"), Lit(cookieName+"=").Op("+").Add(r.varToString(ctx, r.argByName(method, argName))))
	}
}

func (r *ClientRenderer) emitStreamRequestHeaders(bg *Group, ctx context.Context, contract *model.Contract, method *model.Method, requestVar string) {

	implicit := model.HTTPImplicitArgSet(model.BuildHTTPArgMappings(r.project, contract, method))
	for argName, headerName := range common.SortedPairs(model.HTTPHeaderArgMapForRequest(r.project, contract, method)) {
		if _, skip := implicit[argName]; skip {
			continue
		}
		if r.clientArgByName(contract, method, argName) == nil {
			continue
		}
		bg.Id(requestVar).Dot("Header").Dot("Set").Call(Lit(headerName), r.varToString(ctx, r.argByName(method, argName)))
	}
	for argName, cookieName := range common.SortedPairs(model.HTTPCookieArgMapForRequest(r.project, contract, method)) {
		if _, skip := implicit[argName]; skip {
			continue
		}
		if r.clientArgByName(contract, method, argName) == nil {
			continue
		}
		bg.Id(requestVar).Dot("AddCookie").Call(Op("&").Qual(PackageHttp, "Cookie").Values(Dict{
			Id("Name"):  Lit(cookieName),
			Id("Value"): r.varToString(ctx, r.argByName(method, argName)),
		}))
	}
}

func (r *ClientRenderer) wsOutputMethod(ctx context.Context, contract *model.Contract, method *model.Method, name string, withInput bool) (c Code) {

	out, element, ok := model.MethodStreamOutChan(r.project, method)
	if !ok {
		return Empty()
	}
	jsonPkg := r.getPackageJSON(contract)
	return Func().Params(Id("cli").Op("*").Id("Client" + contract.Name)).Id(name).
		Params(r.streamMethodParams(ctx, contract, method, withInput)).
		Params(r.funcDefinitionParams(ctx, r.streamClientResults(method))).BlockFunc(func(bg *Group) {
		bg.Var().Id("params").Qual(jsonPkg, "RawMessage")
		bg.If(List(Id("params"), Err()).Op("=").Qual(jsonPkg, "Marshal").Call(Id(r.requestStructName(contract, method)).Values(DictFunc(func(d Dict) {
			for _, arg := range r.argsForExchangeRequest(contract, method) {
				d[Id(ToCamel(arg.Name))] = Id(ToLowerCamel(arg.Name))
			}
		}))).Op(";").Err().Op("!=").Nil()).Block(Return())
		r.emitStreamPathAndQuery(bg, ctx, contract, method, model.ContractWSPath(r.project, contract))
		r.emitStreamDialHeader(bg, ctx, contract, method)
		bg.Var().Id("conn").Op("*").Qual(packageWebsocket, "Conn")
		bg.If(List(Id("conn"), Id("_"), Err()).Op("=").Qual(packageWebsocket, "DefaultDialer").Dot("DialContext").Call(Id(_ctx_), Id("cli").Dot("wsURL").Call(Id("streamPath"), Id("query")), Id("dialHeader")).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.Id("streamID").Op(":=").Qual(jsonPkg, "RawMessage").Call(Qual(PackageFmt, "Appendf").Call(Nil(), Lit("%q"), Qual(PackageUUID, "NewString").Call()))
		bg.If(Err().Op("=").Id("conn").Dot("WriteJSON").Call(Id("rpcMessage").Values(Dict{
			Id("ID"): Id("streamID"), Id("Version"): Lit("2.0"), Id("Method"): Lit(r.jsonRPCWireMethod(contract, method)), Id("Params"): Id("params"),
		})).Op(";").Err().Op("!=").Nil()).Block(
			Id("_").Op("=").Id("conn").Dot("Close").Call(),
			Return(),
		)
		bg.Id("items").Op(":=").Make(Chan().Add(r.fieldTypeFromTypeRef(ctx, element, false)), Lit(32))
		bg.Id(ToLowerCamel(out.Name)).Op("=").Id("items")
		if withInput {
			if in, _, hasIn := model.MethodStreamInChan(r.project, method); hasIn {
				bg.Go().Func().Params().Block(
					Defer().Func().Params().Block(
						List(Id("endParams"), Id("_")).Op(":=").Qual(jsonPkg, "Marshal").Call(Id("streamEndParams").Values(Dict{Id("ID"): Id("streamID")})),
						Id("_").Op("=").Id("conn").Dot("WriteJSON").Call(Id("rpcMessage").Values(Dict{
							Id("Version"): Lit("2.0"), Id("Method"): Lit(model.JSONRPCStreamEndMethod), Id("Params"): Id("endParams"),
						})),
					).Call(),
					For(Id("item").Op(":=").Range().Id(ToLowerCamel(in.Name))).Block(
						List(Id("chunkParams"), Id("chunkErr")).Op(":=").Qual(jsonPkg, "Marshal").Call(Id("streamChunkParams").Values(Dict{Id("ID"): Id("streamID"), Id("Item"): Id("item")})),
						If(Id("chunkErr").Op("!=").Nil()).Block(Return()),
						If(Err().Op(":=").Id("conn").Dot("WriteJSON").Call(Id("rpcMessage").Values(Dict{
							Id("Version"): Lit("2.0"), Id("Method"): Lit(model.JSONRPCStreamMethod), Id("Params"): Id("chunkParams"),
						})).Op(";").Err().Op("!=").Nil()).Block(Return()),
					),
				).Call()
			}
		}
		bg.Go().Func().Params().BlockFunc(func(gg *Group) {
			gg.Defer().Func().Params().Block(
				Id("_").Op("=").Id("conn").Dot("Close").Call(),
				Close(Id("items")),
			).Call()
			gg.For().Block(
				Var().Id("message").Id("rpcMessage"),
				If(Err().Op(":=").Id("conn").Dot("ReadJSON").Call(Op("&").Id("message")).Op(";").Err().Op("!=").Nil()).Block(Return()),
				If(Id("message").Dot("Method").Op("==").Lit(model.JSONRPCStreamMethod)).Block(
					Var().Id("chunk").Id("rpcStreamChunk"),
					If(Err().Op(":=").Qual(jsonPkg, "Unmarshal").Call(Id("message").Dot("Params"), Op("&").Id("chunk")).Op(";").Err().Op("!=").Nil()).Block(Continue()),
					If(String().Call(Id("chunk").Dot("ID")).Op("!=").String().Call(Id("streamID"))).Block(Continue()),
					Var().Id("item").Add(r.fieldTypeFromTypeRef(ctx, element, false)),
					If(Err().Op(":=").Qual(jsonPkg, "Unmarshal").Call(Id("chunk").Dot("Item"), Op("&").Id("item")).Op(";").Err().Op("!=").Nil()).Block(Continue()),
					Select().Block(
						Case(Id("items").Op("<-").Id("item")),
						Case(Op("<-").Id(_ctx_).Dot("Done").Call()).Block(Return()),
					),
					Continue(),
				),
				If(String().Call(Id("message").Dot("ID")).Op("==").String().Call(Id("streamID"))).Block(Return()),
			)
		}).Call()
		bg.Return()
	})
}

func (r *ClientRenderer) wsClientInputMethod(ctx context.Context, contract *model.Contract, method *model.Method, name string) (c Code) {

	in, _, ok := model.MethodStreamInChan(r.project, method)
	if !ok {
		return Empty()
	}
	jsonPkg := r.getPackageJSON(contract)
	return Func().Params(Id("cli").Op("*").Id("Client" + contract.Name)).Id(name).
		Params(r.streamMethodParams(ctx, contract, method, true)).
		Params(r.funcDefinitionParams(ctx, method.Results)).BlockFunc(func(bg *Group) {
		bg.Var().Id("params").Qual(jsonPkg, "RawMessage")
		bg.If(List(Id("params"), Err()).Op("=").Qual(jsonPkg, "Marshal").Call(Id(r.requestStructName(contract, method)).Values(DictFunc(func(d Dict) {
			for _, arg := range r.argsForExchangeRequest(contract, method) {
				d[Id(ToCamel(arg.Name))] = Id(ToLowerCamel(arg.Name))
			}
		}))).Op(";").Err().Op("!=").Nil()).Block(Return())
		r.emitStreamPathAndQuery(bg, ctx, contract, method, model.ContractWSPath(r.project, contract))
		r.emitStreamDialHeader(bg, ctx, contract, method)
		bg.Var().Id("conn").Op("*").Qual(packageWebsocket, "Conn")
		bg.If(List(Id("conn"), Id("_"), Err()).Op("=").Qual(packageWebsocket, "DefaultDialer").Dot("DialContext").Call(Id(_ctx_), Id("cli").Dot("wsURL").Call(Id("streamPath"), Id("query")), Id("dialHeader")).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.Defer().Func().Params().Block(Id("_").Op("=").Id("conn").Dot("Close").Call()).Call()
		bg.Id("streamID").Op(":=").Qual(jsonPkg, "RawMessage").Call(Qual(PackageFmt, "Appendf").Call(Nil(), Lit("%q"), Qual(PackageUUID, "NewString").Call()))
		bg.If(Err().Op("=").Id("conn").Dot("WriteJSON").Call(Id("rpcMessage").Values(Dict{Id("ID"): Id("streamID"), Id("Version"): Lit("2.0"), Id("Method"): Lit(r.jsonRPCWireMethod(contract, method)), Id("Params"): Id("params")})).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.For(Id("item").Op(":=").Range().Id(ToLowerCamel(in.Name))).Block(
			List(Id("chunkParams"), Id("chunkErr")).Op(":=").Qual(jsonPkg, "Marshal").Call(Id("streamChunkParams").Values(Dict{Id("ID"): Id("streamID"), Id("Item"): Id("item")})),
			If(Id("chunkErr").Op("!=").Nil()).Block(Err().Op("=").Id("chunkErr"), Return()),
			If(Err().Op("=").Id("conn").Dot("WriteJSON").Call(Id("rpcMessage").Values(Dict{Id("Version"): Lit("2.0"), Id("Method"): Lit(model.JSONRPCStreamMethod), Id("Params"): Id("chunkParams")})).Op(";").Err().Op("!=").Nil()).Block(Return()),
		)
		bg.List(Id("endParams"), Id("endErr")).Op(":=").Qual(jsonPkg, "Marshal").Call(Id("streamEndParams").Values(Dict{Id("ID"): Id("streamID")}))
		bg.If(Id("endErr").Op("!=").Nil()).Block(Err().Op("=").Id("endErr"), Return())
		bg.If(Err().Op("=").Id("conn").Dot("WriteJSON").Call(Id("rpcMessage").Values(Dict{Id("Version"): Lit("2.0"), Id("Method"): Lit(model.JSONRPCStreamEndMethod), Id("Params"): Id("endParams")})).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.For().Block(
			Var().Id("message").Id("rpcMessage"),
			If(Err().Op("=").Id("conn").Dot("ReadJSON").Call(Op("&").Id("message")).Op(";").Err().Op("!=").Nil()).Block(Return()),
			If(String().Call(Id("message").Dot("ID")).Op("!=").String().Call(Id("streamID"))).Block(Continue()),
			If(Id("message").Dot("Error").Op("!=").Nil()).Block(Err().Op("=").Qual(PackageFmt, "Errorf").Call(Lit("%d: %s"), Id("message").Dot("Error").Dot("Code"), Id("message").Dot("Error").Dot("Message")), Return()),
			r.streamFinalResult(ctx, contract, method, "message"),
		)
	})
}

func (r *ClientRenderer) sseClientMethod(ctx context.Context, contract *model.Contract, method *model.Method, name string) (c Code) {

	out, element, ok := model.MethodStreamOutChan(r.project, method)
	if !ok {
		return Empty()
	}
	jsonPkg := r.getPackageJSON(contract)
	return Func().Params(Id("cli").Op("*").Id("Client" + contract.Name)).Id(name).
		Params(r.streamMethodParams(ctx, contract, method, false)).
		Params(r.funcDefinitionParams(ctx, r.streamClientResults(method))).BlockFunc(func(bg *Group) {
		bg.Var().Id("params").Qual(jsonPkg, "RawMessage")
		bg.If(List(Id("params"), Err()).Op("=").Qual(jsonPkg, "Marshal").Call(Id(r.requestStructName(contract, method)).Values(DictFunc(func(d Dict) {
			for _, arg := range r.argsForExchangeRequest(contract, method) {
				d[Id(ToCamel(arg.Name))] = Id(ToLowerCamel(arg.Name))
			}
		}))).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.Id("streamID").Op(":=").Qual(jsonPkg, "RawMessage").Call(Qual(PackageFmt, "Appendf").Call(Nil(), Lit("%q"), Qual(PackageUUID, "NewString").Call()))
		bg.Id("body").Op(":=").Id("rpcMessage").Values(Dict{Id("ID"): Id("streamID"), Id("Version"): Lit("2.0"), Id("Method"): Lit(r.jsonRPCWireMethod(contract, method)), Id("Params"): Id("params")})
		bg.Var().Id("bodyBytes").Index().Byte()
		bg.If(List(Id("bodyBytes"), Err()).Op("=").Qual(jsonPkg, "Marshal").Call(Id("body")).Op(";").Err().Op("!=").Nil()).Block(Return())
		r.emitStreamPathAndQuery(bg, ctx, contract, method, model.MethodSSEPath(r.project, contract, method))
		bg.Id("sseURL").Op(":=").Id("cli").Dot("endpoint").Op("+").Id("streamPath")
		bg.If(Len(Id("query")).Op(">").Lit(0)).Block(
			Id("sseURL").Op("=").Id("sseURL").Op("+").Lit("?").Op("+").Id("query").Dot("Encode").Call(),
		)
		bg.Var().Id("request").Op("*").Qual(PackageHttp, "Request")
		bg.If(List(Id("request"), Err()).Op("=").Qual(PackageHttp, "NewRequestWithContext").Call(Id(_ctx_), Qual(PackageHttp, "MethodPost"), Id("sseURL"), Qual(PackageBytes, "NewBuffer").Call(Id("bodyBytes"))).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.Id("request").Dot("Header").Dot("Set").Call(Lit("Accept"), Lit("text/event-stream"))
		bg.Id("request").Dot("Header").Dot("Set").Call(Lit("Content-Type"), Lit("application/json"))
		r.emitStreamRequestHeaders(bg, ctx, contract, method, "request")
		bg.Var().Id("response").Op("*").Qual(PackageHttp, "Response")
		bg.If(List(Id("response"), Err()).Op("=").Id("cli").Dot("httpClient").Dot("Do").Call(Id("request")).Op(";").Err().Op("!=").Nil()).Block(Return())
		bg.If(Id("response").Dot("StatusCode").Op("!=").Qual(PackageHttp, "StatusOK")).Block(Id("_").Op("=").Id("response").Dot("Body").Dot("Close").Call(), Err().Op("=").Qual(PackageFmt, "Errorf").Call(Lit("SSE request failed: %s"), Id("response").Dot("Status")), Return())
		bg.Id("items").Op(":=").Make(Chan().Add(r.fieldTypeFromTypeRef(ctx, element, false)), Lit(32))
		bg.Id(ToLowerCamel(out.Name)).Op("=").Id("items")
		bg.Go().Func().Params().Block(
			Defer().Func().Params().Block(Id("_").Op("=").Id("response").Dot("Body").Dot("Close").Call(), Close(Id("items"))).Call(),
			Id("scanner").Op(":=").Qual("bufio", "NewScanner").Call(Id("response").Dot("Body")),
			For(Id("scanner").Dot("Scan").Call()).Block(
				Id("line").Op(":=").Id("scanner").Dot("Text").Call(),
				If(Op("!").Qual(PackageStrings, "HasPrefix").Call(Id("line"), Lit("data:"))).Block(Continue()),
				Var().Id("message").Id("rpcMessage"),
				If(Err().Op(":=").Qual(jsonPkg, "Unmarshal").Call(Index().Byte().Call(Qual(PackageStrings, "TrimSpace").Call(Id("line").Index(Lit(5), Empty()))), Op("&").Id("message")).Op(";").Err().Op("!=").Nil()).Block(Continue()),
				If(Id("message").Dot("Method").Op("!=").Lit(model.JSONRPCStreamMethod)).Block(If(String().Call(Id("message").Dot("ID")).Op("==").String().Call(Id("streamID"))).Block(Return()), Continue()),
				Var().Id("chunk").Id("rpcStreamChunk"),
				If(Err().Op(":=").Qual(jsonPkg, "Unmarshal").Call(Id("message").Dot("Params"), Op("&").Id("chunk")).Op(";").Err().Op("!=").Nil()).Block(Continue()),
				Var().Id("item").Add(r.fieldTypeFromTypeRef(ctx, element, false)),
				If(Err().Op(":=").Qual(jsonPkg, "Unmarshal").Call(Id("chunk").Dot("Item"), Op("&").Id("item")).Op(";").Err().Op("!=").Nil()).Block(Continue()),
				Select().Block(Case(Id("items").Op("<-").Id("item")), Case(Op("<-").Id(_ctx_).Dot("Done").Call()).Block(Return())),
			),
		).Call()
		bg.Return()
	})
}

func (r *ClientRenderer) streamMethodParams(ctx context.Context, contract *model.Contract, method *model.Method, withInput bool) (st *Statement) {

	st = &Statement{}
	st.ListFunc(func(gr *Group) {
		gr.Id(_ctx_).Qual(PackageContext, "Context")
		for _, arg := range r.argsForClient(contract, method) {
			if model.TypeRefIsChan(r.project, &arg.TypeRef) {
				continue
			}
			gr.Id(ToLowerCamel(arg.Name)).Add(r.fieldTypeFromTypeRef(ctx, &arg.TypeRef, true))
		}
		if withInput {
			if in, _, ok := model.MethodStreamInChan(r.project, method); ok {
				gr.Id(ToLowerCamel(in.Name)).Add(r.fieldTypeFromTypeRef(ctx, &in.TypeRef, true))
			}
		}
	})
	return
}

func (r *ClientRenderer) clientArgByName(contract *model.Contract, method *model.Method, argName string) (v *model.Variable) {

	for _, arg := range r.argsForClient(contract, method) {
		if arg.Name == argName {
			return arg
		}
	}
	return nil
}

// streamClientResults — результаты клиентского API: при out-chan синхронные non-chan не возвращаются
// (они приходят в финальном JSON-RPC result после закрытия потока).
func (r *ClientRenderer) streamClientResults(method *model.Method) (out []*model.Variable) {

	_, _, hasOut := model.MethodStreamOutChan(r.project, method)
	for _, result := range method.Results {
		if hasOut && !model.TypeRefIsChan(r.project, &result.TypeRef) && !r.isErrorResult(result) {
			continue
		}
		out = append(out, result)
	}
	return
}

func (r *ClientRenderer) isErrorResult(result *model.Variable) (ok bool) {

	return result != nil && result.TypeID == "error"
}

func (r *ClientRenderer) streamFinalResult(ctx context.Context, contract *model.Contract, method *model.Method, message string) (c Code) {

	results := r.streamResults(method)
	if len(results) == 0 {
		return Return()
	}
	return Var().Id("response").Id(r.responseStructName(contract, method)).Line().
		If(Err().Op("=").Qual(r.getPackageJSON(contract), "Unmarshal").Call(Id(message).Dot("Result"), Op("&").Id("response")).Op(";").Err().Op("!=").Nil()).Block(Return()).Line().
		ReturnFunc(func(rg *Group) {
			for _, result := range results {
				rg.Id("response").Dot(ToCamel(result.Name))
			}
			rg.Err()
		})
}

func (r *ClientRenderer) streamResults(method *model.Method) (out []*model.Variable) {

	for _, result := range r.resultsWithoutError(method) {
		if !model.TypeRefIsChan(r.project, &result.TypeRef) {
			out = append(out, result)
		}
	}
	return
}
