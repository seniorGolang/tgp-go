// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (r *transportRenderer) RenderTransportContext() error {

	contextPath := path.Join(r.outDir, "context.go")

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(PackageSlog, "slog")

	srcFile.Line().Add(r.getLoggerFunc())

	return srcFile.Save(contextPath)
}

func (r *transportRenderer) getLoggerFunc() Code {

	return Func().Id("GetLogger").
		Params(Id(VarNameCtx).Qual(PackageContext, "Context")).
		Params(Op("*").Qual(PackageSlog, "Logger")).
		Block(
			Return(Id("FromContext").Call(Id(VarNameCtx))),
		)
}
