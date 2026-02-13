// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (r *transportRenderer) RenderTransportContext() error {

	if err := r.pkgCopyTo("srvctx", r.outDir); err != nil {
		return err
	}
	contextPath := path.Join(r.outDir, "context.go")
	srvctxPkgPath := fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir))

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(srvctxPkgPath, "srvctx")
	srcFile.ImportName(PackageSlog, "slog")

	srcFile.Line().Add(r.withMethodLoggerFunc())

	return srcFile.Save(contextPath)
}

func (r *transportRenderer) withMethodLoggerFunc() Code {

	srvctxPkgPath := fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir))

	return Func().Id("withMethodLogger").
		Params(
			Id(VarNameCtx).Qual(PackageContext, "Context"),
			Id("contract").String(),
			Id("method").String(),
		).
		Params(Qual(PackageContext, "Context")).
		BlockFunc(func(bg *Group) {
			bg.Id("log").Op(":=").Qual(srvctxPkgPath, "GetLogger").Call(Id(VarNameCtx))
			bg.If(Id("log").Op("==").Nil()).Block(
				Id("log").Op("=").Qual(PackageSlog, "Default").Call(),
			)
			bg.Return(
				Qual(srvctxPkgPath, "WithCtx").Types(Op("*").Qual(PackageSlog, "Logger")).
					Call(Id(VarNameCtx), Id("log").Dot("With").Call(Lit("contract"), Id("contract"), Lit("method"), Id("method"))),
			)
		})
}
