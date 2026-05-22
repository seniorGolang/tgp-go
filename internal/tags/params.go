// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package tags

import (
	"strings"
)

// paramTagsKey — под-ключ в method.Annotations.Sub(varName); совпадает с model.TagParamTags.
const paramTagsKey = "tags"

// ParseMethodVarTags разбирает теги сериализации аргумента/результата метода.
// Формат в контракте: // @tg <varName>.tags=json:inline|form:filter
func ParseMethodVarTags(methodTags DocTags, varName string) (out map[string]string) {

	out = make(map[string]string)
	if len(methodTags) == 0 || varName == "" {
		return
	}

	value := methodTags.Sub(varName).Value(paramTagsKey, "")
	if value == "" {
		return
	}

	for _, item := range strings.Split(value, "|") {
		tokens := strings.SplitN(strings.TrimSpace(item), ":", 2)
		if len(tokens) < 2 {
			continue
		}
		tagName := strings.TrimSpace(tokens[0])
		tagValue := strings.TrimSpace(tokens[1])
		if tagValue == "inline" {
			tagValue = ",inline"
		}
		out[tagName] = tagValue
	}
	return
}

// HasJSONInline сообщает, задан ли для переменной json:inline.
func HasJSONInline(methodTags DocTags, varName string) (ok bool) {

	tagValue, found := ParseMethodVarTags(methodTags, varName)["json"]
	if !found {
		return
	}

	return tagValue == ",inline" || strings.Contains(tagValue, ",inline")
}

// FormFieldName возвращает имя поля form из тегов переменной (form:<name>).
func FormFieldName(methodTags DocTags, varName string) (name string, ok bool) {

	name, found := ParseMethodVarTags(methodTags, varName)["form"]
	return name, found && name != ""
}
