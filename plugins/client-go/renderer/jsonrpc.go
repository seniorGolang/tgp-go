// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"path"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck
)

type jsonrpcGenerator struct {
	jsonPkg string // Пакет для JSON операций (например, "encoding/json" или "github.com/goccy/go-json")
}

func (r *ClientRenderer) RenderJsonRPCPackage(outDir string) error {

	jsonPkg := r.getPackageJSON(nil)
	gen := &jsonrpcGenerator{jsonPkg: jsonPkg}

	jsonrpcDir := path.Join(outDir, "jsonrpc")

	if err := os.MkdirAll(jsonrpcDir, 0700); err != nil {
		return err
	}

	files := []struct {
		filename  string
		generator func(string) error
	}{
		{"client.go", func(jsonPkg string) error { return gen.renderClient(jsonrpcDir, jsonPkg) }},
		{"error.go", func(jsonPkg string) error { return gen.renderError(jsonrpcDir, jsonPkg) }},
		{"request.go", func(jsonPkg string) error { return gen.renderRequest(jsonrpcDir) }},
		{"response.go", func(jsonPkg string) error { return gen.renderResponse(jsonrpcDir, jsonPkg) }},
		{"internal.go", func(jsonPkg string) error { return gen.renderInternal(jsonrpcDir, jsonPkg) }},
		{"option.go", func(jsonPkg string) error { return gen.renderOption(jsonrpcDir) }},
		{"public.go", func(jsonPkg string) error { return gen.renderPublic(jsonrpcDir) }},
		{"param.go", func(jsonPkg string) error { return gen.renderParam(jsonrpcDir) }},
		{"string.go", func(jsonPkg string) error { return gen.renderString(jsonrpcDir) }},
		{"http2curl.go", func(jsonPkg string) error { return gen.renderHttp2Curl(jsonrpcDir) }},
	}

	for _, file := range files {
		if err := file.generator(jsonPkg); err != nil {
			return err
		}
	}

	return nil
}

func (gen *jsonrpcGenerator) renderClient(outDir, jsonPkg string) error {

	srcFile := NewSrcFile("jsonrpc")
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageHttp, "http")
	srcFile.ImportName(PackageTime, "time")

	srcFile.Line().Const().Defs(
		Id("Version").Op("=").Lit("2.0"),
	)

	srcFile.Line().Type().Id("ID").Op("=").Uint64()

	srcFile.Line().Var().Defs(
		Id("NilID").Id("ID").Op("=").Lit(0),
		Id("requestID").Uint64(),
	)

	srcFile.Line().Func().Id("NewID").Params().Params(Id("ID")).Block(
		Return(Qual("sync/atomic", "AddUint64").Call(Op("&").Id("requestID"), Lit(1))),
	)

	srcFile.Line().Type().Id("ClientRPC").Struct(
		Id("options").Id("options"),
		Id("endpoint").String(),
		Id("httpClient").Op("*").Qual(PackageHttp, "Client"),
	)

	srcFile.Line().Func().Id("NewClient").Params(Id("endpoint").String(), Id("opts").Op("...").Id("Option")).Params(Id("client").Op("*").Id("ClientRPC")).BlockFunc(
		func(bg *Group) {
			bg.Line()
			bg.Id("optsPrepared").Op(":=").Id("prepareOpts").Call(Id("opts"))
			bg.If(Id("optsPrepared").Dot("clientHTTP").Op("==").Nil()).BlockFunc(
				func(ig *Group) {
					ig.Id("defaultTransport").Op(":=").Op("&").Qual(PackageHttp, "Transport").Values(Dict{
						Id("DisableKeepAlives"):     False(),
						Id("ExpectContinueTimeout"): Lit(1).Op("*").Qual(PackageTime, "Second"),
						Id("ForceAttemptHTTP2"):     True(),
						Id("IdleConnTimeout"):       Lit(60).Op("*").Qual(PackageTime, "Second"),
						Id("MaxConnsPerHost"):       Lit(1000),
						Id("MaxIdleConns"):          Lit(500),
						Id("MaxIdleConnsPerHost"):   Lit(100),
						Id("ResponseHeaderTimeout"): Lit(10).Op("*").Qual(PackageTime, "Second"),
						Id("TLSHandshakeTimeout"):   Lit(10).Op("*").Qual(PackageTime, "Second"),
					})
					ig.Id("defaultClient").Op(":=").Op("&").Qual(PackageHttp, "Client").Values(Dict{
						Id("Timeout"):   Lit(30).Op("*").Qual(PackageTime, "Second"),
						Id("Transport"): Id("defaultTransport"),
					})
					ig.Id("client").Op("=").Op("&").Id("ClientRPC").Values(Dict{
						Id("endpoint"):   Id("endpoint"),
						Id("httpClient"): Id("defaultClient"),
						Id("options"):    Id("optsPrepared"),
					})
				},
			).Else().Block(
				Id("client").Op("=").Op("&").Id("ClientRPC").Values(Dict{
					Id("endpoint"):   Id("endpoint"),
					Id("httpClient"): Id("optsPrepared").Dot("clientHTTP"),
					Id("options"):    Id("optsPrepared"),
				}),
			)
			bg.If(Id("client").Dot("options").Dot("tlsConfig").Op("!=").Nil()).BlockFunc(
				func(ig *Group) {
					ig.If(List(Id("transport"), Id("ok")).Op(":=").Id("client").Dot("httpClient").Dot("Transport").Assert(Op("*").Qual(PackageHttp, "Transport")).Op(";").Id("ok")).Block(
						Id("transport").Dot("TLSClientConfig").Op("=").Id("client").Dot("options").Dot("tlsConfig"),
					).Else().BlockFunc(
						func(eg *Group) {
							eg.Id("existingTransport").Op(":=").Id("client").Dot("httpClient").Dot("Transport")
							eg.Id("newTransport").Op(":=").Op("&").Qual(PackageHttp, "Transport").Values(Dict{
								Id("DisableKeepAlives"):     False(),
								Id("ExpectContinueTimeout"): Lit(1).Op("*").Qual(PackageTime, "Second"),
								Id("ForceAttemptHTTP2"):     True(),
								Id("IdleConnTimeout"):       Lit(60).Op("*").Qual(PackageTime, "Second"),
								Id("MaxConnsPerHost"):       Lit(1000),
								Id("MaxIdleConns"):          Lit(500),
								Id("MaxIdleConnsPerHost"):   Lit(100),
								Id("ResponseHeaderTimeout"): Lit(10).Op("*").Qual(PackageTime, "Second"),
								Id("TLSClientConfig"):       Id("client").Dot("options").Dot("tlsConfig"),
								Id("TLSHandshakeTimeout"):   Lit(10).Op("*").Qual(PackageTime, "Second"),
							})
							eg.If(Id("existingTransport").Op("!=").Nil()).BlockFunc(
								func(tg *Group) {
									tg.If(List(Id("existingHTTPTransport"), Id("ok")).Op(":=").Id("existingTransport").Assert(Op("*").Qual(PackageHttp, "Transport")).Op(";").Id("ok")).BlockFunc(
										func(htg *Group) {
											htg.Id("newTransport").Dot("MaxIdleConns").Op("=").Id("existingHTTPTransport").Dot("MaxIdleConns")
											htg.Id("newTransport").Dot("MaxIdleConnsPerHost").Op("=").Id("existingHTTPTransport").Dot("MaxIdleConnsPerHost")
											htg.Id("newTransport").Dot("MaxConnsPerHost").Op("=").Id("existingHTTPTransport").Dot("MaxConnsPerHost")
											htg.Id("newTransport").Dot("IdleConnTimeout").Op("=").Id("existingHTTPTransport").Dot("IdleConnTimeout")
											htg.Id("newTransport").Dot("ResponseHeaderTimeout").Op("=").Id("existingHTTPTransport").Dot("ResponseHeaderTimeout")
											htg.Id("newTransport").Dot("ExpectContinueTimeout").Op("=").Id("existingHTTPTransport").Dot("ExpectContinueTimeout")
											htg.Id("newTransport").Dot("TLSHandshakeTimeout").Op("=").Id("existingHTTPTransport").Dot("TLSHandshakeTimeout")
											htg.Id("newTransport").Dot("DisableKeepAlives").Op("=").Id("existingHTTPTransport").Dot("DisableKeepAlives")
											htg.Id("newTransport").Dot("ForceAttemptHTTP2").Op("=").Id("existingHTTPTransport").Dot("ForceAttemptHTTP2")
										},
									)
								},
							)
							eg.Id("client").Dot("httpClient").Dot("Transport").Op("=").Id("newTransport")
						},
					)
				},
			)
			bg.Return(Id("client"))
		})

	return srcFile.Save(path.Join(outDir, "client.go"))
}

func (gen *jsonrpcGenerator) renderError(outDir, jsonPkg string) error {

	srcFile := NewSrcFile("jsonrpc")
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(jsonPkg, "json")
	srcFile.ImportName(PackageStrconv, "strconv")

	srcFile.Line().Type().Id("RPCError").Struct(
		Id("Code").Int().Tag(map[string]string{"json": "code"}),
		Id("Message").String().Tag(map[string]string{"json": "message"}),
		Id("Data").Qual(jsonPkg, "RawMessage").Tag(map[string]string{"json": "data,omitempty"}),
		Id("RawBytes").Qual(jsonPkg, "RawMessage").Tag(map[string]string{"json": "-"}),
	)

	srcFile.Line().Func().Params(Id("e").Op("*").Id("RPCError")).Id("Raw").Params().Params(Id("data").Qual(jsonPkg, "RawMessage")).Block(
		Return(Id("e").Dot("RawBytes")),
	)

	srcFile.Line().Func().Params(Id("e").Op("*").Id("RPCError")).Id("Error").Params().Params(String()).Block(
		Return(Qual(PackageStrconv, "Itoa").Call(Id("e").Dot("Code")).Op("+").Lit(": ").Op("+").Id("e").Dot("Message")),
	)

	srcFile.Line().Type().Id("HTTPError").Struct(
		Id("Code").Int(),
		Id("err").Error(),
	)

	srcFile.Line().Func().Params(Id("e").Op("*").Id("HTTPError")).Id("Error").Params().Params(String()).Block(
		Return(Id("e").Dot("err").Dot("Error").Call()),
	)

	return srcFile.Save(path.Join(outDir, "error.go"))
}

func (gen *jsonrpcGenerator) renderRequest(outDir string) error {

	srcFile := NewSrcFile("jsonrpc")
	srcFile.PackageComment(DoNotEdit)

	srcFile.Line().Type().Id("RequestRPC").Struct(
		Id("ID").Id("ID").Tag(map[string]string{"json": "id"}),
		Id("Method").String().Tag(map[string]string{"json": "method"}),
		Id("Params").Any().Tag(map[string]string{"json": "params,omitempty"}),
		Id("JSONRPC").String().Tag(map[string]string{"json": "jsonrpc"}),
	)

	srcFile.Line().Type().Id("RequestsRPC").Index().Op("*").Id("RequestRPC")

	srcFile.Line().Func().Id("NewRequest").Params(Id("method").String(), Id("params").Op("...").Any()).Params(Op("*").Id("RequestRPC")).BlockFunc(
		func(bg *Group) {
			bg.Line()
			bg.Id("request").Op(":=").Op("&").Id("RequestRPC").Values(DictFunc(func(d Dict) {
				d[Id("ID")] = Id("NewID").Call()
				d[Id("Method")] = Id("method")
				d[Id("Params")] = Id("Params").Call(Id("params").Op("..."))
				d[Id("JSONRPC")] = Id("Version")
			}))
			bg.Return(Id("request"))
		})

	srcFile.Line().Func().Id("NewRequestWithID").Params(Id("id").Id("ID"), Id("method").String(), Id("params").Op("...").Any()).Params(Op("*").Id("RequestRPC")).BlockFunc(
		func(bg *Group) {
			bg.Line()
			bg.Id("request").Op(":=").Op("&").Id("RequestRPC").Values(Dict{
				Id("ID"):      Id("id"),
				Id("Method"):  Id("method"),
				Id("Params"):  Id("Params").Call(Id("params").Op("...")),
				Id("JSONRPC"): Id("Version"),
			})
			bg.Return(Id("request"))
		})

	return srcFile.Save(path.Join(outDir, "request.go"))
}

func (gen *jsonrpcGenerator) renderResponse(outDir, jsonPkg string) error {

	srcFile := NewSrcFile("jsonrpc")
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(jsonPkg, "json")

	srcFile.Line().Type().Id("ResponseRPC").Struct(
		Id("ID").Id("ID").Tag(map[string]string{"json": "id"}),
		Id("JSONRPC").String().Tag(map[string]string{"json": "jsonrpc"}),
		Id("Error").Op("*").Id("RPCError").Tag(map[string]string{"json": "error,omitempty"}),
		Id("Result").Qual(jsonPkg, "RawMessage").Tag(map[string]string{"json": "result,omitempty"}),
	)

	srcFile.Line().Func().Params(Id("r").Op("*").Id("ResponseRPC")).Id("UnmarshalJSON").Params(Id("data").Index().Byte()).Params(Err().Error()).BlockFunc(
		func(bg *Group) {
			bg.Type().Id("wire").Struct(
				Id("ID").Id("ID").Tag(map[string]string{"json": "id"}),
				Id("JSONRPC").String().Tag(map[string]string{"json": "jsonrpc"}),
				Id("Error").Qual(jsonPkg, "RawMessage").Tag(map[string]string{"json": "error,omitempty"}),
				Id("Result").Qual(jsonPkg, "RawMessage").Tag(map[string]string{"json": "result,omitempty"}),
			)
			bg.Var().Id("w").Id("wire")
			bg.If(Err().Op(":=").Qual(jsonPkg, "Unmarshal").Call(Id("data"), Op("&").Id("w")).Op(";").Err().Op("!=").Nil()).Block(
				Return(Err()),
			)
			bg.Id("r").Dot("ID").Op("=").Id("w").Dot("ID")
			bg.Id("r").Dot("JSONRPC").Op("=").Id("w").Dot("JSONRPC")
			bg.Id("r").Dot("Result").Op("=").Id("w").Dot("Result")
			bg.If(Len(Id("w").Dot("Error")).Op(">").Lit(0)).BlockFunc(
				func(ig *Group) {
					ig.Id("r").Dot("Error").Op("=").Op("&").Id("RPCError").Values()
					ig.If(Err().Op(":=").Qual(jsonPkg, "Unmarshal").Call(Id("w").Dot("Error"), Id("r").Dot("Error")).Op(";").Err().Op("!=").Nil()).Block(
						Return(Err()),
					)
					ig.Id("r").Dot("Error").Dot("RawBytes").Op("=").Id("w").Dot("Error")
				},
			)
			bg.Return(Nil())
		},
	)

	srcFile.Line().Type().Id("ResponsesRPC").Index().Op("*").Id("ResponseRPC")

	srcFile.Line().Func().Params(Id("res").Id("ResponsesRPC")).Id("AsMap").Params().Params(Map(Id("ID")).Op("*").Id("ResponseRPC")).BlockFunc(
		func(bg *Group) {
			bg.Id("resMap").Op(":=").Make(Map(Id("ID")).Op("*").Id("ResponseRPC"), Len(Id("res")))
			bg.For(List(Id("_"), Id("r")).Op(":=").Range().Id("res")).Block(
				Id("resMap").Index(Id("r").Dot("ID")).Op("=").Id("r"),
			)
			bg.Return(Id("resMap"))
		})

	srcFile.Line().Func().Params(Id("res").Id("ResponsesRPC")).Id("GetByID").Params(Id("id").Id("ID")).Params(Op("*").Id("ResponseRPC")).BlockFunc(
		func(bg *Group) {
			bg.For(List(Id("_"), Id("r")).Op(":=").Range().Id("res")).Block(
				If(Id("r").Dot("ID").Op("==").Id("id")).Block(
					Return(Id("r")),
				),
			)
			bg.Return(Nil())
		})

	srcFile.Line().Func().Params(Id("res").Id("ResponsesRPC")).Id("HasError").Params().Params(Bool()).BlockFunc(
		func(bg *Group) {
			bg.For(List(Id("_"), Id("resp")).Op(":=").Range().Id("res")).Block(
				If(Id("resp").Dot("Error").Op("!=").Nil()).Block(
					Return(True()),
				),
			)
			bg.Return(False())
		})

	srcFile.Line().Func().Params(Id("responseRPC").Op("*").Id("ResponseRPC")).Id("GetObject").Params(Id("object").Any()).Params(Err().Error()).Block(
		Return(Qual(jsonPkg, "Unmarshal").Call(Id("responseRPC").Dot("Result"), Id("object"))),
	)

	return srcFile.Save(path.Join(outDir, "response.go"))
}

func (gen *jsonrpcGenerator) renderInternal(outDir, jsonPkg string) error {

	srcFile := NewSrcFile("jsonrpc")
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageBytes, "bytes")
	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(PackageFmt, "fmt")
	srcFile.ImportName(PackageIO, "io")
	srcFile.ImportName(PackageSlog, "slog")
	srcFile.ImportName(PackageHttp, "http")
	srcFile.ImportName(jsonPkg, "json")

	srcFile.Line().Func().Params(Id("client").Op("*").Id("ClientRPC")).Id("newRequest").Params(Id("ctx").Qual(PackageContext, "Context"), Id("reqBody").Any()).Params(Id("request").Op("*").Qual(PackageHttp, "Request"), Id("err").Error()).BlockFunc(
		func(bg *Group) {
			bg.Var().Id("bodyBytes").Index().Byte()
			bg.If(List(Id("bodyBytes"), Err()).Op("=").Qual(jsonPkg, "Marshal").Call(Id("reqBody")).Op(";").Err().Op("!=").Nil()).Block(
				Return(),
			)
			bg.If(List(Id("request"), Id("err")).Op("=").Qual(PackageHttp, "NewRequestWithContext").Call(Id("ctx"), Qual(PackageHttp, "MethodPost"), Id("client").Dot("endpoint"), Qual(PackageBytes, "NewBuffer").Call(Id("bodyBytes"))).Op(";").Id("err").Op("!=").Nil()).Block(
				Return(),
			)
			bg.Id("request").Dot("Header").Dot("Set").Call(Lit("Accept"), Lit("application/json"))
			bg.Id("request").Dot("Header").Dot("Set").Call(Lit("Content-Type"), Lit("application/json"))
			bg.If(Id("client").Dot("options").Dot("clientID").Op("!=").Lit("")).Block(
				Id("request").Dot("Header").Dot("Set").Call(Lit("X-Client-Id"), Id("client").Dot("options").Dot("clientID")),
			)
			bg.For(List(Id("k"), Id("v")).Op(":=").Range().Id("client").Dot("options").Dot("customHeaders")).BlockFunc(
				func(fg *Group) {
					fg.If(Id("k").Op("==").Lit("Host")).Block(
						Id("request").Dot("Host").Op("=").Id("v"),
					).Else().Block(
						Id("request").Dot("Header").Dot("Set").Call(Id("k"), Id("v")),
					)
				},
			)
			bg.For(List(Id("_"), Id("header")).Op(":=").Range().Id("client").Dot("options").Dot("headersFromCtx")).BlockFunc(
				func(fg *Group) {
					fg.If(Id("value").Op(":=").Id("ctx").Dot("Value").Call(Id("header")).Op(";").Id("value").Op("!=").Nil()).BlockFunc(
						func(ig *Group) {
							ig.If(Id("k").Op(":=").Id("toString").Call(Id("header")).Op(";").Id("k").Op("!=").Lit("")).BlockFunc(
								func(jg *Group) {
									jg.If(Id("v").Op(":=").Id("toString").Call(Id("value")).Op(";").Id("v").Op("!=").Lit("")).Block(
										Id("request").Dot("Header").Dot("Set").Call(Id("k"), Id("v")),
									)
								},
							)
						},
					)
				},
			)
			bg.Return()
		})

	srcFile.Line().Func().Params(Id("client").Op("*").Id("ClientRPC")).Id("doCall").Params(Id("ctx").Qual(PackageContext, "Context"), Id("request").Op("*").Id("RequestRPC")).Params(Id("rpcResponse").Op("*").Id("ResponseRPC"), Id("err").Error()).BlockFunc(
		func(bg *Group) {
			bg.Var().Id("httpRequest").Op("*").Qual(PackageHttp, "Request")
			bg.If(List(Id("httpRequest"), Id("err")).Op("=").Id("client").Dot("newRequest").Call(Id("ctx"), Id("request")).Op(";").Id("err").Op("!=").Nil()).Block(
				Id("err").Op("=").Qual(PackageFmt, "Errorf").Call(Lit("rpc call %v() on %v: %v"), Id("request").Dot("Method"), Id("client").Dot("endpoint"), Id("err").Dot("Error").Call()),
				Return(),
			)
			bg.If(Id("client").Dot("options").Dot("before").Op("!=").Nil()).Block(
				Id("ctx").Op("=").Id("client").Dot("options").Dot("before").Call(Id("ctx"), Id("httpRequest")),
			)
			bg.If(Id("client").Dot("options").Dot("logRequests")).BlockFunc(
				func(ig *Group) {
					ig.If(List(Id("cmd"), Id("cmdErr")).Op(":=").Id("ToCurl").Call(Id("httpRequest")).Op(";").Id("cmdErr").Op("==").Nil()).Block(
						Qual(PackageSlog, "DebugContext").Call(Id("ctx"), Lit("call"), Qual(PackageSlog, "String").Call(Lit("method"), Id("request").Dot("Method")), Qual(PackageSlog, "String").Call(Lit("curl"), Id("cmd").Dot("String").Call())),
					)
				},
			)
			bg.Defer().Func().Params().BlockFunc(
				func(dg *Group) {
					dg.If(Id("err").Op("!=").Nil().Op("&&").Id("client").Dot("options").Dot("logOnError")).BlockFunc(
						func(eg *Group) {
							eg.If(List(Id("cmd"), Id("cmdErr")).Op(":=").Id("ToCurl").Call(Id("httpRequest")).Op(";").Id("cmdErr").Op("==").Nil()).Block(
								Qual(PackageSlog, "ErrorContext").Call(Id("ctx"), Lit("call"), Qual(PackageSlog, "String").Call(Lit("method"), Id("request").Dot("Method")), Qual(PackageSlog, "String").Call(Lit("curl"), Id("cmd").Dot("String").Call()), Qual(PackageSlog, "Any").Call(Lit("error"), Id("err"))),
							)
						},
					)
				},
			).Call()
			bg.Var().Id("httpResponse").Op("*").Qual(PackageHttp, "Response")
			bg.If(List(Id("httpResponse"), Id("err")).Op("=").Id("client").Dot("httpClient").Dot("Do").Call(Id("httpRequest")).Op(";").Id("err").Op("!=").Nil()).Block(
				Id("err").Op("=").Qual(PackageFmt, "Errorf").Call(Lit("rpc call %v() on %v: %v"), Id("request").Dot("Method"), Id("httpRequest").Dot("URL").Dot("String").Call(), Id("err").Dot("Error").Call()),
				Return(),
			)
			bg.Defer().Id("httpResponse").Dot("Body").Dot("Close").Call()
			bg.If(Id("client").Dot("options").Dot("after").Op("!=").Nil()).BlockFunc(
				func(ag *Group) {
					ag.If(Err().Op("=").Id("client").Dot("options").Dot("after").Call(Id("ctx"), Id("httpResponse")).Op(";").Err().Op("!=").Nil()).Block(
						Return(),
					)
				},
			)
			bg.If(Id("httpResponse").Dot("StatusCode").Op("!=").Qual(PackageHttp, "StatusOK")).BlockFunc(
				func(sg *Group) {
					sg.List(Id("bodyBytes"), Id("readErr")).Op(":=").Qual(PackageIO, "ReadAll").Call(Qual(PackageIO, "LimitReader").Call(Id("httpResponse").Dot("Body"), Lit(1024)))
					sg.Id("errorMsg").Op(":=").String().Call(Id("bodyBytes"))
					sg.If(Id("readErr").Op("!=").Nil().Op("||").Id("errorMsg").Op("==").Lit("")).Block(
						Id("errorMsg").Op("=").Id("httpResponse").Dot("Status"),
					)
					sg.Return(Nil(), Op("&").Id("HTTPError").Values(Dict{
						Id("Code"): Id("httpResponse").Dot("StatusCode"),
						Id("err"):  Qual(PackageFmt, "Errorf").Call(Lit("rpc call %v() on %v status code: %v. %v"), Id("request").Dot("Method"), Id("httpRequest").Dot("URL").Dot("String").Call(), Id("httpResponse").Dot("StatusCode"), Id("errorMsg")),
					}))
				},
			)
			bg.Id("decoder").Op(":=").Qual(jsonPkg, "NewDecoder").Call(Id("httpResponse").Dot("Body"))
			bg.Id("decoder").Dot("UseNumber").Call()
			bg.If(Err().Op("=").Id("decoder").Dot("Decode").Call(Op("&").Id("rpcResponse")).Op(";").Err().Op("!=").Nil()).Block(
				Return(Nil(), Qual(PackageFmt, "Errorf").Call(Lit("rpc call %v() on %v status code: %v. could not decode body to rpc response: %v"), Id("request").Dot("Method"), Id("httpRequest").Dot("URL").Dot("String").Call(), Id("httpResponse").Dot("StatusCode"), Id("err").Dot("Error").Call())),
			)
			bg.If(Id("rpcResponse").Op("==").Nil()).Block(
				Id("err").Op("=").Qual(PackageFmt, "Errorf").Call(Lit("rpc call %v() on %v status code: %v. rpc response missing"), Id("request").Dot("Method"), Id("httpRequest").Dot("URL").Dot("String").Call(), Id("httpResponse").Dot("StatusCode")),
				Return(),
			)
			bg.If(Id("rpcResponse").Dot("ID").Op("!=").Id("request").Dot("ID")).Block(
				Return(Nil(), Qual(PackageFmt, "Errorf").Call(Lit("rpc call %v() on %v: response ID mismatch. expected: %v, got: %v"), Id("request").Dot("Method"), Id("httpRequest").Dot("URL").Dot("String").Call(), Id("request").Dot("ID"), Id("rpcResponse").Dot("ID"))),
			)
			bg.Return()
		})

	srcFile.Line().Func().Params(Id("client").Op("*").Id("ClientRPC")).Id("doBatchCall").Params(Id("ctx").Qual(PackageContext, "Context"), Id("rpcRequests").Index().Op("*").Id("RequestRPC")).Params(Id("rpcResponses").Id("ResponsesRPC"), Id("err").Error()).BlockFunc(
		func(bg *Group) {
			bg.Defer().Func().Params().BlockFunc(
				func(dg *Group) {
					dg.If(Id("err").Op("!=").Nil()).BlockFunc(
						func(eg *Group) {
							eg.For(List(Id("_"), Id("request")).Op(":=").Range().Id("rpcRequests")).Block(
								If(Id("request").Dot("ID").Op("==").Id("NilID")).Block(
									Continue(),
								),
								Id("rpcResponses").Op("=").Append(Id("rpcResponses"), Op("&").Id("ResponseRPC").Values(Dict{
									Id("ID"):      Id("request").Dot("ID"),
									Id("JSONRPC"): Id("request").Dot("JSONRPC"),
									Id("Error"): Op("&").Id("RPCError").Values(Dict{
										Id("Message"): Id("err").Dot("Error").Call(),
									}),
								})),
							)
						},
					)
				},
			).Call()
			bg.Var().Id("httpRequest").Op("*").Qual(PackageHttp, "Request")
			bg.If(List(Id("httpRequest"), Id("err")).Op("=").Id("client").Dot("newRequest").Call(Id("ctx"), Id("rpcRequests")).Op(";").Id("err").Op("!=").Nil()).Block(
				Id("err").Op("=").Qual(PackageFmt, "Errorf").Call(Lit("rpc batch call on %v: %v"), Id("client").Dot("endpoint"), Id("err").Dot("Error").Call()),
				Return(),
			)
			bg.If(Id("client").Dot("options").Dot("before").Op("!=").Nil()).Block(
				Id("ctx").Op("=").Id("client").Dot("options").Dot("before").Call(Id("ctx"), Id("httpRequest")),
			)
			bg.If(Id("client").Dot("options").Dot("logRequests")).BlockFunc(
				func(ig *Group) {
					ig.If(List(Id("cmd"), Id("cmdErr")).Op(":=").Id("ToCurl").Call(Id("httpRequest")).Op(";").Id("cmdErr").Op("==").Nil()).Block(
						Qual(PackageSlog, "DebugContext").Call(Id("ctx"), Lit("call"), Qual(PackageSlog, "String").Call(Lit("method"), Lit("batch")), Qual(PackageSlog, "Int").Call(Lit("count"), Len(Id("rpcRequests"))), Qual(PackageSlog, "String").Call(Lit("curl"), Id("cmd").Dot("String").Call())),
					)
				},
			)
			bg.Defer().Func().Params().BlockFunc(
				func(dg *Group) {
					dg.If(Id("err").Op("!=").Nil().Op("&&").Id("client").Dot("options").Dot("logOnError")).BlockFunc(
						func(eg *Group) {
							eg.If(List(Id("cmd"), Id("cmdErr")).Op(":=").Id("ToCurl").Call(Id("httpRequest")).Op(";").Id("cmdErr").Op("==").Nil()).Block(
								Qual(PackageSlog, "ErrorContext").Call(Id("ctx"), Lit("call"), Qual(PackageSlog, "String").Call(Lit("method"), Lit("batch")), Qual(PackageSlog, "Int").Call(Lit("count"), Len(Id("rpcRequests"))), Qual(PackageSlog, "String").Call(Lit("curl"), Id("cmd").Dot("String").Call()), Qual(PackageSlog, "Any").Call(Lit("error"), Id("err"))),
							)
						},
					)
				},
			).Call()
			bg.Var().Id("httpResponse").Op("*").Qual(PackageHttp, "Response")
			bg.If(List(Id("httpResponse"), Id("err")).Op("=").Id("client").Dot("httpClient").Dot("Do").Call(Id("httpRequest")).Op(";").Id("err").Op("!=").Nil()).Block(
				Id("err").Op("=").Qual(PackageFmt, "Errorf").Call(Lit("rpc batch call on %v: %v"), Id("httpRequest").Dot("URL").Dot("String").Call(), Id("err").Dot("Error").Call()),
				Return(),
			)
			bg.Defer().Id("httpResponse").Dot("Body").Dot("Close").Call()
			bg.If(Id("client").Dot("options").Dot("after").Op("!=").Nil()).BlockFunc(
				func(ag *Group) {
					ag.If(Err().Op("=").Id("client").Dot("options").Dot("after").Call(Id("ctx"), Id("httpResponse")).Op(";").Err().Op("!=").Nil()).Block(
						Return(),
					)
				},
			)
			bg.If(Id("httpResponse").Dot("StatusCode").Op("!=").Qual(PackageHttp, "StatusOK")).BlockFunc(
				func(sg *Group) {
					sg.List(Id("bodyBytes"), Id("readErr")).Op(":=").Qual(PackageIO, "ReadAll").Call(Id("httpResponse").Dot("Body"))
					sg.Id("errorMsg").Op(":=").String().Call(Id("bodyBytes"))
					sg.If(Id("readErr").Op("!=").Nil().Op("||").Id("errorMsg").Op("==").Lit("")).Block(
						Id("errorMsg").Op("=").Id("httpResponse").Dot("Status"),
					)
					sg.Return(Nil(), Op("&").Id("HTTPError").Values(Dict{
						Id("Code"): Id("httpResponse").Dot("StatusCode"),
						Id("err"):  Qual(PackageFmt, "Errorf").Call(Lit("rpc batch call on %v status code: %v. %v"), Id("httpRequest").Dot("URL").Dot("String").Call(), Id("httpResponse").Dot("StatusCode"), Id("errorMsg")),
					}))
				},
			)
			bg.Id("decoder").Op(":=").Qual(jsonPkg, "NewDecoder").Call(Id("httpResponse").Dot("Body"))
			bg.Id("decoder").Dot("UseNumber").Call()
			bg.If(Err().Op("=").Id("decoder").Dot("Decode").Call(Op("&").Id("rpcResponses")).Op(";").Err().Op("!=").Nil()).Block(
				Id("err").Op("=").Qual(PackageFmt, "Errorf").Call(Lit("rpc batch call on %v status code: %v. could not decode body to rpc response: %v"), Id("httpRequest").Dot("URL").Dot("String").Call(), Id("httpResponse").Dot("StatusCode"), Id("err").Dot("Error").Call()),
				Return(),
			)
			bg.If(Len(Id("rpcResponses")).Op("==").Lit(0)).Block(
				Id("err").Op("=").Qual(PackageFmt, "Errorf").Call(Lit("rpc batch call on %v status code: %v. rpc response missing"), Id("httpRequest").Dot("URL").Dot("String").Call(), Id("httpResponse").Dot("StatusCode")),
				Return(),
			)
			bg.Id("requestIDMap").Op(":=").Make(Map(Id("ID")).Bool(), Len(Id("rpcRequests")))
			bg.For(List(Id("_"), Id("req")).Op(":=").Range().Id("rpcRequests")).Block(
				If(Id("req").Dot("ID").Op("!=").Id("NilID")).Block(
					Id("requestIDMap").Index(Id("req").Dot("ID")).Op("=").True(),
				),
			)
			bg.For(List(Id("_"), Id("resp")).Op(":=").Range().Id("rpcResponses")).Block(
				If(Id("resp").Dot("ID").Op("!=").Id("NilID").Op("&&").Op("!").Id("requestIDMap").Index(Id("resp").Dot("ID"))).Block(
					Id("err").Op("=").Qual(PackageFmt, "Errorf").Call(Lit("rpc batch call on %v: response ID %v does not match any request ID"), Id("httpRequest").Dot("URL").Dot("String").Call(), Id("resp").Dot("ID")),
					Return(),
				),
			)
			bg.Return()
		})

	return srcFile.Save(path.Join(outDir, "internal.go"))
}

func (gen *jsonrpcGenerator) renderOption(outDir string) error {

	srcFile := NewSrcFile("jsonrpc")
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(PackageTLS, "tls")
	srcFile.ImportName(PackageHttp, "http")

	srcFile.Line().Type().Id("options").Struct(
		Id("logOnError").Bool(),
		Id("logRequests").Bool(),
		Id("tlsConfig").Op("*").Qual(PackageTLS, "Config"),
		Id("clientHTTP").Op("*").Qual(PackageHttp, "Client"),
		Id("clientID").String(),
		Id("headersFromCtx").Index().Any(),
		Id("customHeaders").Map(String()).String(),
		Id("before").Func().Params(Qual(PackageContext, "Context"), Op("*").Qual(PackageHttp, "Request")).Params(Qual(PackageContext, "Context")),
		Id("after").Func().Params(Qual(PackageContext, "Context"), Op("*").Qual(PackageHttp, "Response")).Params(Err().Error()),
	)

	srcFile.Line().Type().Id("Option").Op("=").Func().Params(Id("ops").Op("*").Id("options"))

	srcFile.Line().Func().Id("prepareOpts").Params(Id("opts").Index().Id("Option")).Params(Id("options").Id("options")).BlockFunc(
		func(bg *Group) {
			bg.Id("options").Dot("customHeaders").Op("=").Make(Map(String()).String())
			bg.For(List(Id("_"), Id("op")).Op(":=").Range().Id("opts")).Block(
				Id("op").Call(Op("&").Id("options")),
			)
			bg.Return()
		})

	srcFile.Line().Func().Id("BeforeRequest").Params(Id("before").Func().Params(Qual(PackageContext, "Context"), Op("*").Qual(PackageHttp, "Request")).Params(Qual(PackageContext, "Context"))).Params(Id("Option")).Block(
		Return(Func().Params(Id("ops").Op("*").Id("options")).Block(
			Id("ops").Dot("before").Op("=").Id("before"),
		)),
	)

	srcFile.Line().Func().Id("AfterRequest").Params(Id("after").Func().Params(Qual(PackageContext, "Context"), Op("*").Qual(PackageHttp, "Response")).Params(Err().Error())).Params(Id("Option")).Block(
		Return(Func().Params(Id("ops").Op("*").Id("options")).Block(
			Id("ops").Dot("after").Op("=").Id("after"),
		)),
	)

	srcFile.Line().Func().Id("HeaderFromCtx").Params(Id("headers").Op("...").Any()).Params(Id("Option")).Block(
		Return(Func().Params(Id("ops").Op("*").Id("options")).Block(
			Id("ops").Dot("headersFromCtx").Op("=").Append(Id("ops").Dot("headersFromCtx"), Id("headers").Op("...")),
		)),
	)

	srcFile.Line().Func().Id("ClientHTTP").Params(Id("client").Op("*").Qual(PackageHttp, "Client")).Params(Id("Option")).Block(
		Return(Func().Params(Id("ops").Op("*").Id("options")).Block(
			Id("ops").Dot("clientHTTP").Op("=").Id("client"),
		)),
	)

	srcFile.Line().Func().Id("ClientID").Params(Id("id").String()).Params(Id("Option")).Block(
		Return(Func().Params(Id("ops").Op("*").Id("options")).Block(
			Id("ops").Dot("clientID").Op("=").Id("id"),
		)),
	)

	srcFile.Line().Func().Id("ConfigTLS").Params(Id("tlsConfig").Op("*").Qual(PackageTLS, "Config")).Params(Id("Option")).Block(
		Return(Func().Params(Id("ops").Op("*").Id("options")).Block(
			Id("ops").Dot("tlsConfig").Op("=").Id("tlsConfig"),
		)),
	)

	srcFile.Line().Func().Id("LogRequest").Params().Params(Id("Option")).Block(
		Return(Func().Params(Id("ops").Op("*").Id("options")).Block(
			Id("ops").Dot("logRequests").Op("=").True(),
		)),
	)

	srcFile.Line().Func().Id("LogOnError").Params().Params(Id("Option")).Block(
		Return(Func().Params(Id("ops").Op("*").Id("options")).Block(
			Id("ops").Dot("logOnError").Op("=").True(),
		)),
	)

	return srcFile.Save(path.Join(outDir, "option.go"))
}

func (gen *jsonrpcGenerator) renderPublic(outDir string) error {

	srcFile := NewSrcFile("jsonrpc")
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageContext, "context")
	srcFile.ImportName(PackageErrors, "errors")

	srcFile.Line().Func().Params(Id("client").Op("*").Id("ClientRPC")).Id("Call").Params(Id("ctx").Qual(PackageContext, "Context"), Id("method").String(), Id("params").Op("...").Any()).Params(Id("response").Op("*").Id("ResponseRPC"), Id("err").Error()).BlockFunc(
		func(bg *Group) {
			bg.Var().Id("paramsVal").Any()
			bg.If(Len(Id("params")).Op("==").Lit(1)).Block(
				Id("paramsVal").Op("=").Id("ParamsOne").Call(Id("params").Index(Lit(0))),
			).Else().Block(
				Id("paramsVal").Op("=").Id("Params").Call(Id("params").Op("...")),
			)
			bg.Id("request").Op(":=").Op("&").Id("RequestRPC").Values(DictFunc(func(d Dict) {
				d[Id("ID")] = Id("NewID").Call()
				d[Id("Method")] = Id("method")
				d[Id("Params")] = Id("paramsVal")
				d[Id("JSONRPC")] = Id("Version")
			}))
			bg.Return(Id("client").Dot("doCall").Call(Id("ctx"), Id("request")))
		})

	srcFile.Line().Func().Params(Id("client").Op("*").Id("ClientRPC")).Id("CallRaw").Params(Id("ctx").Qual(PackageContext, "Context"), Id("request").Op("*").Id("RequestRPC")).Params(Id("response").Op("*").Id("ResponseRPC"), Id("err").Error()).Block(
		Return(Id("client").Dot("doCall").Call(Id("ctx"), Id("request"))),
	)

	srcFile.Line().Func().Params(Id("client").Op("*").Id("ClientRPC")).Id("CallFor").Params(Id("ctx").Qual(PackageContext, "Context"), Id("out").Any(), Id("method").String(), Id("params").Op("...").Any()).Params(Id("err").Error()).BlockFunc(
		func(bg *Group) {
			bg.Var().Id("rpcResponse").Op("*").Id("ResponseRPC")
			bg.List(Id("rpcResponse"), Id("err")).Op("=").Id("client").Dot("Call").Call(Id("ctx"), Id("method"), Id("params").Op("..."))
			bg.If(Id("err").Op("!=").Nil()).Block(
				Return(Id("err")),
			)
			bg.If(Id("rpcResponse").Dot("Error").Op("!=").Nil()).Block(
				Return(Id("rpcResponse").Dot("Error")),
			)
			bg.Return(Id("rpcResponse").Dot("GetObject").Call(Id("out")))
		})

	srcFile.Line().Func().Params(Id("client").Op("*").Id("ClientRPC")).Id("CallBatch").Params(Id("ctx").Qual(PackageContext, "Context"), Id("requests").Id("RequestsRPC")).Params(Id("responses").Id("ResponsesRPC"), Id("err").Error()).BlockFunc(
		func(bg *Group) {
			bg.If(Len(Id("requests")).Op("==").Lit(0)).Block(
				Id("err").Op("=").Qual(PackageErrors, "New").Call(Lit("empty request list")),
				Return(),
			)
			bg.Return(Id("client").Dot("doBatchCall").Call(Id("ctx"), Id("requests")))
		})

	srcFile.Line().Func().Params(Id("client").Op("*").Id("ClientRPC")).Id("CallBatchRaw").Params(Id("ctx").Qual(PackageContext, "Context"), Id("requests").Id("RequestsRPC")).Params(Id("responses").Id("ResponsesRPC"), Id("err").Error()).BlockFunc(
		func(bg *Group) {
			bg.If(Len(Id("requests")).Op("==").Lit(0)).Block(
				Id("err").Op("=").Qual(PackageErrors, "New").Call(Lit("empty request list")),
				Return(),
			)
			bg.Return(Id("client").Dot("doBatchCall").Call(Id("ctx"), Id("requests")))
		})

	return srcFile.Save(path.Join(outDir, "public.go"))
}

func (gen *jsonrpcGenerator) renderParam(outDir string) error {

	srcFile := NewSrcFile("jsonrpc")
	srcFile.PackageComment(DoNotEdit)

	srcFile.Line().Func().Id("ParamsOne").Params(Id("v").Any()).Params(Any()).Block(
		Return(Id("v")),
	)

	srcFile.Line().Func().Id("Params").Params(Id("params").Op("...").Any()).Params(Any()).BlockFunc(
		func(bg *Group) {
			bg.Var().Id("finalParams").Any()
			bg.If(Id("params").Op("!=").Nil()).BlockFunc(
				func(ig *Group) {
					ig.Switch(Len(Id("params"))).BlockFunc(
						func(sg *Group) {
							sg.Case(Lit(0))
							sg.Case(Lit(1)).Block(
								Id("finalParams").Op("=").Id("ParamsOne").Call(Id("params").Index(Lit(0))),
							)
							sg.Default().Block(
								Id("finalParams").Op("=").Id("params"),
							)
						},
					)
				},
			)
			bg.Return(Id("finalParams"))
		})

	return srcFile.Save(path.Join(outDir, "param.go"))
}

func (gen *jsonrpcGenerator) renderString(outDir string) error {

	srcFile := NewSrcFile("jsonrpc")
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageFmt, "fmt")

	srcFile.Line().Func().Id("toString").Params(Id("v").Any()).Params(String()).BlockFunc(
		func(bg *Group) {
			bg.If(Id("v").Op("==").Nil()).Block(
				Return(Lit("")),
			)
			bg.If(List(Id("s"), Id("ok")).Op(":=").Id("v").Assert(String()).Op(";").Id("ok")).Block(
				Return(Id("s")),
			)
			bg.If(List(Id("s"), Id("ok")).Op(":=").Id("v").Assert(Qual(PackageFmt, "Stringer")).Op(";").Id("ok")).Block(
				Return(Id("s").Dot("String").Call()),
			)
			bg.Return(Qual(PackageFmt, "Sprintf").Call(Lit("%v"), Id("v")))
		})

	return srcFile.Save(path.Join(outDir, "string.go"))
}

func (gen *jsonrpcGenerator) renderHttp2Curl(outDir string) error {

	srcFile := NewSrcFile("jsonrpc")
	srcFile.PackageComment(DoNotEdit)

	srcFile.ImportName(PackageBytes, "bytes")
	srcFile.ImportName(PackageFmt, "fmt")
	srcFile.ImportName(PackageIO, "io")
	srcFile.ImportName(PackageHttp, "http")
	srcFile.ImportName(PackageSort, "sort")
	srcFile.ImportName(PackageStrings, "strings")

	srcFile.Line().Type().Id("CurlCommand").Struct(
		Id("slice").Index().String(),
	)

	srcFile.Line().Type().Id("nopCloser").Struct(
		Qual(PackageIO, "Reader"), // встроенное поле
	)

	srcFile.Line().Func().Id("ToCurl").Params(Id("req").Op("*").Qual(PackageHttp, "Request")).Params(Id("command").Op("*").Id("CurlCommand"), Id("err").Error()).BlockFunc(
		func(bg *Group) {
			bg.Id("command").Op("=").Op("&").Id("CurlCommand").Values()
			bg.Id("command").Dot("append").Call(Lit("curl"))
			bg.Id("command").Dot("append").Call(Lit("-X"), Id("bashEscape").Call(Id("req").Dot("Method")))
			bg.If(Id("req").Dot("Body").Op("!=").Nil()).BlockFunc(
				func(ig *Group) {
					ig.Var().Id("body").Index().Byte()
					ig.If(List(Id("body"), Id("err")).Op("=").Qual(PackageIO, "ReadAll").Call(Id("req").Dot("Body")).Op(";").Id("err").Op("!=").Nil()).Block(
						Return(),
					)
					ig.Id("req").Dot("Body").Op("=").Id("nopCloser").Values(
						Qual(PackageBytes, "NewBuffer").Call(Id("body")),
					)
					ig.Id("bodyEscaped").Op(":=").Id("bashEscape").Call(String().Call(Id("body")))
					ig.Id("command").Dot("append").Call(Lit("-d"), Id("bodyEscaped"))
				},
			)
			bg.Id("keys").Op(":=").Make(Index().String(), Lit(0), Len(Id("req").Dot("Header")))
			bg.For(List(Id("k")).Op(":=").Range().Id("req").Dot("Header")).Block(
				Id("keys").Op("=").Append(Id("keys"), Id("k")),
			)
			bg.Qual(PackageSort, "Strings").Call(Id("keys"))
			bg.For(List(Id("_"), Id("k")).Op(":=").Range().Id("keys")).Block(
				Id("command").Dot("append").Call(Lit("-H"), Id("bashEscape").Call(Qual(PackageFmt, "Sprintf").Call(Lit("%s: %s"), Id("k"), Qual(PackageStrings, "Join").Call(Id("req").Dot("Header").Index(Id("k")), Lit(" "))))),
			)
			bg.Id("command").Dot("append").Call(Id("bashEscape").Call(Id("req").Dot("URL").Dot("String").Call()))
			bg.Return()
		})

	srcFile.Line().Func().Params(Id("c").Op("*").Id("CurlCommand")).Id("append").Params(Id("newSlice").Op("...").String()).Block(
		Id("c").Dot("slice").Op("=").Append(Id("c").Dot("slice"), Id("newSlice").Op("...")),
	)

	srcFile.Line().Func().Params(Id("c").Op("*").Id("CurlCommand")).Id("String").Params().Params(String()).Block(
		Return(Qual(PackageStrings, "Join").Call(Id("c").Dot("slice"), Lit(" "))),
	)

	srcFile.Line().Func().Id("bashEscape").Params(Id("str").String()).Params(String()).Block(
		Return(Lit("'").Op("+").Qual(PackageStrings, "ReplaceAll").Call(Id("str"), Lit("'"), Lit("'\\''")).Op("+").Lit("'")),
	)

	srcFile.Line().Func().Params(Id("nopCloser")).Id("Close").Params().Params(Err().Error()).Block(
		Return(Nil()),
	)

	return srcFile.Save(path.Join(outDir, "http2curl.go"))
}
