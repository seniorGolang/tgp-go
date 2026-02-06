// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
	"tgp/plugins/server/renderer/types"
)

func (r *contractRenderer) RenderServer() error {

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(r.contract.PkgPath, filepath.Base(r.contract.PkgPath))

	typeGen := types.NewGenerator(r.project, &srcFile)

	srcFile.Line().Add(r.serverType(typeGen))
	srcFile.Line().Add(r.middlewareSetType(typeGen))
	srcFile.Line().Add(r.newServerFunc(typeGen))
	srcFile.Line().Add(r.wrapFunc(typeGen))

	for _, method := range r.contract.Methods {
		srcFile.Line().Func().Params(Id("srv").Op("*").Id("server" + r.contract.Name)).
			Id(method.Name).
			Params(typeGen.FuncDefinitionParams(method.Args)).
			Params(typeGen.FuncDefinitionParams(method.Results)).
			Block(
				Return(Id("srv").Dot(toLowerCamel(method.Name)).CallFunc(func(cg *Group) {
					for _, arg := range method.Args {
						argCode := Id(arg.Name)
						if arg.IsEllipsis {
							argCode.Op("...")
						}
						cg.Add(argCode)
					}
				})),
			)
	}

	for _, method := range r.contract.Methods {
		srcFile.Line().Func().Params(Id("srv").Op("*").Id("server" + r.contract.Name)).
			Id("Wrap" + method.Name).
			Params(Id("m").Id("Middleware" + r.contract.Name + method.Name)).
			Block(
				Id("srv").Dot(toLowerCamel(method.Name)).Op("=").Id("m").Call(Id("srv").Dot(toLowerCamel(method.Name))),
			)
	}

	if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagTrace) {
		srcFile.Line().Func().Params(Id("srv").Op("*").Id("server" + r.contract.Name)).Id("WithTrace").Params().Block(
			Id("srv").Dot("Wrap").Call(Id("traceMiddleware" + r.contract.Name)),
		)
	}

	if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagMetrics) {
		srcFile.Line().Func().Params(Id("srv").Op("*").Id("server" + r.contract.Name)).
			Id("WithMetrics").
			Params(Id("metrics").Op("*").Id("Metrics")).
			Block(
				Id("srv").Dot("Wrap").Call(Func().Params(Id("next").Qual(r.contract.PkgPath, r.contract.Name)).Params(Qual(r.contract.PkgPath, r.contract.Name)).Block(
					Return(Id("metricsMiddleware"+r.contract.Name).Call(Id("next"), Id("metrics"))),
				)),
			)
	}

	if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagLogger) {
		srcFile.Line().Func().Params(Id("srv").Op("*").Id("server" + r.contract.Name)).Id("WithLog").Params().Block(
			Id("srv").Dot("Wrap").Call(Id("loggerMiddleware" + r.contract.Name).Call()),
		)
	}

	return srcFile.Save(path.Join(r.outDir, strings.ToLower(r.contract.Name)+"-server.go"))
}

func (r *contractRenderer) serverType(typeGen *types.Generator) Code {

	return Type().Id("server" + r.contract.Name).StructFunc(func(sg *Group) {
		sg.Id("svc").Qual(r.contract.PkgPath, r.contract.Name)
		for _, method := range r.contract.Methods {
			sg.Id(toLowerCamel(method.Name)).Id(r.contract.Name + method.Name)
		}
	})
}

func (r *contractRenderer) middlewareSetType(typeGen *types.Generator) Code {

	return Type().Id("MiddlewareSet" + r.contract.Name).InterfaceFunc(func(ig *Group) {
		ig.Id("Wrap").Params(Id("m").Id("Middleware" + r.contract.Name))
		for _, method := range r.contract.Methods {
			ig.Id("Wrap" + method.Name).Params(Id("m").Id("Middleware" + r.contract.Name + method.Name))
		}
		ig.Line()
		if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagTrace) {
			ig.Id("WithTrace").Params()
		}
		if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagMetrics) {
			ig.Id("WithMetrics").Params(Id("metrics").Op("*").Id("Metrics"))
		}
		if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagLogger) {
			ig.Id("WithLog").Params()
		}
	})
}

func (r *contractRenderer) newServerFunc(typeGen *types.Generator) Code {

	return Func().Id("newServer" + r.contract.Name).
		Params(Id("svc").Qual(r.contract.PkgPath, r.contract.Name)).
		Params(Op("*").Id("server" + r.contract.Name)).
		Block(
			Return(Op("&").Id("server" + r.contract.Name).Values(DictFunc(func(dict Dict) {
				dict[Id("svc")] = Id("svc")
				for _, method := range r.contract.Methods {
					dict[Id(toLowerCamel(method.Name))] = Id("svc").Dot(method.Name)
				}
			}))),
		)
}

func (r *contractRenderer) wrapFunc(typeGen *types.Generator) Code {

	return Func().Params(Id("srv").Op("*").Id("server" + r.contract.Name)).
		Id("Wrap").
		Params(Id("m").Id("Middleware" + r.contract.Name)).
		BlockFunc(func(bg *Group) {
			bg.Id("srv").Dot("svc").Op("=").Id("m").Call(Id("srv").Dot("svc"))
			for _, method := range r.contract.Methods {
				bg.Id("srv").Dot(toLowerCamel(method.Name)).Op("=").Id("srv").Dot("svc").Dot(method.Name)
			}
		})
}
