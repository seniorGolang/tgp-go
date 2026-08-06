// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"

	"tgp/plugins/client-ts/tsg"
)

func (r *ClientRenderer) RenderIdentity() (err error) {

	if !r.clientIdentity {
		return
	}
	file := tsg.NewFile()
	file.ImportNamed("./version", "VersionASTg")
	file.GenerateImports()
	file.Line()
	file.Add(renderIdentityTypes())
	file.Line()
	file.Add(tsg.NewStatement().Var("cachedClientName").Colon().ID(typeString).Op("|").ID(typeUndefined).Semicolon())
	file.Line()
	file.Add(renderIsNodeRuntime())
	file.Line()
	file.Add(renderReadNodeHostname())
	file.Line()
	file.Add(renderFnv1a32())
	file.Line()
	file.Add(renderResolveBrowserToken())
	file.Line()
	file.Add(renderResolveBrowserID())
	file.Line()
	file.Add(renderResolveDefaultClientName())
	file.Line()
	err = file.Save(path.Join(r.outDir, "identity.ts"))
	return
}

func renderIdentityTypes() (stmt *tsg.Statement) {

	stmt = tsg.NewStatement()
	stmt.Type("NodeOS").Op("=").Add(tsg.TypeFromString("{hostname?: () => string}")).Semicolon()
	stmt.Line()
	stmt.Type("NodeProcess").Op("=").Add(tsg.TypeFromString("{versions?: {node?: string}; getBuiltinModule?: (id: string) => NodeOS}")).Semicolon()
	stmt.Line()
	stmt.Type("NodeRequire").Op("=").Add(tsg.TypeFromString("(id: string) => NodeOS")).Semicolon()
	return stmt
}

func renderIsNodeRuntime() (stmt *tsg.Statement) {

	stmt = tsg.NewStatement()
	stmt.Func("isNodeRuntime")
	stmt.Params(func(_ *tsg.Group) {})
	stmt.Colon().ID(typeBoolean)
	stmt.Block(func(bg *tsg.Group) {
		bg.Add(tsg.NewStatement().Const("proc").Op("=").Paren(tsg.NewStatement().ID("globalThis").Op("as").Add(tsg.TypeFromString("{process?: NodeProcess}"))).Dot("process").Semicolon())
		bg.Return(tsg.NewStatement().Typeof(tsg.NewStatement().ID("proc").OptionalChain("versions").OptionalChain("node"), typeString))
	})
	return stmt
}

func renderReadNodeHostname() (stmt *tsg.Statement) {

	stmt = tsg.NewStatement()
	stmt.Func("readNodeHostname")
	stmt.Params(func(_ *tsg.Group) {})
	stmt.Colon().ID(typeString)
	stmt.Block(func(bg *tsg.Group) {
		bg.Add(tsg.NewStatement().Const("proc").Op("=").Paren(tsg.NewStatement().ID("globalThis").Op("as").Add(tsg.TypeFromString("{process?: NodeProcess}"))).Dot("process").Semicolon())
		bg.Add(tsg.NewStatement().Add(tsg.TypeFromString(`const builtin = proc?.getBuiltinModule?.("node:os")`)).Semicolon())
		bg.Add(tsg.NewStatement().If(
			tsg.NewStatement().ID("builtin").Op("&&").Typeof(tsg.NewStatement().ID("builtin").Dot("hostname"), "function"),
			func(ig *tsg.Group) {
				ig.Return(tsg.NewStatement().ID("builtin").Dot("hostname").Bang().Call())
			},
		))
		bg.Add(tsg.NewStatement().Const("req").Op("=").Paren(tsg.NewStatement().ID("globalThis").Op("as").Add(tsg.TypeFromString("{require?: NodeRequire}"))).Dot("require").Semicolon())
		bg.Add(tsg.NewStatement().If(
			tsg.NewStatement().Typeof(tsg.NewStatement().ID("req"), "function"),
			func(ig *tsg.Group) {
				ig.Return(tsg.NewStatement().ID("req").Call(tsg.NewStatement().Lit("node:os")).Dot("hostname").Bang().Call())
			},
		))
		bg.Return(tsg.NewStatement().Lit(""))
	})
	return stmt
}

func renderFnv1a32() (stmt *tsg.Statement) {

	stmt = tsg.NewStatement()
	stmt.Func("fnv1a32")
	stmt.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("input").Colon().ID(typeString))
	})
	stmt.Colon().ID(typeString)
	stmt.Block(func(bg *tsg.Group) {
		bg.Add(tsg.NewStatement().Var("hash").Op("=").Lit(0x811c9dc5).Semicolon())
		bg.Add(tsg.NewStatement().For(
			tsg.NewStatement().ID("let i = 0"),
			tsg.NewStatement().ID("i < input.length"),
			tsg.NewStatement().ID("i++"),
			func(fg *tsg.Group) {
				fg.Add(tsg.NewStatement().ID("hash").Op("^=").ID("input").Dot("charCodeAt").Call(tsg.NewStatement().ID("i")).Semicolon())
				fg.Add(tsg.NewStatement().ID("hash").Op("=").ID("Math").Dot("imul").Call(tsg.NewStatement().ID("hash"), tsg.NewStatement().Lit(0x01000193)).Semicolon())
			},
		))
		bg.Return(tsg.NewStatement().Paren(tsg.NewStatement().ID("hash").Op(">>>").Lit(0)).Dot("toString").Call(tsg.NewStatement().Lit(16)).Dot("padStart").Call(tsg.NewStatement().Lit(8), tsg.NewStatement().Lit("0")))
	})
	return stmt
}

func renderResolveBrowserToken() (stmt *tsg.Statement) {

	stmt = tsg.NewStatement()
	stmt.Func("resolveBrowserToken")
	stmt.Params(func(_ *tsg.Group) {})
	stmt.Colon().ID(typeString)
	stmt.Block(func(bg *tsg.Group) {
		bg.Add(tsg.NewStatement().Const("ua").Op("=").ID("navigator").Dot("userAgent").Dot("toLowerCase").Call().Semicolon())
		bg.If(tsg.NewStatement().ID("ua").Dot("includes").Call(tsg.NewStatement().Lit("edg/")), func(ig *tsg.Group) {
			ig.Return(tsg.NewStatement().Lit("edge"))
		})
		bg.If(tsg.NewStatement().ID("ua").Dot("includes").Call(tsg.NewStatement().Lit("firefox/")), func(ig *tsg.Group) {
			ig.Return(tsg.NewStatement().Lit("firefox"))
		})
		bg.If(tsg.NewStatement().ID("ua").Dot("includes").Call(tsg.NewStatement().Lit("opr/")).Op("||").ID("ua").Dot("includes").Call(tsg.NewStatement().Lit("opera")), func(ig *tsg.Group) {
			ig.Return(tsg.NewStatement().Lit("opera"))
		})
		bg.If(tsg.NewStatement().ID("ua").Dot("includes").Call(tsg.NewStatement().Lit("chrome/")), func(ig *tsg.Group) {
			ig.Return(tsg.NewStatement().Lit("chrome"))
		})
		bg.If(tsg.NewStatement().ID("ua").Dot("includes").Call(tsg.NewStatement().Lit("safari/")), func(ig *tsg.Group) {
			ig.Return(tsg.NewStatement().Lit("safari"))
		})
		bg.Return(tsg.NewStatement().Lit("unknown"))
	})
	return stmt
}

func renderResolveBrowserID() (stmt *tsg.Statement) {

	stmt = tsg.NewStatement()
	stmt.Func("resolveBrowserId")
	stmt.Params(func(_ *tsg.Group) {})
	stmt.Colon().ID(typeString)
	stmt.Block(func(bg *tsg.Group) {
		bg.Return(tsg.NewStatement().ID("fnv1a32").Call(tsg.NewStatement().ID("navigator").Dot("userAgent")))
	})
	return stmt
}

func renderResolveDefaultClientName() (stmt *tsg.Statement) {

	stmt = tsg.NewStatement()
	stmt.Export().Func("resolveDefaultClientName")
	stmt.Params(func(_ *tsg.Group) {})
	stmt.Colon().ID(typeString)
	stmt.Block(func(bg *tsg.Group) {
		bg.If(tsg.NewStatement().ID("cachedClientName").Op("!==").ID(typeUndefined), func(ig *tsg.Group) {
			ig.Return(tsg.NewStatement().ID("cachedClientName"))
		})
		bg.If(tsg.NewStatement().ID("isNodeRuntime").Call(), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Const("hostname").Op("=").ID("readNodeHostname").Call().Semicolon())
			ig.Add(tsg.NewStatement().ID("cachedClientName").Op("=").ID("hostname").Op("!==").Lit("").Op("?").TemplateString([]string{"", "_astg_ts_", ""}, []*tsg.Statement{
				tsg.NewStatement().ID("hostname"),
				tsg.NewStatement().ID("VersionASTg"),
			}).Op(":").TemplateString([]string{"astg_ts_", ""}, []*tsg.Statement{
				tsg.NewStatement().ID("VersionASTg"),
			}).Semicolon())
			ig.Return(tsg.NewStatement().ID("cachedClientName"))
		})
		bg.Add(tsg.NewStatement().Const("browserToken").Op("=").ID("resolveBrowserToken").Call().Semicolon())
		bg.Add(tsg.NewStatement().Const("browserId").Op("=").ID("resolveBrowserId").Call().Semicolon())
		bg.Add(tsg.NewStatement().ID("cachedClientName").Op("=").TemplateString([]string{"", "_", "_astg_ts_", ""}, []*tsg.Statement{
			tsg.NewStatement().ID("browserToken"),
			tsg.NewStatement().ID("browserId"),
			tsg.NewStatement().ID("VersionASTg"),
		}).Semicolon())
		bg.Return(tsg.NewStatement().ID("cachedClientName"))
	})
	return stmt
}
