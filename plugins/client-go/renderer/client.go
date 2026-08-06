// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive

	"tgp/internal/model"
)

func (r *ClientRenderer) RenderClient() (err error) {

	outDir := r.outDir

	if err = r.RenderJSONRPCPackage(outDir); err != nil {
		return fmt.Errorf("не удалось сгенерировать пакет jsonrpc: %w", err)
	}

	srcFile := NewSrcFile(filepath.Base(outDir))
	srcFile.ImportName(PackageHTTP, "http")
	srcFile.ImportName(PackageOS, "os")
	srcFile.ImportName(PackageSync, "sync")
	srcFile.ImportName(PackageTime, PackageTime)
	if r.HasJSONRPC() {
		srcFile.ImportName(PackageContext, "context")
		srcFile.ImportName(PackageURL, "url")
		srcFile.ImportName(PackagePath, "path")
	}
	if r.HasHTTP() || r.HasSSE() {
		srcFile.ImportName(PackageContext, "context")
	}
	if r.hasApp2AppAuth() || r.hasTokenPackage() {
		srcFile.ImportName(PackageContext, "context")
		srcFile.ImportName(PackageOAuth2, "oauth2")
		srcFile.ImportName(r.pkgPath(outDir)+"/token", "token")
	}
	if r.HasJSONRPC() {
		srcFile.ImportName(r.pkgPath(outDir)+"/jsonrpc", pkgJSONRPC)
	}
	if r.HasMetrics() {
		srcFile.ImportName(PackagePrometheus, "prometheus")
	}
	if r.HasWS() {
		srcFile.ImportName(packageWebsocket, "websocket")
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
			bg.Id("defaultTransport").Op(":=").Op("&").Qual(PackageHTTP, "Transport").Values(Dict{
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
			bg.Id("defaultClient").Op(":=").Op("&").Qual(PackageHTTP, "Client").Values(Dict{
				Id("Transport"): Id("defaultTransport"),
			})
			if r.HasWS() {
				bg.Id("defaultWebSocketDialer").Op(":=").Op("*").Qual(packageWebsocket, "DefaultDialer")
			}
			bg.Id("cli").Op("=").Op("&").Id("Client").Values(DictFunc(func(dict Dict) {
				dict[Id("afterRequest")] = Nil()
				dict[Id("beforeRequest")] = Nil()
				dict[Id("endpoint")] = Id("endpoint")
				dict[Id("headersFromCtx")] = Index().Any().Values()
				dict[Id("httpClient")] = Id("defaultClient")
				dict[Id("logOnError")] = False()
				dict[Id("logRequests")] = False()
				dict[Id("name")] = Id("name")
				if r.HasWS() {
					dict[Id("webSocketDialer")] = Op("&").Id("defaultWebSocketDialer")
				}
				if r.HasSSE() {
					dict[Id("maxSSEEventSize")] = Lit(1 << 20)
				}
			}))
			if r.HasJSONRPC() {
				bg.Id("cli").Dot("errorDecoder").Op("=").Id("defaultErrorDecoder")
			}
			if r.HasHTTP() {
				bg.Id("cli").Dot("httpErrorDecoder").Op("=").Id("defaultHTTPErrorDecoder")
			}
			bg.Id("cli").Dot("applyOpts").Call(Id("opts"))
			if r.hasTokenPackage() || r.hasApp2AppAuth() {
				bg.Id("cli").Dot("wireResourceAuth").Call()
			}
			if r.HasJSONRPC() {
				bg.Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(r.pkgPath(outDir)+"/jsonrpc", "ClientHTTP").Call(Id("cli").Dot("httpClient")))
				bg.Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(r.pkgPath(outDir)+"/jsonrpc", "ClientID").Call(Id("cli").Dot("name")))
				bg.Id("cli").Dot("rpc").Op("=").Qual(r.pkgPath(outDir)+"/jsonrpc", "NewClient").Call(Id("endpoint"), Id("cli").Dot("rpcOpts").Op("..."))
			}

			bg.Return()
		})

	if r.hasTokenPackage() || r.hasApp2AppAuth() {
		tokenPkg := r.pkgPath(outDir) + "/token"
		srcFile.Line().Func().Params(Id("cli").Op("*").Id("Client")).Id("wireResourceAuth").Params().BlockFunc(func(bg *Group) {
			bg.If(Id("cli").Dot("httpClient").Op("==").Nil()).Block(Return())
			bg.If(Id("cli").Dot("resourceObtain").Op("!=").Nil()).Block(
				Id("cli").Dot("httpClient").Dot("Transport").Op("=").Qual(tokenPkg, "Resource").Call(
					Id("cli").Dot("httpClient").Dot("Transport"),
					Id("cli").Dot("resourceObtain"),
					Id("cli").Dot("tokenRetry").Op("..."),
				),
				Return(),
			)
			if r.hasApp2AppAuth() {
				bg.If(Id("cli").Dot("authSkip")).Block(Return())
				bg.If(Id("cli").Dot("tokenURL").Op("==").Lit("").Op("||").Id("cli").Dot("assertion").Dot("Subject").Op("==").Lit("").Op("||").Id("cli").Dot("assertion").Dot("KeyID").Op("==").Lit("").Op("||").Id("cli").Dot("assertion").Dot("PrivateKey").Op("==").Lit("")).Block(Return())
				bg.Id("tokenURL").Op(":=").Id("cli").Dot("tokenURL")
				bg.Id("assertion").Op(":=").Id("cli").Dot("assertion")
				bg.Id("cli").Dot("httpClient").Dot("Transport").Op("=").Qual(tokenPkg, "Resource").Call(
					Id("cli").Dot("httpClient").Dot("Transport"),
					Func().Params(Id("ctx").Qual(PackageContext, "Context"), Id("meta").Qual(tokenPkg, "Metadata")).Params(Id("src").Qual(PackageOAuth2, "TokenSource"), Err().Error()).BlockFunc(func(fg *Group) {
						fg.Var().Id("primary").Qual(PackageOAuth2, "TokenSource")
						fg.If(List(Id("primary"), Err()).Op("=").Qual(tokenPkg, "JWTBearer").Call(Id("tokenURL"), Id("assertion")).Op(";").Err().Op("!=").Nil()).Block(Return())
						fg.Return(Qual(tokenPkg, "Exchange").Call(Id("tokenURL"), Id("primary"), Id("meta").Dot("Audience")))
					}),
					Id("cli").Dot("tokenRetry").Op("..."),
				)
			}
		})
	}
	if (r.hasTokenPackage() || r.hasApp2AppAuth()) && r.HasWS() {
		tokenPkg := r.pkgPath(outDir) + "/token"
		srcFile.Line().Func().Params(Id("cli").Op("*").Id("Client")).Id("applyDialAuth").
			Params(Id("ctx").Qual(PackageContext, "Context"), Id("dialHeader").Qual(PackageHTTP, "Header")).
			Params(Id("err").Error()).
			Block(
				Line(),
				If(Id("cli").Dot("httpClient").Op("==").Nil()).Block(Return()),
				Return(Qual(tokenPkg, "SetAuthorization").Call(Id("ctx"), Id("cli").Dot("httpClient").Dot("Transport"), Id("dialHeader"), Id("cli").Dot("endpoint"))),
			)
	}
	for _, contractName := range r.ContractKeys() {
		contract := r.FindContract(contractName)
		if contract == nil {
			continue
		}
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJSONRPC) ||
			model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) ||
			model.ContractHasWS(r.project, contract) ||
			model.ContractHasSSE(r.project, contract) {
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
	err = srcFile.Save(path.Join(outDir, "client.go"))
	return
}

func (r *ClientRenderer) clientStructFunc(outDir string) Code {

	return Type().Id("Client").StructFunc(func(sg *Group) {
		sg.Id("name").String()
		sg.Id("endpoint").String()
		if r.HasJSONRPC() || r.HasHTTP() || r.HasSSE() {
			sg.Line().Id("httpClient").Op("*").Qual(PackageHTTP, "Client")
		}
		if r.HasWS() {
			sg.Line().Id("webSocketDialer").Op("*").Qual(packageWebsocket, "Dialer")
		}
		if r.HasSSE() {
			sg.Id("maxSSEEventSize").Int()
		}
		if r.HasJSONRPC() {
			sg.Line().Id("rpc").Op("*").Qual(r.pkgPath(outDir)+"/jsonrpc", "ClientRPC")
			sg.Id("rpcOpts").Op("[]").Qual(r.pkgPath(outDir)+"/jsonrpc", "Option")
			sg.Line().Id("errorDecoder").Id("ErrorDecoder")
		}
		if r.HasHTTP() {
			sg.Line().Id("httpErrorDecoder").Id("HTTPErrorDecoder")
		}
		if r.HasJSONRPC() || r.HasHTTP() || r.HasSSE() {
			sg.Line().Id("logRequests").Bool()
			sg.Id("logOnError").Bool()
			sg.Id("headersFromCtx").Op("[]").Any()
			sg.Id("beforeRequest").Func().Params(Qual(PackageContext, "Context"), Op("*").Qual(PackageHTTP, "Request")).Params(Qual(PackageContext, "Context"))
			sg.Id("afterRequest").Func().Params(Qual(PackageContext, "Context"), Op("*").Qual(PackageHTTP, "Response")).Params(Err().Error())
		}
		if r.hasTokenPackage() {
			tokenPkg := r.pkgPath(outDir) + "/token"
			sg.Line().Id("tokenRetry").Index().Qual(tokenPkg, "RetryOption")
			sg.Id("resourceObtain").Qual(tokenPkg, "ObtainFunc")
		}
		if r.hasApp2AppAuth() {
			tokenPkg := r.pkgPath(outDir) + "/token"
			sg.Line().Id("tokenURL").String()
			sg.Id("assertion").Qual(tokenPkg, "Assertion")
			sg.Id("authSkip").Bool()
		}
		if r.HasMetrics() {
			sg.Line().Id("metrics").Op("*").Id("Metrics")
			sg.Id("metricsReg").Op("*").Qual(PackagePrometheus, "Registry")
			sg.Id("latencyBuckets").Index().Float64()
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
