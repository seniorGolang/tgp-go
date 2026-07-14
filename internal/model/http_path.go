// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"path"
	"strings"
)

// DefaultMethodHTTPPath — путь метода по умолчанию, если @tg http-path не задан.
func DefaultMethodHTTPPath(contract *Contract, method *Method) (methodPath string) {

	if contract == nil || method == nil {
		return "/"
	}

	return "/" + LowerCamel(contract.Name) + "/" + LowerCamel(method.Name)
}

// MethodHTTPPathValue возвращает @tg http-path или DefaultMethodHTTPPath.
func MethodHTTPPathValue(project *Project, contract *Contract, method *Method) (methodPath string) {

	return GetAnnotationValue(project, contract, method, nil, TagHttpPath, DefaultMethodHTTPPath(contract, method))
}

// JoinHTTPPath склеивает @tg http-prefix и path метода (как server, swagger, HTTP-клиенты).
func JoinHTTPPath(prefix string, pathValue string) (fullPath string) {

	trimmed := strings.TrimPrefix(pathValue, "/")
	if prefix == "" {
		return "/" + trimmed
	}

	return path.Join("/", prefix, trimmed)
}

// MethodHTTPFullPath — итоговый HTTP-маршрут: prefix + path.
func MethodHTTPFullPath(project *Project, contract *Contract, method *Method) (fullPath string) {

	prefix := GetAnnotationValue(project, contract, nil, nil, TagHttpPrefix, "")
	methodPath := MethodHTTPPathValue(project, contract, method)

	return JoinHTTPPath(prefix, methodPath)
}

func lowerCamel(name string) (result string) {

	if name == "" {
		return name
	}

	if isAllUpperASCII(name) {
		return name
	}

	result = strings.ToLower(string(name[0])) + name[1:]

	parts := strings.FieldsFunc(result, func(r rune) bool {
		return r == '_' || r == '-'
	})
	if len(parts) > 1 {
		result = parts[0]
		for i := 1; i < len(parts); i++ {
			if parts[i] != "" {
				result += strings.ToUpper(string(parts[i][0])) + parts[i][1:]
			}
		}
	}

	return result
}

func isAllUpperASCII(s string) (allUpper bool) {

	if s == "" {
		return false
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			return false
		}
	}

	return true
}
