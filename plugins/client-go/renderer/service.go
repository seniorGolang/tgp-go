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

	"tgp/internal/model"
)

func (r *ClientRenderer) RenderServiceClient(contract *model.Contract) error {

	outDir := r.outDir
	pkgName := filepath.Base(outDir)
	srcFile := NewSrcFile(pkgName)
	srcFile.PackageComment(DoNotEdit)

	ctx := context.WithValue(context.Background(), keyCode, srcFile) // nolint
	ctx = context.WithValue(ctx, keyPackage, pkgName)                // nolint

	// Импорты для JSON-RPC
	if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerJsonRPC) {
		srcFile.ImportName(PackageUUID, "uuid")
		srcFile.ImportName(PackageFiber, "fiber")
		srcFile.ImportName(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "jsonrpc")
		// Импорт кастомного JSON пакета, если указан
		jsonPkg := r.getPackageJSON(contract)
		srcFile.ImportName(jsonPkg, "json")
	}

	// Импорты для HTTP
	if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerHTTP) {
		srcFile.ImportName(PackageContext, "context")
		srcFile.ImportName(PackageFmt, "fmt")
		srcFile.ImportName(PackageTime, "time")
		srcFile.ImportName(PackageHttp, "http")
		srcFile.ImportName("net/url", "url")
		srcFile.ImportName(PackageBytes, "bytes")
		srcFile.ImportName(PackageIO, "io")
		srcFile.ImportName(PackageStrings, "strings")
		// Всегда импортируем jsonrpc для toCurl, если он доступен
		srcFile.ImportName(fmt.Sprintf("%s/jsonrpc", r.pkgPath(outDir)), "jsonrpc")
		srcFile.ImportName(PackageSlog, "slog")
		// Импорт кастомного JSON пакета, если указан
		jsonPkg := r.getPackageJSON(contract)
		srcFile.ImportName(jsonPkg, "json")
	}

	// Импорты для метрик
	if r.HasMetrics() && model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
		srcFile.ImportName(PackageStrconv, "strconv")
		srcFile.ImportName(PackageTime, "time")
	}

	// Структура клиента сервиса
	srcFile.Line().Type().Id("Client" + contract.Name).StructFunc(func(sg *Group) {
		sg.Op("*").Id("Client")
	}).Line()

	for _, method := range contract.Methods {
		if r.methodIsJsonRPC(contract, method) {
			srcFile.Type().Id("ret" + contract.Name + method.Name).Op("=").Func().Params(r.funcDefinitionParams(ctx, method.Results))
		}
	}

	for _, method := range contract.Methods {
		if r.methodIsJsonRPC(contract, method) {
			srcFile.Line().Add(r.jsonrpcClientMethodFunc(ctx, contract, method, outDir))
			srcFile.Line().Add(r.jsonrpcClientRequestFunc(ctx, contract, method, outDir))
		} else if r.methodIsHTTP(method) {
			srcFile.Line().Add(r.httpClientMethodFunc(ctx, contract, method, outDir))
		}
	}

	return srcFile.Save(path.Join(outDir, strings.ToLower(contract.Name)+"-client.go"))
}
