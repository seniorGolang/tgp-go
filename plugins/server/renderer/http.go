// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

func (r *contractRenderer) RenderHTTP() error {

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageCors, "cors")
	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(r.contract.PkgPath, filepath.Base(r.contract.PkgPath))

	r.renderHTTPTypes(&srcFile)
	r.renderHTTPNewFunc(&srcFile)
	r.renderHTTPServiceFunc(&srcFile)
	r.renderHTTPWithFuncs(&srcFile)
	r.renderHTTPSetRoutes(&srcFile)

	return srcFile.Save(path.Join(r.outDir, strings.ToLower(r.contract.Name)+"-http.go"))
}

func (r *contractRenderer) renderHTTPTypes(srcFile *GoFile) {

	fields := []Code{
		Id("errorHandler").Id("ErrorHandler"),
		Id("svc").Op("*").Id("server" + r.contract.Name),
		Id("base").Qual(r.contract.PkgPath, r.contract.Name),
	}
	if r.hasJsonRPC() {
		fields = append(fields, Id("maxBatchSize").Int(), Id("maxParallelBatch").Int())
	}
	if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagServerJsonRPC) {
		fields = append(fields, Id("srv").Op("*").Id("Server"))
	}
	srcFile.Type().Id("http" + r.contract.Name).Struct(fields...)
}

func (r *contractRenderer) renderHTTPNewFunc(srcFile *GoFile) {

	srcFile.Line().Func().Id("new"+r.contract.Name).
		Params(Id("svc"+r.contract.Name).Qual(r.contract.PkgPath, r.contract.Name)).
		Params(Id("srv").Op("*").Id("http"+r.contract.Name)).
		Block(
			Line().Id("srv").Op("=").Op("&").Id("http"+r.contract.Name).Values(Dict{
				Id("base"): Id("svc" + r.contract.Name),
				Id("svc"):  Id("newServer" + r.contract.Name).Call(Id("svc" + r.contract.Name)),
			}),
			Return(),
		)
}

func (r *contractRenderer) renderHTTPServiceFunc(srcFile *GoFile) {

	srcFile.Line().Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).
		Id("Service").
		Params().
		Params(Op("*").Id("server" + r.contract.Name)).
		Block(
			Return(Id("http").Dot("svc")),
		)
}

func (r *contractRenderer) renderHTTPWithFuncs(srcFile *GoFile) {

	srcFile.Line().Add(r.httpWithLogFunc())
	if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagTrace) {
		srcFile.Line().Add(r.httpWithTraceFunc())
	}
	if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagMetrics) {
		srcFile.Line().Add(r.httpWithMetricsFunc())
	}
	srcFile.Line().Add(r.httpWithErrorHandler())
	if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagServerHTTP) {
		srcFile.Line().Add(r.httpWithRedirectFunc())
	}
}

func (r *contractRenderer) renderHTTPSetRoutes(srcFile *GoFile) {

	srcFile.Line().Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).
		Id("SetRoutes").
		Params(Id("route").Op("*").Qual(PackageFiber, "App")).
		BlockFunc(func(bg *Group) {
			if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagServerJsonRPC) {
				bg.Id("route").Dot("Post").Call(Lit(r.batchPath()), Id("http").Dot("serveBatch"))
				for _, method := range r.contract.Methods {
					if !r.methodIsJsonRPC(method) {
						continue
					}
					bg.Id("route").Dot("Post").Call(Lit(r.methodJsonRPCPath(method)), Id("http").Dot("serve"+method.Name))
				}
			}
			if model.IsAnnotationSet(r.project, r.contract, nil, nil, TagServerHTTP) {
				for _, method := range r.contract.Methods {
					if !r.methodIsHTTP(method) {
						continue
					}
					if model.IsAnnotationSet(r.project, r.contract, method, nil, TagHandler) {
						handlerQual := r.methodHandlerQual(srcFile, method)
						bg.Id("route").Dot(toCamel(r.methodHTTPMethod(method))).
							Call(Lit(r.methodHTTPPath(method)), Func().Params(Id(VarNameFtx).Op("*").Qual(PackageFiber, "Ctx")).Params(Err().Error()).Block(
								Return().Add(handlerQual).Call(Id(VarNameFtx), Id("http").Dot("base")),
							))
						continue
					}
					bg.Id("route").Dot(toCamel(r.methodHTTPMethod(method))).
						Call(Lit(r.methodHTTPPath(method)), Id("http").Dot("serve"+method.Name))
				}
			}
		})
}

func (r *contractRenderer) httpWithErrorHandler() Code {

	return Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).
		Id("WithErrorHandler").
		Params(Id("handler").Id("ErrorHandler")).
		Params(Op("*").Id("http" + r.contract.Name)).
		BlockFunc(func(bg *Group) {
			bg.Id("http").Dot("errorHandler").Op("=").Id("handler")
			bg.Return(Id("http"))
		})
}

func (r *contractRenderer) httpWithLogFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).
		Id("WithLog").
		Params().
		Params(Op("*").Id("http" + r.contract.Name)).
		BlockFunc(func(bg *Group) {
			bg.Id("http").Dot("svc").Dot("WithLog").Call()
			bg.Return(Id("http"))
		})
}

func (r *contractRenderer) httpWithTraceFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).
		Id("WithTrace").
		Params().
		Params(Op("*").Id("http" + r.contract.Name)).
		BlockFunc(func(bg *Group) {
			bg.Id("http").Dot("svc").Dot("WithTrace").Call()
			bg.Return(Id("http"))
		})
}

func (r *contractRenderer) httpWithMetricsFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).
		Id("WithMetrics").
		Params(Id("metrics").Op("*").Id("Metrics")).
		Params(Op("*").Id("http" + r.contract.Name)).
		BlockFunc(func(bg *Group) {
			bg.Id("http").Dot("svc").Dot("WithMetrics").Call(Id("metrics"))
			bg.Return(Id("http"))
		})
}

func (r *contractRenderer) httpWithRedirectFunc() Code {

	return Func().Params(Id("http").Op("*").Id("http" + r.contract.Name)).
		Id("WithRedirect").
		Params().
		Params(Op("*").Id("http" + r.contract.Name)).
		BlockFunc(func(bg *Group) {
			bg.Return(Id("http"))
		})
}
