// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"strings"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func isRequiredGeneratedRequestField(variable *model.Variable, methodTags tags.DocTags) (required bool) {

	if variable == nil {
		return
	}
	if hasRequiredAnnotation(variable.Annotations) {
		return true
	}
	if isPointerTypeRef(variable.TypeRef) {
		return
	}
	return !hasJSONOmitemptyForVariable(variable, methodTags)
}

func isRequiredGeneratedResponseField(variable *model.Variable, methodTags tags.DocTags) (required bool) {

	if variable == nil {
		return
	}
	if hasRequiredAnnotation(variable.Annotations) {
		return true
	}
	return !hasJSONOmitemptyForVariable(variable, methodTags)
}

func isRequiredStructField(field *model.StructField) (required bool) {

	if field == nil {
		return
	}
	if hasRequiredAnnotation(field.Annotations) {
		return true
	}
	return !hasJSONOmitemptyForStructField(field)
}

func isRequiredQueryParameter(variable *model.Variable) (required bool) {

	if variable == nil {
		return
	}
	if hasRequiredAnnotation(variable.Annotations) {
		return true
	}
	return !isPointerTypeRef(variable.TypeRef)
}

func isRequiredHeaderOrCookie(variable *model.Variable) (required bool) {

	if variable == nil {
		return
	}
	return hasRequiredAnnotation(variable.Annotations)
}

func hasRequiredAnnotation(annotations tags.DocTags) (required bool) {

	return annotations != nil && annotations.IsSet(model.TagRequired)
}

func isPointerTypeRef(typeRef model.TypeRef) (isPointer bool) {

	return typeRef.NumberOfPointers > 0
}

func hasJSONOmitemptyForStructField(field *model.StructField) (hasOmitempty bool) {

	tagValues, ok := field.Tags["json"]
	if !ok || len(tagValues) < 2 {
		return
	}
	for _, option := range tagValues[1:] {
		if strings.TrimSpace(option) == "omitempty" {
			return true
		}
	}
	return
}

func hasJSONOmitemptyForVariable(variable *model.Variable, methodTags tags.DocTags) (hasOmitempty bool) {

	if variable != nil && variable.Annotations != nil {
		if hasOmitempty := hasJSONOmitemptyTag(variable.Annotations.Value("json", "")); hasOmitempty {
			return true
		}
	}
	if variable != nil {
		if jsonTag, ok := tags.ParseMethodVarTags(methodTags, variable.Name)["json"]; ok {
			if hasOmitempty := hasJSONOmitemptyTag(jsonTag); hasOmitempty {
				return true
			}
		}
	}
	return
}

func hasJSONOmitemptyTag(jsonTag string) (hasOmitempty bool) {

	if jsonTag == "" {
		return
	}
	for _, part := range strings.Split(jsonTag, ",") {
		if strings.TrimSpace(part) == "omitempty" {
			return true
		}
	}
	return
}
