// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (r *transportRenderer) RenderTransportErrors() error {

	errorsPath := path.Join(r.outDir, "errors.go")

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageSlog, "slog")

	srcFile.Line().Add(r.withErrorCodeInterface())
	srcFile.Line().Add(r.withRedirectInterface())
	srcFile.Line().Add(r.errValidationType())
	srcFile.Line().Add(r.errBadRequestDataFunc())
	srcFile.Line().Add(r.exitOnErrorFunc())

	return srcFile.Save(errorsPath)
}

func (r *transportRenderer) errValidationType() Code {

	return Line().
		Type().Id("errValidation").Struct(
		Id("TrKey").String().Tag(map[string]string{"json": "trKey"}),
		Id("Data").String().Tag(map[string]string{"json": "data,omitempty"}),
		Id("code").Int(),
	).
		Line().
		Func().Params(Id("e").Op("*").Id("errValidation")).Id("Error").Params().String().
		Block(Return(Id("e").Dot("Data"))).
		Line().
		Func().Params(Id("e").Op("*").Id("errValidation")).Id("Code").Params().Int().
		Block(Return(Id("e").Dot("code")))
}

func (r *transportRenderer) errBadRequestDataFunc() Code {

	return Func().Id("errBadRequestData").Params(Id("data").String()).Op("*").Id("errValidation").
		Block(
			Return(Op("&").Id("errValidation").Values(Dict{
				Id("TrKey"): Lit("badRequest"),
				Id("Data"):  Id("data"),
				Id("code"):  Lit(400),
			})),
		)
}

func (r *transportRenderer) withErrorCodeInterface() Code {

	return Type().Id("withErrorCode").Interface(
		Id("Code").Params().Int(),
	)
}

func (r *transportRenderer) withRedirectInterface() Code {

	return Type().Id("withRedirect").Interface(
		Id("RedirectTo").Params().String(),
	)
}

func (r *transportRenderer) exitOnErrorFunc() Code {

	return Func().Id("ExitOnError").
		Params(Id("log").Op("*").Qual(PackageSlog, "Logger"), Id("err").Error(), Id("msg").String()).
		Block(
			If(Id("err").Op("!=").Nil()).Block(
				Id("log").Dot("Error").Call(Id("msg"), Qual(PackageSlog, "Any").Call(Lit("error"), Id("err").Dot("Error").Call())),
				Panic(Err()),
			),
		)
}
