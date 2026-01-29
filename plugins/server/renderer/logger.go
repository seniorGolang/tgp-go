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

	if err := r.pkgCopyTo("viewer", r.outDir); err != nil {
		return fmt.Errorf("copy viewer package: %w", err)
	}

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(r.contract.PkgPath, filepath.Base(r.contract.PkgPath))
	srcFile.ImportName(fmt.Sprintf("%s/viewer", r.pkgPath(r.outDir)), "viewer")

	typeGen := types.NewGenerator(r.project, &srcFile)

	srcFile.Line().Const().Id("logService" + r.contract.Name).Op("=").Lit(r.contract.Name)
	for _, method := range r.contract.Methods {
		srcFile.Const().Id("logMethod" + r.contract.Name + method.Name).Op("=").Lit(toLowerCamel(method.Name))
	}

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

	return func(bg *Group) {
		bg.Id("sLogger").Op(":=").Id("FromContext").Call(Id(VarNameCtx))
		bg.If(Id("sLogger").Op("==").Nil()).Block(
			Id("sLogger").Op("=").Qual(PackageSlog, "Default").Call(),
		)
		bg.Id("_begin_").Op(":=").Qual(PackageTime, "Now").Call()
		bg.Defer().Func().Params().BlockFunc(func(bg *Group) {
			// Ленивое форматирование: проверяем уровень логирования перед форматированием
			bg.If(Op("!").Id("sLogger").Dot("Enabled").Call(Id(VarNameCtx), Qual(PackageSlog, "LevelInfo")).Op("&&").Id("err").Op("==").Nil()).Block(
				Return(), // Логирование отключено, не форматируем
			)
			bg.If(Op("!").Id("sLogger").Dot("Enabled").Call(Id(VarNameCtx), Qual(PackageSlog, "LevelError")).Op("&&").Id("err").Op("!=").Nil()).Block(
				Return(), // Логирование ошибок отключено
			)

			// Форматируем только если нужно логировать
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

			// Базовые attrs (всегда нужны)
			bg.Var().Id("_attrs_").Index().Qual(PackageSlog, "Attr")
			bg.Id("_attrs_").Op("=").Append(Id("_attrs_"),
				Qual(PackageSlog, "String").Call(Lit("service"), Id("logService"+r.contract.Name)),
				Qual(PackageSlog, "String").Call(Lit("method"), Id("logMethod"+r.contract.Name+method.Name)),
				Qual(PackageSlog, "String").Call(Lit("took"), Qual(PackageTime, "Since").Call(Id("_begin_")).Dot("String").Call()),
			)

			// Обработка ошибки - ленивое форматирование только здесь
			bg.If(Id("err").Op("!=").Id("nil")).BlockFunc(func(bg *Group) {
				// Ленивое форматирование request только если не пропущен
				if !skipRequest {
					params := removeSkippedFields(r.ArgsFieldsWithoutContext(method), skipFields)
					originParams := removeSkippedFields(argsWithoutContext(method), skipFields)
					bg.Id("_attrs_").Op("=").Append(Id("_attrs_"), Qual(PackageSlog, "String").Call(Lit("request"), Qual(fmt.Sprintf("%s/viewer", r.pkgPath(r.outDir)), "Sprintf").Call(Lit("%+v"), Id(requestStructName(r.contract.Name, method.Name)).Values(r.dictByNormalVariables(params, originParams)))))
				}
				// Ленивое форматирование response только если не пропущен
				if !skipResponse {
					returns := r.ResultFieldsWithoutError(method)
					originReturns := resultsWithoutError(method)
					bg.Id("_attrs_").Op("=").Append(Id("_attrs_"), Qual(PackageSlog, "String").Call(Lit("response"), Qual(fmt.Sprintf("%s/viewer", r.pkgPath(r.outDir)), "Sprintf").Call(Lit("%+v"), Id(responseStructName(r.contract.Name, method.Name)).Values(r.dictByNormalVariables(returns, originReturns)))))
				}
				bg.Id("_attrs_").Op("=").Append(Id("_attrs_"), Qual(PackageSlog, "Any").Call(Lit("error"), Err()))
				bg.Var().Id("args").Index().Any()
				bg.For(List(Id("_"), Id("attr")).Op(":=").Range().Id("_attrs_")).Block(
					Id("args").Op("=").Append(Id("args"), Id("attr")),
				)
				bg.Id("sLogger").Dot("Error").Call(Lit(fmt.Sprintf("call %s", toLowerCamel(method.Name))), Id("args").Op("..."))
				bg.Return()
			})

			// Успешное выполнение - ленивое форматирование только здесь (не выполняется при ошибке)
			if !skipRequest {
				params := removeSkippedFields(r.ArgsFieldsWithoutContext(method), skipFields)
				originParams := removeSkippedFields(argsWithoutContext(method), skipFields)
				bg.Id("_attrs_").Op("=").Append(Id("_attrs_"), Qual(PackageSlog, "String").Call(Lit("request"), Qual(fmt.Sprintf("%s/viewer", r.pkgPath(r.outDir)), "Sprintf").Call(Lit("%+v"), Id(requestStructName(r.contract.Name, method.Name)).Values(r.dictByNormalVariables(params, originParams)))))
			}
			if !skipResponse {
				returns := r.ResultFieldsWithoutError(method)
				originReturns := resultsWithoutError(method)
				bg.Id("_attrs_").Op("=").Append(Id("_attrs_"), Qual(PackageSlog, "String").Call(Lit("response"), Qual(fmt.Sprintf("%s/viewer", r.pkgPath(r.outDir)), "Sprintf").Call(Lit("%+v"), Id(responseStructName(r.contract.Name, method.Name)).Values(r.dictByNormalVariables(returns, originReturns)))))
			}
			bg.Var().Id("args").Index().Any()
			bg.For(List(Id("_"), Id("attr")).Op(":=").Range().Id("_attrs_")).Block(
				Id("args").Op("=").Append(Id("args"), Id("attr")),
			)
			bg.Id("sLogger").Dot("Info").Call(Lit(fmt.Sprintf("call %s", toLowerCamel(method.Name))), Id("args").Op("..."))
		}).Call()
		bg.Return().Id("m").Dot(VarNameNext).Dot(method.Name).Call(r.paramNames(method.Args))
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
