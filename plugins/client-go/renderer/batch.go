// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive
)

func (r *ClientRenderer) RenderClientBatch() (err error) {

	outDir := r.outDir
	srcFile := NewSrcFile(filepath.Base(outDir))
	srcFile.ImportName(PackageFmt, "fmt")
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(r.pkgPath(outDir)+"/jsonrpc", pkgJSONRPC)

	srcFile.Line().Type().Id("RequestRPC").Struct(
		Id("retHandler").Id("rpcCallback"),
		Id("rpcRequest").Op("*").Qual(r.pkgPath(outDir)+"/jsonrpc", "RequestRPC"),
	)

	srcFile.Line().Type().Id("rpcCallback").Func().Params(Err().Error(), Id("response").Op("*").Qual(r.pkgPath(outDir)+"/jsonrpc", "ResponseRPC"))

	srcFile.Line().Func().Params(Id("cli").Op("*").
		Id("Client")).Id("Batch").
		Params(Id(_ctx).Qual(PackageContext, "Context"), Id("requests").Op("...").Id("RequestRPC")).BlockFunc(func(bg *Group) {
		bg.Line()
		bg.If(Len(Id("requests")).Op("==").Lit(0)).Block(Return())
		bg.Line()
		bg.Var().Id("rpcRequests").Qual(r.pkgPath(outDir)+"/jsonrpc", "RequestsRPC")
		bg.Id("callbacks").Op(":=").Make(Map(Qual(r.pkgPath(outDir)+"/jsonrpc", "ID")).Id("rpcCallback"))
		bg.For(List(Id("_"), Id("request")).Op(":=").Range().Id("requests")).BlockFunc(func(fg *Group) {
			fg.Id("id").Op(":=").Id("request").Dot("rpcRequest").Dot("ID")
			fg.If(List(Id("_"), Id("exists")).Op(":=").Id("callbacks").Index(Id("id")).Op(";").Id("exists")).Block(
				For(List(Id("_"), Id("callback")).Op(":=").Range().Id("callbacks")).Block(
					Id("callback").Call(Qual(PackageFmt, "Errorf").Call(Lit("duplicate batch request id: %v"), Id("id")), Nil()),
				),
				If(Id("request").Dot("retHandler").Op("!=").Nil()).Block(
					Id("request").Dot("retHandler").Call(Qual(PackageFmt, "Errorf").Call(Lit("duplicate batch request id: %v"), Id("id")), Nil()),
				),
				Return(),
			)
			fg.Id("rpcRequests").Op("=").Append(Id("rpcRequests"), Id("request").Dot("rpcRequest"))
			fg.Id("callbacks").Index(Id("id")).Op("=").Id("request").Dot("retHandler")
		})
		bg.Var().Err().Error()
		bg.Var().Id("rpcResponses").Qual(r.pkgPath(outDir)+"/jsonrpc", "ResponsesRPC")
		bg.If(List(Id("rpcResponses"), Err()).Op("=").Id("cli").Dot("rpc").Dot("CallBatch").Call(Id(_ctx), Id("rpcRequests")).Op(";").Err().Op("!=").Nil().Op("||").Id("rpcResponses").Op("==").Nil()).Block(
			For(List(Id("_"), Id("callback")).Op(":=").Range().Id("callbacks")).Block(
				Id("callback").Call(Err(), Nil()),
			),
			Return(),
		)
		bg.For(List(Id("_"), Id("response")).Op(":=").Range().Id("rpcResponses")).Block(
			If(Id("response").Op("==").Nil()).Block(Continue()),
			If(Id("callback").Op(":=").Id("callbacks").Op("[").Id("response").Dot("ID").Op("]").Op(";").Id("callback").Op("!=").Nil()).Block(
				If(Id("response").Dot("Error").Op("!=").Nil()).Block(
					Var().Id("cbErr").Id(typeError),
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
		jsonrpcPkg := r.pkgPath(outDir) + "/jsonrpc"
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
	err = srcFile.Save(path.Join(outDir, "batch.go"))
	return
}
