// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"

	"tgp/plugins/client-ts/tsg"
)

func (r *ClientRenderer) RenderClientError() (err error) {

	outDir := r.outDir
	file := tsg.NewFile()
	stmt := tsg.NewStatement()
	stmt.Export().Interface("ErrorJsonRPC", func(grp *tsg.Group) {
		grp.Add(tsg.NewStatement().ID("code").Colon().ID(typeNumber).Semicolon())
		grp.Add(tsg.NewStatement().ID("message").Colon().ID(typeString).Semicolon())
		grp.Add(tsg.NewStatement().ID("data").Optional().Colon().ID(typeAny).Semicolon())
	})
	file.Add(stmt)
	file.Line()

	stmt2 := tsg.NewStatement()
	stmt2.Export().Interface("WithErrorCode", func(grp *tsg.Group) {
		grp.Add(tsg.NewStatement().ID("code").Call().Colon().ID(typeNumber).Semicolon())
	})
	file.Add(stmt2)
	file.Line()

	stmt3 := tsg.NewStatement()
	stmt3.Export().Const("internalError").Op("=").Lit(-32603).Semicolon()
	file.Add(stmt3)
	file.Line()

	if r.HasJSONRPC() {
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
		fg.Add(tsg.NewStatement().ID("errData").Colon().ID("unknown"))
	}).Op("=>").ID("Error")
	stmt.Add(fnType).Semicolon()
	return stmt
}

func (r *ClientRenderer) renderDefaultRPCErrorDecoderFunc() *tsg.Statement {

	stmt := tsg.NewStatement()
	stmt.Export().Func("defaultErrorDecoder")
	stmt.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("errData").Colon().ID("unknown"))
	})
	stmt.Colon().ID("Error")
	stmt.Block(func(bg *tsg.Group) {
		bg.Add(tsg.NewStatement().Const("jsonrpcError").Colon().ID("ErrorJsonRPC").Op("=").ID("errData").Op("as").ID("ErrorJsonRPC").Semicolon())
		bg.Return(tsg.NewStatement().New("Error").Call(tsg.NewStatement().ID("jsonrpcError").Dot("message").Op("??").Lit("unknown error")))
	})
	return stmt
}

func (r *ClientRenderer) renderResponseErrorClass() *tsg.Statement {

	stmt := tsg.NewStatement().Export()
	stmt.ClassDecl("ResponseError")
	stmt.Extends("Error")
	stmt.Implements("WithErrorCode")
	stmt.Block(func(grp *tsg.Group) {
		grp.Add(tsg.NewStatement().ID("statusCode").Colon().ID(typeNumber).Semicolon())
		grp.Add(tsg.NewStatement().ID("contentType").Colon().ID(typeString).Semicolon())
		grp.Add(tsg.NewStatement().ID("body").Colon().ID(typeString).Semicolon())
		grp.Line()

		ctor := tsg.NewStatement()
		ctor.ID("constructor")
		ctor.Params(func(pg *tsg.Group) {
			pg.Add(tsg.NewStatement().ID("statusCode").Colon().ID(typeNumber))
			pg.Add(tsg.NewStatement().ID("contentType").Colon().ID(typeString))
			pg.Add(tsg.NewStatement().ID("body").Colon().ID(typeString))
		})
		ctor.Block(func(bg *tsg.Group) {
			bg.Add(tsg.NewStatement().ID("super").Call(tsg.NewStatement().TemplateString([]string{"HTTP error: ", ""}, []*tsg.Statement{
				tsg.NewStatement().ID("statusCode"),
			})).Semicolon())
			bg.Add(tsg.NewStatement().This().Dot("statusCode").Op("=").ID("statusCode").Semicolon())
			bg.Add(tsg.NewStatement().This().Dot("contentType").Op("=").ID("contentType").Semicolon())
			bg.Add(tsg.NewStatement().This().Dot("body").Op("=").ID("body").Semicolon())
		})
		grp.Add(ctor)
		grp.Line()

		grp.Add(tsg.NewStatement().ID("code").Call().Colon().ID(typeNumber).Block(func(mg *tsg.Group) {
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
		fg.Add(tsg.NewStatement().ID("statusCode").Colon().ID(typeNumber))
		fg.Add(tsg.NewStatement().ID("contentType").Colon().ID(typeString))
		fg.Add(tsg.NewStatement().ID("body").Colon().ID(typeString))
	}).Op("=>").ID("Error")
	stmt.Add(fnType).Semicolon()
	return stmt
}

func (r *ClientRenderer) renderDefaultHTTPErrorDecoderFunc() *tsg.Statement {

	stmt := tsg.NewStatement()
	stmt.Export().Func("defaultHTTPErrorDecoder")
	stmt.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("statusCode").Colon().ID(typeNumber))
		pg.Add(tsg.NewStatement().ID("contentType").Colon().ID(typeString))
		pg.Add(tsg.NewStatement().ID("body").Colon().ID(typeString))
	})
	stmt.Colon().ID("Error")
	stmt.Block(func(bg *tsg.Group) {
		bg.Return(tsg.NewStatement().New("ResponseError").Call(
			tsg.NewStatement().ID("statusCode"),
			tsg.NewStatement().ID("contentType"),
			tsg.NewStatement().ID("body"),
		))
	})
	return stmt
}
