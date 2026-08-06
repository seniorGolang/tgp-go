// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive

	"tgp/internal/common"
	"tgp/internal/model"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) RenderWebSocket() (err error) {

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(PackageStdJSON, keyJSON)
	srcFile.ImportName(PackageFmt, "fmt")
	srcFile.ImportName(PackageErrors, "errors")
	srcFile.ImportName(PackageStrings, "strings")
	srcFile.ImportName(PackageTime, "time")
	srcFile.ImportName(r.contract.PkgPath, filepath.Base(r.contract.PkgPath))
	srcFile.ImportName(PackageFiberWebsocket, "websocket")
	srcFile.ImportName(r.pkgPath(r.outDir)+"/stream", "stream")

	typeGen := types.NewGenerator(r.project, &srcFile)
	srcFile.Type().Id("ws" + r.contract.Name + "Conn").Struct(Id("c").Op("*").Qual(PackageFiberWebsocket, "Conn"))
	srcFile.Line().Add(r.wsReadJSON())
	srcFile.Line().Add(r.wsWriteJSON())
	srcFile.Line().Add(r.wsSetWriteDeadline())
	srcFile.Line().Add(r.wsClose())
	srcFile.Line().Add(r.wsServe())

	for _, method := range r.contract.Methods {
		if model.MethodIsWS(r.project, r.contract, method) {
			srcFile.Line().Add(r.wsHandler(&srcFile, typeGen, method))
		}
	}
	return srcFile.Save(path.Join(r.outDir, strings.ToLower(r.contract.Name)+"-websocket.go"))
}

func (r *contractRenderer) wsReadJSON() (c Code) {

	return Func().Params(Id("conn").Op("*").Id("ws" + r.contract.Name + "Conn")).Id("ReadJSON").
		Params(Id("value").Any()).Params(Id("err").Error()).Block(
		Return(Id("conn").Dot("c").Dot("ReadJSON").Call(Id("value"))),
	)
}

func (r *contractRenderer) wsWriteJSON() (c Code) {

	return Func().Params(Id("conn").Op("*").Id("ws" + r.contract.Name + "Conn")).Id("WriteJSON").
		Params(Id("value").Any()).Params(Id("err").Error()).Block(
		Return(Id("conn").Dot("c").Dot("WriteJSON").Call(Id("value"))),
	)
}

func (r *contractRenderer) wsSetWriteDeadline() (c Code) {

	return Func().Params(Id("conn").Op("*").Id("ws" + r.contract.Name + "Conn")).Id("SetWriteDeadline").
		Params(Id("deadline").Qual(PackageTime, "Time")).Params(Id("err").Error()).Block(
		Return(Id("conn").Dot("c").Dot("SetWriteDeadline").Call(Id("deadline"))),
	)
}

func (r *contractRenderer) wsClose() (c Code) {

	return Func().Params(Id("conn").Op("*").Id("ws" + r.contract.Name + "Conn")).Id("Close").
		Params().Params(Id("err").Error()).Block(
		Return(Id("conn").Dot("c").Dot("Close").Call()),
	)
}

func (r *contractRenderer) wsServe() (c Code) {

	streamPath := r.pkgPath(r.outDir) + "/stream"
	headerNames, cookieNames, queryNames, pathNames := r.streamWSHTTPKeys()
	needsOverlay := len(headerNames)+len(cookieNames)+len(queryNames)+len(pathNames) > 0
	return Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).Id("serveWS").
		Params(Id("conn").Op("*").Qual(PackageFiberWebsocket, "Conn")).BlockFunc(func(bg *Group) {
		bg.If(Id("http").Dot("srv").Op("!=").Nil()).Block(
			Defer().Id("http").Dot("srv").Dot("releaseWS").Call(),
			If(Id("http").Dot("srv").Dot("maxWSMessageSize").Op(">").Lit(0)).Block(
				Id("conn").Dot("SetReadLimit").Call(Id("http").Dot("srv").Dot("maxWSMessageSize")),
			),
		)
		bg.Id("handlers").Op(":=").Map(String()).Qual(streamPath, "Handler").Values(DictFunc(func(dg Dict) {
			for _, method := range r.contract.Methods {
				if !model.MethodIsWS(r.project, r.contract, method) {
					continue
				}
				dg[Lit(strings.ToLower(model.JSONRPCWireMethod(r.contract.Name, method.Name)))] = Id("http").Dot("stream" + method.Name)
			}
		}))
		bg.Id("baseCtx").Op(":=").Qual(PackageContext, "Background").Call()
		bg.If(List(Id("userCtx"), Id("ok")).Op(":=").Id("conn").Dot("Locals").Call(Lit("wsCtx")).Assert(Qual(PackageContext, "Context")).Op(";").Id("ok").Op("&&").Id("userCtx").Op("!=").Nil()).Block(
			Id("baseCtx").Op("=").Id("userCtx"),
		)
		bg.List(Id("sessionCtx"), Id("cancel")).Op(":=").Qual(PackageContext, "WithCancel").Call(Id("baseCtx"))
		bg.Defer().Id("cancel").Call()
		bg.If(Id("http").Dot("srv").Op("!=").Nil()).Block(
			Go().Func().Params().Block(
				Select().Block(
					Case(Op("<-").Id("http").Dot("srv").Dot("rootCtx").Dot("Done").Call()).Block(
						Id("cancel").Call(),
					),
					Case(Op("<-").Id("sessionCtx").Dot("Done").Call()).Block(),
				),
			).Call(),
		)
		if needsOverlay {
			bg.Id("overlay").Op(":=").Map(String()).String().Values()
			for _, name := range headerNames {
				bg.Id("overlay").Index(Lit(name)).Op("=").Qual(PackageStrings, "Clone").Call(Id("conn").Dot("Headers").Call(Lit(name)))
				bg.If(Id("overlay").Index(Lit(name)).Op("==").Lit("")).Block(
					Id("overlay").Index(Lit(name)).Op("=").Qual(PackageStrings, "Clone").Call(Id("conn").Dot("Query").Call(Lit(name))),
				)
			}
			for _, name := range cookieNames {
				bg.Id("overlay").Index(Lit(name)).Op("=").Qual(PackageStrings, "Clone").Call(Id("conn").Dot("Cookies").Call(Lit(name)))
				bg.If(Id("overlay").Index(Lit(name)).Op("==").Lit("")).Block(
					Id("overlay").Index(Lit(name)).Op("=").Qual(PackageStrings, "Clone").Call(Id("conn").Dot("Query").Call(Lit(name))),
				)
			}
			for _, name := range queryNames {
				bg.Id("overlay").Index(Lit(name)).Op("=").Qual(PackageStrings, "Clone").Call(Id("conn").Dot("Query").Call(Lit(name)))
			}
			for _, name := range pathNames {
				bg.Id("overlay").Index(Lit(name)).Op("=").Qual(PackageStrings, "Clone").Call(Id("conn").Dot("Params").Call(Lit(name)))
			}
			bg.Id("ctx").Op(":=").Qual(PackageContext, "WithValue").Call(Id("sessionCtx"), Qual(streamPath, "KeyOverlay"), Id("overlay"))
		} else {
			bg.Id("ctx").Op(":=").Id("sessionCtx")
		}
		bg.Id("maxOpenStreams").Op(":=").Lit(0)
		bg.Id("writeTimeout").Op(":=").Qual(PackageTime, "Duration").Call(Lit(0))
		bg.If(Id("http").Dot("srv").Op("!=").Nil()).Block(
			Id("maxOpenStreams").Op("=").Id("http").Dot("srv").Dot("maxOpenStreams"),
			Id("writeTimeout").Op("=").Id("http").Dot("srv").Dot("streamWriteTimeout"),
		)
		bg.Id("session").Op(":=").Qual(streamPath, "NewSession").Call(
			Op("&").Id("ws"+r.contract.Name+"Conn").Values(Dict{Id("c"): Id("conn")}),
			Id("handlers"),
			Id("maxOpenStreams"),
			Id("writeTimeout"),
		)
		bg.Id("streamID").Op(":=").Uint64().Call(Lit(0))
		bg.If(Id("http").Dot("srv").Op("!=").Nil()).Block(
			Id("streamID").Op("=").Id("http").Dot("srv").Dot("registerStream").Call(Id("session")),
			Defer().Id("http").Dot("srv").Dot("unregisterStream").Call(Id("streamID")),
		)
		bg.Id("session").Dot("Serve").Call(Id("ctx"))
	})
}

func (r *contractRenderer) wsHandler(srcFile *GoFile, typeGen *types.Generator, method *model.Method) (c Code) {

	streamPath := r.pkgPath(r.outDir) + "/stream"
	mode := model.MethodStreamMode(r.project, r.contract, method)
	hasFeedIn := mode == model.StreamModeClient || mode == model.StreamModeBidi
	errBody := func(_arg, _header string) []Code {
		return []Code{
			Return(Nil(), Qual(PackageFmt, "Errorf").Call(Lit("http value could not be decoded: %w"), Err())),
		}
	}
	return Func().Params(Id("http").Op("*").Id("http"+r.contract.Name)).Id("stream"+method.Name).
		Params(
			Id(VarNameCtx).Qual(PackageContext, "Context"),
			Id("requestBase").Qual(streamPath, "Message"),
			Id("session").Op("*").Qual(streamPath, "Session"),
		).
		Params(Id("result").Qual(PackageStdJSON, "RawMessage"), Id("err").Error()).
		BlockFunc(func(bg *Group) {
			bg.Var().Id("request").Id(requestStructName(r.contract.Name, method.Name))
			bg.If(Len(Id("requestBase").Dot("Params")).Op(">").Lit(0)).Block(
				If(Err().Op("=").Qual(PackageStdJSON, "Unmarshal").Call(Id("requestBase").Dot("Params"), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).Block(
					Return(Nil(), Qual(PackageFmt, "Errorf").Call(Lit("invalid params: %w"), Err())),
				),
			)
			bg.Add(r.applyOverlayFromContext(srcFile, typeGen, method, errBody, false))
			callCtx := Id(VarNameCtx)
			if hasFeedIn {
				inArg, element, _ := model.MethodStreamInChan(r.project, method)
				bg.Id("rawIn").Op(",").Id("ok").Op(":=").Id("session").Dot("Incoming").Call(Id("requestBase").Dot("ID"))
				bg.If(Op("!").Id("ok")).Block(
					Return(Nil(), Qual(PackageFmt, "Errorf").Call(Lit("stream input is unavailable"))),
				)
				bg.List(Id("feedCtx"), Id("feedCancel")).Op(":=").Qual(PackageContext, "WithCancel").Call(Id(VarNameCtx))
				bg.Id(inArg.Name).Op(":=").Make(Chan().Add(typeGen.FieldTypeFromTypeRef(element, false)))
				bg.Id("feedErr").Op(":=").Make(Chan().Error(), Lit(1))
				bg.Go().Func().Params().Block(
					Id("e").Op(":=").Qual(streamPath, "FeedIn").Types(typeGen.FieldTypeFromTypeRef(element, false)).
						Call(Id("feedCtx"), Id("rawIn"), Id(inArg.Name)),
					If(Id("e").Op("!=").Nil()).Block(Id("feedCancel").Call()),
					Id("feedErr").Op("<-").Id("e"),
				).Call()
				bg.Defer().Func().Params().Block(
					Id("feedCancel").Call(),
					Id("e").Op(":=").Op("<-").Id("feedErr"),
					If(Id("e").Op("==").Nil().Op("||").Qual(PackageErrors, "Is").Call(Id("e"), Qual(PackageContext, "Canceled"))).Block(
						Return(),
					),
					If(Err().Op("==").Nil().Op("||").Qual(PackageErrors, "Is").Call(Err(), Qual(PackageContext, "Canceled"))).Block(
						Err().Op("=").Id("e"),
					),
				).Call()
				callCtx = Id("feedCtx")
			}
			nonChanResults := streamVariables(r.project, resultsWithoutError(method), false)
			if len(nonChanResults) > 0 {
				bg.Var().Id("response").Id(responseStructName(r.contract.Name, method.Name))
			}
			outName := ""
			var outElement *model.TypeRef
			if mode == model.StreamModeServer || mode == model.StreamModeBidi {
				outResult, element, _ := model.MethodStreamOutChan(r.project, method)
				outName = outResult.Name
				outElement = element
				bg.Var().Id(outName).Add(typeGen.FieldTypeFromTypeRef(&outResult.TypeRef, false))
			}
			bg.ListFunc(func(lg *Group) {
				for _, variable := range resultsWithoutError(method) {
					if model.TypeRefIsChan(r.project, &variable.TypeRef) {
						lg.Id(variable.Name)
						continue
					}
					lg.Id("response").Dot(r.responseStructFieldName(method, variable))
				}
				lg.Err()
			}).Op("=").Id("http").Dot("svc").Dot(method.Name).CallFunc(func(cg *Group) {
				cg.Add(callCtx)
				for _, arg := range argsWithoutContext(method) {
					if model.TypeRefIsChan(r.project, &arg.TypeRef) {
						cg.Id(arg.Name)
						continue
					}
					cg.Id("request").Dot(r.requestStructFieldName(method, arg))
				}
			})
			bg.If(Err().Op("!=").Nil()).Block(
				Return(Nil(), Err()),
			)
			if outName != "" {
				bg.If(Err().Op("=").Qual(streamPath, "PumpOutTyped").Types(typeGen.FieldTypeFromTypeRef(outElement, false)).
					Call(callCtx, Id("session"), Id("requestBase").Dot("ID"), Id(outName)).Op(";").Err().Op("!=").Nil()).Block(
					Return(Nil(), Err()),
				)
			}
			if len(nonChanResults) == 0 {
				bg.Return(Qual(streamPath, "EmptyResult").Call(), Nil())
				return
			}
			bg.Return(Qual(streamPath, "MarshalResult").Call(Id("response")))
		})
}

func (r *contractRenderer) streamWSHTTPKeys() (headerNames []string, cookieNames []string, queryNames []string, pathNames []string) {

	headers := make(map[string]struct{})
	cookies := make(map[string]struct{})
	queries := make(map[string]struct{})
	paths := make(map[string]struct{})
	for _, method := range r.contract.Methods {
		if !model.MethodIsWS(r.project, r.contract, method) {
			continue
		}
		for _, name := range usedHeaderNamesForRequestOverlay(r.project, r.contract, method) {
			headers[name] = struct{}{}
		}
		for _, name := range usedCookieNamesForRequestOverlay(r.project, r.contract, method) {
			cookies[name] = struct{}{}
		}
		for _, key := range model.HTTPArgQueryMapForRequest(r.project, r.contract, method) {
			queries[key] = struct{}{}
		}
		for _, segment := range model.StreamPathParamArgMap(r.project, r.contract, method) {
			paths[segment] = struct{}{}
		}
	}
	return common.SortedKeys(headers), common.SortedKeys(cookies), common.SortedKeys(queries), common.SortedKeys(paths)
}
