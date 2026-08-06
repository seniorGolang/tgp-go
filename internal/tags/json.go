// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package tags

import (
	"slices"
	"strings"
)

// StructFieldJSONTag собирает значение json-тега struct field из map тегов.
func StructFieldJSONTag(fieldTags map[string][]string) (tag string) {

	if values, ok := fieldTags["json"]; ok && len(values) > 0 {
		return strings.Join(values, ",")
	}

	return
}

// StructFieldHasJSONInline сообщает, задан ли json:inline в тегах struct field.
func StructFieldHasJSONInline(fieldTags map[string][]string) (ok bool) {

	if values, found := fieldTags["json"]; found {
		if slices.Contains(values, "inline") {
			return true
		}
	}

	return
}

// StructFieldJSONName возвращает json-имя поля и признак inline.
func StructFieldJSONName(fieldTags map[string][]string, fieldName string) (jsonName string, inline bool) {

	inline = StructFieldHasJSONInline(fieldTags)

	if values, ok := fieldTags["json"]; ok {
		for _, val := range values {
			if val == "inline" || val == "omitempty" {
				continue
			}
			if val == "-" {
				return "-", inline
			}
			if val != "" {
				return val, inline
			}
		}
		if inline {
			return "", inline
		}
	}

	if fieldName == "" || fieldName == "-" {
		return fieldName, inline
	}

	if strings.HasPrefix(fieldName, strings.ToLower(string(fieldName[0]))) {
		return "", inline
	}

	return fieldName, inline
}
