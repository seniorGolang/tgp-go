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

func (r *contractRenderer) expandInlineFields(vars []*model.Variable, methodTags tags.DocTags) (out []*model.Variable) {

	out = make([]*model.Variable, 0, len(vars))
	for _, v := range vars {
		if r.isInlineField(v, methodTags) {
			expanded := r.expandInlineStruct(v)
			out = append(out, expanded...)
		} else {
			out = append(out, v)
		}
	}
	return
}

func (r *contractRenderer) isInlineField(v *model.Variable, methodTags tags.DocTags) (ok bool) {

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

func (r *contractRenderer) expandInlineStruct(v *model.Variable) (out []*model.Variable) {

	typ, ok := r.project.Types[v.TypeID]
	if !ok {
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

	out = make([]*model.Variable, 0)
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
			expanded := r.expandInlineStruct(fieldVar)
			out = append(out, expanded...)
		} else {
			out = append(out, fieldVar)
		}
	}

	return
}

func (r *contractRenderer) requestStructFieldName(method *model.Method, v *model.Variable) (s string) {

	if r.isInlineField(v, method.Annotations) {
		return typeNameFromTypeID(r.project, v.TypeID)
	}
	return toCamel(v.Name)
}

func (r *contractRenderer) responseStructFieldName(method *model.Method, ret *model.Variable) (s string) {

	results := resultsWithoutError(method)
	if len(results) == 1 && model.IsAnnotationSet(r.project, r.contract, method, nil, model.TagHttpEnableInlineSingle) {
		if typeIsEmbeddable(r.project, ret.TypeID) {
			return typeNameFromTypeID(r.project, ret.TypeID)
		}

		return toCamel(ret.Name)
	}
	return toCamel(ret.Name)
}

func (r *contractRenderer) ArgsFieldsWithoutContext(method *model.Method) (out []*model.Variable) {

	argVars := r.expandInlineFields(method.Args, method.Annotations)
	if len(argVars) > 0 && argVars[0].TypeID == "context:Context" {
		return argVars[1:]
	}
	return argVars
}

func (r *contractRenderer) ResultFieldsWithoutError(method *model.Method) (out []*model.Variable) {

	resultVars := r.expandInlineFields(method.Results, method.Annotations)
	if len(resultVars) > 0 && resultVars[len(resultVars)-1].TypeID == "error" {
		return resultVars[:len(resultVars)-1]
	}
	return resultVars
}
