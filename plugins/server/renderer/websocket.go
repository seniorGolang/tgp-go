// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/common"
	"tgp/internal/generated"
	"tgp/internal/model"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) RenderWebSocket() (err error) {

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(generated.ByToolGateway)
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(PackageStdJSON, "json")
	srcFile.ImportName(PackageFmt, "fmt")
	srcFile.ImportName(PackageStrings, "strings")
	srcFile.ImportName(r.contract.PkgPath, filepath.Base(r.contract.PkgPath))
	srcFile.ImportName(PackageFiberWebsocket, "websocket")
	srcFile.ImportName(fmt.Sprintf("%s/stream", r.pkgPath(r.outDir)), "stream")

	typeGen := types.NewGenerator(r.project, &srcFile)
	srcFile.Type().Id("ws" + r.contract.Name + "Conn").Struct(Id("c").Op("*").Qual(PackageFiberWebsocket, "Conn"))
	srcFile.Line().Add(r.wsReadJSON())
	srcFile.Line().Add(r.wsWriteJSON())
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

func (r *contractRenderer) wsClose() (c Code) {

	return Func().Params(Id("conn").Op("*").Id("ws" + r.contract.Name + "Conn")).Id("Close").
		Params().Params(Id("err").Error()).Block(
		Return(Id("conn").Dot("c").Dot("Close").Call()),
	)
}

func (r *contractRenderer) wsServe() (c Code) {

	streamPath := fmt.Sprintf("%s/stream", r.pkgPath(r.outDir))
	headerNames, cookieNames, queryNames, pathNames := r.streamWSHTTPKeys()
	return Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).Id("serveWS").
		Params(Id("conn").Op("*").Qual(PackageFiberWebsocket, "Conn")).BlockFunc(func(bg *Group) {
		bg.Id("handlers").Op(":=").Map(String()).Qual(streamPath, "Handler").Values(DictFunc(func(dg Dict) {
			for _, method := range r.contract.Methods {
				if !model.MethodIsWS(r.project, r.contract, method) {
					continue
				}
				dg[Lit(strings.ToLower(model.JsonRPCWireMethod(r.contract.Name, method.Name)))] = Id("http").Dot("stream" + method.Name)
			}
		}))
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
		bg.Id("ctx").Op(":=").Qual(PackageContext, "WithValue").Call(Qual(PackageContext, "Background").Call(), Qual(streamPath, "KeyOverlay"), Id("overlay"))
		bg.Id("session").Op(":=").Qual(streamPath, "NewSession").Call(
			Op("&").Id("ws"+r.contract.Name+"Conn").Values(Dict{Id("c"): Id("conn")}),
			Id("handlers"),
		)
		bg.Id("session").Dot("Serve").Call(Id("ctx"))
	})
}

func (r *contractRenderer) wsHandler(srcFile *GoFile, typeGen *types.Generator, method *model.Method) (c Code) {

	streamPath := fmt.Sprintf("%s/stream", r.pkgPath(r.outDir))
	mode := model.MethodStreamMode(r.project, r.contract, method)
	errBody := func(arg, header string) []Code {
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
			if mode == model.StreamModeClient || mode == model.StreamModeBidi {
				inArg, element, _ := model.MethodStreamInChan(r.project, method)
				bg.Id("rawIn").Op(",").Id("ok").Op(":=").Id("session").Dot("Incoming").Call(Id("requestBase").Dot("ID"))
				bg.If(Op("!").Id("ok")).Block(
					Return(Nil(), Qual(PackageFmt, "Errorf").Call(Lit("stream input is unavailable"))),
				)
				bg.Id(inArg.Name).Op(":=").Make(Chan().Add(typeGen.FieldTypeFromTypeRef(element, false)))
				bg.Go().Func().Params().Block(
					Id("_").Op("=").Qual(streamPath, "FeedIn").Types(typeGen.FieldTypeFromTypeRef(element, false)).
						Call(Id(VarNameCtx), Id("rawIn"), Id(inArg.Name)),
				).Call()
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
				cg.Id(VarNameCtx)
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
					Call(Id(VarNameCtx), Id("session"), Id("requestBase").Dot("ID"), Id(outName)).Op(";").Err().Op("!=").Nil()).Block(
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
