// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package validate

import (
	"fmt"
	"strconv"
	"strings"

	"tgp/internal/content"
	"tgp/internal/model"
	"tgp/internal/tags"
)

var allowedHTTPMethods = map[string]struct{}{
	"GET":     {},
	"POST":    {},
	"PUT":     {},
	"PATCH":   {},
	"DELETE":  {},
	"OPTIONS": {},
}

func contractHTTPAnnotations(project *model.Project, contract *model.Contract) (err error) {

	if contract == nil {
		return nil
	}
	if !model.IsAnnotationSet(project, contract, nil, nil, model.TagServerHTTP) {
		return nil
	}

	for _, method := range contract.Methods {
		if err = methodHTTPAnnotations(project, contract, method); err != nil {
			return
		}
	}
	return nil
}

func methodHTTPAnnotations(project *model.Project, contract *model.Contract, method *model.Method) (err error) {

	if method == nil {
		return nil
	}
	if !model.IsAnnotationSet(project, contract, method, nil, model.TagHTTPMethod) {
		return nil
	}

	methodName := method.Name
	contractName := contract.Name

	if httpMethod := strings.TrimSpace(model.GetAnnotationValue(project, contract, method, nil, model.TagHTTPMethod, "")); httpMethod != "" {
		if _, ok := allowedHTTPMethods[strings.ToUpper(httpMethod)]; !ok {
			return fmt.Errorf("contract %q: method %q: http-method %q is not supported", contractName, methodName, httpMethod)
		}
	}

	if successValue := model.GetAnnotationValue(project, contract, method, nil, model.TagHttpSuccess, ""); successValue != "" {
		var code int
		if code, err = strconv.Atoi(successValue); err != nil || code <= 0 {
			return fmt.Errorf("contract %q: method %q: http-success must be a positive integer", contractName, methodName)
		}
	}

	if err = validateArgMapAnnotation(project, contract, method, model.TagHttpArg, "http-args"); err != nil {
		return
	}
	if err = validateArgMapAnnotation(project, contract, method, model.TagHttpHeader, "http-headers"); err != nil {
		return
	}
	if err = validateArgMapAnnotation(project, contract, method, model.TagHttpCookies, "http-cookies"); err != nil {
		return
	}
	if err = validateFormRequestAnnotations(project, contract, method); err != nil {
		return
	}
	return nil
}

func validateFormRequestAnnotations(project *model.Project, contract *model.Contract, method *model.Method) (err error) {

	requestContentType := model.GetAnnotationValue(project, contract, method, nil, model.TagRequestContentType, content.CanonicalMIME(content.KindJSON))
	if content.Kind(requestContentType) != content.KindForm {
		return nil
	}

	for _, arg := range model.HTTPArgsFromRequestBody(project, contract, method) {
		if model.ArgFieldEmbedded(project, method, arg) {
			continue
		}
		if _, ok := tags.FormFieldName(method.Annotations, arg.Name); !ok {
			return fmt.Errorf("contract %q: method %q: argument %q requires form:<name> tag when requestContentType is %s", contract.Name, method.Name, arg.Name, requestContentType)
		}
	}
	return nil
}

func validateArgMapAnnotation(project *model.Project, contract *model.Contract, method *model.Method, tagName string, label string) (err error) {

	value := model.GetAnnotationValue(project, contract, method, nil, tagName, "")
	if value == "" {
		return nil
	}

	if _, err = model.ParseArgMapEntriesStrict(value); err != nil {
		return fmt.Errorf("contract %q: method %q: %s: %w", contract.Name, method.Name, label, err)
	}
	return nil
}
