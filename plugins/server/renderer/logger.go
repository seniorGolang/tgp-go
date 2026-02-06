// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) RenderLogger() error {

	if err := r.pkgCopyTo("srvctx", r.outDir); err != nil {
		return fmt.Errorf("copy srvctx package: %w", err)
	}
	if err := r.pkgCopyTo("viewer", r.outDir); err != nil {
		return fmt.Errorf("copy viewer package: %w", err)
	}

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir)), "srvctx")
	srcFile.ImportName(PackageTime, "time")
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

	return srcFile.Save(path.Join(r.outDir, strings.ToLower(r.contract.Name)+"-logger.go"))
}

func (r *contractRenderer) loggerMiddleware() Code {

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

	skipFields := strings.Split(model.GetAnnotationValue(r.project, r.contract, method, nil, TagLogSkip, ""), ",")
	skipRequest := false
	skipResponse := false
	for _, field := range skipFields {
		field = strings.TrimSpace(field)
		if field == "request" {
			skipRequest = true
		}
		if field == "response" {
			skipResponse = true
		}
	}

	capSuccess := 1
	if !skipRequest {
		capSuccess++
	}
	if !skipResponse {
		capSuccess++
	}
	capError := capSuccess + 1

	return func(bg *Group) {
		bg.Id("sLogger").Op(":=").Qual(fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir)), "FromCtx").Types(Op("*").Qual(PackageSlog, "Logger")).Call(Id(VarNameCtx))
		bg.If(Id("sLogger").Op("==").Nil()).Block(
			Id("sLogger").Op("=").Qual(PackageSlog, "Default").Call(),
		)
		bg.Id("_begin_").Op(":=").Qual(PackageTime, "Now").Call()
		bg.Defer().Func().Params().BlockFunc(func(bg *Group) {
			// При панике не выполняем логирование ниже — только пробрасываем (обработка в recoverHandler)
			bg.Id("r").Op(":=").Id("recover").Call()
			bg.If(Id("r").Op("!=").Nil()).Block(
				Id("panic").Call(Id("r")),
			)

			bg.If(Op("!").Id("sLogger").Dot("Enabled").Call(Id(VarNameCtx), Qual(PackageSlog, "LevelInfo")).Op("&&").Id("err").Op("==").Nil()).Block(
				Return(),
			)
			bg.If(Op("!").Id("sLogger").Dot("Enabled").Call(Id(VarNameCtx), Qual(PackageSlog, "LevelError")).Op("&&").Id("err").Op("!=").Nil()).Block(
				Return(),
			)

			r.loggerDeferAttrsBlock(bg, method, skipRequest, skipResponse, skipFields)
			bg.If(Id("err").Op("!=").Nil()).Block(
				Id("_attrs_").Op("=").Append(Id("_attrs_"), Qual(PackageSlog, "Any").Call(Lit("error"), Err())),
			)
			bg.Id("args").Op(":=").Make(Index().Any(), Lit(0), Lit(capError))
			bg.For(List(Id("_"), Id("attr")).Op(":=").Range().Id("_attrs_")).Block(
				Id("args").Op("=").Append(Id("args"), Id("attr")),
			)
			bg.If(Id("err").Op("!=").Nil()).Block(
				Id("sLogger").Dot("Error").Call(Lit(fmt.Sprintf("call %s", toLowerCamel(method.Name))), Id("args").Op("...")),
				Return(),
			)
			bg.Id("sLogger").Dot("Info").Call(Lit(fmt.Sprintf("call %s", toLowerCamel(method.Name))), Id("args").Op("..."))
		}).Call()
		bg.Return().Id("m").Dot(VarNameNext).Dot(method.Name).Call(r.paramNames(method.Args))
	}
}

func (r *contractRenderer) loggerDeferAttrsBlock(bg *Group, method *model.Method, skipRequest bool, skipResponse bool, skipFields []string) {

	bg.Var().Id("_attrs_").Index().Qual(PackageSlog, "Attr")
	bg.Id("_attrs_").Op("=").Append(Id("_attrs_"),
		Qual(PackageSlog, "String").Call(Lit("took"), Qual(PackageTime, "Since").Call(Id("_begin_")).Dot("String").Call()),
	)
	if !skipRequest {
		params := removeSkippedFields(r.ArgsFieldsWithoutContext(method), skipFields)
		originParams := removeSkippedFields(argsWithoutContext(method), skipFields)
		bg.Id("_attrs_").Op("=").Append(Id("_attrs_"), Qual(PackageSlog, "Any").Call(Lit("request"), Id(requestStructName(r.contract.Name, method.Name)).Values(r.dictByNormalVariables(params, originParams))))
	}
	if !skipResponse {
		returns := r.ResultFieldsWithoutError(method)
		originReturns := resultsWithoutError(method)
		bg.Id("_attrs_").Op("=").Append(Id("_attrs_"), Qual(PackageSlog, "Any").Call(Lit("response"), Id(responseStructName(r.contract.Name, method.Name)).Values(r.dictByNormalVariables(returns, originReturns))))
	}
}

func (r *contractRenderer) dictByNormalVariables(fields []*model.Variable, normals []*model.Variable) Dict {

	if len(fields) != len(normals) {
		panic("len of fields and normals not the same")
	}
	return DictFunc(func(d Dict) {
		for i, field := range fields {
			normalVar := normals[i]
			normalVarCode := Id(toLowerCamel(normalVar.Name))

			// Если поле в структуре НЕ указатель, а нормальная переменная - указатель,
			// то нужно разыменовать указатель при использовании в struct literal
			if field.NumberOfPointers == 0 && normalVar.NumberOfPointers > 0 {
				normalVarCode = Op("*").Add(normalVarCode)
			}

			d[Id(toCamel(field.Name))] = normalVarCode
		}
	})
}
