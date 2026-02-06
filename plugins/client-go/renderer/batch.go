// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

func (r *ClientRenderer) RenderClientBatch() error {

	outDir := r.outDir
	srcFile := NewSrcFile(filepath.Base(outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageFmt, "fmt")
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "jsonrpc")

	srcFile.Line().Type().Id("RequestRPC").Struct(
		Id("retHandler").Id("rpcCallback"),
		Id("rpcRequest").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "RequestRPC"),
	)

	srcFile.Line().Type().Id("rpcCallback").Func().Params(Err().Error(), Id("response").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "ResponseRPC"))

	srcFile.Line().Func().Params(Id("cli").Op("*").
		Id("Client")).Id("Batch").
		Params(Id(_ctx_).Qual(PackageContext, "Context"), Id("requests").Op("...").Id("RequestRPC")).BlockFunc(func(bg *Group) {
		bg.Line()
		bg.If(Len(Id("requests")).Op("==").Lit(0)).Block(Return())
		bg.Line()
		bg.Var().Id("rpcRequests").Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "RequestsRPC")
		bg.Id("callbacks").Op(":=").Make(Map(Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "ID")).Id("rpcCallback"))
		bg.For(List(Id("_"), Id("request")).Op(":=").Range().Id("requests")).Block(
			Id("rpcRequests").Op("=").Append(Id("rpcRequests"), Id("request").Dot("rpcRequest")),
			Id("callbacks").Op("[").Id("request").Dot("rpcRequest").Dot("ID").Op("]").Op("=").Id("request").Dot("retHandler"),
		)
		bg.Var().Err().Error()
		bg.Var().Id("rpcResponses").Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "ResponsesRPC")
		bg.List(Id("rpcResponses"), Err()).Op("=").Id("cli").Dot("rpc").Dot("CallBatch").Call(Id(_ctx_), Id("rpcRequests"))
		bg.If(Err().Op("!=").Nil().Op("||").Id("rpcResponses").Op("==").Nil()).Block(
			For(List(Id("_"), Id("callback")).Op(":=").Range().Id("callbacks")).Block(
				Id("callback").Call(Err(), Nil()),
			),
			Return(),
		)
		bg.For(List(Id("_"), Id("response")).Op(":=").Range().Id("rpcResponses")).Block(
			If(Id("response").Op("==").Nil()).Block(Continue()),
			If(Id("callback").Op(":=").Id("callbacks").Op("[").Id("response").Dot("ID").Op("]").Op(";").Id("callback").Op("!=").Nil()).Block(
				If(Id("response").Dot("Error").Op("!=").Nil()).Block(
					Var().Id("cbErr").Id("error"),
					If(Id("cli").Dot("errorDecoder").Op("!=").Nil()).Block(
						Id("cbErr").Op("=").Id("cli").Dot("errorDecoder").Call(Id("response").Dot("Error").Dot("Raw").Call()),
					).Else().Block(
						Id("cbErr").Op("=").Qual(PackageFmt, "Errorf").Call(Lit("%s"), Id("response").Dot("Error").Dot("Message")),
					),
					Id("callback").Call(Id("cbErr"), Id("response")),
				).Else().Block(
					Id("callback").Call(Nil(), Id("response")),
				),
			),
		)
		jsonrpcPkg := fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir))
		bg.Id("respondedIDs").Op(":=").Make(Map(Qual(jsonrpcPkg, "ID")).Bool())
		bg.For(List(Id("_"), Id("response")).Op(":=").Range().Id("rpcResponses")).Block(
			If(Id("response").Op("!=").Nil()).Block(
				Id("respondedIDs").Index(Id("response").Dot("ID")).Op("=").True(),
			),
		)
		bg.For(List(Id("_"), Id("request")).Op(":=").Range().Id("requests")).BlockFunc(func(miss *Group) {
			miss.Id("id").Op(":=").Id("request").Dot("rpcRequest").Dot("ID")
			miss.If(Id("callback").Op(":=").Id("callbacks").Index(Id("id")).Op(";").Id("callback").Op("!=").Nil().Op("&&").Op("!").Id("respondedIDs").Index(Id("id"))).Block(
				Id("callback").Call(Qual(PackageFmt, "Errorf").Call(Lit("missing response for request ID %v"), Id("id")), Nil()),
			)
		})
	})
	return srcFile.Save(path.Join(outDir, "batch.go"))
}
