// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path"
	"path/filepath"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive
)

func (r *ClientRenderer) RenderClientOptions() (err error) {

	outDir := r.outDir
	srcFile := NewSrcFile(filepath.Base(outDir))
	srcFile.ImportName(r.pkgPath(outDir)+"/jsonrpc", pkgJSONRPC)
	if r.HasJSONRPC() || r.HasHTTP() || r.HasSSE() || r.HasWS() {
		srcFile.ImportName(PackageHTTP, "http")
		srcFile.ImportName(PackageTLS, "tls")
		srcFile.ImportName(PackageContext, "context")
		srcFile.ImportName(PackageSlog, "slog")
	}
	if r.HasWS() {
		srcFile.ImportName(packageWebsocket, "websocket")
	}
	tokenPkg := r.pkgPath(outDir) + "/token"
	if r.hasTokenPackage() {
		srcFile.ImportName(PackageOAuth2, "oauth2")
		srcFile.ImportName(tokenPkg, "token")
		srcFile.ImportName(PackageTime, "time")
	}
	if r.hasApp2AppAuth() {
		srcFile.ImportName(PackageStrings, "strings")
	}

	srcFile.Line().Type().Id("Option").Func().Params(Id("cli").Op("*").Id("Client"))

	srcFile.Line().Func().Params(Id("cli").Op("*").Id("Client")).Id("applyOpts").Params(Id("opts").Op("[]").Id("Option")).Block(
		For(List(Id("_"), Id("op")).Op(":=").Range().Id("opts")).Block(
			Id("op").Call(Id("cli")),
		),
	)

	if r.HasJSONRPC() {
		srcFile.Line().Func().Id("DecodeError").Params(Id("decoder").Id("ErrorDecoder")).Params(Id("Option")).Block(
			Return(Func().Params(Id("cli").Op("*").Id("Client"))).Block(
				Id("cli").Dot("errorDecoder").Op("=").Id("decoder"),
			),
		)
	}

	if r.HasHTTP() {
		srcFile.Line().Func().Id("DecodeHTTPError").Params(Id("decoder").Id("HTTPErrorDecoder")).Params(Id("Option")).Block(
			Return(Func().Params(Id("cli").Op("*").Id("Client"))).Block(
				Id("cli").Dot("httpErrorDecoder").Op("=").Id("decoder"),
			),
		)
	}

	srcFile.Line().Func().Id("Name").Params(Id("name").String()).Params(Id("Option")).Block(
		Return(Func().Params(Id("cli").Op("*").Id("Client"))).Block(
			Id("cli").Dot("name").Op("=").Id("name"),
		),
	)

	if r.HasJSONRPC() || r.HasHTTP() || r.HasWS() {
		srcFile.Line().Func().Id("Headers").Params(Id("headers").Op("...").Any()).Params(Id("Option")).BlockFunc(func(bg *Group) {
			bg.Return(Func().Params(Id("cli").Op("*").Id("Client"))).BlockFunc(func(returnBg *Group) {
				returnBg.Id("cli").Dot("headersFromCtx").Op("=").Append(Id("cli").Dot("headersFromCtx"), Id("headers").Op("..."))
				if r.HasJSONRPC() {
					returnBg.Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(r.pkgPath(outDir)+"/jsonrpc", "HeaderFromCtx").Call(Id("headers").Op("...")))
				}
			})
		})
	}

	if r.HasJSONRPC() || r.HasHTTP() || r.HasWS() {
		srcFile.Line().Func().Id("ConfigTLS").Params(Id("tlsConfig").Op("*").Qual(PackageTLS, "Config")).Params(Id("Option")).BlockFunc(func(bg *Group) {
			bg.Return(Func().Params(Id("cli").Op("*").Id("Client"))).BlockFunc(func(returnBg *Group) {
				returnBg.If(Id("cli").Dot("httpClient").Op("!=").Nil()).Block(
					If(List(Id("transport"), Id("ok")).Op(":=").Id("httpTransportOf").Call(Id("cli").Dot("httpClient").Dot("Transport")).Op(";").Id("ok")).Block(
						Id("transport").Dot("TLSClientConfig").Op("=").Id("tlsConfig"),
					),
				)
				if r.HasJSONRPC() {
					returnBg.Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(r.pkgPath(outDir)+"/jsonrpc", "ConfigTLS").Call(Id("tlsConfig")))
				}
				if r.HasWS() {
					returnBg.If(Id("cli").Dot("webSocketDialer").Op("!=").Nil()).Block(
						Id("cli").Dot("webSocketDialer").Dot("TLSClientConfig").Op("=").Id("tlsConfig"),
					)
				}
			})
		})
	}

	if r.HasWS() {
		srcFile.Line().Func().Id("WebSocketDialer").Params(Id("dialer").Op("*").Qual(packageWebsocket, "Dialer")).Params(Id("Option")).Block(
			Return(Func().Params(Id("cli").Op("*").Id("Client")).Block(
				If(Id("dialer").Op("==").Nil()).Block(Return()),
				Id("cloned").Op(":=").Op("*").Id("dialer"),
				Id("cli").Dot("webSocketDialer").Op("=").Op("&").Id("cloned"),
			)),
		)
	}

	if r.HasSSE() {
		srcFile.Line().Func().Id("MaxSSEEventSize").Params(Id("size").Int()).Params(Id("Option")).Block(
			Return(Func().Params(Id("cli").Op("*").Id("Client")).Block(
				If(Id("size").Op(">").Lit(0)).Block(
					Id("cli").Dot("maxSSEEventSize").Op("=").Id("size"),
				),
			)),
		)
	}

	if r.HasJSONRPC() || r.HasHTTP() {
		srcFile.Line().Func().Id("LogRequest").Params().Params(Id("Option")).BlockFunc(func(bg *Group) {
			bg.Return(Func().Params(Id("cli").Op("*").Id("Client"))).BlockFunc(func(returnBg *Group) {
				returnBg.Id("cli").Dot("logRequests").Op("=").True()
				if r.HasJSONRPC() {
					returnBg.Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(r.pkgPath(outDir)+"/jsonrpc", "LogRequest").Call())
				}
			})
		})
	}

	if r.HasJSONRPC() || r.HasHTTP() {
		srcFile.Line().Func().Id("LogOnError").Params().Params(Id("Option")).BlockFunc(func(bg *Group) {
			bg.Return(Func().Params(Id("cli").Op("*").Id("Client"))).BlockFunc(func(returnBg *Group) {
				returnBg.Id("cli").Dot("logOnError").Op("=").True()
				if r.HasJSONRPC() {
					returnBg.Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(r.pkgPath(outDir)+"/jsonrpc", "LogOnError").Call())
				}
			})
		})
	}

	if r.HasJSONRPC() || r.HasHTTP() {
		srcFile.Line().Func().Id("ClientHTTP").Params(Id(pkgClient).Op("*").Qual(PackageHTTP, "Client")).Params(Id("Option")).BlockFunc(func(bg *Group) {
			bg.Return(Func().Params(Id("cli").Op("*").Id("Client"))).BlockFunc(func(returnBg *Group) {
				returnBg.If(Id(pkgClient).Op("!=").Nil()).Block(
					Id("cli").Dot("httpClient").Op("=").Id(pkgClient),
				)
				if r.HasJSONRPC() {
					returnBg.Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(r.pkgPath(outDir)+"/jsonrpc", "ClientHTTP").Call(Id(pkgClient)))
				}
			})
		})
	}

	if r.HasJSONRPC() || r.HasHTTP() {
		srcFile.Line().Func().Id("Transport").Params(Id("transport").Qual(PackageHTTP, "RoundTripper")).Params(Id("Option")).BlockFunc(func(bg *Group) {
			bg.Return(Func().Params(Id("cli").Op("*").Id("Client"))).BlockFunc(func(returnBg *Group) {
				returnBg.If(Id("cli").Dot("httpClient").Op("!=").Nil()).Block(
					Id("cli").Dot("httpClient").Dot("Transport").Op("=").Id("transport"),
				)
				if r.HasJSONRPC() {
					returnBg.Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(r.pkgPath(outDir)+"/jsonrpc", "ClientHTTP").Call(Id("cli").Dot("httpClient")))
				}
			})
		})
	}

	if r.hasTokenPackage() {
		srcFile.Line().Func().Id("TokenSource").Params(Id("src").Qual(PackageOAuth2, "TokenSource")).Params(Id("Option")).BlockFunc(func(bg *Group) {
			bg.Return(Func().Params(Id("cli").Op("*").Id("Client"))).BlockFunc(func(returnBg *Group) {
				if r.hasApp2AppAuth() {
					returnBg.Id("cli").Dot("authSkip").Op("=").True()
				}
				returnBg.If(Id("cli").Dot("httpClient").Op("==").Nil().Op("||").Id("src").Op("==").Nil()).Block(
					Return(),
				)
				returnBg.Id("cli").Dot("httpClient").Dot("Transport").Op("=").Op("&").Qual(PackageOAuth2, "Transport").Values(Dict{
					Id("Base"):   Id("cli").Dot("httpClient").Dot("Transport"),
					Id("Source"): Id("src"),
				})
				if r.HasJSONRPC() {
					returnBg.Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(r.pkgPath(outDir)+"/jsonrpc", "ClientHTTP").Call(Id("cli").Dot("httpClient")))
				}
			})
		})
	}

	if r.hasTokenPackage() || r.hasApp2AppAuth() {
		srcFile.Line().Type().Id("AuthOption").Func().Params(Id("cli").Op("*").Id("Client"))
		srcFile.Line().Func().Id("Auth").Params(Id("opts").Op("...").Id("AuthOption")).Params(Id("Option")).Block(
			Return(Func().Params(Id("cli").Op("*").Id("Client")).Block(
				For(List(Id("_"), Id("opt")).Op(":=").Range().Id("opts")).Block(
					Id("opt").Call(Id("cli")),
				),
			)),
		)
	}

	if r.hasTokenPackage() {
		srcFile.Line().Func().Id("ResourceTokens").
			Params(Id("obtain").Qual(tokenPkg, "ObtainFunc")).
			Params(Id("opt").Id("AuthOption")).
			BlockFunc(func(bg *Group) {
				bg.Return(Func().Params(Id("cli").Op("*").Id("Client")).BlockFunc(func(returnBg *Group) {
					if r.hasApp2AppAuth() {
						returnBg.Id("cli").Dot("authSkip").Op("=").True()
					}
					returnBg.If(Id("obtain").Op("==").Nil()).Block(Return())
					returnBg.Id("cli").Dot("resourceObtain").Op("=").Id("obtain")
				}))
			})
		srcFile.Line().Func().Id("TokenRetry").Params(Id("opts").Op("...").Qual(tokenPkg, "RetryOption")).Params(Id("opt").Id("AuthOption")).Block(
			Return(Func().Params(Id("cli").Op("*").Id("Client")).Block(
				Id("cli").Dot("tokenRetry").Op("=").Append(Id("cli").Dot("tokenRetry"), Id("opts").Op("...")),
			)),
		)
		srcFile.Line().Func().Id("MaxAttempts").Params(Id("n").Int()).Params(Id("opt").Qual(tokenPkg, "RetryOption")).Block(
			Return(Qual(tokenPkg, "MaxAttempts").Call(Id("n"))),
		)
		srcFile.Line().Func().Id("Backoff").Params(Id("base"), Id("cap").Qual(PackageTime, "Duration")).Params(Id("opt").Qual(tokenPkg, "RetryOption")).Block(
			Return(Qual(tokenPkg, "Backoff").Call(Id("base"), Id("cap"))),
		)
		srcFile.Line().Func().Id("Disabled").Params().Params(Id("opt").Qual(tokenPkg, "RetryOption")).Block(
			Return(Qual(tokenPkg, "Disabled").Call()),
		)
		srcFile.Line().Func().Id("TokenFlightTimeout").Params(Id("timeout").Qual(PackageTime, "Duration")).Params(Id("opt").Qual(tokenPkg, "RetryOption")).Block(
			Return(Qual(tokenPkg, "FlightTimeout").Call(Id("timeout"))),
		)
	}

	if r.hasApp2AppAuth() {
		srcFile.Line().Func().Id("TokenURL").Params(Id("url").String()).Params(Id("opt").Id("AuthOption")).Block(
			Return(Func().Params(Id("cli").Op("*").Id("Client")).Block(
				Id("cli").Dot("tokenURL").Op("=").Qual(PackageStrings, "TrimSpace").Call(Id("url")),
			)),
		)
		srcFile.Line().Func().Id("Assertion").Params(Id("subject"), Id("keyID"), Id("privateKey").String()).Params(Id("opt").Id("AuthOption")).Block(
			Return(Func().Params(Id("cli").Op("*").Id("Client")).Block(
				Id("cli").Dot("assertion").Dot("Subject").Op("=").Qual(PackageStrings, "TrimSpace").Call(Id("subject")),
				Id("cli").Dot("assertion").Dot("KeyID").Op("=").Qual(PackageStrings, "TrimSpace").Call(Id("keyID")),
				Id("cli").Dot("assertion").Dot("PrivateKey").Op("=").Id("privateKey"),
			)),
		)
	}

	if r.HasJSONRPC() || r.HasHTTP() {
		srcFile.Line().Func().Id("BeforeRequest").Params(Id("before").Func().Params(Qual(PackageContext, "Context"), Op("*").Qual(PackageHTTP, "Request")).Params(Qual(PackageContext, "Context"))).Params(Id("Option")).BlockFunc(func(bg *Group) {
			bg.Return(Func().Params(Id("cli").Op("*").Id("Client"))).BlockFunc(func(returnBg *Group) {
				returnBg.Id("cli").Dot("beforeRequest").Op("=").Id("before")
				if r.HasJSONRPC() {
					returnBg.Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(r.pkgPath(outDir)+"/jsonrpc", "BeforeRequest").Call(Id("before")))
				}
			})
		})
	}

	if r.HasJSONRPC() || r.HasHTTP() {
		srcFile.Line().Func().Id("AfterRequest").Params(Id("after").Func().Params(Qual(PackageContext, "Context"), Op("*").Qual(PackageHTTP, "Response")).Params(Err().Error())).Params(Id("Option")).BlockFunc(func(bg *Group) {
			bg.Return(Func().Params(Id("cli").Op("*").Id("Client"))).BlockFunc(func(returnBg *Group) {
				returnBg.Id("cli").Dot("afterRequest").Op("=").Id("after")
				if r.HasJSONRPC() {
					returnBg.Id("cli").Dot("rpcOpts").Op("=").Append(Id("cli").Dot("rpcOpts"), Qual(r.pkgPath(outDir)+"/jsonrpc", "AfterRequest").Call(Id("after")))
				}
			})
		})
	}

	if r.HasMetrics() {
		srcFile.Line().Func().Id("LatencyBuckets").
			Params(Id("buckets").Op("...").Float64()).
			Params(Id("opt").Id("Option")).
			Block(
				Return(Func().Params(Id("cli").Op("*").Id("Client")).Block(
					If(Op("!").Id("validBuckets").Call(Id("buckets"))).Block(
						Return(),
					),
					Id("cli").Dot("latencyBuckets").Op("=").Append(Index().Float64().Values(), Id("buckets").Op("...")),
				)),
			)
		srcFile.Line().Func().Id("WithMetrics").Params().Params(Id("opt").Id("Option")).Block(
			Return(Func().Params(Id("cli").Op("*").Id("Client")).Block(
				If(Id("cli").Dot("metrics").Op("==").Nil()).Block(
					List(Id("cli").Dot("metrics"), Id("cli").Dot("metricsReg")).Op("=").Id("cli").Dot("newMetrics").Call(),
				),
			)),
		)
	}
	if r.HasJSONRPC() || r.HasHTTP() {
		srcFile.Line().Func().Id("httpTransportOf").Params(Id("rt").Qual(PackageHTTP, "RoundTripper")).Params(Id("transport").Op("*").Qual(PackageHTTP, "Transport"), Id("ok").Bool()).Block(
			For(Id("rt").Op("!=").Nil()).Block(
				If(List(Id("transport"), Id("ok")).Op("=").Id("rt").Assert(Op("*").Qual(PackageHTTP, "Transport")).Op(";").Id("ok")).Block(
					Return(),
				),
				If(List(Id("unwrapper"), Id("hasUnwrap")).Op(":=").Id("rt").Assert(Interface(Id("Unwrap").Params().Params(Qual(PackageHTTP, "RoundTripper")))).Op(";").Id("hasUnwrap")).Block(
					Id("rt").Op("=").Id("unwrapper").Dot("Unwrap").Call(),
					Continue(),
				),
				Return(Nil(), False()),
			),
			Return(Nil(), False()),
		)
	}
	return srcFile.Save(path.Join(outDir, "options.go"))
}
