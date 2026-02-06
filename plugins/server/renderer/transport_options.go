// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

func (r *transportRenderer) RenderTransportOptions() error {

	optionsPath := path.Join(r.outDir, "options.go")

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(PackageUUID, "uuid")
	srcFile.ImportName(PackageTime, "time")

	r.renderOptionsTypes(&srcFile)
	r.renderOptionsService(&srcFile)
	r.renderOptionsForContracts(&srcFile)
	r.renderOptionsConfig(&srcFile)
	r.renderOptionsTimeouts(&srcFile)
	r.renderOptionsHeaders(&srcFile)
	r.renderOptionsUse(&srcFile)

	return srcFile.Save(optionsPath)
}

func (r *transportRenderer) renderOptionsTypes(srcFile *GoFile) {

	srcFile.Line().Type().Id("ServiceRoute").Interface(
		Id("SetRoutes").Params(Id("route").Op("*").Qual(PackageFiber, "App")),
	)

	srcFile.Line().Type().Id("Option").Func().Params(Id("srv").Op("*").Id("Server"))
	srcFile.Type().Id("Handler").Op("=").Qual(PackageFiber, "Handler")
	srcFile.Type().Id("ErrorHandler").Func().Params(Err().Error()).Params(Error())
}

func (r *transportRenderer) renderOptionsService(srcFile *GoFile) {

	srcFile.Line().Func().Id("Service").
		Params(Id("svc").Id("ServiceRoute")).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).Block(
					Id("svc").Dot("SetRoutes").Call(Id("srv").Dot("Fiber").Call()),
				),
			)),
		)
}

func (r *transportRenderer) renderOptionsForContracts(srcFile *GoFile) {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerHTTP) {
			srcFile.ImportName(contract.PkgPath, filepath.Base(contract.PkgPath))
			block := func(gr *Group) {
				gr.Id("httpSvc").Op(":=").Id("new" + contract.Name).Call(Id("svc"))
				gr.Id("srv").Dot("http" + contract.Name).Op("=").Id("httpSvc")
				if r.hasJsonRPC() {
					gr.Id("httpSvc").Dot("maxBatchSize").Op("=").Id("srv").Dot("maxBatchSize")
					gr.Id("httpSvc").Dot("maxParallelBatch").Op("=").Id("srv").Dot("maxParallelBatch")
				}
				gr.Id("httpSvc").Dot("SetRoutes").Call(Id("srv").Dot("Fiber").Call())
			}
			srcFile.Line().Func().Id(contract.Name).
				Params(Id("svc").Qual(contract.PkgPath, contract.Name)).
				Id("Option").
				Block(
					Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
						If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).BlockFunc(block),
					)),
				)
		}
	}
	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerJsonRPC) {
			srcFile.ImportName(contract.PkgPath, filepath.Base(contract.PkgPath))
			srcFile.Line().Func().Id(contract.Name).
				Params(Id("svc").Qual(contract.PkgPath, contract.Name)).
				Id("Option").
				Block(
					Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
						If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).BlockFunc(func(gr *Group) {
							gr.Id("httpSvc").Op(":=").Id("new" + contract.Name).Call(Id("svc"))
							gr.Id("srv").Dot("http" + contract.Name).Op("=").Id("httpSvc")
							gr.Id("httpSvc").Dot("srv").Op("=").Id("srv")
							gr.Id("httpSvc").Dot("SetRoutes").Call(Id("srv").Dot("Fiber").Call())
						}),
					)),
				)
		}
	}
}

func (r *transportRenderer) renderOptionsConfig(srcFile *GoFile) {

	srcFile.Line().Func().Id("SetFiberCfg").
		Params(Id("cfg").Qual(PackageFiber, "Config")).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("config").Op("=").Id("cfg"),
				Id("srv").Dot("config").Dot("DisableStartupMessage").Op("=").True(),
			)),
		)
	srcFile.Line().Func().Id("SetReadBufferSize").
		Params(Id("size").Int()).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("config").Dot("ReadBufferSize").Op("=").Id("size"),
			)),
		)
	srcFile.Line().Func().Id("SetWriteBufferSize").
		Params(Id("size").Int()).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("config").Dot("WriteBufferSize").Op("=").Id("size"),
			)),
		)
	srcFile.Line().Func().Id("MaxBodySize").
		Params(Id("size").Int()).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("config").Dot("BodyLimit").Op("=").Id("size"),
			)),
		)
	if r.hasJsonRPC() {
		srcFile.Line().Func().Id("MaxBatchSize").
			Params(Id("size").Int()).
			Id("Option").
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					Id("srv").Dot("maxBatchSize").Op("=").Id("size"),
				)),
			)
		srcFile.Line().Func().Id("MaxBatchWorkers").
			Params(Id("size").Int()).
			Id("Option").
			Block(
				Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
					Id("srv").Dot("maxParallelBatch").Op("=").Id("size"),
				)),
			)
	}
}

func (r *transportRenderer) renderOptionsTimeouts(srcFile *GoFile) {

	srcFile.Line().Func().Id("ReadTimeout").
		Params(Id("timeout").Qual(PackageTime, "Duration")).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("config").Dot("ReadTimeout").Op("=").Id("timeout"),
			)),
		)
	srcFile.Line().Func().Id("WriteTimeout").
		Params(Id("timeout").Qual(PackageTime, "Duration")).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("config").Dot("WriteTimeout").Op("=").Id("timeout"),
			)),
		)
}

func (r *transportRenderer) renderOptionsHeaders(srcFile *GoFile) {

	srcFile.Line().Func().Id("WithRequestID").
		Params(Id("headerName").String()).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("headerHandlers").Op("[").Id("headerName").Op("]").Op("=").
					Func().Params(Id("value").String()).Params(Id("Header")).Block(
					If(Id("value").Op("==").Lit("")).Block(
						Id("value").Op("=").Qual(PackageUUID, "New").Call().Dot("String").Call(),
					),
					Return(Id("Header").Block(Dict{
						Id("SpanKey"):       Lit("requestID"),
						Id("SpanValue"):     Id("value"),
						Id("ResponseKey"):   Id("headerName"),
						Id("ResponseValue"): Id("value"),
						Id("LogKey"):        Lit("requestID"),
						Id("LogValue"):      Id("value"),
					})),
				),
			)),
		)
	srcFile.Line().Func().Id("WithHeader").
		Params(Id("headerName").String(), Id("handler").Id("HeaderHandler")).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				Id("srv").Dot("headerHandlers").Op("[").Id("headerName").Op("]").Op("=").Id("handler"),
			)),
		)
}

func (r *transportRenderer) renderOptionsUse(srcFile *GoFile) {

	srcFile.Line().Func().Id("Use").
		Params(Id("args").Op("...").Any()).
		Id("Option").
		Block(
			Return(Func().Params(Id("srv").Op("*").Id("Server")).Block(
				If(Id("srv").Dot("srvHTTP").Op("!=").Nil()).Block(
					Id("srv").Dot("srvHTTP").Dot("Use").Call(Id("args").Op("...")),
				),
			)),
		)
}
