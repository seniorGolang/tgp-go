// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive

	"tgp/internal/model"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) RenderSSE() (err error) {

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.ImportName("bufio", "bufio")
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(PackageStdJSON, keyJSON)
	srcFile.ImportName(PackageTime, "time")
	srcFile.ImportName(PackageErrors, "errors")
	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(PackageSync, "sync")
	if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagTrace) {
		srcFile.ImportName(PackageTrace, "trace")
	}
	srcFile.ImportName(r.pkgPath(r.outDir)+"/stream", "stream")

	typeGen := types.NewGenerator(r.project, &srcFile)
	for _, method := range r.contract.Methods {
		if model.MethodIsSSE(r.project, r.contract, method) {
			srcFile.Line().Add(r.sseHandler(&srcFile, typeGen, method))
		}
	}
	return srcFile.Save(path.Join(r.outDir, strings.ToLower(r.contract.Name)+"-sse.go"))
}

func (r *contractRenderer) sseHandler(srcFile *GoFile, typeGen *types.Generator, method *model.Method) (c Code) {

	streamPath := r.pkgPath(r.outDir) + "/stream"
	outResult, element, _ := model.MethodStreamOutChan(r.project, method)
	needsServerRef := model.IsAnnotationSet(r.project, r.contract, nil, nil, model.TagServerJSONRPC) ||
		model.ContractHasSSE(r.project, r.contract)
	withTrace := model.IsAnnotationSet(r.project, r.contract, nil, nil, TagTrace)
	fiberErr := func(_arg, _header string) []Code {
		return []Code{
			Return(Qual(PackageFiber, "NewError").Call(Qual(PackageFiber, "StatusBadRequest"), Lit("http value could not be decoded: ").Op("+").Err().Dot("Error").Call())),
		}
	}
	return Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).Id("serveSSE" + method.Name).
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).Params(Id("err").Error()).
		BlockFunc(func(bg *Group) {
			if needsServerRef {
				bg.Var().Id("release").Func().Params()
				bg.If(Id("http").Dot("srv").Op("!=").Nil()).Block(
					If(Op("!").Id("http").Dot("srv").Dot("acquireSSE").Call()).Block(
						Return(Qual(PackageFiber, "NewError").Call(Qual(PackageFiber, "StatusServiceUnavailable"), Lit("too many SSE connections"))),
					),
					Var().Id("once").Qual(PackageSync, "Once"),
					Id("release").Op("=").Func().Params().Block(
						Id("once").Dot("Do").Call(Id("http").Dot("srv").Dot("releaseSSE")),
					),
					Defer().Id("release").Call(),
				)
			}
			bg.Var().Id("requestBase").Qual(streamPath, "Message")
			bg.If(Id(VarNameFtx).Dot("Request").Call().Dot("Header").Dot("ContentLength").Call().Op("!=").Lit(0)).Block(
				If(Err().Op("=").Qual(PackageStdJSON, "NewDecoder").Call(Id("decodeBodyReader").Call(Id(VarNameFtx))).Dot("Decode").Call(Op("&").Id("requestBase")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
					ig.If(Qual(PackageErrors, "Is").Call(Err(), Id("errBodyTooLarge"))).Block(
						Return(Qual(PackageFiber, "NewError").Call(Qual(PackageFiber, "StatusRequestEntityTooLarge"), Lit("request body too large"))),
					)
					ig.Return(Qual(PackageFiber, "NewError").Call(Qual(PackageFiber, "StatusBadRequest"), Lit("invalid stream request: ").Op("+").Err().Dot("Error").Call()))
				}),
			)
			bg.Var().Id("request").Id(requestStructName(r.contract.Name, method.Name))
			bg.If(Len(Id("requestBase").Dot("Params")).Op(">").Lit(0)).Block(
				If(Err().Op("=").Qual(PackageStdJSON, "Unmarshal").Call(Id("requestBase").Dot("Params"), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).Block(
					Return(Qual(PackageFiber, "NewError").Call(Qual(PackageFiber, "StatusBadRequest"), Lit("invalid stream parameters: ").Op("+").Err().Dot("Error").Call())),
				),
			)
			bg.Add(r.httpArgHeadersBodyMode(srcFile, typeGen, method, fiberErr))
			bg.Add(r.httpCookiesBodyMode(srcFile, typeGen, method, fiberErr))
			bg.Add(r.urlArgs(srcFile, typeGen, method, fiberErr))
			bg.Add(r.urlParams(srcFile, typeGen, method, fiberErr))
			bg.Add(r.httpArgHeaders(srcFile, typeGen, method, fiberErr))
			bg.Add(r.httpCookies(srcFile, typeGen, method, fiberErr))
			nonChanResults := streamVariables(r.project, resultsWithoutError(method), false)
			bg.Id("heartbeat").Op(":=").Qual(streamPath, "DefaultSSEHeartbeat")
			if needsServerRef {
				bg.If(Id("http").Dot("srv").Op("!=").Nil()).Block(
					Id("heartbeat").Op("=").Id("http").Dot("srv").Dot("effectiveSSEHeartbeat").Call(),
				)
			}
			bg.Id(VarNameFtx).Dot("Set").Call(Lit("Content-Type"), Lit("text/event-stream"))
			bg.Id(VarNameFtx).Dot("Set").Call(Lit("Cache-Control"), Lit("no-cache"))
			bg.Id(VarNameFtx).Dot("Set").Call(Lit("X-Accel-Buffering"), Lit("no"))
			bg.Id("parentCtx").Op(":=").Id(VarNameFtx).Dot("UserContext").Call()
			bg.Id("conn").Op(":=").Id(VarNameFtx).Dot("Context").Call().Dot("Conn").Call()
			bg.Id(VarNameFtx).Dot("Context").Call().Dot("SetBodyStreamWriter").Call(
				Func().Params(Id("writer").Op("*").Qual("bufio", "Writer")).BlockFunc(func(wg *Group) {
					if needsServerRef {
						wg.If(Id("release").Op("!=").Nil()).Block(
							Defer().Id("release").Call(),
						)
					}
					wg.Var().Id("writeErr").Error()
					wg.List(Id("streamCtx"), Id("cancel")).Op(":=").Qual(PackageContext, "WithCancel").Call(Id("parentCtx"))
					wg.Defer().Id("cancel").Call()
					if withTrace {
						wg.Defer().Qual(PackageTrace, "SpanFromContext").Call(Id("streamCtx")).Dot("End").Call()
					}
					if needsServerRef {
						wg.Id("streamID").Op(":=").Uint64().Call(Lit(0))
						wg.If(Id("http").Dot("srv").Op("!=").Nil()).Block(
							Go().Func().Params().Block(
								Select().Block(
									Case(Op("<-").Id("http").Dot("srv").Dot("rootCtx").Dot("Done").Call()).Block(
										Id("cancel").Call(),
									),
									Case(Op("<-").Id("streamCtx").Dot("Done").Call()).Block(),
								),
							).Call(),
							Id("streamID").Op("=").Id("http").Dot("srv").Dot("registerStream").Call(Id("cancelCloser").Values(Dict{Id("cancel"): Id("cancel")})),
							Defer().Id("http").Dot("srv").Dot("unregisterStream").Call(Id("streamID")),
						)
					}
					wg.Id("beforeWrite").Op(":=").Func().Params().Params(Id("deadlineErr").Error()).Block(
						If(Id("http").Dot("srv").Op("==").Nil().Op("||").Id("http").Dot("srv").Dot("streamWriteTimeout").Op("<=").Lit(0)).Block(
							Return(),
						),
						Return(Id("conn").Dot("SetWriteDeadline").Call(Qual(PackageTime, "Now").Call().Dot("Add").Call(Id("http").Dot("srv").Dot("streamWriteTimeout")))),
					)
					wg.Defer().Func().Params().Block(
						If(Id("recovered").Op(":=").Recover().Op(";").Id("recovered").Op("!=").Nil()).Block(
							Id("_").Op("=").Qual(streamPath, "WriteSSEError").Call(
								Id("writer"),
								Id("requestBase").Dot("ID"),
								Qual(PackageErrors, "New").Call(Lit("internal error")),
								Id("beforeWrite"),
							),
						),
					).Call()
					wg.If(Id("writeErr").Op("=").Qual(streamPath, "OpenSSE").Call(Id("writer"), Id("beforeWrite")).Op(";").Id("writeErr").Op("!=").Nil()).Block(Return())
					if len(nonChanResults) > 0 {
						wg.Var().Id("response").Id(responseStructName(r.contract.Name, method.Name))
					}
					wg.Var().Id(outResult.Name).Add(typeGen.FieldTypeFromTypeRef(&outResult.TypeRef, false))
					wg.ListFunc(func(lg *Group) {
						for _, variable := range resultsWithoutError(method) {
							if model.TypeRefIsChan(r.project, &variable.TypeRef) {
								lg.Id(variable.Name)
								continue
							}
							lg.Id("response").Dot(r.responseStructFieldName(method, variable))
						}
						lg.Id("writeErr")
					}).Op("=").Id("http").Dot("svc").Dot(method.Name).CallFunc(func(cg *Group) {
						cg.Id("streamCtx")
						for _, arg := range argsWithoutContext(method) {
							if model.TypeRefIsChan(r.project, &arg.TypeRef) {
								continue
							}
							cg.Id("request").Dot(r.requestStructFieldName(method, arg))
						}
					})
					wg.If(Id("writeErr").Op("!=").Nil()).Block(
						Id("_").Op("=").Qual(streamPath, "WriteSSEError").Call(Id("writer"), Id("requestBase").Dot("ID"), Id("writeErr"), Id("beforeWrite")),
						Return(),
					)
					if len(nonChanResults) == 0 {
						wg.If(Id("writeErr").Op("=").Qual(streamPath, "PumpSSEServerStreamTyped").Types(typeGen.FieldTypeFromTypeRef(element, false)).
							Call(
								Id("streamCtx"),
								Id("writer"),
								Id("requestBase").Dot("ID"),
								Id(outResult.Name),
								Qual(streamPath, "EmptyResult").Call(),
								Id("heartbeat"),
								Id("beforeWrite"),
							).Op(";").Id("writeErr").Op("!=").Nil()).Block(Return())
					} else {
						wg.Var().Id("final").Qual(PackageStdJSON, "RawMessage")
						wg.If(List(Id("final"), Id("writeErr")).Op("=").Qual(streamPath, "MarshalResult").Call(Id("response")).Op(";").Id("writeErr").Op("!=").Nil()).Block(
							Id("_").Op("=").Qual(streamPath, "WriteSSEError").Call(Id("writer"), Id("requestBase").Dot("ID"), Id("writeErr"), Id("beforeWrite")),
							Return(),
						)
						wg.If(Id("writeErr").Op("=").Qual(streamPath, "PumpSSEServerStreamTyped").Types(typeGen.FieldTypeFromTypeRef(element, false)).
							Call(
								Id("streamCtx"),
								Id("writer"),
								Id("requestBase").Dot("ID"),
								Id(outResult.Name),
								Id("final"),
								Id("heartbeat"),
								Id("beforeWrite"),
							).Op(";").Id("writeErr").Op("!=").Nil()).Block(Return())
					}
				}),
			)
			bg.Return(Nil())
		})
}
