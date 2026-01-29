// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

func (r *transportRenderer) RenderTransportServer() error {

	serverPath := path.Join(r.outDir, "server.go")

	if err := r.pkgCopyTo("logger", r.outDir); err != nil {
		return fmt.Errorf("copy logger package: %w", err)
	}
	if r.hasTrace() {
		if err := r.pkgCopyTo("tracer", r.outDir); err != nil {
			return fmt.Errorf("copy tracer package: %w", err)
		}
	}

	srcFile := NewSrcFile(filepath.Base(r.outDir))
	srcFile.PackageComment(DoNotEdit)

	r.renderServerImports(&srcFile)
	r.renderServerTypes(&srcFile)
	r.renderServerConstants(&srcFile)
	r.renderServerFunctions(&srcFile)

	return srcFile.Save(serverPath)
}

func (r *transportRenderer) renderServerImports(srcFile *GoFile) {

	// Стандартные библиотеки (будут отсортированы goimports)
	srcFile.ImportName("context", "context")
	jsonPkg := r.getPackageJSON()
	srcFile.ImportName(jsonPkg, "json")
	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(PackageTime, "time")

	// Локальные пакеты
	srcFile.ImportName(fmt.Sprintf("%s/logger", r.pkgPath(r.outDir)), "logger")
	if r.hasTrace() {
		srcFile.ImportName(fmt.Sprintf("%s/tracer", r.pkgPath(r.outDir)), "tracer")
		srcFile.ImportName(PackageTraceSDK, "trace")
	}

	// Внешние пакеты
	srcFile.ImportName(PackageFiber, "fiber")
	srcFile.ImportName(PackagePrometheus, "prometheus")
	srcFile.ImportName(PackagePrometheusAuto, "promauto")
	srcFile.ImportName(PackagePrometheusHttp, "promhttp")

	for _, contract := range r.project.Contracts {
		srcFile.ImportName(contract.PkgPath, filepath.Base(contract.PkgPath))
	}
}

func (r *transportRenderer) renderServerTypes(srcFile *GoFile) {

	srcFile.Line().Add(r.transportServerType())
}

func (r *transportRenderer) renderServerConstants(srcFile *GoFile) {

	srcFile.Const().Id("defaultShutdownTimeout").Op("=").Lit(30).Op("*").Qual(PackageTime, "Second")
	srcFile.Line()
	srcFile.Const().Id("defaultBodyLimit").Op("=").Lit(8).Op("*").Lit(1024).Op("*").Lit(1024)
	srcFile.Const().Id("defaultReadBufferSize").Op("=").Lit(4096)
	srcFile.Const().Id("defaultWriteBufferSize").Op("=").Lit(4096)
	srcFile.Const().Id("defaultReadTimeout").Op("=").Lit(30).Op("*").Qual(PackageTime, "Second")
	srcFile.Const().Id("defaultWriteTimeout").Op("=").Lit(30).Op("*").Qual(PackageTime, "Second")
	srcFile.Const().Id("defaultIdleTimeout").Op("=").Lit(120).Op("*").Qual(PackageTime, "Second")
	srcFile.Const().Id("defaultConcurrency").Op("=").Lit(256).Op("*").Lit(1024)

	srcFile.Line().Add(r.healthServerType())
}

func (r *transportRenderer) renderServerFunctions(srcFile *GoFile) {

	srcFile.Line().Add(r.healthServerStopMethod())
	srcFile.Line().Add(r.serverNewFunc())
	srcFile.Line().Add(r.requiresHTTPFunc())
	srcFile.Line().Add(r.fiberFunc())
	srcFile.Line().Add(r.withLogFunc())
	srcFile.Line().Add(r.serveHealthFunc())
	srcFile.Line().Add(r.sendResponseFunc())
	srcFile.Line().Add(r.sendHTTPErrorFunc())
	srcFile.Line().Add(r.shutdownFunc())
	if r.hasTrace() {
		srcFile.Line().Add(r.withTraceFunc())
	}
	if r.hasMetrics() {
		srcFile.Line().Add(r.withMetricsFunc())
	}
	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerHTTP) {
			srcFile.Line().Add(r.httpServiceAccessorFunc(contract))
		}
	}
}

func (r *transportRenderer) transportServerType() Code {

	return Type().Id("Server").StructFunc(func(bg *Group) {
		bg.Id("log").Op("*").Qual(PackageSlog, "Logger")
		bg.Line().Id("config").Qual(PackageFiber, "Config")
		bg.Line().Id("srvHTTP").Op("*").Qual(PackageFiber, "App")
		bg.Id("srvMetrics").Op("*").Qual(PackageFiber, "App")
		if r.hasMetrics() {
			bg.Line().Id("metrics").Op("*").Id("Metrics")
		}
		if r.hasJsonRPC() {
			bg.Line().Id("maxBatchSize").Int()
			bg.Id("maxParallelBatch").Int()
			bg.Id("methodTimeout").Qual(PackageTime, "Duration")
			bg.Id("jsonRPCMethodMap").Map(String()).Id("methodJsonRPC").Line()
		}
		for _, contract := range r.project.Contracts {
			if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerHTTP) ||
				model.IsAnnotationSet(r.project, contract, nil, nil, TagServerJsonRPC) {
				bg.Line()
				bg.Id("http" + contract.Name).Op("*").Id("http" + contract.Name)
			}
		}
		bg.Line()
		bg.Id("headerHandlers").Map(String()).Id("HeaderHandler")
	})
}

func (r *transportRenderer) healthServerType() Code {

	return Type().Id("HealthServer").StructFunc(func(bg *Group) {
		bg.Id("srv").Op("*").Qual(PackageFiber, "App")
		bg.Id("responseBody").Index().Byte()
	})
}

func (r *transportRenderer) healthServerStopMethod() Code {

	return Func().Params(Id("hs").Op("*").Id("HealthServer")).
		Id("Stop").
		Params().
		Block(
			If(Id("hs").Dot("srv").Op("!=").Nil()).Block(
				If(Err().Op(":=").Id("hs").Dot("srv").Dot("ShutdownWithTimeout").Call(Id("defaultShutdownTimeout")).Op(";").Err().Op("!=").Nil()).Block(),
			),
		)
}

func (r *transportRenderer) httpServiceAccessorFunc(contract *model.Contract) Code {

	return Func().Params(Id("srv").Op("*").Id("Server")).
		Id(contract.Name).
		Params().
		Params(Op("*").Id("http" + contract.Name)).
		Block(
			Return(Id("srv").Dot("http" + contract.Name)),
		)
}
