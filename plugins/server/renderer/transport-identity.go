// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive
)

func (r *transportRenderer) RenderTransportLimits() (err error) {

	limitsPath := path.Join(r.outDir, "limits.go")
	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.ImportName(PackageStrings, "strings")
	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(PackageMapHash, "maphash")

	srcFile.Const().Defs(
		Id("maxClientIDLength").Op("=").Lit(64),
		Id("maxLogFieldLength").Op("=").Lit(256),
	)

	srcFile.Line().Var().Id("clientIDHashSeed").Op("=").Qual(PackageMapHash, "MakeSeed").Call()

	r.renderIdentityHelpers(&srcFile)
	return srcFile.Save(limitsPath)
}

func (r *transportRenderer) renderIdentityHelpers(srcFile *GoFile) {

	srcFile.Line().Func().Id("normalizeClientID").
		Params(Id("raw").String()).
		Params(Id("id").String()).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.Id("id").Op("=").Qual(PackageStrings, "TrimSpace").Call(Id("raw"))
			bg.If(Id("id").Op("==").Lit("")).Block(
				Id("id").Op("=").Lit("unknown"),
				Return(),
			)
			bg.If(Len(Id("id")).Op(">").Id("maxClientIDLength")).Block(
				Id("id").Op("=").Lit("invalid"),
				Return(),
			)
			bg.For(List(Id("_"), Id("r")).Op(":=").Range().Id("id")).Block(
				If(Op("!").Id("isClientIDRune").Call(Id("r"))).Block(
					Id("id").Op("=").Lit("invalid"),
					Return(),
				),
			)
			bg.Return()
		})

	srcFile.Line().Func().Id("hashClientID").
		Params(Id("id").String()).
		Params(Id("h").Uint64()).
		Block(
			Line(),
			Id("h").Op("=").Qual(PackageMapHash, "String").Call(Id("clientIDHashSeed"), Id("id")),
			If(Id("h").Op("==").Lit(0)).Block(
				Id("h").Op("=").Lit(1),
			),
			Return(),
		)

	srcFile.Line().Func().Params(Id("srv").Op("*").Id("Server")).
		Id("boundClientID").
		Params(Id("id").String()).
		Params(Id("out").String()).
		BlockFunc(func(bg *Group) {
			bg.Line()
			bg.If(Id("srv").Dot("maxClientIDs").Op("==").Lit(0).Op("||").Id("id").Op("==").Lit("unknown").Op("||").Id("id").Op("==").Lit("invalid")).Block(
				Id("out").Op("=").Id("id"),
				Return(),
			)
			bg.Id("h").Op(":=").Id("hashClientID").Call(Id("id"))
			bg.Id("srv").Dot("clientIDsMu").Dot("RLock").Call()
			bg.List(Id("_"), Id("ok")).Op(":=").Id("srv").Dot("clientIDs").Index(Id("h"))
			bg.Id("srv").Dot("clientIDsMu").Dot("RUnlock").Call()
			bg.If(Id("ok")).Block(
				Id("out").Op("=").Id("id"),
				Return(),
			)
			bg.Id("srv").Dot("clientIDsMu").Dot("Lock").Call()
			bg.Defer().Id("srv").Dot("clientIDsMu").Dot("Unlock").Call()
			bg.If(List(Id("_"), Id("ok")).Op("=").Id("srv").Dot("clientIDs").Index(Id("h")).Op(";").Id("ok")).Block(
				Id("out").Op("=").Id("id"),
				Return(),
			)
			bg.If(Len(Id("srv").Dot("clientIDs")).Op(">=").Id("srv").Dot("maxClientIDs")).Block(
				Id("out").Op("=").Lit("other"),
				Return(),
			)
			bg.Id("srv").Dot("clientIDs").Index(Id("h")).Op("=").Struct().Block()
			bg.Id("out").Op("=").Id("id")
			bg.Return()
		})

	srcFile.Line().Func().Id("isClientIDRune").
		Params(Id("r").Rune()).
		Params(Id("ok").Bool()).
		Block(
			Line(),
			Id("ok").Op("=").Parens(
				Id("r").Op(">=").Lit('a').Op("&&").Id("r").Op("<=").Lit('z'),
			).Op("||").Parens(
				Id("r").Op(">=").Lit('A').Op("&&").Id("r").Op("<=").Lit('Z'),
			).Op("||").Parens(
				Id("r").Op(">=").Lit('0').Op("&&").Id("r").Op("<=").Lit('9'),
			).Op("||").Id("r").Op("==").Lit('.').Op("||").Id("r").Op("==").Lit('_').Op("||").Id("r").Op("==").Lit('-').Op("||").Id("r").Op("==").Lit(':'),
			Return(),
		)

	srcFile.Line().Func().Id("clipString").
		Params(Id("value").String(), Id("maxLen").Int()).
		Params(Id("out").String()).
		Block(
			Line(),
			If(Len(Id("value")).Op("<=").Id("maxLen")).Block(
				Id("out").Op("=").Id("value"),
				Return(),
			),
			Id("out").Op("=").Id("value").Index(Op(":").Id("maxLen")),
			Return(),
		)

	srcFile.Line().Func().Id("metricRoutePath").
		Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).
		Params(Id("path").String()).
		Block(
			Line(),
			Id("path").Op("=").Id(VarNameFtx).Dot("Route").Call().Dot("Path"),
			If(Id("path").Op("==").Lit("")).Block(
				Id("path").Op("=").Lit("unmatched"),
			),
			Return(),
		)
}
