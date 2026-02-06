// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (r *transportRenderer) RenderTransportFiber() error {

	fiberPath := path.Join(r.outDir, "fiber.go")
	srvctxPkgPath := fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir))

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(PackageErrors, "errors")
	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(srvctxPkgPath, "srvctx")
	srcFile.ImportName(PackageFmt, "fmt")
	srcFile.ImportName("runtime/debug", "debug")

	srcFile.Line().Const().Id("logLevelHeader").Op("=").Lit("X-Log-Level")

	srcFile.Line().Type().Id("levelHandler").Struct(
		Id("handler").Qual(PackageSlog, "Handler"),
		Id("level").Op("*").Qual(PackageSlog, "LevelVar"),
	)
	srcFile.Line().Func().Params(Id("h").Op("*").Id("levelHandler")).
		Id("Enabled").
		Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("level").Qual(PackageSlog, "Level")).
		Bool().
		Block(
			Return(Id("level").Op(">=").Id("h").Dot("level").Dot("Level").Call().Op("&&").Id("h").Dot("handler").Dot("Enabled").Call(Id(VarNameCtx), Id("level"))),
		)
	srcFile.Line().Func().Params(Id("h").Op("*").Id("levelHandler")).
		Id("Handle").
		Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("r").Qual(PackageSlog, "Record")).
		Error().
		Block(
			Return(Id("h").Dot("handler").Dot("Handle").Call(Id(VarNameCtx), Id("r"))),
		)
	srcFile.Line().Func().Params(Id("h").Op("*").Id("levelHandler")).
		Id("WithAttrs").
		Params(Id("attrs").Index().Qual(PackageSlog, "Attr")).
		Qual(PackageSlog, "Handler").
		Block(
			Return(Op("&").Id("levelHandler").Values(Dict{
				Id("handler"): Id("h").Dot("handler").Dot("WithAttrs").Call(Id("attrs")),
				Id("level"):   Id("h").Dot("level"),
			})),
		)
	srcFile.Line().Func().Params(Id("h").Op("*").Id("levelHandler")).
		Id("WithGroup").
		Params(Id("name").String()).
		Qual(PackageSlog, "Handler").
		Block(
			Return(Op("&").Id("levelHandler").Values(Dict{
				Id("handler"): Id("h").Dot("handler").Dot("WithGroup").Call(Id("name")),
				Id("level"):   Id("h").Dot("level"),
			})),
		)

	r.renderFiberLogger(&srcFile)
	r.renderFiberRecover(&srcFile)

	return srcFile.Save(fiberPath)
}

func (r *transportRenderer) renderFiberLogger(srcFile *GoFile) {

	srvctxPkgPath := fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir))
	fromCtxLogger := func(ctx Code) *Statement {

		return Qual(srvctxPkgPath, "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(ctx)
	}
	withCtxLogger := func(ctx, logger Code) Code {

		return Qual(srvctxPkgPath, "WithCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(ctx, logger)
	}

	srcFile.Line().Func().Params(Id("srv").Op("*").Id("Server")).
		Id("setLogger").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Error().
		BlockFunc(func(bg *Group) {
			bg.Id("ctx").Op(":=").Id(VarNameFtx).Dot("UserContext").Call()
			bg.If(fromCtxLogger(Id("ctx")).Op("!=").Nil()).Block(
				Return(Id(VarNameFtx).Dot("Next").Call()),
			)
			bg.Id("levelName").Op(":=").String().Call(Id(VarNameFtx).Dot("Request").Call().Dot("Header").Dot("Peek").Call(Id("logLevelHeader")))
			bg.If(Id("levelName").Op("==").Lit("")).Block(
				Id(VarNameFtx).Dot("SetUserContext").Call(withCtxLogger(Id("ctx"), Id("srv").Dot("log"))),
				Return(Id(VarNameFtx).Dot("Next").Call()),
			)
			bg.Var().Id("level").Qual(PackageSlog, "Level")
			bg.Switch(Id("levelName")).Block(
				Case(Lit("debug"), Lit("DEBUG")).Block(Id("level").Op("=").Qual(PackageSlog, "LevelDebug")),
				Case(Lit("info"), Lit("INFO")).Block(Id("level").Op("=").Qual(PackageSlog, "LevelInfo")),
				Case(Lit("warn"), Lit("WARN")).Block(Id("level").Op("=").Qual(PackageSlog, "LevelWarn")),
				Case(Lit("error"), Lit("ERROR")).Block(Id("level").Op("=").Qual(PackageSlog, "LevelError")),
				Default().Block(
					Id(VarNameFtx).Dot("SetUserContext").Call(withCtxLogger(Id("ctx"), Id("srv").Dot("log"))),
					Return(Id(VarNameFtx).Dot("Next").Call()),
				),
			)
			bg.Id("levelVar").Op(":=").Op("new").Call(Qual(PackageSlog, "LevelVar"))
			bg.Id("levelVar").Dot("Set").Call(Id("level"))
			bg.Id("baseHandler").Op(":=").Id("srv").Dot("log").Dot("Handler").Call()
			bg.Id("requestLogger").Op(":=").Qual(PackageSlog, "New").Call(Op("&").Id("levelHandler").Values(Dict{
				Id("handler"): Id("baseHandler"),
				Id("level"):   Id("levelVar"),
			}))
			bg.Id(VarNameFtx).Dot("SetUserContext").Call(withCtxLogger(Id("ctx"), Id("requestLogger")))
			bg.Return(Id(VarNameFtx).Dot("Next").Call())
		})
}

func (r *transportRenderer) renderFiberRecover(srcFile *GoFile) {

	srcFile.Line().Func().Id("recoverHandler").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Error().
		BlockFunc(func(bg *Group) {
			bg.Defer().Func().Params().BlockFunc(func(dg *Group) {
				dg.If(Id("r").Op(":=").Recover().Op(";").Id("r").Op("!=").Nil()).BlockFunc(func(ig *Group) {
					ig.List(Err(), Id("ok")).Op(":=").Id("r").Op(".").Call(Error())
					ig.If(Op("!").Id("ok")).Block(
						Err().Op("=").Qual(PackageErrors, "New").Call(Qual(PackageFmt, "Sprintf").Call(Lit("%v"), Id("r"))),
					)
					ig.If(List(Id("server"), Id("ok")).Op(":=").Id(VarNameFtx).Dot("Locals").Call(Lit("server")).Assert(Op("*").Id("Server")).Op(";").Id("ok").Op("&&").Id("server").Dot("metrics").Op("!=").Nil()).Block(
						Id("server").Dot("metrics").Dot("PanicsTotal").Dot("Inc").Call(),
					)
					ig.If(Id("logger").Op(":=").Qual(fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir)), "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(Id(VarNameFtx).Dot("UserContext").Call()).Op(";").Id("logger").Op("!=").Nil()).Block(
						Id("logger").Dot("Error").Call(Lit("panic occurred"),
							Qual(PackageSlog, "Any").Call(Lit("error"), Qual(PackageErrors, "Wrap").Call(Err(), Lit("recover"))),
							Qual(PackageSlog, "String").Call(Lit("method"), Id(VarNameFtx).Dot("Method").Call()),
							Qual(PackageSlog, "String").Call(Lit("path"), Id(VarNameFtx).Dot("OriginalURL").Call()),
							Qual(PackageSlog, "String").Call(Lit("stack"), Id("string").Call(Qual("runtime/debug", "Stack").Call())),
						),
					)
					ig.Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusInternalServerError"))
					ig.Id("_").Op("=").Id(VarNameFtx).Dot("JSON").Call(Map(String()).String().Values(Dict{Lit("message"): Lit("internal server error")}))
				})
			}).Call()
			bg.Return(Id(VarNameFtx).Dot("Next").Call())
		})
}
