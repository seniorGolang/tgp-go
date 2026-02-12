// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (r *transportRenderer) RenderTransportJsonRPC() error {

	jsonrpcPath := path.Join(r.outDir, "jsonrpc.go")

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	r.renderJsonRPCImports(&srcFile)
	r.renderJsonRPCTypes(&srcFile)
	r.renderJsonRPCConstants(&srcFile)
	r.renderJsonRPCFunctions(&srcFile)

	return srcFile.Save(jsonrpcPath)
}

func (r *transportRenderer) renderJsonRPCImports(srcFile *GoFile) {

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
	srcFile.ImportName(jsonPkg, "json")
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir)), "srvctx")
}

func (r *transportRenderer) renderJsonRPCTypes(srcFile *GoFile) {

	srcFile.Line().Add(r.jsonrpcConstants())
	srcFile.Add(r.idJsonRPC()).Line()
	srcFile.Add(r.baseJsonRPC()).Line()
	srcFile.Add(r.errorJsonRPC()).Line()
	srcFile.Line()
	srcFile.Add(r.requestOverlayKeyType()).Line()
	srcFile.Add(r.requestOverlayStructType()).Line()
	srcFile.Add(r.requestOverlayGetMethod()).Line()
	srcFile.Add(r.requestOverlayGetterType()).Line()
	srcFile.Add(r.requestOverlayFromFiberFunc())
	srcFile.Add(r.requestOverlayMiddlewareFunc())
}

func (r *transportRenderer) renderJsonRPCConstants(srcFile *GoFile) {

	// Константы уже добавлены в renderJsonRPCTypes
}

func (r *transportRenderer) renderJsonRPCFunctions(srcFile *GoFile) {

	srcFile.Add(r.sendResponseJsonRPCFunc())
	srcFile.Line()
	srcFile.Line().Type().Id("methodJsonRPC").
		Func().
		Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("requestBase").Id("baseJsonRPC")).
		Params(Id("responseBase").Op("*").Id("baseJsonRPC"))
	srcFile.Line()
	srcFile.Line().Type().Id("methodJsonRPCWithFiber").
		Func().
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx"), Id("requestBase").Id("baseJsonRPC")).
		Params(Id("responseBase").Op("*").Id("baseJsonRPC"))
	srcFile.Line()
	srcFile.Line().Add(r.initJsonRPCMethodMap())
	srcFile.Line()
	srcFile.Add(r.serveBatchFunc())
	srcFile.Add(r.batchFunc())
	srcFile.Add(r.singleBatchFunc())
	srcFile.Line()
	srcFile.Line()
	srcFile.Add(r.toLowercaseMethodFunc())
	srcFile.Add(r.sanitizeErrorMessageFunc())
	srcFile.Add(r.validateJsonRPCRequestFunc())
	srcFile.Line()
	srcFile.Line().Add(r.makeErrorResponseJsonRPCFunc())
	srcFile.Line().Add(r.readUntilFirstNonWhitespaceFunc())
}

func (r *transportRenderer) jsonrpcConstants() Code {

	return Const().Op("(").
		Line().Id("defaultMaxBatchSize").Op("=").Lit(100).
		Line().Id("defaultMaxParallelBatch").Op("=").Lit(10).
		Line().Id("maxPeekBytes").Op("=").Lit(16).
		Line().Id("Version").Op("=").Lit("2.0").
		Line().Id("syncHeader").Op("=").Lit("X-Sync-On").
		Line().Id("parseError").Op("=").Lit(-32700).
		Line().Id("invalidRequestError").Op("=").Lit(-32600).
		Line().Id("methodNotFoundError").Op("=").Lit(-32601).
		Line().Id("invalidParamsError").Op("=").Lit(-32602).
		Line().Id("internalError").Op("=").Lit(-32603).
		Op(")")
}

func (r *transportRenderer) idJsonRPC() Code {

	jsonPkg := r.getPackageJSON()
	return Type().Id("idJsonRPC").Op("=").Qual(jsonPkg, "RawMessage")
}

func (r *transportRenderer) baseJsonRPC() Code {

	jsonPkg := r.getPackageJSON()
	return Type().Id("baseJsonRPC").StructFunc(func(tg *Group) {
		tg.Id("ID").Id("idJsonRPC").Tag(map[string]string{"json": "id"})
		tg.Id("Version").Id("string").Tag(map[string]string{"json": "jsonrpc"})
		tg.Id("Method").Id("string").Tag(map[string]string{"json": "method,omitempty"})
		tg.Id("Error").Op("*").Id("errorJsonRPC").Tag(map[string]string{"json": "error,omitempty"})
		tg.Id("Params").Qual(jsonPkg, "RawMessage").Tag(map[string]string{"json": "params,omitempty"})
		tg.Id("Result").Qual(jsonPkg, "RawMessage").Tag(map[string]string{"json": "result,omitempty"})
	})
}

func (r *transportRenderer) errorJsonRPC() Code {

	return Type().Id("errorJsonRPC").Struct(
		Id("Code").Id("int").Tag(map[string]string{"json": "code"}),
		Id("Message").Id("string").Tag(map[string]string{"json": "message"}),
		Id("Data").Id("any").Tag(map[string]string{"json": "data,omitempty"}),
	).Line().Func().Params(Err().Id("errorJsonRPC")).
		Id("Error").
		Params().
		Params(String()).
		Block(
			Return(Err().Dot("Message")),
		)
}

func (r *transportRenderer) requestOverlayKeyType() Code {

	return Line().
		Type().Id("requestOverlayKey").Struct().Line().
		Var().Id("keyRequestOverlay").Op("=").Id("requestOverlayKey").Values()
}

func (r *transportRenderer) requestOverlayStructType() Code {

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

func (r *transportRenderer) requestOverlayGetMethod() Code {

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

func (r *transportRenderer) requestOverlayGetterType() Code {

	return Type().Id("requestOverlayGetter").Func().Params().Params(Id("requestOverlay"))
}

func (r *transportRenderer) requestOverlayFromFiberFunc() Code {

	headerNames, cookieNames := r.jsonRPCUsedOverlayKeys()
	return Line().Func().Id("requestOverlayFromFiber").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Id("requestOverlay")).
		BlockFunc(func(block *Group) {
			block.Id("o").Op(":=").Id("requestOverlay").Values()
			for _, name := range headerNames {
				block.Id("o").Dot(overlayKeyToFieldName(name)).Op("=").Qual(PackageStrings, "Clone").Call(Id("string").Call(Id(VarNameFtx).Dot("Request").Call().Dot("Header").Dot("Peek").Call(Lit(name))))
			}
			for _, name := range cookieNames {
				block.Id("o").Dot(overlayKeyToFieldName(name)).Op("=").Qual(PackageStrings, "Clone").Call(Id(VarNameFtx).Dot("Cookies").Call(Lit(name)))
			}
			block.Return(Id("o"))
		})
}

func (r *transportRenderer) requestOverlayMiddlewareFunc() Code {

	return Line().Func().Id("requestOverlayMiddleware").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Error()).
		BlockFunc(func(bg *Group) {
			bg.Id(VarNameFtx).Dot("SetUserContext").Call(
				Qual(PackageContext, "WithValue").Call(
					Id(VarNameFtx).Dot("UserContext").Call(),
					Id("keyRequestOverlay"),
					Id("requestOverlayGetter").Call(Func().Params().Params(Id("requestOverlay")).Block(Return(Id("requestOverlayFromFiber").Call(Id(VarNameFtx))))),
				),
			)
			bg.Return(Id(VarNameFtx).Dot("Next").Call())
		})
}
