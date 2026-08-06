// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"context"
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" //nolint:staticcheck,revive

	"tgp/internal/content"
	"tgp/internal/model"
)

func (r *ClientRenderer) RenderServiceClient(contract *model.Contract) (err error) {

	outDir := r.outDir
	pkgName := filepath.Base(outDir)
	srcFile := NewSrcFile(pkgName)
	ctx := context.WithValue(context.Background(), keyCode, srcFile)
	ctx = context.WithValue(ctx, keyPackage, pkgName)

	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJSONRPC) {
		srcFile.ImportName(PackageContext, "context")
		srcFile.ImportName(PackageUUID, "uuid")
		srcFile.ImportName(PackageFiber, "fiber")
		srcFile.ImportName(r.pkgPath(outDir)+"/jsonrpc", pkgJSONRPC)
		jsonPkg := r.getPackageJSON(contract)
		srcFile.ImportName(jsonPkg, keyJSON)
	}

	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
		srcFile.ImportName(PackageContext, "context")
		srcFile.ImportName(PackageFmt, "fmt")
		srcFile.ImportName(PackageTime, PackageTime)
		srcFile.ImportName(PackageHTTP, "http")
		srcFile.ImportName(PackageURL, "url")
		srcFile.ImportName(PackagePath, "path")
		srcFile.ImportName(PackageBytes, "bytes")
		srcFile.ImportName(PackageIO, "io")
		srcFile.ImportName(PackageStrings, "strings")
		jsonPkg := r.getPackageJSON(contract)
		srcFile.ImportName(jsonPkg, keyJSON)
		for _, method := range contract.Methods {
			if r.methodIsHTTP(contract, method) && (r.methodRequestMultipart(contract, method) || r.methodResponseMultipart(contract, method)) {
				srcFile.ImportName(PackageMime, "mime")
				srcFile.ImportName(PackageMimeMultipart, "multipart")
				srcFile.ImportName(PackageNetTextproto, "textproto")
				break
			}
		}
		if r.contractHasResponseMultipart(contract) {
			srcFile.ImportName(PackageSync, "sync")
		}
		kindsUsed := make(map[string]struct{})
		for _, method := range contract.Methods {
			if !r.methodIsHTTP(contract, method) {
				continue
			}
			kindsUsed[content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagRequestContentType, "application/json"))] = struct{}{}
			kindsUsed[content.Kind(model.GetAnnotationValue(r.project, contract, method, nil, model.TagResponseContentType, "application/json"))] = struct{}{}
		}
		for k := range kindsUsed {
			switch k {
			case content.KindForm:
				// form.go генерируется в outDir
			case content.KindXML:
				srcFile.ImportName(PackageXML, "xml")
			case content.KindMsgpack:
				srcFile.ImportName(PackageMsgpack, "msgpack")
			case content.KindCBOR:
				srcFile.ImportName(PackageCBOR, "cbor")
			case content.KindYAML:
				srcFile.ImportName(PackageYAML, "yaml")
			}
		}
		if _, formUsed := kindsUsed[content.KindForm]; formUsed {
			if err = r.pkgRenderTo("schema", outDir, newPkgTemplateData()); err != nil {
				return
			}
			srcFile.ImportName(r.pkgPath(outDir)+"/schema", "schema")
		}
	}

	hasStreams := false
	hasWS := false
	hasSSE := false
	for _, method := range contract.Methods {
		if model.MethodIsWS(r.project, contract, method) {
			hasStreams = true
			hasWS = true
		}
		if model.MethodIsSSE(r.project, contract, method) {
			hasStreams = true
			hasSSE = true
		}
	}
	if hasStreams {
		srcFile.ImportName(PackageContext, "context")
		srcFile.ImportName(PackageFmt, "fmt")
		srcFile.ImportName(PackageStrings, "strings")
		srcFile.ImportName(PackageURL, "url")
		srcFile.ImportName(PackageUUID, "uuid")
		srcFile.ImportName(r.getPackageJSON(contract), keyJSON)
		if hasWS {
			srcFile.ImportName(packageWebsocket, "websocket")
			srcFile.ImportName(PackageHTTP, "http")
			srcFile.ImportName(PackageIO, "io")
		}
		if hasSSE {
			srcFile.ImportName(PackageBytes, "bytes")
			srcFile.ImportName("bufio", "bufio")
			srcFile.ImportName(PackageHTTP, "http")
		}
	}

	if r.HasMetrics() && model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) && r.contractHasHTTPMethods(contract) {
		srcFile.ImportName(PackageTime, PackageTime)
	}

	srcFile.Line().Type().Id("Client" + contract.Name).StructFunc(func(sg *Group) {
		sg.Op("*").Id("Client")
	}).Line()

	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) && r.contractHasResponseMultipart(contract) {
		srcFile.Add(r.StreamingMultipartHelperTypes())
	}
	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJSONRPC) && r.HasMetrics() && model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
		srcFile.Add(r.rpcRecordMetricsHelper(contract))
	}

	for _, method := range contract.Methods {
		if r.methodIsJSONRPC(contract, method) {
			srcFile.Type().Id("ret" + contract.Name + method.Name).Op("=").Func().Params(r.funcDefinitionParams(ctx, method.Results))
		}
	}

	for _, method := range contract.Methods {
		if r.methodIsJSONRPC(contract, method) {
			srcFile.Line().Add(r.jsonrpcClientMethodFunc(ctx, contract, method, outDir))
			srcFile.Line().Add(r.jsonrpcClientRequestFunc(ctx, contract, method, outDir))
		} else if r.methodIsHTTP(contract, method) {
			srcFile.Line().Add(r.httpClientMethodFunc(ctx, contract, method, outDir))
		}
		if model.MethodIsWS(r.project, contract, method) || model.MethodIsSSE(r.project, contract, method) {
			srcFile.Line().Add(r.streamClientMethod(ctx, contract, method))
			if model.MethodIsWS(r.project, contract, method) && model.MethodIsSSE(r.project, contract, method) {
				srcFile.Line().Add(r.streamSSEAlternateMethod(ctx, contract, method))
			}
		}
	}

	return srcFile.Save(path.Join(outDir, strings.ToLower(contract.Name)+"-client.go"))
}
