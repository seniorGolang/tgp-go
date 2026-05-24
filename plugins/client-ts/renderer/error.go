// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"

	"tgp/internal/generated"
	"tgp/plugins/client-ts/tsg"
)

func (r *ClientRenderer) RenderClientError() (err error) {

	outDir := r.outDir
	file := tsg.NewFile()
	file.Comment(generated.ByToolGatewayComment)

	stmt := tsg.NewStatement()
	stmt.Export().Interface("ErrorJsonRPC", func(grp *tsg.Group) {
		grp.Add(tsg.NewStatement().Id("code").Colon().Id("number").Semicolon())
		grp.Add(tsg.NewStatement().Id("message").Colon().Id("string").Semicolon())
		grp.Add(tsg.NewStatement().Id("data").Optional().Colon().Id("any").Semicolon())
	})
	file.Add(stmt)
	file.Line()

	stmt2 := tsg.NewStatement()
	stmt2.Export().Interface("WithErrorCode", func(grp *tsg.Group) {
		grp.Add(tsg.NewStatement().Id("code").Call().Colon().Id("number").Semicolon())
	})
	file.Add(stmt2)
	file.Line()

	stmt3 := tsg.NewStatement()
	stmt3.Export().Const("internalError").Op("=").Lit(-32603).Semicolon()
	file.Add(stmt3)
	file.Line()

	if r.HasJsonRPC() {
		file.Add(r.renderRPCErrorDecoderType())
		file.Line()
		file.Add(r.renderDefaultRPCErrorDecoderFunc())
		file.Line()
	}

	if r.HasHTTP() {
		file.Add(r.renderResponseErrorClass())
		file.Line()
		file.Add(r.renderHTTPErrorDecoderType())
		file.Line()
		file.Add(r.renderDefaultHTTPErrorDecoderFunc())
		file.Line()
	}

	file.GenerateImports()

	return file.Save(path.Join(outDir, "error.ts"))
}

func (r *ClientRenderer) renderRPCErrorDecoderType() *tsg.Statement {

	stmt := tsg.NewStatement()
	stmt.Export().Type("ErrorDecoder")
	stmt.Op("=")
	fnType := tsg.NewStatement()
	fnType.Params(func(fg *tsg.Group) {
		fg.Add(tsg.NewStatement().Id("errData").Colon().Id("unknown"))
	}).Op("=>").Id("Error")
	stmt.Add(fnType).Semicolon()
	return stmt
}

func (r *ClientRenderer) renderDefaultRPCErrorDecoderFunc() *tsg.Statement {

	stmt := tsg.NewStatement()
	stmt.Export().Func("defaultErrorDecoder")
	stmt.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().Id("errData").Colon().Id("unknown"))
	})
	stmt.Colon().Id("Error")
	stmt.Block(func(bg *tsg.Group) {
		bg.Add(tsg.NewStatement().Const("jsonrpcError").Colon().Id("ErrorJsonRPC").Op("=").Id("errData").Op("as").Id("ErrorJsonRPC").Semicolon())
		bg.Return(tsg.NewStatement().New("Error").Call(tsg.NewStatement().Id("jsonrpcError").Dot("message").Op("??").Lit("unknown error")))
	})
	return stmt
}

func (r *ClientRenderer) renderResponseErrorClass() *tsg.Statement {

	stmt := tsg.NewStatement().Export()
	stmt.Add(tsg.TypeFromString("class ResponseError extends Error implements WithErrorCode"))
	stmt.Block(func(grp *tsg.Group) {
		grp.Add(tsg.NewStatement().Id("statusCode").Colon().Id("number").Semicolon())
		grp.Add(tsg.NewStatement().Id("contentType").Colon().Id("string").Semicolon())
		grp.Add(tsg.NewStatement().Id("body").Colon().Id("string").Semicolon())
		grp.Line()

		ctor := tsg.NewStatement()
		ctor.Id("constructor")
		ctor.Params(func(pg *tsg.Group) {
			pg.Add(tsg.NewStatement().Id("statusCode").Colon().Id("number"))
			pg.Add(tsg.NewStatement().Id("contentType").Colon().Id("string"))
			pg.Add(tsg.NewStatement().Id("body").Colon().Id("string"))
		})
		ctor.Block(func(bg *tsg.Group) {
			bg.Add(tsg.TypeFromString("super(`HTTP error: ${statusCode}`);"))
			bg.Add(tsg.NewStatement().This().Dot("statusCode").Op("=").Id("statusCode").Semicolon())
			bg.Add(tsg.NewStatement().This().Dot("contentType").Op("=").Id("contentType").Semicolon())
			bg.Add(tsg.NewStatement().This().Dot("body").Op("=").Id("body").Semicolon())
		})
		grp.Add(ctor)
		grp.Line()

		grp.Add(tsg.NewStatement().Id("code").Call().Colon().Id("number").Block(func(mg *tsg.Group) {
			mg.Return(tsg.NewStatement().This().Dot("statusCode"))
		}))
	})
	return stmt
}

func (r *ClientRenderer) renderHTTPErrorDecoderType() *tsg.Statement {

	stmt := tsg.NewStatement()
	stmt.Export().Type("HTTPErrorDecoder")
	stmt.Op("=")
	fnType := tsg.NewStatement()
	fnType.Params(func(fg *tsg.Group) {
		fg.Add(tsg.NewStatement().Id("statusCode").Colon().Id("number"))
		fg.Add(tsg.NewStatement().Id("contentType").Colon().Id("string"))
		fg.Add(tsg.NewStatement().Id("body").Colon().Id("string"))
	}).Op("=>").Id("Error")
	stmt.Add(fnType).Semicolon()
	return stmt
}

func (r *ClientRenderer) renderDefaultHTTPErrorDecoderFunc() *tsg.Statement {

	stmt := tsg.NewStatement()
	stmt.Export().Func("defaultHTTPErrorDecoder")
	stmt.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().Id("statusCode").Colon().Id("number"))
		pg.Add(tsg.NewStatement().Id("contentType").Colon().Id("string"))
		pg.Add(tsg.NewStatement().Id("body").Colon().Id("string"))
	})
	stmt.Colon().Id("Error")
	stmt.Block(func(bg *tsg.Group) {
		bg.Return(tsg.NewStatement().New("ResponseError").Call(
			tsg.NewStatement().Id("statusCode"),
			tsg.NewStatement().Id("contentType"),
			tsg.NewStatement().Id("body"),
		))
	})
	return stmt
}
