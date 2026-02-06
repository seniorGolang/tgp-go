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

func (r *contractRenderer) RenderMetrics() error {

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName("context", "context")
	srcFile.ImportName(PackageStrconv, "strconv")
	srcFile.ImportName(PackageTime, "time")
	srcFile.ImportName(r.contract.PkgPath, filepath.Base(r.contract.PkgPath))
	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(PackagePrometheus, "metrics")
	srcFile.ImportName("fmt", "fmt")
	srcFile.ImportName(fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir)), "srvctx")

	typeGen := types.NewGenerator(r.project, &srcFile)

	srcFile.Line().Const().Defs(
		Id("metricService" + r.contract.Name).Op("=").Lit(toLowerCamel(r.contract.Name)),
	)
	for _, method := range r.contract.Methods {
		srcFile.Const().Id("metricMethod" + r.contract.Name + method.Name).Op("=").Lit(toLowerCamel(method.Name))
	}

	srcFile.Line().Type().Id("metrics"+r.contract.Name).Struct(
		Id(VarNameNext).Qual(r.contract.PkgPath, r.contract.Name),
		Id("metrics").Op("*").Id("Metrics"),
	)

	srcFile.Line().Add(r.metricsMiddleware())

	for _, method := range r.contract.Methods {
		srcFile.Line().Func().Params(Id("m").Id("metrics" + r.contract.Name)).
			Id(method.Name).
			Params(typeGen.FuncDefinitionParams(method.Args)).
			Params(typeGen.FuncDefinitionParams(method.Results)).
			BlockFunc(r.metricFuncBody(method))
	}

	return srcFile.Save(path.Join(r.outDir, strings.ToLower(r.contract.Name)+"-metrics.go"))
}

func (r *contractRenderer) metricsMiddleware() Code {

	return Func().Id("metricsMiddleware"+r.contract.Name).
		Params(Id(VarNameNext).Qual(r.contract.PkgPath, r.contract.Name), Id("metrics").Op("*").Id("Metrics")).
		Params(Qual(r.contract.PkgPath, r.contract.Name)).
		BlockFunc(func(bg *Group) {
			bg.Return(Op("&").Id("metrics" + r.contract.Name).Values(
				Dict{
					Id(VarNameNext): Id(VarNameNext),
					Id("metrics"):   Id("metrics"),
				},
			))
		})
}

func (r *contractRenderer) metricFuncBody(method *model.Method) func(bg *Group) {

	return func(bg *Group) {

		errCodeAssignment := Id("errCode").Op("=")

		if r.methodIsHTTP(method) {
			errCodeAssignment.Qual(PackageFiber, "StatusInternalServerError")
		} else {
			errCodeAssignment.Id("internalError")
		}

		bg.Line().Defer().Func().Params(Id("_begin_").Qual(PackageTime, "Time")).Block(
			// Проверка на nil для метрик
			If(Id("m").Dot("metrics").Op("==").Nil()).Block(
				Return(),
			),
			Var().Defs(
				Id("success").Op("=").True(),
				Id("errCode").Int(),
			),
			If(Err().Op("!=").Nil()).Block(
				Id("success").Op("=").False(),
				errCodeAssignment,
				List(Id("ec"), Id("ok")).Op(":=").Err().Assert(Id("withErrorCode")),
				If(Id("ok")).Block(
					Id("errCode").Op("=").Id("ec").Dot("Code").Call(),
				),
			),
			Id("clientID").Op(":=").Qual(fmt.Sprintf("%s/srvctx", r.pkgPath(r.outDir)), "GetClientID").Call(Id(VarNameCtx)),
			If(Id("success")).Block(
				Id("m").Dot("metrics").Dot("RequestCount").Dot("WithLabelValues").Call(
					Id("metricService"+r.contract.Name),
					Id("metricMethod"+r.contract.Name+method.Name),
					Id("metricSuccessTrue"),
					Id("metricErrCodeSuccess"),
					Id("clientID")).
					Dot("Add").Call(Lit(1)),
				Id("m").Dot("metrics").Dot("RequestLatency").Dot("WithLabelValues").Call(
					Id("metricService"+r.contract.Name),
					Id("metricMethod"+r.contract.Name+method.Name),
					Id("metricSuccessTrue"),
					Id("metricErrCodeSuccess"),
					Id("clientID")).
					Dot("Observe").Call(Id("float64").Call(Qual(PackageTime, "Since").Call(Id("_begin_")).Dot("Seconds").Call())),
			).Else().Block(
				Id("errCodeStr").Op(":=").Qual(PackageStrconv, "Itoa").Call(Id("errCode")),
				Id("m").Dot("metrics").Dot("RequestCount").Dot("WithLabelValues").Call(
					Id("metricService"+r.contract.Name),
					Id("metricMethod"+r.contract.Name+method.Name),
					Id("metricSuccessFalse"),
					Id("errCodeStr"),
					Id("clientID")).
					Dot("Add").Call(Lit(1)),
				Id("m").Dot("metrics").Dot("RequestLatency").Dot("WithLabelValues").Call(
					Id("metricService"+r.contract.Name),
					Id("metricMethod"+r.contract.Name+method.Name),
					Id("metricSuccessFalse"),
					Id("errCodeStr"),
					Id("clientID")).
					Dot("Observe").Call(Id("float64").Call(Qual(PackageTime, "Since").Call(Id("_begin_")).Dot("Seconds").Call())),
			),
		).Call(Qual(PackageTime, "Now").Call())

		bg.Line().Return().Id("m").Dot(VarNameNext).Dot(method.Name).Call(r.paramNames(method.Args))
	}
}

func (r *contractRenderer) methodIsHTTP(method *model.Method) bool {

	contractHasHTTP := model.IsAnnotationSet(r.project, r.contract, nil, nil, TagServerHTTP)
	if !contractHasHTTP {
		return false
	}
	contractHasJsonRPC := model.IsAnnotationSet(r.project, r.contract, nil, nil, TagServerJsonRPC)
	methodHasExplicitHTTP := model.IsAnnotationSet(r.project, r.contract, method, nil, TagMethodHTTP)
	return !contractHasJsonRPC || methodHasExplicitHTTP
}

func (r *contractRenderer) paramNames(vars []*model.Variable) *Statement {

	var list = make([]Code, 0, len(vars))
	for _, v := range vars {
		paramCode := Id(toLowerCamel(v.Name))
		if v.IsEllipsis {
			paramCode.Op("...")
		}
		list = append(list, paramCode)
	}
	return List(list...)
}
