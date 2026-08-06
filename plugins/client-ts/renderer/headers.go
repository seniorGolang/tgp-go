// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"

	"tgp/plugins/client-ts/tsg"
)

func (r *ClientRenderer) RenderHeaders() (err error) {

	file := tsg.NewFile()
	file.ImportType("./options", "ClientOptions")
	file.GenerateImports()
	file.Line()
	if r.clientIdentity {
		file.Add(tsg.NewStatement().Export().Const("headerClientId").Op("=").Lit("X-Client-Id").Semicolon())
		file.Line()
		file.Add(renderBuildClientHeaders(true))
	} else {
		file.Add(renderBuildClientHeaders(false))
	}
	file.Line()
	err = file.Save(path.Join(r.outDir, "headers.ts"))
	return
}

func renderBuildClientHeaders(withIdentity bool) (stmt *tsg.Statement) {

	stmt = tsg.NewStatement()
	stmt.Export()
	stmt.Async()
	stmt.Func("buildClientHeaders")
	stmt.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("options").Colon().ID("ClientOptions"))
	})
	stmt.Colon().ID("Promise").Generic("Record<string, string>")
	stmt.Block(func(mg *tsg.Group) {
		if !withIdentity {
			mg.If(tsg.NewStatement().ID("options").Dot("headers").Op("&&").Typeof(tsg.NewStatement().ID("options").Dot("headers"), "function"), func(ig *tsg.Group) {
				ig.Return(tsg.NewStatement().ID("options").Dot("headers").Call())
			})
			mg.If(tsg.NewStatement().ID("options").Dot("headers"), func(ig *tsg.Group) {
				ig.Return(tsg.NewStatement().ID("options").Dot("headers"))
			})
			mg.Return(tsg.NewStatement().ObjectLiteral(nil))
			return
		}
		mg.Add(tsg.NewStatement().Var("userHeaders").Colon().ID("Record").Generic(typeString, typeString).Op("=").ObjectLiteral(nil).Semicolon())
		mg.If(tsg.NewStatement().ID("options").Dot("headers").Op("&&").Typeof(tsg.NewStatement().ID("options").Dot("headers"), "function"), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().ID("userHeaders").Op("=").Await(tsg.NewStatement().ID("options").Dot("headers").Call()).Semicolon())
		})
		mg.ElseIf(tsg.NewStatement().ID("options").Dot("headers"), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().ID("userHeaders").Op("=").ID("options").Dot("headers").Semicolon())
		})
		mg.Add(tsg.NewStatement().Const(nameResult).Colon().ID("Record").Generic(typeString, typeString).Op("=").ObjectLiteral(func(og *tsg.Group) {
			og.Add(tsg.NewStatement().Spread(tsg.NewStatement().ID("userHeaders")))
		}).Semicolon())
		mg.If(tsg.NewStatement().Bang().Paren(tsg.NewStatement().InExpr(tsg.NewStatement().ID("headerClientId"), tsg.NewStatement().ID(nameResult))), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().ID(nameResult).Index(tsg.NewStatement().ID("headerClientId")).Op("=").ID("options").Dot("clientName").Semicolon())
		})
		mg.Return(tsg.NewStatement().ID(nameResult))
	})
	return stmt
}
