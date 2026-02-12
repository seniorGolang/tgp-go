// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"strings"

	"tgp/internal/model"
)

func (r *ClientRenderer) methodRequestBodyStreamArg(method *model.Method) *model.Variable {

	for _, arg := range r.argsWithoutContext(method) {
		if arg.TypeID == TypeIDIOReader {
			return arg
		}
	}
	return nil
}

func (r *ClientRenderer) methodResponseBodyStreamResult(method *model.Method) *model.Variable {

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
	return args
}

func (r *ClientRenderer) methodResponseBodyStreamResults(method *model.Method) (results []*model.Variable) {

	for _, res := range r.resultsWithoutError(method) {
		if res.TypeID == TypeIDIOReadCloser {
			results = append(results, res)
		}
	}
	return results
}

func (r *ClientRenderer) methodRequestMultipart(contract *model.Contract, method *model.Method) bool {

	readerArgs := r.methodRequestBodyStreamArgs(method)
	if len(readerArgs) > 1 {
		return true
	}
	if len(readerArgs) == 1 && model.IsAnnotationSet(r.project, contract, method, nil, model.TagHttpMultipart) {
		return true
	}
	return false
}

func (r *ClientRenderer) methodResponseMultipart(contract *model.Contract, method *model.Method) bool {

	readCloserResults := r.methodResponseBodyStreamResults(method)
	if len(readCloserResults) > 1 {
		return true
	}
	if len(readCloserResults) == 1 && model.IsAnnotationSet(r.project, contract, method, nil, model.TagHttpMultipart) {
		return true
	}
	return false
}

func (r *ClientRenderer) streamPartName(contract *model.Contract, method *model.Method, v *model.Variable) string {

	if v != nil && v.Annotations != nil {
		if val, found := v.Annotations[model.TagHttpPartName]; found && val != "" {
			return val
		}
	}
	if method != nil && method.Annotations != nil {
		if val, found := method.Annotations[model.TagHttpPartName]; found && val != "" {
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

func (r *ClientRenderer) varValueFromMethodMap(annotationValue string, varName string) string {

	for _, pair := range strings.Split(annotationValue, ",") {
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
