// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive
)

func (r *transportRenderer) RenderTransportJSONRPC() (err error) {

	jsonrpcPath := path.Join(r.outDir, "jsonrpc.go")

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	r.renderJSONRPCImports(&srcFile)
	r.renderJSONRPCTypes(&srcFile)
	r.renderJSONRPCConstants(&srcFile)
	r.renderJSONRPCFunctions(&srcFile)

	return srcFile.Save(jsonrpcPath)
}

func (r *transportRenderer) renderJSONRPCImports(srcFile *GoFile) {

	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName("io", "io")
	srcFile.ImportName(PackageSync, "sync")
	srcFile.ImportName(PackageBytes, "bytes")
	srcFile.ImportName(PackageStrings, "strings")
	srcFile.ImportName(PackageErrors, "errors")
	srcFile.ImportName(PackageFmt, "fmt")
	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(PackageStrconv, "strconv")
	jsonPkg := r.getPackageJSON()
	srcFile.ImportName(jsonPkg, keyJSON)
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(r.pkgPath(r.outDir)+"/srvctx", "srvctx")
	srcFile.ImportName(r.pkgPath(r.outDir)+"/stream", "stream")
}

func (r *transportRenderer) renderJSONRPCTypes(srcFile *GoFile) {

	srcFile.Line().Add(r.jsonrpcConstants())
	srcFile.Add(r.idJSONRPC()).Line()
	srcFile.Add(r.baseJSONRPC()).Line()
	srcFile.Add(r.errorJSONRPC()).Line()
	srcFile.Add(r.batchResultType()).Line()
	if !r.jsonRPCNeedsOverlay(nil) {
		return
	}
	srcFile.Line()
	srcFile.Add(r.requestOverlayKeyType()).Line()
	srcFile.Add(r.requestOverlayStructType()).Line()
	srcFile.Add(r.requestOverlayGetMethod()).Line()
	srcFile.Add(r.requestOverlayFromFiberFunc())
}

func (r *transportRenderer) renderJSONRPCConstants(_srcFile *GoFile) {

}

func (r *transportRenderer) renderJSONRPCFunctions(srcFile *GoFile) {

	srcFile.Add(r.sendResponseJSONRPCFunc())
	srcFile.Line()
	srcFile.Add(r.sendJSONRPCResponseFunc())
	srcFile.Line()
	srcFile.Line().Type().Id("methodJsonRPC").
		Func().
		Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("requestBase").Id("baseJSONRPC")).
		Params(Id("responseBase").Op("*").Id("baseJSONRPC"))
	srcFile.Line()
	srcFile.Line().Type().Id("methodJsonRPCWithFiber").
		Func().
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx"), Id("requestBase").Id("baseJSONRPC")).
		Params(Id("responseBase").Op("*").Id("baseJSONRPC"))
	srcFile.Line()
	srcFile.Line().Add(r.initJSONRPCMethodMaps())
	srcFile.Line()
	srcFile.Add(r.batchHandlerFunc())
	srcFile.Add(r.serveBatchFunc())
	srcFile.Add(r.batchFunc())
	srcFile.Add(r.singleBatchFunc())
	srcFile.Line()
	srcFile.Line()
	srcFile.Add(r.toLowercaseMethodFunc())
	srcFile.Add(r.validateJSONRPCRequestFunc())
	srcFile.Line()
	srcFile.Line().Add(r.makeErrorResponseJSONRPCFunc())
}

func (r *transportRenderer) jsonrpcConstants() (c Code) {

	return Const().Op("(").
		Line().Id("defaultMaxBatchSize").Op("=").Lit(100).
		Line().Id("defaultMaxParallelBatch").Op("=").Lit(10).
		Line().Id("Version").Op("=").Lit("2.0").
		Line().Id("syncHeader").Op("=").Lit("X-Sync-On").
		Line().Id("parseError").Op("=").Lit(-32700).
		Line().Id("invalidRequestError").Op("=").Lit(-32600).
		Line().Id("methodNotFoundError").Op("=").Lit(-32601).
		Line().Id("invalidParamsError").Op("=").Lit(-32602).
		Line().Id("internalError").Op("=").Lit(-32603).
		Op(")")
}

func (r *transportRenderer) idJSONRPC() (c Code) {

	jsonPkg := r.getPackageJSON()
	return Type().Id("idJSONRPC").Op("=").Qual(jsonPkg, "RawMessage")
}

func (r *transportRenderer) baseJSONRPC() (c Code) {

	jsonPkg := r.getPackageJSON()
	return Type().Id("baseJSONRPC").StructFunc(func(tg *Group) {
		tg.Id("ID").Id("idJSONRPC").Tag(map[string]string{keyJSON: "id"})
		tg.Id("Version").Id(typeString).Tag(map[string]string{keyJSON: "jsonrpc"})
		tg.Id("Method").Id(typeString).Tag(map[string]string{keyJSON: "method,omitempty"})
		tg.Id("Error").Op("*").Id("errorJSONRPC").Tag(map[string]string{keyJSON: "error,omitempty"})
		tg.Id("Params").Qual(jsonPkg, "RawMessage").Tag(map[string]string{keyJSON: "params,omitempty"})
		tg.Id("Result").Qual(jsonPkg, "RawMessage").Tag(map[string]string{keyJSON: "result,omitempty"})
	})
}

func (r *transportRenderer) errorJSONRPC() (c Code) {

	return Type().Id("errorJSONRPC").Struct(
		Id("Code").Id("int").Tag(map[string]string{keyJSON: "code"}),
		Id("Message").Id(typeString).Tag(map[string]string{keyJSON: "message"}),
		Id("Data").Id("any").Tag(map[string]string{keyJSON: "data,omitempty"}),
	).Line().Func().Params(Err().Id("errorJSONRPC")).
		Id("Error").
		Params().
		Params(String()).
		Block(
			Return(Err().Dot("Message")),
		)
}

func (r *transportRenderer) batchResultType() (c Code) {

	return Type().Id("batchResult").Struct(
		Id("idx").Int(),
		Id("response").Op("*").Id("baseJSONRPC"),
	)
}

func (r *transportRenderer) requestOverlayKeyType() (c Code) {

	return Line().
		Type().Id("requestOverlayKey").Struct().Line().
		Var().Id("keyRequestOverlay").Op("=").Id("requestOverlayKey").Values()
}

func (r *transportRenderer) requestOverlayStructType() (c Code) {

	headerNames, cookieNames := r.jsonRPCUsedOverlayKeys()
	return Type().Id("requestOverlay").StructFunc(func(tg *Group) {
		for _, name := range headerNames {
			tg.Id(overlayKeyToFieldName(name)).String()
		}
		for _, name := range cookieNames {
			tg.Id(overlayKeyToFieldName(name)).String()
		}
	})
}

func (r *transportRenderer) requestOverlayGetMethod() (c Code) {

	headerNames, cookieNames := r.jsonRPCUsedOverlayKeys()
	cases := make([]Code, 0, len(headerNames)+len(cookieNames)+1)
	for _, name := range headerNames {
		cases = append(cases, Case(Lit(name)).Block(Return(Id("o").Dot(overlayKeyToFieldName(name)))))
	}
	for _, name := range cookieNames {
		cases = append(cases, Case(Lit(name)).Block(Return(Id("o").Dot(overlayKeyToFieldName(name)))))
	}
	cases = append(cases, Default().Block(Return(Lit(""))))
	return Func().Params(Id("o").Id("requestOverlay")).Id("Get").Params(Id("key").String()).Params(String()).Block(
		Switch(Id("key")).Block(cases...),
	)
}

func (r *transportRenderer) requestOverlayFromFiberFunc() (c Code) {

	headerNames, cookieNames := r.jsonRPCUsedOverlayKeys()
	return Line().Func().Id("requestOverlayFromFiber").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Id("overlay").Id("requestOverlay")).
		BlockFunc(func(block *Group) {
			for _, name := range headerNames {
				block.Id("overlay").Dot(overlayKeyToFieldName(name)).Op("=").Id(VarNameFtx).Dot("Get").Call(Lit(name))
			}
			for _, name := range cookieNames {
				block.Id("overlay").Dot(overlayKeyToFieldName(name)).Op("=").Id(VarNameFtx).Dot("Cookies").Call(Lit(name))
			}
			block.Return()
		})
}

func setRequestOverlayContext() (c Code) {

	return Id(VarNameFtx).Dot("SetUserContext").Call(
		Qual(PackageContext, "WithValue").Call(
			Id(VarNameFtx).Dot("UserContext").Call(),
			Id("keyRequestOverlay"),
			Id("requestOverlayFromFiber").Call(Id(VarNameFtx)),
		),
	)
}
