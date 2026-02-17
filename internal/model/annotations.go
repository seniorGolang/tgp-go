// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"strings"
)

const (
	TagHTTPMethod             = "http-method"
	DefaultHTTPMethod         = "POST"
	TagHttpPrefix             = "http-prefix"
	TagHttpPath               = "http-path"
	TagHttpSuccess            = "http-success"
	TagHttpArg                = "http-args"
	TagHttpHeader             = "http-headers"
	TagHttpCookies            = "http-cookies"
	TagRequestContentType     = "requestContentType"
	TagResponseContentType    = "responseContentType"
	TagHttpMultipart          = "http-multipart"
	TagHttpPartName           = "http-part-name"
	TagHttpPartContent        = "http-part-content"
	TagServerJsonRPC          = "jsonRPC-server"
	TagServerHTTP             = "http-server"
	TagHttpEnableInlineSingle = "enableInlineSingle"
	TagParamTags              = "tags"
	TagRequired               = "required"
)

// Если аннотация http-method не задана, возвращает DefaultHTTPMethod.
func GetHTTPMethod(project *Project, contract *Contract, method *Method) (methodName string) {
	return strings.TrimSpace(GetAnnotationValue(project, contract, method, nil, TagHTTPMethod, DefaultHTTPMethod))
}

// Поиск снизу вверх (variable → method → contract → project). Приоритет у ближайшего к месту использования.
func GetAnnotationValue(project *Project, contract *Contract, method *Method, variable *Variable, tagName string, defaultValue ...string) (value string) {

	if variable != nil && variable.Annotations != nil {
		if val, found := variable.Annotations[tagName]; found && val != "" {
			return val
		}
	}

	if method != nil && method.Annotations != nil {
		if val, found := method.Annotations[tagName]; found && val != "" {
			return val
		}
	}

	if contract != nil && contract.Annotations != nil {
		if val, found := contract.Annotations[tagName]; found && val != "" {
			return val
		}
	}

	if project != nil && project.Annotations != nil {
		return project.Annotations.Value(tagName, defaultValue...)
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

func GetAnnotationValueInt(project *Project, contract *Contract, method *Method, variable *Variable, tagName string, defaultValue ...int) (value int) {

	if variable != nil && variable.Annotations != nil && variable.Annotations.IsSet(tagName) {
		return variable.Annotations.ValueInt(tagName, defaultValue...)
	}

	if method != nil && method.Annotations != nil && method.Annotations.IsSet(tagName) {
		return method.Annotations.ValueInt(tagName, defaultValue...)
	}

	if contract != nil && contract.Annotations != nil && contract.Annotations.IsSet(tagName) {
		return contract.Annotations.ValueInt(tagName, defaultValue...)
	}

	if project != nil && project.Annotations != nil {
		return project.Annotations.ValueInt(tagName, defaultValue...)
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return
}

func GetAnnotationValueBool(project *Project, contract *Contract, method *Method, variable *Variable, tagName string, defaultValue ...bool) (value bool) {

	if variable != nil && variable.Annotations != nil && variable.Annotations.IsSet(tagName) {
		return variable.Annotations.ValueBool(tagName, defaultValue...)
	}

	if method != nil && method.Annotations != nil && method.Annotations.IsSet(tagName) {
		return method.Annotations.ValueBool(tagName, defaultValue...)
	}

	if contract != nil && contract.Annotations != nil && contract.Annotations.IsSet(tagName) {
		return contract.Annotations.ValueBool(tagName, defaultValue...)
	}

	if project != nil && project.Annotations != nil {
		return project.Annotations.ValueBool(tagName, defaultValue...)
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return
}

func IsAnnotationSet(project *Project, contract *Contract, method *Method, variable *Variable, tagName string) (found bool) {

	if variable != nil && variable.Annotations != nil {
		if variable.Annotations.IsSet(tagName) {
			return true
		}
	}

	if method != nil && method.Annotations != nil {
		if method.Annotations.IsSet(tagName) {
			return true
		}
	}

	if contract != nil && contract.Annotations != nil {
		if contract.Annotations.IsSet(tagName) {
			return true
		}
	}

	if project != nil && project.Annotations != nil {
		return project.Annotations.IsSet(tagName)
	}

	return
}
