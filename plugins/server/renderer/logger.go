// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive

	"tgp/internal/model"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) RenderLogger() (err error) {

	if err = r.pkgRenderTo("srvctx", r.outDir, r.srvctxTemplateData()); err != nil {
		return fmt.Errorf("render srvctx package: %w", err)
	}
	if err = r.pkgRenderTo("viewer", r.outDir, newPkgTemplateData()); err != nil {
		return fmt.Errorf("render viewer package: %w", err)
	}

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(r.pkgPath(r.outDir)+"/srvctx", "srvctx")
	srcFile.ImportName(PackageTime, "time")
	srcFile.ImportName(r.pkgPath(r.outDir)+"/viewer", "viewer")
	srcFile.ImportName(r.contract.PkgPath, filepath.Base(r.contract.PkgPath))

	typeGen := types.NewGenerator(r.project, &srcFile)

	srcFile.Type().Id("logger" + r.contract.Name).Struct(
		Id(VarNameNext).Qual(r.contract.PkgPath, r.contract.Name),
	)

	srcFile.Line().Add(r.loggerMiddleware())

	for _, method := range r.contract.Methods {
		srcFile.Line().Func().Params(Id("m").Id("logger" + r.contract.Name)).
			Id(method.Name).
			Params(typeGen.FuncDefinitionParams(method.Args)).
			Params(typeGen.FuncDefinitionParams(method.Results)).
			BlockFunc(r.loggerFuncBody(method))
	}

	err = srcFile.Save(path.Join(r.outDir, strings.ToLower(r.contract.Name)+"-logger.go"))
	return
}

func (r *contractRenderer) loggerMiddleware() (c Code) {

	return Func().Id("loggerMiddleware" + r.contract.Name).
		Params().
		Params(Id("Middleware" + r.contract.Name)).
		Block(
			Return(Func().Params(Id(VarNameNext).Qual(r.contract.PkgPath, r.contract.Name)).Params(Qual(r.contract.PkgPath, r.contract.Name)).Block(
				Return(Op("&").Id("logger" + r.contract.Name).Values(Dict{
					Id(VarNameNext): Id(VarNameNext),
				})),
			)),
		)
}

func (r *contractRenderer) loggerFuncBody(method *model.Method) func(bg *Group) {

	sides, names := model.MethodLogSkip(r.project, r.contract, method)
	reqValue := r.requestLogValue(method, names)
	respValue := r.responseLogValues(method, names)
	skipRequest := sides.Request || reqValue == nil
	skipResponse := sides.Response || respValue == nil

	capacity := 1
	if !skipRequest {
		capacity++
	}
	if !skipResponse {
		capacity++
	}
	capacity++

	return func(bg *Group) {
		bg.Id("sLogger").Op(":=").Qual(r.pkgPath(r.outDir)+"/srvctx", "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(Id(VarNameCtx))
		bg.If(Id("sLogger").Op("==").Nil()).Block(
			Id("sLogger").Op("=").Qual(PackageSlog, "Default").Call(),
		)
		bg.Id("_begin_").Op(":=").Qual(PackageTime, "Now").Call()
		bg.Defer().Func().Params().BlockFunc(func(bg *Group) {
			bg.Id("r").Op(":=").Id("recover").Call()
			bg.If(Id("r").Op("!=").Nil()).Block(
				Id("panic").Call(Id("r")),
			)

			bg.If(
				Op("!").Id("sLogger").Dot("Enabled").Call(Id(VarNameCtx), Qual(PackageSlog, "LevelInfo")).Op("&&").Id("err").Op("==").Nil().
					Op("||").
					Op("!").Id("sLogger").Dot("Enabled").Call(Id(VarNameCtx), Qual(PackageSlog, "LevelError")).Op("&&").Id("err").Op("!=").Nil(),
			).Block(Return())

			r.loggerDeferAttrsBlock(bg, skipRequest, skipResponse, reqValue, respValue, capacity)
			bg.If(Id("err").Op("!=").Nil()).Block(
				Id("_attrs_").Op("=").Append(Id("_attrs_"), Qual(PackageSlog, "Any").Call(Lit("error"), Err())),
			)
			bg.If(Id("err").Op("!=").Nil()).Block(
				Id("sLogger").Dot("Error").Call(Lit("call "+toLowerCamel(method.Name)), Id("_attrs_").Op("...")),
				Return(),
			)
			bg.Id("sLogger").Dot("Info").Call(Lit("call "+toLowerCamel(method.Name)), Id("_attrs_").Op("..."))
		}).Call()
		bg.Return().Id("m").Dot(VarNameNext).Dot(method.Name).Call(r.paramNames(method.Args))
	}
}

func (r *contractRenderer) loggerDeferAttrsBlock(bg *Group, skipRequest bool, skipResponse bool, reqValue Code, respValue Code, capacity int) {

	bg.Id("_attrs_").Op(":=").Make(Index().Any(), Lit(0), Lit(capacity))
	bg.Id("_attrs_").Op("=").Append(Id("_attrs_"),
		Qual(PackageSlog, "String").Call(Lit("took"), Qual(PackageTime, "Since").Call(Id("_begin_")).Dot("String").Call()),
	)
	if !skipRequest && reqValue != nil {
		bg.Id("_attrs_").Op("=").Append(Id("_attrs_"), Qual(r.pkgPath(r.outDir)+"/viewer", "Any").Call(Lit("request"), reqValue))
	}
	if !skipResponse && respValue != nil {
		bg.Id("_attrs_").Op("=").Append(Id("_attrs_"), Qual(r.pkgPath(r.outDir)+"/viewer", "Any").Call(Lit("response"), respValue))
	}
}

func (r *contractRenderer) requestLogValue(method *model.Method, names map[string]struct{}) (value Code) {

	logged := r.loggerLoggedArgs(method, names)
	if len(logged) == 0 {
		return nil
	}
	if r.varsNeedLogProjection(logged) {
		return Map(String()).Any().Values(DictFunc(func(d Dict) {
			for _, arg := range logged {
				d[Lit(r.requestStructFieldName(method, arg))] = r.logViewExpr(&arg.TypeRef, Id(arg.Name))
			}
		}))
	}
	return Id(requestStructName(r.contract.Name, method.Name)).Values(DictFunc(func(d Dict) {
		for _, arg := range logged {
			d[Id(r.requestStructFieldName(method, arg))] = Id(arg.Name)
		}
	}))
}

func (r *contractRenderer) loggerLoggedArgs(method *model.Method, names map[string]struct{}) (logged []*model.Variable) {

	for _, arg := range streamVariables(r.project, argsWithoutContext(method), false) {
		if model.VariableLogSkipped(method, arg, names) {
			continue
		}
		logged = append(logged, arg)
	}
	return
}

func (r *contractRenderer) varsNeedLogProjection(vars []*model.Variable) (needed bool) {

	for _, v := range vars {
		if model.TypeRefNeedsLogProjection(r.project, &v.TypeRef) {
			return true
		}
	}
	return false
}
