// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive

	"tgp/internal/model"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) RenderJSONRPC() (err error) {

	if err = r.pkgRenderTo("srvctx", r.outDir, r.srvctxTemplateData()); err != nil {
		return fmt.Errorf("render srvctx package: %w", err)
	}

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	jsonPkg := r.getPackageJSON()
	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(PackageErrors, "errors")
	srcFile.ImportName("io", "io")
	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(PackageSync, "sync")
	srcFile.ImportName(PackageBytes, "bytes")
	srcFile.ImportName(PackageTime, "time")
	srcFile.ImportName(r.contract.PkgPath, filepath.Base(r.contract.PkgPath))
	srcFile.ImportName(jsonPkg, keyJSON)
	srcFile.ImportName(PackageReflect, "reflect")
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(r.pkgPath(r.outDir)+"/srvctx", "srvctx")
	srcFile.ImportName(r.pkgPath(r.outDir)+"/stream", "stream")

	typeGen := types.NewGenerator(r.project, &srcFile)

	for _, method := range r.contract.Methods {
		if !r.methodIsJSONRPC(method) {
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

	err = srcFile.Save(path.Join(r.outDir, strings.ToLower(r.contract.Name)+"-jsonrpc.go"))
	return
}

func (r *contractRenderer) rpcMethodFuncWithFiberCtx(_srcFile *GoFile, _typeGen *types.Generator, method *model.Method, _jsonPkg string) (c Code) {

	return Func().Params(Id("http").Op("*").Id("http"+r.contract.Name)).
		Id(toLowerCamel(method.Name)).
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx"), Id("requestBase").Id("baseJSONRPC")).
		Params(Id("responseBase").Op("*").Id("baseJSONRPC")).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Return(Id("http").Dot(toLowerCamel(method.Name)+"WithContext").Call(Id(VarNameFtx).Dot("UserContext").Call(), Id("requestBase")))
		})
}

func (r *contractRenderer) rpcMethodFuncWithContext(srcFile *GoFile, typeGen *types.Generator, method *model.Method, jsonPkg string) (c Code) {

	return Func().Params(Id("http").Op("*").Id("http"+r.contract.Name)).
		Id(toLowerCamel(method.Name)+"WithContext").
		Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("requestBase").Id("baseJSONRPC")).
		Params(Id("responseBase").Op("*").Id("baseJSONRPC")).
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
				for _, target := range r.responseAssignmentTargets(method, "response") {
					lg.Add(target)
				}
				lg.Err()
			}).Op("=").Id("http").Dot("svc").Dot(method.Name).CallFunc(func(cg *Group) {
				cg.Id(VarNameCtx)
				for _, arg := range argsWithoutContext(method) {
					argCode := Id("request").Dot(r.requestStructFieldName(method, arg))
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
				ig.If(Id("code").Op("==").Id("internalError")).Block(
					If(Id("logger").Op(":=").Qual(r.pkgPath(r.outDir)+"/srvctx", "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(Id(VarNameCtx)).Op(";").Id("logger").Op("!=").Nil()).Block(
						Id("logger").Dot("Error").Call(Lit("JSON-RPC internal error"), Qual(PackageSlog, "Any").Call(Lit("error"), Err())),
					),
				)
				ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("code"), Err().Dot("Error").Call(), Nil()))
			})
			bg.Id("responseBase").Op("=").Op("&").Id("baseJSONRPC").Values(Dict{
				Id("ID"):      Id("requestBase").Dot("ID"),
				Id("Version"): Id("Version"),
			})
			if len(r.resultNamesExcludeFromBody(method)) > 0 {
				bg.Id("resultForMarshal").Op(":=").Add(r.resultForMarshalValues(method, "response"))
				bg.If(List(Id("responseBase").Dot("Result"), Err()).Op("=").Qual(jsonPkg, "Marshal").Call(Op("&").Id("resultForMarshal")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
					ig.If(Id("logger").Op(":=").Qual(r.pkgPath(r.outDir)+"/srvctx", "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(Id(VarNameCtx)).Op(";").Id("logger").Op("!=").Nil()).Block(
						Id("logger").Dot("Error").Call(Lit("JSON-RPC response marshal error"), Qual(PackageSlog, "Any").Call(Lit("error"), Err())),
					)
					ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("internalError"), Lit("response body could not be encoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
				})
			} else {
				bg.If(List(Id("responseBase").Dot("Result"), Err()).Op("=").Qual(jsonPkg, "Marshal").Call(r.responseMarshalArg(method, "response")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
					ig.If(Id("logger").Op(":=").Qual(r.pkgPath(r.outDir)+"/srvctx", "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(Id(VarNameCtx)).Op(";").Id("logger").Op("!=").Nil()).Block(
						Id("logger").Dot("Error").Call(Lit("JSON-RPC response marshal error"), Qual(PackageSlog, "Any").Call(Lit("error"), Err())),
					)
					ig.Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("internalError"), Lit("response body could not be encoded: ").Op("+").Err().Dot("Error").Call(), Nil()))
				})
			}
			bg.Line()
			bg.Return()
		})
}

func (r *contractRenderer) serveMethodFunc(jsonPkg string) (c Code) {

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
			if r.jsonRPCNeedsOverlay(r.contract) {
				bg.Add(setRequestOverlayContext())
			}
			bg.Id("methodHTTP").Op(":=").Id(VarNameFtx).Dot("Method").Call()
			bg.If(Id("methodHTTP").Op("!=").Qual(PackageFiber, "MethodPost")).BlockFunc(func(ig *Group) {
				ig.Id(VarNameFtx).Dot("Response").Call().Dot("SetStatusCode").Call(Qual(PackageFiber, "StatusMethodNotAllowed"))
				ig.If(List(Id("_"), Err()).Op("=").Id(VarNameFtx).Dot("WriteString").Call(Lit("only POST method supported")).Op(";").Err().Op("!=").Nil()).Block(
					Return(),
				)
			})
			bg.Var().Id("request").Id("baseJSONRPC")
			bg.Var().Id("response").Op("*").Id("baseJSONRPC")
			bg.Id("bodyStream").Op(":=").Id("decodeBodyReader").Call(Id(VarNameFtx))
			bg.If(Err().Op("=").Qual(jsonPkg, "NewDecoder").Call(Id("bodyStream")).Dot("Decode").Call(Op("&").Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.If(Qual(PackageErrors, "Is").Call(Err(), Id("errBodyTooLarge"))).Block(
					Return().Id("sendHTTPError").Call(Id(VarNameFtx), Qual(PackageFiber, "StatusRequestEntityTooLarge"), Lit("request body too large")),
				)
				ig.Return().Id("sendHTTPError").Call(Id(VarNameFtx), Qual(PackageFiber, "StatusBadRequest"), Lit("request body could not be decoded: ").Op("+").Err().Dot("Error").Call())
			})
			bg.If(Err().Op("=").Id("validateJsonRPCRequest").Call(Id("request")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(ig *Group) {
				ig.If(Id("request").Dot("ID").Op("==").Nil()).Block(
					Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusNoContent")),
					Return(Nil()),
				)
				ig.Return().Id("sendHTTPError").Call(Id(VarNameFtx), Qual(PackageFiber, "StatusBadRequest"), Lit("invalid JSON-RPC request: ").Op("+").Err().Dot("Error").Call())
			})
			bg.If(Id(VarNameFtx).Dot("UserContext").Call().Dot("Err").Call().Op("!=").Nil()).Block(
				Return().Id("sendJSONRPCResponse").Call(Id(VarNameFtx), Id("request").Dot("ID"), Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("invalidRequestError"), Lit("request context cancelled"), Nil())),
			)
			bg.Id("methodNameOrigin").Op(":=").Id("request").Dot("Method")
			bg.Id("method").Op(":=").Id("toLowercaseMethod").Call(Id("request").Dot("Method"))

			bg.If(Id("method").Op("!=").Lit("").Op("&&").Id("method").Op("!=").Id("methodName")).BlockFunc(func(ig *Group) {
				ig.Return().Id("sendJSONRPCResponse").Call(Id(VarNameFtx), Id("request").Dot("ID"), Id("makeErrorResponseJsonRPC").Call(Id("request").Dot("ID"), Id("methodNotFoundError"), Lit("invalid method ").Op("+").Id("methodNameOrigin"), Nil()))
			})
			bg.Id("response").Op("=").Id("methodHandler").Call(Id(VarNameFtx), Id("request"))
			bg.If(Id("response").Op("!=").Nil()).Block(
				Return().Id("sendJSONRPCResponse").Call(Id(VarNameFtx), Id("request").Dot("ID"), Id("response")),
			)
			bg.Return()
		})
}

func (r *contractRenderer) serviceServeBatchFunc(_jsonPkg string) (c Code) {

	srvctxPkgPath := r.pkgPath(r.outDir) + "/srvctx"
	streamPkg := r.pkgPath(r.outDir) + "/stream"
	servicePath := r.batchPath()
	return Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).
		Id("serveBatch").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Id("err").Error()).
		BlockFunc(func(bg *Group) {
			bg.Line()
			if r.jsonRPCNeedsOverlay(r.contract) {
				bg.Add(setRequestOverlayContext())
			}
			bg.Var().Id("single").Bool()
			bg.Var().Id("requests").Op("[]").Id("baseJSONRPC")
			bg.Id("clientID").Op(":=").Qual(srvctxPkgPath, "GetClientID").Call(Id(VarNameFtx).Dot("UserContext").Call())
			bg.Id("methods").Op(":=").Id("http").Dot("srv").Dot("jsonRPCMethodMaps").Index(Lit(servicePath))
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
			writeJSONRPCEnvelopeDecode(bg, streamPkg, Id("http").Dot("srv").Dot("maxBatchSize"), func(g *Group, stmt Code) {
				g.If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(stmt)
			}, func(reason string) Code {
				return Id("http").Dot("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit(reason), Id("clientID")).Dot("Inc").Call()
			})
			bg.If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(
				Id("http").Dot("srv").Dot("metrics").Dot("BatchSize").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit(servicePath), Id("clientID")).Dot("Observe").Call(Id("float64").Call(Len(Id("requests")))),
			)
			bg.If(Id("single")).BlockFunc(func(ig *Group) {
				ig.If(Err().Op("=").Id("validateJsonRPCRequest").Call(Id("requests").Op("[").Lit(0).Op("]")).Op(";").Err().Op("!=").Nil()).BlockFunc(func(vg *Group) {
					vg.If(Id("requests").Op("[").Lit(0).Op("]").Dot("ID").Op("==").Nil()).Block(
						Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusNoContent")),
						Return(Nil()),
					)
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
				ig.Return(Id("sendJSONRPCResponse").Call(Id(VarNameFtx), Id("requests").Op("[").Lit(0).Op("]").Dot("ID"), Id("http").Dot("srv").Dot("doSingleBatch").
					Call(Id(VarNameFtx).Dot("UserContext").Call(), Id("requests").Op("[").Lit(0).Op("]"), Id("methods")),
				))
			})
			bg.Defer().Func().Params().Block(
				If(Id("http").Dot("srv").Op("!=").Nil().Op("&&").Id("http").Dot("srv").Dot("metrics").Op("!=").Nil()).Block(
					Id("http").Dot("srv").Dot("metrics").Dot("EntryRequestsTotal").Dot("WithLabelValues").Call(Lit("json-rpc"), Lit("ok"), Id("clientID")).Dot("Inc").Call(),
				),
			).Call()
			bg.Return(Id("sendResponse").Call(Id(VarNameFtx), Id("http").Dot("srv").Dot("doBatch").
				Call(Id(VarNameFtx), Id("requests"), Id("methods")),
			))
		})
}

func (r *contractRenderer) jsonRPCArgErrReturn() func(arg, header string) []Code {

	return func(_arg, _header string) []Code {
		return []Code{
			Return(Id("makeErrorResponseJsonRPC").Call(Id("requestBase").Dot("ID"), Id("invalidParamsError"), Lit("http header could not be decoded: ").Op("+").Err().Dot("Error").Call(), Nil())),
		}
	}
}
