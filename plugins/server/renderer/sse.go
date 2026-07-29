// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/generated"
	"tgp/internal/model"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) RenderSSE() (err error) {

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(generated.ByToolGateway)
	srcFile.ImportName("bufio", "bufio")
	srcFile.ImportName(PackageStdJSON, "json")
	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(fmt.Sprintf("%s/stream", r.pkgPath(r.outDir)), "stream")

	typeGen := types.NewGenerator(r.project, &srcFile)
	for _, method := range r.contract.Methods {
		if model.MethodIsSSE(r.project, r.contract, method) {
			srcFile.Line().Add(r.sseHandler(&srcFile, typeGen, method))
		}
	}
	return srcFile.Save(path.Join(r.outDir, strings.ToLower(r.contract.Name)+"-sse.go"))
}

func (r *contractRenderer) sseHandler(srcFile *GoFile, typeGen *types.Generator, method *model.Method) (c Code) {

	streamPath := fmt.Sprintf("%s/stream", r.pkgPath(r.outDir))
	outResult, element, _ := model.MethodStreamOutChan(r.project, method)
	needsServerRef := model.IsAnnotationSet(r.project, r.contract, nil, nil, model.TagServerJsonRPC) ||
		model.ContractHasSSE(r.project, r.contract)
	fiberErr := func(arg, header string) []Code {
		return []Code{
			Return(Qual(PackageFiber, "NewError").Call(Qual(PackageFiber, "StatusBadRequest"), Lit("http value could not be decoded: ").Op("+").Err().Dot("Error").Call())),
		}
	}
	return Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).Id("serveSSE" + method.Name).
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).Params(Id("err").Error()).
		BlockFunc(func(bg *Group) {
			bg.Var().Id("requestBase").Qual(streamPath, "Message")
			bg.If(Len(Id(VarNameFtx).Dot("Body").Call()).Op(">").Lit(0)).Block(
				If(Err().Op("=").Qual(PackageStdJSON, "Unmarshal").Call(Id(VarNameFtx).Dot("Body").Call(), Op("&").Id("requestBase")).Op(";").Err().Op("!=").Nil()).Block(
					Return(Qual(PackageFiber, "NewError").Call(Qual(PackageFiber, "StatusBadRequest"), Lit("invalid stream request: ").Op("+").Err().Dot("Error").Call())),
				),
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
			bg.Id(VarNameFtx).Dot("Set").Call(Lit("Content-Type"), Lit("text/event-stream"))
			bg.Id(VarNameFtx).Dot("Set").Call(Lit("Cache-Control"), Lit("no-cache"))
			bg.Id(VarNameFtx).Dot("Set").Call(Lit("X-Accel-Buffering"), Lit("no"))
			bg.Comment("Capture request context before SetBodyStreamWriter: Fiber Ctx is recycled while the writer runs.")
			bg.Id("streamCtx").Op(":=").Id(VarNameFtx).Dot("UserContext").Call()
			bg.Id("heartbeat").Op(":=").Qual(streamPath, "DefaultSSEHeartbeat")
			if needsServerRef {
				bg.If(Id("http").Dot("srv").Op("!=").Nil()).Block(
					Id("heartbeat").Op("=").Id("http").Dot("srv").Dot("sseHeartbeat"),
				)
			}
			bg.Id(VarNameFtx).Dot("Context").Call().Dot("SetBodyStreamWriter").Call(
				Func().Params(Id("writer").Op("*").Qual("bufio", "Writer")).BlockFunc(func(wg *Group) {
					wg.If(Err().Op("=").Qual(streamPath, "OpenSSE").Call(Id("writer")).Op(";").Err().Op("!=").Nil()).Block(Return())
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
						lg.Err()
					}).Op("=").Id("http").Dot("svc").Dot(method.Name).CallFunc(func(cg *Group) {
						cg.Id("streamCtx")
						for _, arg := range argsWithoutContext(method) {
							if model.TypeRefIsChan(r.project, &arg.TypeRef) {
								continue
							}
							cg.Id("request").Dot(r.requestStructFieldName(method, arg))
						}
					})
					wg.If(Err().Op("!=").Nil()).Block(
						Id("_").Op("=").Qual(streamPath, "WriteSSEError").Call(Id("writer"), Id("requestBase").Dot("ID"), Err()),
						Return(),
					)
					if len(nonChanResults) == 0 {
						wg.If(Err().Op("=").Qual(streamPath, "PumpSSEServerStreamTyped").Types(typeGen.FieldTypeFromTypeRef(element, false)).
							Call(
								Id("streamCtx"),
								Id("writer"),
								Id("requestBase").Dot("ID"),
								Id(outResult.Name),
								Qual(streamPath, "EmptyResult").Call(),
								Id("heartbeat"),
							).Op(";").Err().Op("!=").Nil()).Block(Return())
					} else {
						wg.Var().Id("final").Qual(PackageStdJSON, "RawMessage")
						wg.If(List(Id("final"), Err()).Op("=").Qual(streamPath, "MarshalResult").Call(Id("response")).Op(";").Err().Op("!=").Nil()).Block(Return())
						wg.If(Err().Op("=").Qual(streamPath, "PumpSSEServerStreamTyped").Types(typeGen.FieldTypeFromTypeRef(element, false)).
							Call(
								Id("streamCtx"),
								Id("writer"),
								Id("requestBase").Dot("ID"),
								Id(outResult.Name),
								Id("final"),
								Id("heartbeat"),
							).Op(";").Err().Op("!=").Nil()).Block(Return())
					}
				}),
			)
			bg.Return()
		})
}
