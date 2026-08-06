// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive
)

func (r *ClientRenderer) RenderPathHelpers() (err error) {

	if !r.HasHTTP() && !r.HasWS() && !r.HasSSE() {
		return
	}

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.ImportName(PackageStrings, "strings")
	srcFile.Line().Add(r.joinEndpointPathFunc())
	srcFile.Line().Add(r.splitEndpointPathFunc())
	srcFile.Line().Add(r.endpointPathOverlapFunc())
	srcFile.Line().Add(r.endpointPathSegmentsEqualFunc())

	err = srcFile.Save(path.Join(r.outDir, "path.go"))
	return
}

func (r *ClientRenderer) joinEndpointPathFunc() (c Code) {

	return Func().Id("joinEndpointPath").
		Params(Id("basePath").String(), Id("routePath").String()).
		Params(Id("joined").String()).
		Block(
			Id("base").Op(":=").Id("splitEndpointPath").Call(Id("basePath")),
			Id("route").Op(":=").Id("splitEndpointPath").Call(Id("routePath")),
			Id("overlap").Op(":=").Id("endpointPathOverlap").Call(Id("base"), Id("route")),
			Id("out").Op(":=").Make(Index().String(), Lit(0), Len(Id("base")).Op("-").Id("overlap").Op("+").Len(Id("route"))),
			Id("out").Op("=").Append(Id("out"), Id("base").Index(Op(":").Len(Id("base")).Op("-").Id("overlap")).Op("...")),
			Id("out").Op("=").Append(Id("out"), Id("route").Op("...")),
			If(Len(Id("out")).Op("==").Lit(0)).Block(
				Return(Lit("/")),
			),
			Id("joined").Op("=").Lit("/").Op("+").Qual(PackageStrings, "Join").Call(Id("out"), Lit("/")),
			Return(),
		)
}

func (r *ClientRenderer) splitEndpointPathFunc() (c Code) {

	return Func().Id("splitEndpointPath").
		Params(Id("raw").String()).
		Params(Id("segments").Index().String()).
		Block(
			For(List(Id("_"), Id("part")).Op(":=").Range().Qual(PackageStrings, "Split").Call(Id("raw"), Lit("/"))).Block(
				If(Id("part").Op("==").Lit("")).Block(
					Continue(),
				),
				Id("segments").Op("=").Append(Id("segments"), Id("part")),
			),
			Return(),
		)
}

func (r *ClientRenderer) endpointPathOverlapFunc() (c Code) {

	return Func().Id("endpointPathOverlap").
		Params(Id("base").Index().String(), Id("route").Index().String()).
		Params(Id("overlap").Int()).
		Block(
			Id("max").Op(":=").Len(Id("base")),
			If(Len(Id("route")).Op("<").Id("max")).Block(
				Id("max").Op("=").Len(Id("route")),
			),
			For(Id("n").Op(":=").Id("max"), Id("n").Op(">=").Lit(1), Id("n").Op("--")).Block(
				If(Id("endpointPathSegmentsEqual").Call(
					Id("base").Index(Len(Id("base")).Op("-").Id("n").Op(":")),
					Id("route").Index(Op(":").Id("n")),
				)).Block(
					Return(Id("n")),
				),
			),
			Return(Lit(0)),
		)
}

func (r *ClientRenderer) endpointPathSegmentsEqualFunc() (c Code) {

	return Func().Id("endpointPathSegmentsEqual").
		Params(Id("left").Index().String(), Id("right").Index().String()).
		Params(Id("equal").Bool()).
		Block(
			If(Len(Id("left")).Op("!=").Len(Id("right"))).Block(
				Return(False()),
			),
			For(Id("i").Op(":=").Range().Id("left")).Block(
				If(Id("left").Index(Id("i")).Op("!=").Id("right").Index(Id("i"))).Block(
					Return(False()),
				),
			),
			Return(True()),
		)
}
