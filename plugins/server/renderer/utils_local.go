// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"strings"
	"unicode"

	"tgp/internal/model"
)

func requestStructName(contractName string, methodName string) string {

	return "request" + contractName + methodName
}

func responseStructName(contractName string, methodName string) string {

	return "response" + contractName + methodName
}

func argsWithoutContext(method *model.Method) []*model.Variable {

	if len(method.Args) == 0 {
		return method.Args
	}
	if method.Args[0].TypeID == "context:Context" {
		return method.Args[1:]
	}
	return method.Args
}

func resultsWithoutError(method *model.Method) []*model.Variable {

	if len(method.Results) == 0 {
		return method.Results
	}
	if method.Results[len(method.Results)-1].TypeID == "error" {
		return method.Results[:len(method.Results)-1]
	}
	return method.Results
}

func typeNameFromTypeID(project *model.Project, typeID string) string {

	if typ, ok := project.Types[typeID]; ok && typ.TypeName != "" {
		return typ.TypeName
	}
	if idx := strings.Index(typeID, ":"); idx >= 0 && idx+1 < len(typeID) {
		return typeID[idx+1:]
	}
	return typeID
}

func removeSkippedFields(vars []*model.Variable, skipFields []string) []*model.Variable {

	result := make([]*model.Variable, 0, len(vars))
	for _, v := range vars {
		found := false
		for _, skip := range skipFields {
			if v.Name == skip {
				found = true
				break
			}
		}
		if !found {
			result = append(result, v)
		}
	}
	return result
}

func toCamel(s string) string {

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

func toLowerCamel(s string) string {

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
