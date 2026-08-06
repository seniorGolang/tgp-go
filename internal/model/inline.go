// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"strings"

	"tgp/internal/tags"
)

// TypeIsEmbeddable reports whether typeID is a named type usable as an embedded struct field.
func TypeIsEmbeddable(project *Project, typeID string) (ok bool) {

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
		case TypeKindStruct, TypeKindInterface:
			return true
		case TypeKindAlias:
			typeID = typ.AliasOf
		default:
			return false
		}
	}

	return false
}

// TypeNameFromTypeID returns the Go type name for composite literals and field selectors (pkg:Item → Item).
func TypeNameFromTypeID(project *Project, typeID string) (name string) {

	if typ, ok := project.Types[typeID]; ok && typ.TypeName != "" {
		return typ.TypeName
	}
	if idx := strings.Index(typeID, ":"); idx >= 0 && idx+1 < len(typeID) {
		return typeID[idx+1:]
	}
	return typeID
}

// ArgFieldEmbedded reports whether the method argument is serialized as json:,inline in exchange.
func ArgFieldEmbedded(project *Project, method *Method, arg *Variable) (ok bool) {

	if method == nil || arg == nil {
		return false
	}

	jsonTag := tags.ParseMethodVarTags(method.Annotations, arg.Name)["json"]
	return jsonTag == jsonTagInline && TypeIsEmbeddable(project, arg.TypeID)
}

// ResultFieldEmbedded reports whether the method result is serialized as json:,inline in exchange.
func ResultFieldEmbedded(project *Project, contract *Contract, method *Method, ret *Variable) (ok bool) {

	if method == nil || ret == nil {
		return false
	}

	jsonTag := tags.ParseMethodVarTags(method.Annotations, ret.Name)["json"]
	if contract != nil {
		nonErr := 0
		for _, r := range method.Results {
			if r.TypeID != string(TypeKindError) {
				nonErr++
			}
		}
		if nonErr == 1 && IsAnnotationSet(project, contract, method, nil, TagHTTPEnableInlineSingle) && jsonTag == "" {
			jsonTag = jsonTagInline
		}
	}

	return jsonTag == jsonTagInline && TypeIsEmbeddable(project, ret.TypeID)
}

// StructFieldIsAnonymousEmbed reports whether the struct field is an anonymous Go embed.
func StructFieldIsAnonymousEmbed(field *StructField) (ok bool) {

	return field != nil && field.Anonymous
}

// StructFieldHasJSONInline reports whether struct field tags contain json:inline.
func StructFieldHasJSONInline(field *StructField) (ok bool) {

	if field == nil {
		return false
	}

	return tags.StructFieldHasJSONInline(field.Tags)
}

// StructFieldEmbedded reports whether the struct field is serialized as json:,inline embed.
func StructFieldEmbedded(project *Project, field *StructField) (ok bool) {

	if field == nil {
		return false
	}

	jsonTag := tags.StructFieldJSONTag(field.Tags)
	return jsonTag == jsonTagInline && TypeIsEmbeddable(project, field.TypeID)
}
