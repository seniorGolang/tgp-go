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

func (r *ClientRenderer) RenderClient() error {

	outDir := r.outDir

	if err := r.RenderJsonRPCPackage(outDir); err != nil {
		return fmt.Errorf("не удалось сгенерировать пакет jsonrpc: %w", err)
	}

	srcFile := NewSrcFile(filepath.Base(outDir))
	srcFile.PackageComment(DoNotEdit)
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(PackageHttp, "http")
	srcFile.ImportName(PackageOS, "os")
	srcFile.ImportName(PackageSync, "sync")
	srcFile.ImportName(PackageTime, "time")
	srcFile.ImportName(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "jsonrpc")
	if r.HasMetrics() {
		srcFile.ImportName(PackagePrometheus, "prometheus")
	}

	srcFile.Line().Add(r.clientStructFunc(outDir))

	srcFile.Line().Var().Defs(
		Id("clientHostnameOnce").Qual(PackageSync, "Once"),
		Id("cachedClientHostname").String(),
	)
	srcFile.Line().Func().Id("getClientHostname").Params().String().Block(
		Id("clientHostnameOnce").Dot("Do").Call(Func().Params().Block(
			List(Id("cachedClientHostname"), Id("_")).Op("=").Qual(PackageOS, "Hostname").Call(),
		)),
		Return(Id("cachedClientHostname")),
	)
	srcFile.Line().Func().Id("New").Params(Id("endpoint").String(), Id("opts").Op("...").Id("Option")).Params(Id("cli").Op("*").Id("Client")).BlockFunc(
		func(bg *Group) {
			bg.Line()
			bg.Id("name").Op(":=").Id("getClientHostname").Call().Op("+").Lit("_astg_go_").Op("+").Id("VersionASTg")
			bg.If(Id("name").Op("==").Lit("_astg_go_").Op("+").Id("VersionASTg")).Block(
				Id("name").Op("=").Lit("astg_go_").Op("+").Id("VersionASTg"),
			)
			bg.Id("defaultTransport").Op(":=").Op("&").Qual(PackageHttp, "Transport").Values(Dict{
				Id("DisableKeepAlives"):     False(),
				Id("ExpectContinueTimeout"): Qual(PackageTime, "Second").Op("*").Lit(1),
				Id("ForceAttemptHTTP2"):     True(),
				Id("IdleConnTimeout"):       Qual(PackageTime, "Second").Op("*").Lit(60),
				Id("MaxConnsPerHost"):       Lit(1000),
				Id("MaxIdleConns"):          Lit(500),
				Id("MaxIdleConnsPerHost"):   Lit(100),
				Id("ResponseHeaderTimeout"): Qual(PackageTime, "Second").Op("*").Lit(10),
				Id("TLSHandshakeTimeout"):   Qual(PackageTime, "Second").Op("*").Lit(10),
			})
			bg.Id("defaultClient").Op(":=").Op("&").Qual(PackageHttp, "Client").Values(Dict{
				Id("Timeout"):   Qual(PackageTime, "Second").Op("*").Lit(30),
				Id("Transport"): Id("defaultTransport"),
			})
			bg.Id("cli").Op("=").Op("&").Id("Client").Values(DictFunc(func(dict Dict) {
				dict[Id("afterRequest")] = Nil()
				dict[Id("beforeRequest")] = Nil()
				dict[Id("endpoint")] = Id("endpoint")
				dict[Id("errorDecoder")] = Id("defaultErrorDecoder")
				dict[Id("headersFromCtx")] = Index().Any().Values()
				dict[Id("httpClient")] = Id("defaultClient")
				dict[Id("logOnError")] = False()
				dict[Id("logRequests")] = False()
				dict[Id("name")] = Id("name")
			}))
			bg.Id("cli").Dot("applyOpts").Call(Id("opts"))
			if r.HasJsonRPC() {
				bg.Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "ClientHTTP").Call(Id("cli").Dot("httpClient")))
				bg.Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "ClientID").Call(Id("cli").Dot("name")))
				bg.Id("cli").Dot("rpc").Op("=").Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "NewClient").Call(Id("endpoint"), Id("cli").Dot("rpcOpts").Op("..."))
			}

			bg.Return()
		})

	for _, contractName := range r.ContractKeys() {
		contract := r.FindContract(contractName)
		if contract == nil {
			continue
		}
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) || model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
			srcFile.Line().Func().Params(Id("cli").Op("*").Id("Client")).Id(contract.Name).Params().Params(Op("*").Id("Client" + contract.Name)).Block(
				Return(Op("&").Id("Client" + contract.Name).Values(Dict{
					Id("Client"): Id("cli"),
				})),
			)
		}
	}
	if r.HasMetrics() {
		srcFile.Line().Func().Params(Id("cli").Op("*").Id("Client")).Id("GetMetricsRegistry").Params().Params(Id("reg").Op("*").Qual(PackagePrometheus, "Registry")).Block(
			Return(Id("cli").Dot("metricsReg")),
		)
	}
	return srcFile.Save(path.Join(outDir, "client.go"))
}

func (r *ClientRenderer) clientStructFunc(outDir string) Code {

	return Type().Id("Client").StructFunc(func(sg *Group) {
		sg.Id("name").String()
		sg.Id("endpoint").String()
		if r.HasJsonRPC() || r.HasHTTP() {
			sg.Line().Id("httpClient").Op("*").Qual(PackageHttp, "Client")
		}
		if r.HasJsonRPC() {
			sg.Line().Id("rpc").Op("*").Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "ClientRPC")
			sg.Id("rpcOpts").Op("[]").Qual(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "Option")
		}
		sg.Line().Id("errorDecoder").Id("ErrorDecoder")
		if r.HasJsonRPC() || r.HasHTTP() {
			sg.Line().Id("logRequests").Bool()
			sg.Id("logOnError").Bool()
			sg.Id("headersFromCtx").Op("[]").Any()
			sg.Id("beforeRequest").Func().Params(Qual(PackageContext, "Context"), Op("*").Qual(PackageHttp, "Request")).Params(Qual(PackageContext, "Context"))
			sg.Id("afterRequest").Func().Params(Qual(PackageContext, "Context"), Op("*").Qual(PackageHttp, "Response")).Params(Err().Error())
		}
		if r.HasMetrics() {
			sg.Line().Id("metrics").Op("*").Id("Metrics")
			sg.Id("metricsReg").Op("*").Qual(PackagePrometheus, "Registry")
		}
	})
}

func (r *ClientRenderer) FindContract(name string) *model.Contract {
	for _, contract := range r.project.Contracts {
		if contract.Name == name {
			return contract
		}
	}
	return nil
}
