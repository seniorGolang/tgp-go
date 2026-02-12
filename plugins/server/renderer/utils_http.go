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

func (r *contractRenderer) batchPath() string {

	prefix := model.GetAnnotationValue(r.project, r.contract, nil, nil, model.TagHttpPrefix, "")
	pathValue := model.GetAnnotationValue(r.project, r.contract, nil, nil, model.TagHttpPath, "/"+toLowerCamel(r.contract.Name))
	if prefix != "" {
		return path.Join("/", prefix, strings.TrimPrefix(pathValue, "/"))
	}
	return pathValue
}

func (r *contractRenderer) methodHTTPMethod(method *model.Method) string {

	httpMethod := model.GetHTTPMethod(r.project, r.contract, method)
	if httpMethod == "" {
		return "post"
	}
	switch strings.ToUpper(httpMethod) {
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

func (r *contractRenderer) methodHTTPPath(method *model.Method) string {

	prefix := model.GetAnnotationValue(r.project, r.contract, nil, nil, model.TagHttpPrefix, "")
	methodPath := model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagHttpPath, "/"+toLowerCamel(method.Name))
	if prefix != "" {
		return path.Join("/", prefix, strings.TrimPrefix(methodPath, "/"))
	}
	return methodPath
}

func (r *contractRenderer) methodJsonRPCPath(method *model.Method) string {

	elements := make([]string, 0, 3)
	elements = append(elements, "/")
	prefix := model.GetAnnotationValue(r.project, r.contract, nil, nil, model.TagHttpPrefix, "")
	urlPath := model.GetAnnotationValue(r.project, r.contract, method, nil, model.TagHttpPath, "/"+toLowerCamel(method.Name))
	urlPath = strings.Split(urlPath, ":")[0]
	return path.Join(append(elements, prefix, urlPath)...)
}

func (r *contractRenderer) methodIsJsonRPC(method *model.Method) bool {

	if method == nil {
		return false
	}
	return r.contract != nil && model.IsAnnotationSet(r.project, r.contract, nil, nil, model.TagServerJsonRPC) && !model.IsAnnotationSet(r.project, r.contract, method, nil, model.TagHTTPMethod)
}

func (r *contractRenderer) methodHandlerQual(srcFile *GoFile, method *model.Method) Code {

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
