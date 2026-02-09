// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) RenderJsonRPC() error {

	if err := r.pkgCopyTo("srvctx", r.outDir); err != nil {
		return fmt.Errorf("copy srvctx package: %w", err)
	}

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	jsonPkg := r.getPackageJSON()
	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(PackageErrors, "errors")
	srcFile.ImportName("io", "io")
	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(PackageSync, "sync")
	srcFile.ImportName(PackageBytes, "bytes")
	srcFile.ImportName(PackageTime, "time")
	srcFile.ImportName(r.contract.PkgPath, filepath.Base(r.contract.PkgPath))
	srcFile.ImportName(jsonPkg, "json")
	srcFile.ImportName(PackageReflect, "reflect")
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir)), "srvctx")

	typeGen := types.NewGenerator(r.project, &srcFile)

	for _, method := range r.contract.Methods {
		if !r.methodIsJsonRPC(method) {
			continue
		}
		methodName := strings.ToLower(method.Name)
		srcFile.Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).
			Id("serve" + method.Name).
			Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
			Params(Err().Error()).
			Block(
				Return().Id("http").Dot("_serveMethod").Call(Id(VarNameFtx), Lit(methodName), Id("http").Dot(toLowerCamel(method.Name))),
			)
		srcFile.Add(r.rpcMethodFuncWithFiberCtx(&srcFile, typeGen, method, jsonPkg))
		srcFile.Add(r.rpcMethodFuncWithContext(&srcFile, typeGen, method, jsonPkg))
	}
	srcFile.Add(r.serveMethodFunc(jsonPkg))
	srcFile.Add(r.serviceServeBatchFunc(jsonPkg))
	srcFile.Add(r.serviceSingleBatchFunc(typeGen))

	return srcFile.Save(path.Join(r.outDir, strings.ToLower(r.contract.Name)+"-jsonrpc.go"))
}

func (r *contractRenderer) rpcMethodFuncWithFiberCtx(srcFile *GoFile, typeGen *types.Generator, method *model.Method, jsonPkg string) Code {

	return Func().Params(Id("http").Op("*").Id("http"+r.contract.Name)).
		Id(toLowerCamel(method.Name)).
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx"), Id("requestBase").Id("baseJsonRPC")).
		Params(Id("responseBase").Op("*").Id("baseJsonRPC")).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Return(Id("http").Dot(toLowerCamel(method.Name)+"WithContext").Call(Id(VarNameFtx).Dot("UserContext").Call(), Id("requestBase")))
		})
}

func (r *contractRenderer) rpcMethodFuncWithContext(srcFile *GoFile, typeGen *types.Generator, method *model.Method, jsonPkg string) Code {

	return Func().Params(Id("http").Op("*").Id("http"+r.contract.Name)).
		Id(toLowerCamel(method.Name)+"WithContext").
		Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("requestBase").Id("baseJsonRPC")).
		Params(Id("responseBase").Op("*").Id("baseJsonRPC")).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Id(VarNameCtx).Op("=").Id("withMethodLogger").Call(Id(VarNameCtx), Lit(toLowerCamel(r.contract.Name)), Lit(toLowerCamel(method.Name)))
			bg.Line()
			bg.Var().Err().Error()
			bg.Var().Id("request").Id(requestStructName(r.contract.Name, method.Name))
			bg.Var().Id("response").Id(responseStructName(r.contract.Name, method.Name))
			bg.Line()
			bg.If(Id("requestBase").Dot("Params").Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.If(Err().Op("=").Qual(jsonPkg, "Unmarshal").Call(Id("requestBase").Dot("Params"), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(eg *Group) {
					eg.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("invalidParamsError"), Lit("invalid params: ").Op("+").Err().Dot("Error").Call(), Nil()))
				})
			})
			bg.Add(r.applyOverlayFromContext(srcFile, typeGen, method, r.jsonRPCArgErrReturn(), true))
			bg.Line()
			bg.ListFunc(func(lg *Group) {
				for _, ret := range resultsWithoutError(method) {
					lg.Id("response").Dot(toCamel(ret.Name))
				}
				lg.Err()
			}).Op("=").Id("http").Dot("svc").Dot(method.Name).CallFunc(func(cg *Group) {
				cg.Id(VarNameCtx)
				for _, arg := range argsWithoutContext(method) {
					argCode := Id("request").Dot(toCamel(arg.Name))
					if arg.IsEllipsis {
						argCode.Op("...")
					}
					cg.Add(argCode)
				}
			})
			bg.If(Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.If(Id("http").Dot("errorHandler").Op("!=").Nil()).Block(
					Err().Op("=").Id("http").Dot("errorHandler").Call(Err()),
				)
				ig.Id("code").Op(":=").Id("internalError")
				ig.If(List(Id("errCoder"), Id("ok")).Op(":=").Err().Op(".").Call(Id("withErrorCode")).Op(";").Id("ok")).Block(
					Id("code").Op("=").Id("errCoder").Dot("Code").Call(),
				)
				ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("code"), Id("sanitizeErrorMessage").Call(Err()), Nil()))
			})
			bg.Id("responseBase").Op("=").Op("&").Id("baseJsonRPC").Values(Dict{
				Id("ID"):      Id("requestBase").Dot("ID"),
				Id("Version"): Id("Version"),
			})
			resp := Id("response")
			if len(resultsWithoutError(method)) == 1 && model.IsAnnotationSet(r.project, r.contract, method, nil, TagHttpEnableInlineSingle) {
				resp = Id("response").Dot(toCamel(resultsWithoutError(method)[0].Name))
			}
			bg.If(List(Id("responseBase").Dot("Result"), Err()).Op("=").Qual(jsonPkg, "Marshal").Call(resp).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("parseError"), Lit("response body could not be encoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
			})
			bg.Line()
			bg.Return()
		})
}

func (r *contractRenderer) serveMethodFunc(jsonPkg string) Code {

	return Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).
		Id("_serveMethod").
		ParamsFunc(func(pg *Group) {
			pg.Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")
			pg.Id("methodName").String()
			pg.Id("methodHandler").Id("methodJsonRPCWithFiber")
		}).
		Params(Err().Error()).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Id("methodHTTP").Op(":=").Id(VarNameFtx).Dot("Method").Call()
			bg.If(Id("methodHTTP").Op("!=").Qual(PackageFiber, "MethodPost")).BlockFunc(func(ig *Group) {
				ig.Id(VarNameFtx).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(PackageFiber, "StatusMethodNotAllowed"))
				ig.If(List(Id("_"), Err()).Op("=").Id(VarNameFtx).Dot("WriteString").Call(Lit("only POST method supported")).Op(";").Err().Op("!=").Nil()).Block(
					Return(),
				)
			})
			bg.Var().Id("request").Id("baseJsonRPC")
			bg.Var().Id("response").Op("*").Id("baseJsonRPC")
			bg.List(Id("body"), Err()).Op(":=").Qual("io", "ReadAll").Call(Id("ensureBodyReader").Call(Id(VarNameFtx).Dot("Context").Call().Dot("RequestBodyStream").Call()))
			bg.If(Err().Op("!=").Nil()).Block(
				Return().Id("sendHTTPError").Call(Id(VarNameFtx), Qual(PackageFiber, "StatusBadRequest"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call()),
			)
			bg.If(Err().Op("=").Qual(jsonPkg, "Unmarshal").Call(Id("body"), Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Return().Id("sendHTTPError").Call(Id(VarNameFtx), Qual(PackageFiber, "StatusBadRequest"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call())
			})
			bg.If(Err().Op("=").Id("validateJsonRPCRequest").Call(Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.Return().Id("sendHTTPError").Call(Id(VarNameFtx), Qual(PackageFiber, "StatusBadRequest"), Lit("invalid JSON-RPC request: ").Op("+").Err().Dot("Error").Call())
			})
			bg.If(Id(VarNameFtx).Dot("UserContext").Call().Dot("Err").Call().Op("!=").Nil()).Block(
				Return().Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("invalidRequestError"), Lit("request context cancelled"), Nil())),
			)
			bg.Id("methodNameOrigin").Op(":=").Id("request").Dot("Method")
			bg.Id("method").Op(":=").Id("toLowercaseMethod").Call(Id("request").Dot("Method"))

			bg.If(Id("method").Op("!=").Lit("").Op("&&").Id("method").Op("!=").Id("methodName")).BlockFunc(func(ig *Group) {
				ig.Return().Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("methodNotFoundError"), Lit("invalid method ").Op("+").Id("methodNameOrigin"), Nil()))
			})
			bg.Id("response").Op("=").Id("methodHandler").Call(Id(VarNameFtx), Id("request"))
			bg.If(Id("response").Op("!=").Nil()).Block(
				Return().Id("sendResponse").Call(Id(VarNameFtx), Id("response")),
			)
			bg.Return()
		})
}

func (r *contractRenderer) serviceServeBatchFunc(jsonPkg string) Code {

	srvctxPkgPath := fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir))
	return Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).
		Id("serveBatch").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Id("err").Error()).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Var().Id("single").Bool()
			bg.Var().Id("requests").Op("[]").Id("baseJsonRPC")
			bg.Id("clientID").Op(":=").Qual(srvctxPkgPath, "GetClientID").Call(Id(VarNameFtx).Dot("UserContext").Call())
			bg.Id("methodHTTP").Op(":=").Id(VarNameFtx).Dot("Method").Call()
			bg.If(Id("methodHTTP").Op("!=").Qual(PackageFiber, "MethodPost")).BlockFunc(func(ig *Group) {
				ig.Id(VarNameFtx).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(PackageFiber, "StatusMethodNotAllowed"))
				ig.If(List(Id("_"), Err()).Op("=").Id(VarNameFtx).Dot("WriteString").Call(Lit("only POST method supported")).Op(";").Err().Op("!=").Nil()).Block(
					Return(),
				)
				ig.If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(
					Id("http").Dot("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("method_not_allowed"), Id("clientID")).Dot("Inc").Call(),
				)
				ig.Return()
			})
			bg.Id("bodyStream").Op(":=").Id("ensureBodyReader").Call(Id(VarNameFtx).Dot("Context").Call().Dot("RequestBodyStream").Call())
			bg.List(Id("firstByte"), Err()).Op(":=").Id("readUntilFirstNonWhitespace").Call(Id("bodyStream"))
			bg.If(Err().Op("!=").Nil().Op("&&").Err().Op("!=").Qual("io", "EOF")).BlockFunc(func(ig *Group) {
				ig.If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(
					Id("http").Dot("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("parse_error"), Id("clientID")).Dot("Inc").Call(),
				)
				ig.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil())))
			})
			bg.If(Id("firstByte").Op("==").Lit(0)).BlockFunc(func(ig *Group) {
				ig.If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(
					Id("http").Dot("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("empty_body"), Id("clientID")).Dot("Inc").Call(),
				)
				ig.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("parseError"), Lit("request body could not be decoded: empty body"), Nil())))
			})
			bg.Id("r").Op(":=").Qual("io", "MultiReader").Call(Qual(PackageBytes, "NewReader").Call(Index().Byte().Values(Id("firstByte"))), Id("bodyStream"))
			bg.Switch(Id("firstByte")).BlockFunc(func(sg *Group) {
				sg.Case(Lit(123)).BlockFunc(func(cg *Group) {
					cg.Var().Id("request").Id("baseJsonRPC")
					cg.If(Err().Op("=").Qual(jsonPkg, "NewDecoder").Call(Id("r")).Dot("Decode").Call(Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(dg *Group) {
						dg.If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(
							Id("http").Dot("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("parse_error"), Id("clientID")).Dot("Inc").Call(),
						)
						dg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil())))
					})
					cg.Id("single").Op("=").True()
					cg.Id("requests").Op("=").Append(Id("requests"), Id("request"))
				})
				sg.Case(Lit(91)).BlockFunc(func(cg *Group) {
					cg.If(Err().Op("=").Qual(jsonPkg, "NewDecoder").Call(Id("r")).Dot("Decode").Call(Op("&").Id("requests")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(dg *Group) {
						dg.If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(
							Id("http").Dot("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("parse_error"), Id("clientID")).Dot("Inc").Call(),
						)
						dg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("parseError"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil())))
					})
					cg.If(Len(Id("requests")).Op("==").Lit(0)).BlockFunc(func(eg *Group) {
						eg.If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(
							Id("http").Dot("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("invalid_request"), Id("clientID")).Dot("Inc").Call(),
						)
						eg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("invalidRequestError"), Lit("empty batch request"), Nil())))
					})
				})
				sg.Default().BlockFunc(func(dg *Group) {
					dg.If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(
						Id("http").Dot("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("parse_error"), Id("clientID")).Dot("Inc").Call(),
					)
					dg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("makeErrorResponseJsonRPC").Call(Nil(), Id("parseError"), Lit("request body could not be decoded: expected { or ["), Nil())))
				})
			})
			bg.If(Len(Id("requests")).Op(">").Id("http").Dot("srv").Dot("maxBatchSize")).BlockFunc(func(ig *Group) {
				ig.If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(
					Id("http").Dot("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("batch_size_exceeded"), Id("clientID")).Dot("Inc").Call(),
				)
				ig.Return(Id("sendHTTPError").Call(Id(VarNameFtx), Qual(PackageFiber, "StatusBadRequest"), Lit("batch size exceeded")))
			})
			bg.If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(
				Id("http").Dot("srv").Dot("metrics").Dot("BatchSize").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit(r.batchPath()), Id("clientID")).Dot("Observe").Call(Id("float64").Call(Len(Id("requests")))),
			)
			bg.If(Id("single")).BlockFunc(func(ig *Group) {
				ig.If(Err().Op("=").Id("validateJsonRPCRequest").Call(Id("requests").Op("[").Lit(0).Op("]")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(vg *Group) {
					vg.If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(
						Id("http").Dot("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("invalid_request"), Id("clientID")).Dot("Inc").Call(),
					)
					vg.Return(Id("sendHTTPError").Call(Id(VarNameFtx), Qual(PackageFiber, "StatusBadRequest"), Lit("invalid JSON-RPC request: ").Op("+").Err().Dot("Error").Call()))
				})
				ig.Defer().Func().Params().Block(
					If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(
						Id("http").Dot("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("ok"), Id("clientID")).Dot("Inc").Call(),
					),
				).Call()
				ig.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("http").Dot("doSingleBatch").
					Call(Id(VarNameFtx).Dot("UserContext").Call(), Id("requests").Op("[").Lit(0).Op("]")),
				))
			})
			bg.Defer().Func().Params().Block(
				If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(
					Id("http").Dot("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("ok"), Id("clientID")).Dot("Inc").Call(),
				),
			).Call()
			bg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("http").Dot("srv").Dot("doBatch").
				Call(Id(VarNameFtx), Id("requests")),
			))
		})
}

func (r *contractRenderer) serviceSingleBatchFunc(typeGen *types.Generator) Code {

	return Func().Params(Id("http").Op("*").Id("http"+r.contract.Name)).
		Id("doSingleBatch").
		Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("request").Id("baseJsonRPC")).
		Params(Id("response").Op("*").Id("baseJsonRPC")).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Var().Err().Error()
			bg.If(Err().Op("=").Id("validateJsonRPCRequest").Call(Id("request")).Op(";").Err().Op("!=").Nil()).Block(
				Return(Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("invalidRequestError"), Lit("invalid JSON-RPC request: ").Op("+").Err().Dot("Error").Call(), Nil())),
			)
			bg.If(Id(VarNameCtx).Dot("Err").Call().Op("!=").Nil()).Block(
				Return(Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("invalidRequestError"), Lit("request context cancelled"), Nil())),
			)
			bg.Id("methodNameOrigin").Op(":=").Id("request").Dot("Method")
			bg.Id("method").Op(":=").Id("toLowercaseMethod").Call(Id("request").Dot("Method"))
			bg.Switch(Id("method")).BlockFunc(func(sg *Group) {
				for _, method := range r.contract.Methods {
					if !r.methodIsJsonRPC(method) {
						continue
					}
					methodNameLC := strings.ToLower(method.Name)
					sg.Case(Lit(methodNameLC)).Block(
						Return(Id("http").Dot(toLowerCamel(method.Name)+"WithContext").Call(Id(VarNameCtx), Id("request"))),
					)
				}
				sg.Default().BlockFunc(func(dg *Group) {
					dg.Return(Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("methodNotFoundError"), Lit("invalid method '").Op("+").Id("methodNameOrigin").Op("+").Lit("'"), Nil()))
				})
			})
		})
}

func (r *contractRenderer) jsonRPCArgErrReturn() func(arg, header string) *Statement {

	return func(arg, header string) *Statement {
		return Line().If(Err().Op("!=").Nil()).Block(
			Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("invalidParamsError"), Lit("http header could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil())),
		)
	}
}
