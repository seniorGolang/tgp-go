// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive
)

func (r *transportRenderer) RenderTransportFiber() (err error) {

	fiberPath := path.Join(r.outDir, "fiber.go")
	srvctxPkgPath := r.pkgPath(r.outDir) + "/srvctx"

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(srvctxPkgPath, "srvctx")
	srcFile.ImportName(PackageFmt, "fmt")
	srcFile.ImportName(PackageBytes, "bytes")
	srcFile.ImportName(PackageErrors, "errors")
	srcFile.ImportName("io", "io")
	srcFile.ImportName("runtime/debug", "debug")

	srcFile.Line().Add(r.ensureBodyReaderFunc())
	srcFile.Line().Add(r.errBodyTooLargeVar())
	srcFile.Line().Add(r.maxBytesReaderType())
	srcFile.Line().Add(r.limitBodyReaderFunc())
	srcFile.Line().Add(r.decodeBodyReaderFunc())
	srcFile.Line().Add(r.streamBodyReaderFunc())
	srcFile.Line().Add(r.bodyLimitFunc())
	srcFile.Line().Add(r.streamBodyLimitFunc())
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

	r.renderRequestLogger(&srcFile)
	r.renderFiberLogger(&srcFile)
	r.renderFiberRecover(&srcFile)

	return srcFile.Save(fiberPath)
}

func (r *transportRenderer) ensureBodyReaderFunc() Code {

	return Func().Id("ensureBodyReader").
		Params(Id("r").Qual("io", "Reader")).
		Params(Qual("io", "Reader")).
		Block(
			If(Id("r").Op("==").Nil()).Block(
				Return(Qual(PackageBytes, "NewReader").Call(Nil())),
			),
			Return(Id("r")),
		)
}

func (r *transportRenderer) errBodyTooLargeVar() Code {

	return Var().Id("errBodyTooLarge").Op("=").Qual(PackageErrors, "New").Call(Lit("request body too large"))
}

func (r *transportRenderer) maxBytesReaderType() Code {

	return Type().Id("maxBytesReader").Struct(
		Id("r").Qual("io", "Reader"),
		Id("n").Int64(),
	).Line().
		Func().Params(Id("l").Op("*").Id("maxBytesReader")).
		Id("Read").
		Params(Id("p").Index().Byte()).
		Params(Id("n").Int(), Err().Error()).
		Block(
			If(Id("l").Dot("n").Op("<").Lit(0)).Block(
				Return(Lit(0), Id("errBodyTooLarge")),
			),
			If(Int64().Call(Len(Id("p"))).Op(">").Id("l").Dot("n").Op("+").Lit(1)).Block(
				Id("p").Op("=").Id("p").Index(Empty(), Id("l").Dot("n").Op("+").Lit(1)),
			),
			List(Id("n"), Err()).Op("=").Id("l").Dot("r").Dot("Read").Call(Id("p")),
			Id("l").Dot("n").Op("-=").Int64().Call(Id("n")),
			If(Id("l").Dot("n").Op("<").Lit(0)).Block(
				Id("n").Op("+=").Int().Call(Id("l").Dot("n")),
				Err().Op("=").Id("errBodyTooLarge"),
			),
			Return(),
		)
}

func (r *transportRenderer) limitBodyReaderFunc() Code {

	return Func().Id("limitBodyReader").
		Params(Id("r").Qual("io", "Reader"), Id("limit").Int64()).
		Params(Qual("io", "Reader")).
		Block(
			Id("r").Op("=").Id("ensureBodyReader").Call(Id("r")),
			If(Id("limit").Op("<=").Lit(0)).Block(
				Return(Id("r")),
			),
			Return(Op("&").Id("maxBytesReader").Values(Dict{Id("r"): Id("r"), Id("n"): Id("limit")})),
		)
}

func (r *transportRenderer) bodyLimitFunc() Code {

	return Func().Id("bodyLimit").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Id("limit").Int64()).
		Block(
			Id("limit").Op("=").Id("defaultBodyLimit"),
			If(List(Id("server"), Id("ok")).Op(":=").Id(VarNameFtx).Dot("Locals").Call(Lit("server")).Assert(Op("*").Id("Server")).Op(";").Id("ok")).Block(
				Id("limit").Op("=").Int64().Call(Id("server").Dot("config").Dot("BodyLimit")),
			),
			Return(),
		)
}

func (r *transportRenderer) streamBodyLimitFunc() Code {

	return Func().Id("streamBodyLimit").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Id("limit").Int64()).
		Block(
			Id("limit").Op("=").Id("defaultBodyLimit"),
			If(List(Id("server"), Id("ok")).Op(":=").Id(VarNameFtx).Dot("Locals").Call(Lit("server")).Assert(Op("*").Id("Server")).Op(";").Id("ok")).Block(
				Id("limit").Op("=").Id("server").Dot("maxStreamBodySize"),
			),
			Return(),
		)
}

func (r *transportRenderer) decodeBodyReaderFunc() Code {

	return Func().Id("decodeBodyReader").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Qual("io", "Reader")).
		Block(
			Return(Id("limitBodyReader").Call(Id("ensureBodyReader").Call(Id(VarNameFtx).Dot("Context").Call().Dot("RequestBodyStream").Call()), Id("bodyLimit").Call(Id(VarNameFtx)))),
		)
}

func (r *transportRenderer) streamBodyReaderFunc() Code {

	return Func().Id("streamBodyReader").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Qual("io", "Reader")).
		Block(
			Return(Id("limitBodyReader").Call(Id("ensureBodyReader").Call(Id(VarNameFtx).Dot("Context").Call().Dot("RequestBodyStream").Call()), Id("streamBodyLimit").Call(Id(VarNameFtx)))),
		)
}

func (r *transportRenderer) renderRequestLogger(srcFile *GoFile) {

	srvctxPkgPath := r.pkgPath(r.outDir) + "/srvctx"

	srcFile.Line().Func().Params(Id("srv").Op("*").Id("Server")).
		Id("requestLogger").
		Params(Id(VarNameCtx).Qual(PackageContext, "Context")).
		Params(Id("log").Op("*").Qual(PackageSlog, "Logger")).
		Block(
			Id("log").Op("=").Id("srv").Dot("log").Dot("With").Call(Lit("clientID"), Qual(srvctxPkgPath, "GetClientID").Call(Id(VarNameCtx))),
			Return(Id("log")),
		)
}

func (r *transportRenderer) renderFiberLogger(srcFile *GoFile) {

	srvctxPkgPath := r.pkgPath(r.outDir) + "/srvctx"
	fromCtxLogger := func(ctx Code) *Statement {

		return Qual(srvctxPkgPath, "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(ctx)
	}
	withCtxLogger := func(ctx, logger Code) Code {

		return Qual(srvctxPkgPath, "WithCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(ctx, logger)
	}
	requestLoggerCall := func(ctx Code) Code {

		return Id("srv").Dot("requestLogger").Call(ctx)
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
			bg.Id("levelName").Op(":=").Lit("")
			bg.If(Id("srv").Dot("allowClientLogLevel")).Block(
				Id("levelName").Op("=").Id(VarNameFtx).Dot("Get").Call(Id("logLevelHeader")),
			)
			bg.If(Id("levelName").Op("==").Lit("")).Block(
				Id(VarNameFtx).Dot("SetUserContext").Call(withCtxLogger(Id("ctx"), requestLoggerCall(Id("ctx")))),
				Return(Id(VarNameFtx).Dot("Next").Call()),
			)
			bg.Var().Id("level").Qual(PackageSlog, "Level")
			bg.Switch(Id("levelName")).Block(
				Case(Lit("debug"), Lit("DEBUG")).Block(Id("level").Op("=").Qual(PackageSlog, "LevelDebug")),
				Case(Lit("info"), Lit("INFO")).Block(Id("level").Op("=").Qual(PackageSlog, "LevelInfo")),
				Case(Lit("warn"), Lit("WARN")).Block(Id("level").Op("=").Qual(PackageSlog, "LevelWarn")),
				Case(Lit("error"), Lit("ERROR")).Block(Id("level").Op("=").Qual(PackageSlog, "LevelError")),
				Default().Block(
					Id(VarNameFtx).Dot("SetUserContext").Call(withCtxLogger(Id("ctx"), requestLoggerCall(Id("ctx")))),
					Return(Id(VarNameFtx).Dot("Next").Call()),
				),
			)
			bg.Id("levelVar").Op(":=").Op("new").Call(Qual(PackageSlog, "LevelVar"))
			bg.Id("levelVar").Dot("Set").Call(Id("level"))
			bg.Id("log").Op(":=").Id("srv").Dot("requestLogger").Call(Id("ctx"))
			bg.Id("baseHandler").Op(":=").Id("log").Dot("Handler").Call()
			bg.Id("levelLogger").Op(":=").Qual(PackageSlog, "New").Call(Op("&").Id("levelHandler").Values(Dict{
				Id("handler"): Id("baseHandler"),
				Id("level"):   Id("levelVar"),
			}))
			bg.Id(VarNameFtx).Dot("SetUserContext").Call(withCtxLogger(Id("ctx"), Id("levelLogger")))
			bg.Return(Id(VarNameFtx).Dot("Next").Call())
		})
}

func (r *transportRenderer) renderFiberRecover(srcFile *GoFile) {

	srcFile.Line().Func().Id("recoverHandler").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Error().
		BlockFunc(func(bg *Group) {
			bg.Id("path").Op(":=").Id("metricRoutePath").Call(Id(VarNameFtx))
			bg.Id("method").Op(":=").Id(VarNameFtx).Dot("Method").Call()
			bg.Id("originalURL").Op(":=").Id("clipString").Call(Id(VarNameFtx).Dot("OriginalURL").Call(), Id("maxLogFieldLength"))
			bg.Defer().Func().Params().BlockFunc(func(dg *Group) {
				dg.If(Id("r").Op(":=").Recover().Op(";").Id("r").Op("!=").Nil()).BlockFunc(func(ig *Group) {
					ig.List(Err(), Id("ok")).Op(":=").Id("r").Op(".").Call(Error())
					ig.If(Op("!").Id("ok")).Block(
						Err().Op("=").Qual(PackageFmt, "Errorf").Call(Lit("%v"), Id("r")),
					)
					ig.If(List(Id("server"), Id("ok")).Op(":=").Id(VarNameFtx).Dot("Locals").Call(Lit("server")).Assert(Op("*").Id("Server")).Op(";").Id("ok").Op("&&").Id("server").Dot("metrics").Op("!=").Nil()).BlockFunc(func(pg *Group) {
						pg.Id("clientID").Op(":=").Qual(r.pkgPath(r.outDir)+"/srvctx", "GetClientID").Call(Id(VarNameFtx).Dot("UserContext").Call())
						pg.Id("server").Dot("metrics").Dot("PanicsTotal").Dot("WithLabelValues").Call(Id("path"), Id("method"), Id("clientID")).Dot("Inc").Call()
					})
					ig.If(Id("logger").Op(":=").Qual(r.pkgPath(r.outDir)+"/srvctx", "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(Id(VarNameFtx).Dot("UserContext").Call()).Op(";").Id("logger").Op("!=").Nil()).Block(
						Id("logger").Dot("Error").Call(Lit("panic occurred"),
							Qual(PackageSlog, "Any").Call(Lit("error"), Qual(PackageFmt, "Errorf").Call(Lit("recover: %w"), Err())),
							Qual(PackageSlog, "String").Call(Lit("method"), Id("method")),
							Qual(PackageSlog, "String").Call(Lit("path"), Id("originalURL")),
							Qual(PackageSlog, "String").Call(Lit("stack"), Id(typeString).Call(Qual("runtime/debug", "Stack").Call())),
						),
					)
					ig.Id(VarNameFtx).Dot("Status").Call(Qual(PackageFiber, "StatusInternalServerError"))
					ig.Id("_").Op("=").Id(VarNameFtx).Dot("JSON").Call(Map(String()).String().Values(Dict{Lit("message"): Lit("internal server error")}))
				})
			}).Call()
			bg.Return(Id(VarNameFtx).Dot("Next").Call())
		})
}
