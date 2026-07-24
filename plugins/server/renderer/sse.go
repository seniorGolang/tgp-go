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
	srcFile.ImportName(PackageFmt, "fmt")
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
	outResult, _, _ := model.MethodStreamOutChan(r.project, method)
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
			if len(nonChanResults) > 0 {
				bg.Var().Id("response").Id(responseStructName(r.contract.Name, method.Name))
			}
			bg.Var().Id(outResult.Name).Add(typeGen.FieldTypeFromTypeRef(&outResult.TypeRef, false))
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
				cg.Id(VarNameFtx).Dot("UserContext").Call()
				for _, arg := range argsWithoutContext(method) {
					if model.TypeRefIsChan(r.project, &arg.TypeRef) {
						continue
					}
					cg.Id("request").Dot(r.requestStructFieldName(method, arg))
				}
			})
			bg.If(Err().Op("!=").Nil()).Block(
				Return(Err()),
			)
			bg.Id(VarNameFtx).Dot("Set").Call(Lit("Content-Type"), Lit("text/event-stream"))
			bg.Id(VarNameFtx).Dot("Set").Call(Lit("Cache-Control"), Lit("no-cache"))
			// Capture request context before SetBodyStreamWriter: Fiber Ctx is recycled while the writer runs.
			bg.Id("streamCtx").Op(":=").Id(VarNameFtx).Dot("UserContext").Call()
			bg.Id(VarNameFtx).Dot("Context").Call().Dot("SetBodyStreamWriter").Call(
				Func().Params(Id("writer").Op("*").Qual("bufio", "Writer")).BlockFunc(func(wg *Group) {
					wg.Var().Id("seq").Int64()
					wg.For().BlockFunc(func(fg *Group) {
						fg.Select().Block(
							Case(Id("item").Op(",").Id("open").Op(":=").Op("<-").Id(outResult.Name)).BlockFunc(func(cg *Group) {
								cg.If(Op("!").Id("open")).BlockFunc(func(ig *Group) {
									if len(nonChanResults) == 0 {
										ig.Id("final").Op(":=").Qual(streamPath, "Message").Values(Dict{
											Id("ID"):      Id("requestBase").Dot("ID"),
											Id("Version"): Qual(streamPath, "Version"),
											Id("Result"):  Qual(streamPath, "EmptyResult").Call(),
										})
									} else {
										ig.Var().Id("finalResult").Qual(PackageStdJSON, "RawMessage")
										ig.If(List(Id("finalResult"), Err()).Op("=").Qual(streamPath, "MarshalResult").Call(Id("response")).Op(";").Err().Op("!=").Nil()).Block(Return())
										ig.Id("final").Op(":=").Qual(streamPath, "Message").Values(Dict{
											Id("ID"):      Id("requestBase").Dot("ID"),
											Id("Version"): Qual(streamPath, "Version"),
											Id("Result"):  Id("finalResult"),
										})
									}
									ig.Id("payload").Op(",").Id("marshalErr").Op(":=").Qual(PackageStdJSON, "Marshal").Call(Id("final"))
									ig.If(Id("marshalErr").Op("!=").Nil()).Block(Return())
									ig.Id("_").Op(",").Id("_").Op("=").Qual(PackageFmt, "Fprintf").Call(Id("writer"), Lit("data: %s\n\n"), Id("payload"))
									ig.Id("_").Op("=").Id("writer").Dot("Flush").Call()
									ig.Return()
								})
								cg.Id("seq").Op("++")
								cg.Id("raw").Op(",").Id("marshalErr").Op(":=").Qual(PackageStdJSON, "Marshal").Call(Id("item"))
								cg.If(Id("marshalErr").Op("!=").Nil()).Block(Return())
								cg.Id("params").Op(",").Id("marshalErr").Op(":=").Qual(PackageStdJSON, "Marshal").Call(Qual(streamPath, "ChunkParams").Values(Dict{
									Id("ID"):   Id("requestBase").Dot("ID"),
									Id("Seq"):  Id("seq"),
									Id("Item"): Id("raw"),
								}))
								cg.If(Id("marshalErr").Op("!=").Nil()).Block(Return())
								cg.Id("payload").Op(",").Id("marshalErr").Op(":=").Qual(PackageStdJSON, "Marshal").Call(Qual(streamPath, "Message").Values(Dict{
									Id("Version"): Qual(streamPath, "Version"),
									Id("Method"):  Qual(streamPath, "MethodStream"),
									Id("Params"):  Id("params"),
								}))
								cg.If(Id("marshalErr").Op("!=").Nil()).Block(Return())
								cg.Id("_").Op(",").Id("_").Op("=").Qual(PackageFmt, "Fprintf").Call(Id("writer"), Lit("data: %s\n\n"), Id("payload"))
								cg.Id("_").Op("=").Id("writer").Dot("Flush").Call()
							}),
							Case(Op("<-").Id("streamCtx").Dot("Done").Call()).Block(
								Return(),
							),
						)
					})
				}),
			)
			bg.Return()
		})
}
