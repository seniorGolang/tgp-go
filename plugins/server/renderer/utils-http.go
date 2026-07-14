// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path/filepath"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/internal/model"
)

func (r *contractRenderer) batchPath() (path string) {

	prefix := model.GetAnnotationValue(r.project, r.contract, nil, nil, model.TagHttpPrefix, "")
	pathValue := model.GetAnnotationValue(r.project, r.contract, nil, nil, model.TagHttpPath, "/"+toLowerCamel(r.contract.Name))

	return model.JoinHTTPPath(prefix, pathValue)
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

func (r *contractRenderer) methodHTTPPath(method *model.Method) (path string) {

	return model.MethodHTTPFullPath(r.project, r.contract, method)
}

func (r *contractRenderer) methodJsonRPCPath(method *model.Method) (path string) {

	prefix := model.GetAnnotationValue(r.project, r.contract, nil, nil, model.TagHttpPrefix, "")
	pathValue := model.MethodHTTPPathValue(r.project, r.contract, method)
	pathBase := strings.TrimPrefix(strings.Split(pathValue, ":")[0], "/")

	return model.JoinHTTPPath(prefix, "/"+pathBase)
}

func methodIsHTTP(project *model.Project, contract *model.Contract, method *model.Method) (ok bool) {

	if contract == nil || method == nil {
		return false
	}
	contractHasHTTP := model.IsAnnotationSet(project, contract, nil, nil, model.TagServerHTTP)
	if !contractHasHTTP {
		return false
	}
	contractHasJsonRPC := model.IsAnnotationSet(project, contract, nil, nil, model.TagServerJsonRPC)
	methodHasExplicitHTTP := model.IsAnnotationSet(project, contract, method, nil, model.TagHTTPMethod)
	return !contractHasJsonRPC || methodHasExplicitHTTP
}

func (r *contractRenderer) methodIsHTTP(method *model.Method) (ok bool) {

	return methodIsHTTP(r.project, r.contract, method)
}

func (r *contractRenderer) methodIsJsonRPC(method *model.Method) (ok bool) {

	if method == nil {
		return false
	}
	return r.contract != nil && model.IsAnnotationSet(r.project, r.contract, nil, nil, model.TagServerJsonRPC) && !model.IsAnnotationSet(r.project, r.contract, method, nil, model.TagHTTPMethod)
}

func (r *contractRenderer) methodHandlerQual(srcFile *GoFile, method *model.Method) (stmt *Statement) {

	handlerValue := model.GetAnnotationValue(r.project, r.contract, method, nil, TagHandler, "")
	if handlerValue == "" {
		return Id("")
	}
	if tokens := strings.Split(handlerValue, ":"); len(tokens) == 2 {
		pkgPath := strings.TrimSpace(tokens[0])
		funcName := strings.TrimSpace(tokens[1])
		baseName := filepath.Base(pkgPath)
		srcFile.ImportName(pkgPath, baseName)
		return Qual(pkgPath, funcName)
	}
	return Id(handlerValue)
}
