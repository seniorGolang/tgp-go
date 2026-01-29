// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (r *transportRenderer) RenderTransportLogger() error {

	loggerPath := path.Join(r.outDir, "logger.go")

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(PackageSlog, "slog")

	srcFile.Line().Add(r.loggerContextKey())
	srcFile.Line().Add(r.loggerWithLoggerFunc())
	srcFile.Line().Add(r.loggerFromContextFunc())

	return srcFile.Save(loggerPath)
}

func (r *transportRenderer) loggerContextKey() Code {

	return Type().Id("loggerContextKey").String().Line().
		Line().Var().Id("loggerKey").Id("loggerContextKey").Op("=").Lit("logger")
}

func (r *transportRenderer) loggerWithLoggerFunc() Code {

	return Func().Id("WithLogger").
		Params(Id(VarNameCtx).Qual(PackageContext, "Context"), Id("logger").Op("*").Qual(PackageSlog, "Logger")).
		Params(Qual(PackageContext, "Context")).
		Block(
			Return(Qual(PackageContext, "WithValue").Call(Id(VarNameCtx), Id("loggerKey"), Id("logger"))),
		)
}

func (r *transportRenderer) loggerFromContextFunc() Code {

	return Func().Id("FromContext").
		Params(Id(VarNameCtx).Qual(PackageContext, "Context")).
		Params(Op("*").Qual(PackageSlog, "Logger")).
		Block(
			If(List(Id("logger"), Id("ok")).Op(":=").Id(VarNameCtx).Dot("Value").Call(Id("loggerKey")).Op(".").Call(Op("*").Qual(PackageSlog, "Logger")).Op(";").Id("ok")).Block(
				Return(Id("logger")),
			),
			Return(Nil()),
		)
}
