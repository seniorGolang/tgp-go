// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/content"
	"tgp/internal/model"
)

func (r *ClientRenderer) RenderServiceClient(contract *model.Contract) error {

	outDir := r.outDir
	pkgName := filepath.Base(outDir)
	srcFile := NewSrcFile(pkgName)
	srcFile.PackageComment(DoNotEdit)

	ctx := context.WithValue(context.Background(), keyCode, srcFile) // nolint
	ctx = context.WithValue(ctx, keyPackage, pkgName)                // nolint

	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
		srcFile.ImportName(PackageUUID, "uuid")
		srcFile.ImportName(PackageFiber, "fiber")
		srcFile.ImportName(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "jsonrpc")
		jsonPkg := r.getPackageJSON(contract)
		srcFile.ImportName(jsonPkg, "json")
	}

	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
		srcFile.ImportName(PackageContext, "context")
		srcFile.ImportName(PackageFmt, "fmt")
		srcFile.ImportName(PackageTime, "time")
		srcFile.ImportName(PackageHttp, "http")
		srcFile.ImportName(PackageURL, "url")
		srcFile.ImportName(PackagePath, "path")
		srcFile.ImportName(PackageBytes, "bytes")
		srcFile.ImportName(PackageIO, "io")
		srcFile.ImportName(PackageStrings, "strings")
		srcFile.ImportName(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "jsonrpc")
		srcFile.ImportName(PackageSlog, "slog")
		jsonPkg := r.getPackageJSON(contract)
		srcFile.ImportName(jsonPkg, "json")
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
			if err := r.copySchemaTo(outDir); err != nil {
				return err
			}
			srcFile.ImportName(fmt.Sprintf("%s/schema", r.pkgPath(outDir)), "schema")
		}
	}

	if r.HasMetrics() && model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
		srcFile.ImportName(PackageStrconv, "strconv")
		srcFile.ImportName(PackageTime, "time")
	}

	srcFile.Line().Type().Id("Client" + contract.Name).StructFunc(func(sg *Group) {
		sg.Op("*").Id("Client")
	}).Line()

	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) && r.contractHasResponseMultipart(contract) {
		srcFile.Add(r.StreamingMultipartHelperTypes())
	}
	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) && r.contractHasHTTPMethods(contract) {
		srcFile.Add(r.httpApplyHeadersFromCtxHelper(contract))
		srcFile.Add(r.httpDoRoundTripHelper(contract, outDir))
		if r.HasMetrics() && model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
			srcFile.Add(r.httpRecordHTTPMetricsHelper(contract))
		}
	}
	if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) && r.HasMetrics() && model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
		srcFile.Add(r.rpcRecordMetricsHelper(contract))
	}

	for _, method := range contract.Methods {
		if r.methodIsJsonRPC(contract, method) {
			srcFile.Type().Id("ret" + contract.Name + method.Name).Op("=").Func().Params(r.funcDefinitionParams(ctx, method.Results))
		}
	}

	for _, method := range contract.Methods {
		if r.methodIsJsonRPC(contract, method) {
			srcFile.Line().Add(r.jsonrpcClientMethodFunc(ctx, contract, method, outDir))
			srcFile.Line().Add(r.jsonrpcClientRequestFunc(ctx, contract, method, outDir))
		} else if r.methodIsHTTP(contract, method) {
			srcFile.Line().Add(r.httpClientMethodFunc(ctx, contract, method, outDir))
		}
	}

	return srcFile.Save(path.Join(outDir, strings.ToLower(contract.Name)+"-client.go"))
}
