// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"strings"
	"unicode"

	"tgp/internal/model"
)

func requestStructName(contractName string, methodName string) (s string) {

	return "request" + contractName + methodName
}

func responseStructName(contractName string, methodName string) (s string) {

	return "response" + contractName + methodName
}

func responseBodyStructName(contractName string, methodName string) (s string) {

	return "responseBody" + contractName + methodName
}

func responseResultStructName(contractName string, methodName string) (s string) {

	return "responseResult" + contractName + methodName
}

func argsWithoutContext(method *model.Method) (out []*model.Variable) {

	if len(method.Args) == 0 {
		return method.Args
	}
	if method.Args[0].TypeID == "context:Context" {
		return method.Args[1:]
	}
	return method.Args
}

func resultsWithoutError(method *model.Method) (out []*model.Variable) {

	if len(method.Results) == 0 {
		return method.Results
	}
	if method.Results[len(method.Results)-1].TypeID == "error" {
		return method.Results[:len(method.Results)-1]
	}
	return method.Results
}

func typeIsEmbeddable(project *model.Project, typeID string) (ok bool) {

	seen := make(map[string]struct{})
	for typeID != "" {
		if _, repeated := seen[typeID]; repeated {
			return false
		}
		seen[typeID] = struct{}{}

		typ, found := project.Types[typeID]
		if !found {
			return false
		}
		switch typ.Kind {
		case model.TypeKindStruct, model.TypeKindInterface:
			return true
		case model.TypeKindAlias:
			typeID = typ.AliasOf
		default:
			return false
		}
	}

	return false
}

func typeNameFromTypeID(project *model.Project, typeID string) (s string) {

	if typ, ok := project.Types[typeID]; ok && typ.TypeName != "" {
		return typ.TypeName
	}
	if idx := strings.Index(typeID, ":"); idx >= 0 && idx+1 < len(typeID) {
		return typeID[idx+1:]
	}
	return typeID
}

func toCamel(s string) (out string) {

	if s == "" {
		return s
	}
	s = strings.Trim(s, " ")
	result := strings.Builder{}
	capNext := true
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r)
			capNext = false
		case r >= '0' && r <= '9':
			result.WriteRune(r)
			capNext = false
		case r >= 'a' && r <= 'z':
			if capNext {
				result.WriteRune(unicode.ToUpper(r))
			} else {
				result.WriteRune(r)
			}
			capNext = false
		case r == '_' || r == ' ' || r == '-' || r == '.':
			capNext = true
		default:
			result.WriteRune(r)
			capNext = false
		}
	}
	return result.String()
}

func toLowerCamel(s string) (out string) {

	if s == "" {
		return s
	}
	allUpper := true
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			allUpper = false
			break
		}
	}
	if allUpper {
		return s
	}
	// Если первый символ в верхнем регистре, делаем его нижним
	runes := []rune(s)
	if len(runes) > 0 && runes[0] >= 'A' && runes[0] <= 'Z' {
		runes[0] = unicode.ToLower(runes[0])
		s = string(runes)
	}
	// Применяем toCamel, но с capNext = false
	s = strings.Trim(s, " ")
	result := strings.Builder{}
	capNext := false
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			result.WriteRune(r)
			capNext = false
		case r >= '0' && r <= '9':
			result.WriteRune(r)
			capNext = false
		case r >= 'a' && r <= 'z':
			if capNext {
				result.WriteRune(unicode.ToUpper(r))
			} else {
				result.WriteRune(r)
			}
			capNext = false
		case r == '_' || r == ' ' || r == '-' || r == '.':
			capNext = true
		default:
			result.WriteRune(r)
			capNext = false
		}
	}
	return result.String()
}
