// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"

	"tgp/plugins/client-ts/tsg"
)

// RenderForm пишет изоморфный form.ts (browser + Node): парсинг form/header/cookie строк без Node API.
func (r *ClientRenderer) RenderForm() (err error) {

	file := tsg.NewFile()
	file.Line()
	file.Add(renderFormParseHelpers())
	file.Line()
	err = file.Save(path.Join(r.outDir, "form.ts"))
	return
}

func renderFormParseHelpers() (stmt *tsg.Statement) {

	stmt = tsg.NewStatement()
	stmt.Export().Func("parseFormValue")
	stmt.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("val").Colon().ID(typeString).Op("|").ID("null"))
		pg.Add(tsg.NewStatement().ID("kind").Colon().Lit(typeString).Op("|").Lit(typeNumber).Op("|").Lit(typeBoolean).Op("|").Lit("json"))
	})
	stmt.Colon().ID(typeString).Op("|").ID(typeNumber).Op("|").ID(typeBoolean).Op("|").ID("unknown").Op("|").ID(typeUndefined)
	stmt.Block(func(bg *tsg.Group) {
		bg.Add(tsg.NewStatement().If(tsg.NewStatement().ID("val").Op("==").ID("null"), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Return(tsg.NewStatement().ID(typeUndefined)).Semicolon())
		}))
		bg.Add(tsg.NewStatement().If(tsg.NewStatement().ID("kind").Op("===").Lit(typeString), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Return(tsg.NewStatement().ID("val")).Semicolon())
		}))
		bg.Add(tsg.NewStatement().If(tsg.NewStatement().ID("kind").Op("===").Lit(typeNumber), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Const(tsLocalVar("n")).Op("=").ID("Number").Call(tsg.NewStatement().ID("val")).Semicolon())
			ig.Add(tsg.NewStatement().Return(tsg.NewStatement().ID("Number").Dot("isNaN").Call(tsg.NewStatement().ID(tsLocalVar("n"))).Op("?").ID(typeUndefined).Op(":").ID(tsLocalVar("n"))).Semicolon())
		}))
		bg.Add(tsg.NewStatement().If(tsg.NewStatement().ID("kind").Op("===").Lit(typeBoolean), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Return(tsg.NewStatement().ID("val").Op("===").Lit("true")).Semicolon())
		}))
		bg.Add(tsg.NewStatement().If(tsg.NewStatement().ID("kind").Op("===").Lit("json"), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Try(func(tg *tsg.Group) {
				tg.Add(tsg.NewStatement().Return(tsg.NewStatement().ID("JSON").Dot("parse").Call(tsg.NewStatement().ID("val"))).Semicolon())
			}, func(cg *tsg.Group) {
				cg.Add(tsg.NewStatement().Return(tsg.NewStatement().ID(typeUndefined)).Semicolon())
			}))
		}))
		bg.Add(tsg.NewStatement().Return(tsg.NewStatement().ID("val")).Semicolon())
	})
	stmt.Line()

	csv := tsg.NewStatement()
	csv.Export().Func("parseCSVFormValue")
	csv.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("val").Colon().ID(typeString).Op("|").ID("null"))
		pg.Add(tsg.NewStatement().ID("kind").Colon().Lit(typeString).Op("|").Lit(typeNumber).Op("|").Lit(typeBoolean).Op("|").Lit("json"))
	})
	csv.Colon().Add(tsg.TypeFromString("(string | number | boolean | unknown)[] | undefined"))
	csv.Block(func(bg *tsg.Group) {
		bg.Add(tsg.NewStatement().If(tsg.NewStatement().ID("val").Op("==").ID("null"), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Return(tsg.NewStatement().ID(typeUndefined)).Semicolon())
		}))
		bg.Add(tsg.NewStatement().Const("out").Colon().Add(tsg.TypeFromString("(string | number | boolean | unknown)[]")).Op("=").Index(nil).Semicolon())
		bg.Add(tsg.NewStatement().ForOf("part", "val.split(',')", func(fg *tsg.Group) {
			fg.Add(tsg.NewStatement().Const("elemStr").Op("=").ID("part").Dot("trim").Call().Semicolon())
			fg.Add(tsg.NewStatement().If(tsg.NewStatement().ID("elemStr").Op("===").Lit(""), func(ig *tsg.Group) {
				ig.Add(tsg.NewStatement().ID("continue").Semicolon())
			}))
			fg.Add(tsg.NewStatement().Const("elem").Op("=").ID("parseFormValue").Call(tsg.NewStatement().ID("elemStr"), tsg.NewStatement().ID("kind")).Semicolon())
			fg.Add(tsg.NewStatement().If(tsg.NewStatement().ID("kind").Op("===").Lit(typeNumber), func(ig *tsg.Group) {
				ig.Add(tsg.NewStatement().ID("out").Dot("push").Call(
					tsg.NewStatement().Typeof(tsg.NewStatement().ID("elem"), typeNumber).Op("?").ID("elem").Op(":").Lit(0),
				).Semicolon())
				ig.Add(tsg.NewStatement().ID("continue").Semicolon())
			}))
			fg.Add(tsg.NewStatement().If(tsg.NewStatement().ID("kind").Op("===").Lit(typeBoolean), func(ig *tsg.Group) {
				ig.Add(tsg.NewStatement().ID("out").Dot("push").Call(tsg.NewStatement().ID("elem").Op("===").ID("true")).Semicolon())
				ig.Add(tsg.NewStatement().ID("continue").Semicolon())
			}))
			fg.Add(tsg.NewStatement().If(tsg.NewStatement().ID("elem").Op("!==").ID(typeUndefined), func(ig *tsg.Group) {
				ig.Add(tsg.NewStatement().ID("out").Dot("push").Call(tsg.NewStatement().ID("elem")).Semicolon())
			}))
		}).Semicolon())
		bg.Add(tsg.NewStatement().Return(tsg.NewStatement().ID("out")).Semicolon())
	})
	csv.Line()
	stmt.Add(csv)
	return stmt
}
