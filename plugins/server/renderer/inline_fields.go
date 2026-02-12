// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"slices"
	"strings"

	"tgp/internal/common"
	"tgp/internal/model"
	"tgp/internal/tags"
)

func (r *contractRenderer) expandInlineFields(vars []*model.Variable, methodTags tags.DocTags) []*model.Variable {

	result := make([]*model.Variable, 0, len(vars))
	for _, v := range vars {
		if r.isInlineField(v, methodTags) {
			// Разворачиваем inline структуру
			expanded := r.expandInlineStruct(v)
			result = append(result, expanded...)
		} else {
			result = append(result, v)
		}
	}
	return result
}

func (r *contractRenderer) isInlineField(v *model.Variable, methodTags tags.DocTags) bool {

	for key, value := range common.SortedPairs(methodTags.Sub(v.Name)) {
		if key == model.TagParamTags {
			// Формат: tag:json:fieldName,inline|tag:xml:fieldName
			if list := strings.Split(value, "|"); len(list) > 0 {
				for _, item := range list {
					if tokens := strings.Split(item, ":"); len(tokens) >= 2 {
						tagName := tokens[0]
						tagValue := strings.Join(tokens[1:], ":")
						if tagName == "json" && (tagValue == "inline" || strings.Contains(tagValue, ",inline")) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func (r *contractRenderer) expandInlineStruct(v *model.Variable) []*model.Variable {

	typ, ok := r.project.Types[v.TypeID]
	if !ok {
		// Тип не найден, возвращаем как есть
		return []*model.Variable{v}
	}

	if typ.Kind != model.TypeKindStruct {
		return []*model.Variable{v}
	}

	// Если это алиас структуры, получаем базовый тип
	if typ.Kind == model.TypeKindAlias && typ.AliasOf != "" {
		baseTyp, ok := r.project.Types[typ.AliasOf]
		if ok && baseTyp.Kind == model.TypeKindStruct {
			typ = baseTyp
		}
	}

	// Разворачиваем поля структуры
	result := make([]*model.Variable, 0)
	for _, field := range typ.StructFields {
		fieldVar := &model.Variable{
			TypeRef: field.TypeRef,
			Name:    field.Name,
		}

		isInline := false
		if field.Tags != nil {
			if jsonTags, ok := field.Tags["json"]; ok {
				isInline = slices.ContainsFunc(jsonTags, func(tag string) bool {
					return tag == "inline" || strings.Contains(tag, ",inline")
				})
			}
		}

		if isInline {
			// Рекурсивно разворачиваем вложенные inline поля
			expanded := r.expandInlineStruct(fieldVar)
			result = append(result, expanded...)
		} else {
			result = append(result, fieldVar)
		}
	}

	return result
}

func (r *contractRenderer) requestStructFieldName(method *model.Method, v *model.Variable) string {

	if r.isInlineField(v, method.Annotations) {
		return typeNameFromTypeID(r.project, v.TypeID)
	}
	return toCamel(v.Name)
}

func (r *contractRenderer) responseStructFieldName(method *model.Method, ret *model.Variable) string {

	results := resultsWithoutError(method)
	if len(results) == 1 && model.IsAnnotationSet(r.project, r.contract, method, nil, model.TagHttpEnableInlineSingle) {
		return typeNameFromTypeID(r.project, ret.TypeID)
	}
	return toCamel(ret.Name)
}

func (r *contractRenderer) ArgsFieldsWithoutContext(method *model.Method) []*model.Variable {

	argVars := r.expandInlineFields(method.Args, method.Annotations)
	// Убираем context, если он первый
	if len(argVars) > 0 && argVars[0].TypeID == "context:Context" {
		return argVars[1:]
	}
	return argVars
}

func (r *contractRenderer) ResultFieldsWithoutError(method *model.Method) []*model.Variable {

	resultVars := r.expandInlineFields(method.Results, method.Annotations)
	// Убираем error, если он последний
	if len(resultVars) > 0 && resultVars[len(resultVars)-1].TypeID == "error" {
		return resultVars[:len(resultVars)-1]
	}
	return resultVars
}
