// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/generated"
)

func (r *ClientRenderer) RenderClientError() (err error) {

	outDir := r.outDir
	srcFile := NewSrcFile(filepath.Base(outDir))
	srcFile.PackageComment(generated.ByToolGateway)

	jsonPkg := r.getPackageJSON(nil)

	if r.HasJsonRPC() {
		srcFile.ImportName(jsonPkg, "json")
	}
	if r.HasHTTP() {
		srcFile.ImportName(PackageFmt, "fmt")
	}

	srcFile.Line().Const().Id("internalError").Op("=").Lit(-32603) // JSON-RPC: Internal error

	srcFile.Line().Type().Id("withErrorCode").Interface(
		Id("Code").Call().Int(),
	)

	if r.HasJsonRPC() {
		srcFile.Line().Add(r.errorJsonRPCType()).Line()
		srcFile.Line().Type().Id("ErrorDecoder").Func().Params(
			Id("errData").Qual(jsonPkg, "RawMessage"),
		).Params(Error())
	}

	if r.HasHTTP() {
		srcFile.Line().Type().Id("ResponseError").Struct(
			Id("StatusCode").Int(),
			Id("ContentType").String(),
			Id("Body").Index().Byte(),
		)
		srcFile.Line().Type().Id("HTTPErrorDecoder").Func().Params(
			Id("statusCode").Int(),
			Id("contentType").String(),
			Id("body").Index().Byte(),
		).Params(Id("err").Error())
	}

	if r.HasJsonRPC() {
		srcFile.Line().Add(r.errorJsonRPCErrorMethod()).Line()
	}

	if r.HasHTTP() {
		srcFile.Line().Func().Params(Id("e").Op("*").Id("ResponseError")).Id("Error").Params().Params(Id("msg").String()).Block(
			If(Len(Id("e").Dot("Body")).Op("==").Lit(0)).Block(
				Return(Qual(PackageFmt, "Sprintf").Call(Lit("HTTP error: %d"), Id("e").Dot("StatusCode"))),
			),
			Return(Qual(PackageFmt, "Sprintf").Call(
				Lit("HTTP error: %d: %s"),
				Id("e").Dot("StatusCode"),
				String().Call(Id("e").Dot("Body")),
			)),
		)
		srcFile.Line().Func().Params(Id("e").Op("*").Id("ResponseError")).Id("Code").Params().Params(Id("code").Int()).Block(
			Return(Id("e").Dot("StatusCode")),
		)
	}

	if r.HasJsonRPC() {
		srcFile.Line().Func().Id("defaultErrorDecoder").Params(
			Id("errData").Qual(jsonPkg, "RawMessage"),
		).Params(Id("err").Error()).Block(
			Var().Id("jsonrpcError").Id("errorJsonRPC"),
			If(Err().Op("=").Qual(jsonPkg, "Unmarshal").Call(Id("errData"), Op("&").Id("jsonrpcError")).Op(";").Err().Op("!=").Nil()).Block(
				Return(),
			),
			Return(Id("jsonrpcError")),
		)
	}

	if r.HasHTTP() {
		srcFile.Line().Func().Id("defaultHTTPErrorDecoder").Params(
			Id("statusCode").Int(),
			Id("contentType").String(),
			Id("body").Index().Byte(),
		).Params(Id("err").Error()).Block(
			Return(Op("&").Id("ResponseError").Values(Dict{
				Id("StatusCode"):  Id("statusCode"),
				Id("ContentType"): Id("contentType"),
				Id("Body"):        Id("append").Call(Index().Byte().Call(Nil()), Id("body").Op("...")),
			})),
		)
	}

	return srcFile.Save(path.Join(outDir, "error.go"))
}

func (r *ClientRenderer) errorJsonRPCType() (c Code) {

	return Type().Id("errorJsonRPC").Struct(
		Id("Code").Int().Tag(map[string]string{"json": "code"}),
		Id("Message").String().Tag(map[string]string{"json": "message"}),
		Id("Data").Any().Tag(map[string]string{"json": "data,omitempty"}),
	)
}

func (r *ClientRenderer) errorJsonRPCErrorMethod() (c Code) {

	return Func().Params(Err().Id("errorJsonRPC")).Id("Error").Params().Params(String()).Block(
		Return(Err().Dot("Message")),
	)
}
