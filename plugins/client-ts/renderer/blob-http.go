// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"strings"

	"tgp/internal/model"
)

func (r *ClientRenderer) methodRequestBodyStreamArg(method *model.Method) (v *model.Variable) {

	for _, arg := range r.argsWithoutContext(method) {
		if arg.TypeID == TypeIDIOReader {
			return arg
		}
	}
	return nil
}

func (r *ClientRenderer) methodResponseBodyStreamResult(method *model.Method) (v *model.Variable) {

	for _, res := range r.resultsWithoutError(method) {
		if res.TypeID == TypeIDIOReadCloser {
			return res
		}
	}
	return nil
}

func (r *ClientRenderer) methodRequestBodyStreamArgs(method *model.Method) (args []*model.Variable) {

	for _, arg := range r.argsWithoutContext(method) {
		if arg.TypeID == TypeIDIOReader {
			args = append(args, arg)
		}
	}
	return
}

func (r *ClientRenderer) methodResponseBodyStreamResults(method *model.Method) (results []*model.Variable) {

	for _, res := range r.resultsWithoutError(method) {
		if res.TypeID == TypeIDIOReadCloser {
			results = append(results, res)
		}
	}
	return
}

func (r *ClientRenderer) methodRequestMultipart(contract *model.Contract, method *model.Method) (ok bool) {

	readerArgs := r.methodRequestBodyStreamArgs(method)
	if len(readerArgs) > 1 {
		return true
	}
	if len(readerArgs) == 1 && model.IsAnnotationSet(r.project, contract, method, nil, model.TagHTTPMultipart) {
		return true
	}
	return false
}

func (r *ClientRenderer) methodResponseMultipart(contract *model.Contract, method *model.Method) (ok bool) {

	readCloserResults := r.methodResponseBodyStreamResults(method)
	if len(readCloserResults) > 1 {
		return true
	}
	if len(readCloserResults) == 1 && model.IsAnnotationSet(r.project, contract, method, nil, model.TagHTTPMultipart) {
		return true
	}
	return false
}

func (r *ClientRenderer) streamPartName(_contract *model.Contract, method *model.Method, v *model.Variable) (s string) {

	if v != nil && v.Annotations != nil {
		if val, found := v.Annotations[model.TagHTTPPartName]; found && val != "" {
			return val
		}
	}
	if method != nil && method.Annotations != nil {
		if val, found := method.Annotations[model.TagHTTPPartName]; found && val != "" {
			if v != nil && v.Name != "" {
				if partName := r.varValueFromMethodMap(val, v.Name); partName != "" {
					return partName
				}
			}
		}
	}
	if v != nil && v.Name != "" {
		return v.Name
	}
	return ""
}

func (r *ClientRenderer) varValueFromMethodMap(annotationValue string, varName string) (s string) {

	for pair := range strings.SplitSeq(annotationValue, ",") {
		if pairTokens := strings.Split(strings.TrimSpace(pair), "|"); len(pairTokens) == 2 {
			arg := strings.TrimSpace(pairTokens[0])
			value := strings.TrimSpace(pairTokens[1])
			if arg == varName {
				return value
			}
		}
	}
	return ""
}
