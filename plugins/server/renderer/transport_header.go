// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (r *transportRenderer) RenderTransportHeader() error {

	headerPath := path.Join(r.outDir, "header.go")
	srvctxPkgPath := fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir))

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(PackageErrors, "errors")
	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(srvctxPkgPath, "srvctx")
	jsonPkg := r.getPackageJSON()
	srcFile.ImportName(jsonPkg, "json")
	srcFile.ImportName(PackageFmt, "fmt")
	srcFile.ImportName(PackageStrings, "strings")

	r.renderHeaderTypes(&srcFile)
	r.renderHeaderHandler(&srcFile)
	r.renderHeaderValue(&srcFile, jsonPkg)
	r.renderHeaderValueInterface(&srcFile)

	return srcFile.Save(headerPath)
}

func (r *transportRenderer) renderHeaderTypes(srcFile *GoFile) {

	srcFile.Line().Type().Id("Header").Struct(
		Id("SpanKey").String(),
		Id("SpanValue").Any(),
		Id("RequestKey").String(),
		Id("RequestValue").Any(),
		Id("ResponseKey").String(),
		Id("ResponseValue").Any(),
		Id("LogKey").String(),
		Id("LogValue").Any(),
	).Line().
		Line().Type().Id("HeaderHandler").Func().Params(Id("value").String()).Params(Id("Header"))
}

func (r *transportRenderer) renderHeaderHandler(srcFile *GoFile) {

	srcFile.Line().Func().Params(Id("srv").Op("*").Id("Server")).
		Id("headersHandler").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Error()).
		BlockFunc(func(bg *Group) {
			bg.Line()
			// Ранний выход, если нет обработчиков
			bg.If(Len(Id("srv").Dot("headerHandlers")).Op("==").Lit(0)).Block(
				Return(Id(VarNameFtx).Dot("Next").Call()),
			)
			bg.Line()
			bg.Id("req").Op(":=").Id(VarNameFtx).Dot("Request").Call()
			bg.Id("resp").Op(":=").Id(VarNameFtx).Dot("Response").Call()
			bg.Id("ctx").Op(":=").Id(VarNameFtx).Dot("UserContext").Call()
			bg.Id("logger").Op(":=").Qual(fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir)), "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(Id("ctx"))
			bg.Line()
			bg.Var().Id("logAttrs").Index().Qual(PackageSlog, "Attr")
			bg.Id("updatedCtx").Op(":=").Id("ctx")
			bg.For(List(Id("headerName"), Id("handler")).Op(":=").Range().Id("srv").Dot("headerHandlers")).Block(
				Id("value").Op(":=").Id("req").Dot("Header").Dot("Peek").Call(Id("headerName")),
				Id("header").Op(":=").Id("handler").Call(Qual(PackageStrings, "Clone").Call(String().Call(Id("value")))),
				If(Id("header").Dot("RequestValue").Op("!=").Nil()).Block(
					Id("req").Dot("Header").Dot("Set").Call(Id("header").Dot("RequestKey"), Id("headerValue").Call(Id("header").Dot("RequestValue"))),
				),
				If(Id("header").Dot("ResponseValue").Op("!=").Nil()).Block(
					Id("resp").Dot("Header").Dot("Set").Call(Id("header").Dot("ResponseKey"), Id("headerValue").Call(Id("header").Dot("ResponseValue"))),
				),
				If(Id("header").Dot("LogValue").Op("!=").Nil()).Block(
					If(Id("logger").Op("!=").Nil()).Block(
						Id("logAttrs").Op("=").Append(Id("logAttrs"), Qual(PackageSlog, "Any").Call(Id("header").Dot("LogKey"), Id("header").Dot("LogValue"))),
					),
				),
			)
			bg.If(Len(Id("logAttrs")).Op(">").Lit(0)).Block(
				If(Id("logger").Op("!=").Nil()).Block(
					Id("args").Op(":=").Make(Index().Any(), Lit(0), Len(Id("logAttrs"))),
					For(List(Id("_"), Id("attr")).Op(":=").Range().Id("logAttrs")).Block(
						Id("args").Op("=").Append(Id("args"), Id("attr")),
					),
					Id("requestLogger").Op(":=").Id("logger").Dot("With").Call(Id("args").Op("...")),
					Id("updatedCtx").Op("=").Qual(fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir)), "WithCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(Id("updatedCtx"), Id("requestLogger")),
				),
			)
			bg.If(Id("updatedCtx").Op("!=").Id("ctx")).Block(
				Id(VarNameFtx).Dot("SetUserContext").Call(Id("updatedCtx")),
			)
			bg.Return(Id(VarNameFtx).Dot("Next").Call())
		})
}

func (r *transportRenderer) renderHeaderValue(srcFile *GoFile, jsonPkg string) {

	srcFile.Line().Func().Id("headerValue").
		Params(Id("src").Any()).
		Params(Id("value").String()).
		Block(
			Line(),
			If(List(Id("v"), Id("ok")).Op(":=").Id("src").Assert(String()).Op(";").Id("ok")).Block(
				Return(Id("v")),
			),
			If(List(Id("v"), Id("ok")).Op(":=").Id("src").Assert(Id("iHeaderValue")).Op(";").Id("ok")).Block(
				Return(Id("v").Dot("Header").Call()),
			),
			If(List(Id("v"), Id("ok")).Op(":=").Id("src").Assert(Qual(PackageFmt, "Stringer")).Op(";").Id("ok")).Block(
				Return(Id("v").Dot("String").Call()),
			),
			List(Id("bytes"), Id("err")).Op(":=").Qual(jsonPkg, "Marshal").Call(Id("src")),
			If(Id("err").Op("!=").Nil()).Block(
				Return(Qual(PackageFmt, "Sprint").Call(Id("src"))),
			),
			Return(String().Call(Id("bytes"))),
		)
}

func (r *transportRenderer) renderHeaderValueInterface(srcFile *GoFile) {

	srcFile.Line().Type().Id("iHeaderValue").Interface(
		Id("Header").Params().Params(String()),
	).Line().
		Line().Type().Id("cookieType").Interface(
		Id("Cookie").Params().Params(Qual(PackageFiber, "Cookie")),
	)
}
