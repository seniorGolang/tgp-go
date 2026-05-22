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

func buildFullPath(prefix string, pathValue string) (result string) {

	trimmed := strings.TrimPrefix(pathValue, "/")
	if prefix == "" {
		return "/" + trimmed
	}
	return path.Join("/", prefix, trimmed)
}

func (r *contractRenderer) batchPath() (path string) {

	prefix := model.GetAnnotationValue(r.project, r.contract, nil, nil, model.TagHttpPrefix, "")
	pathValue := model.GetAnnotationValue(r.project, r.contract, nil, nil, model.TagHttpPath, "/"+toLowerCamel(r.contract.Name))
	return buildFullPath(prefix, pathValue)
}

func (r *transportRenderer) generalBatchPath() (path string) {

	var seen map[string]struct{}
	for _, c := range r.contractsSorted() {
		if !model.IsAnnotationSet(r.project, c, nil, nil, model.TagServerJsonRPC) {
			continue
		}
		p := model.GetAnnotationValue(r.project, c, nil, nil, model.TagHttpPrefix, "")
		if p == "" {
			continue
		}
		if seen == nil {
			seen = make(map[string]struct{})
		}
		seen[p] = struct{}{}
	}
	if len(seen) != 1 {
		return "/"
	}
	var single string
	for p := range seen {
		single = p
		break
	}
	return buildFullPath(single, "/")
}

func (r *contractRenderer) methodHTTPMethod(method *model.Method) (httpMethod string) {

	m := model.GetHTTPMethod(r.project, r.contract, method)
	if m == "" {
		return "post"
	}
	switch strings.ToUpper(m) {
	case "GET":
		return "get"
	case "PUT":
		return "put"
	case "PATCH":
		return "patch"
	case "DELETE":
		return "delete"
	case "OPTIONS":
		return "options"
	default:
		return "post"
	}
}

func (r *contractRenderer) defaultMethodPathValue(method *model.Method) (path string) {
	return "/" + toLowerCamel(r.contract.Name) + "/" + toLowerCamel(method.Name)
}

func (r *contractRenderer) methodHTTPPath(method *model.Method) (path string) {

	prefix := model.GetAnnotationValue(r.project, r.contract, nil, nil, model.TagHttpPrefix, "")
	methodPath := model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagHttpPath, r.defaultMethodPathValue(method))
	return buildFullPath(prefix, methodPath)
}

func (r *contractRenderer) methodJsonRPCPath(method *model.Method) (path string) {

	prefix := model.GetAnnotationValue(r.project, r.contract, nil, nil, model.TagHttpPrefix, "")
	pathValue := model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagHttpPath, r.defaultMethodPathValue(method))
	pathBase := strings.TrimPrefix(strings.Split(pathValue, ":")[0], "/")
	return buildFullPath(prefix, "/"+pathBase)
}

func (r *contractRenderer) methodIsJsonRPC(method *model.Method) (ok bool) {

	if method == nil {
		return false
	}
	return r.contract != nil && model.IsAnnotationSet(r.project, r.contract, nil, nil, model.TagServerJsonRPC) && !model.IsAnnotationSet(r.project, r.contract, method, nil, model.TagHTTPMethod)
}

func (r *contractRenderer) methodHandlerQual(srcFile *GoFile, method *model.Method) (code Code) {

	handlerValue := model.GetAnnotationValue(r.project, r.contract, method, nil, TagHandler, "")
	if handlerValue == "" {
		return Id("")
	}
	if tokens := strings.Split(handlerValue, ":"); len(tokens) == 2 {
		pkgPath := tokens[0]
		funcName := tokens[1]
		baseName := filepath.Base(pkgPath)
		srcFile.ImportName(pkgPath, baseName)
		return Qual(pkgPath, funcName)
	}
	return Id(handlerValue)
}
