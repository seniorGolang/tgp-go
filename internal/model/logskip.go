// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package model

import (
	"strings"

	"tgp/internal/tags"
)

const (
	logSkipRequest  = "request"
	logSkipResponse = "response"
	dumperTagName   = "dumper"
	dumperHideWord  = "hide"
)

// LogSkipSides — пропуск целых атрибутов request/response в логах.
type LogSkipSides struct {
	Request  bool
	Response bool
}

// MethodLogSkip разбирает method-level log-skip=a,b,request,response.
func MethodLogSkip(project *Project, contract *Contract, method *Method) (sides LogSkipSides, names map[string]struct{}) {

	names = make(map[string]struct{})
	raw := strings.TrimSpace(GetAnnotationValue(project, contract, method, nil, TagLogSkip, ""))
	if raw == "" {
		return
	}
	for part := range strings.SplitSeq(raw, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		switch name {
		case logSkipRequest:
			sides.Request = true
		case logSkipResponse:
			sides.Response = true
		default:
			names[name] = struct{}{}
		}
	}
	return
}

// VariableLogSkipped — аргумент/результат исключён из логов.
// Не использует IsAnnotationSet(method): method-level log-skip=secret не должен скипнуть все переменные.
func VariableLogSkipped(method *Method, variable *Variable, names map[string]struct{}) (skipped bool) {

	if variable == nil {
		return false
	}
	if variable.Annotations.IsSet(TagLogSkip) {
		return true
	}
	if method != nil {
		if method.Annotations.Sub(variable.Name).IsSet(TagLogSkip) {
			return true
		}
		if DumperHidesFully(tags.ParseMethodVarTags(method.Annotations, variable.Name)[dumperTagName]) {
			return true
		}
	}
	if names != nil {
		if _, found := names[variable.Name]; found {
			return true
		}
	}
	return false
}

// StructFieldLogSkipped — поле DTO исключено из логов (log-skip или полный dumper:hide).
func StructFieldLogSkipped(field *StructField) (skipped bool) {

	if field == nil {
		return false
	}
	if field.Annotations.IsSet(TagLogSkip) {
		return true
	}
	if DumperHidesFully(structTagFirst(field.Tags[dumperTagName])) {
		return true
	}
	return dumperHidesFullyFromPipe(field.Annotations.Value(TagParamTags, ""))
}

// TypeRefNeedsLogProjection — значение typeRef нужно проецировать перед логом.
func TypeRefNeedsLogProjection(project *Project, typeRef *TypeRef) (needed bool) {

	return typeRefNeedsLogProjection(project, typeRef, make(map[string]bool))
}

// TypeNeedsLogProjection — именованный тип содержит skip-поля (прямо или вложенно).
func TypeNeedsLogProjection(project *Project, typeID string) (needed bool) {

	return typeNeedsLogProjection(project, typeID, make(map[string]bool))
}

// DumperHidesFully — значение тега dumper означает полный omit (hide / hide,-).
func DumperHidesFully(dumperValue string) (full bool) {

	dumperValue = strings.TrimSpace(dumperValue)
	if dumperValue == "" {
		return false
	}
	parts := strings.SplitN(dumperValue, ",", 2)
	if strings.TrimSpace(parts[0]) != dumperHideWord {
		return false
	}
	if len(parts) == 1 {
		return true
	}
	formula := strings.TrimSpace(parts[1])
	return formula == "" || formula == "-"
}

func dumperHidesFullyFromPipe(tagsValue string) (full bool) {

	for item := range strings.SplitSeq(tagsValue, "|") {
		tokens := strings.SplitN(strings.TrimSpace(item), ":", 2)
		if len(tokens) < 2 {
			continue
		}
		if strings.TrimSpace(tokens[0]) != dumperTagName {
			continue
		}
		if DumperHidesFully(strings.TrimSpace(tokens[1])) {
			return true
		}
	}
	return false
}

func structTagFirst(values []string) (value string) {

	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, ",")
}

func typeRefNeedsLogProjection(project *Project, typeRef *TypeRef, visiting map[string]bool) (needed bool) {

	if typeRef == nil {
		return false
	}
	if typeRef.IsSlice || typeRef.ArrayLen > 0 {
		if typeRef.TypeID != "" {
			return typeNeedsLogProjection(project, typeRef.TypeID, visiting)
		}
		if typeRef.MapValue != nil {
			return typeRefNeedsLogProjection(project, typeRef.MapValue, visiting)
		}
		return false
	}
	if typeRef.MapKey != nil || typeRef.MapValue != nil {
		if typeRefNeedsLogProjection(project, typeRef.MapKey, visiting) {
			return true
		}
		return typeRefNeedsLogProjection(project, typeRef.MapValue, visiting)
	}
	if typeRef.ChanOf != nil {
		return typeRefNeedsLogProjection(project, typeRef.ChanOf, visiting)
	}
	return typeNeedsLogProjection(project, typeRef.TypeID, visiting)
}

func typeNeedsLogProjection(project *Project, typeID string, visiting map[string]bool) (needed bool) {

	if project == nil || typeID == "" {
		return false
	}
	typ, found := project.Types[typeID]
	if !found || typ == nil {
		return false
	}
	if visiting[typeID] {
		return false
	}
	visiting[typeID] = true
	defer delete(visiting, typeID)

	switch typ.Kind {
	case TypeKindAlias:
		return typeNeedsLogProjection(project, typ.AliasOf, visiting)
	case TypeKindStruct:
		for _, field := range typ.StructFields {
			if StructFieldLogSkipped(field) {
				return true
			}
			if typeRefNeedsLogProjection(project, &field.TypeRef, visiting) {
				return true
			}
		}
	}
	if typ.IsSlice && typ.ArrayOfID != "" {
		return typeNeedsLogProjection(project, typ.ArrayOfID, visiting)
	}
	if typ.MapValue != nil && typeRefNeedsLogProjection(project, typ.MapValue, visiting) {
		return true
	}
	if typ.MapKey != nil && typeRefNeedsLogProjection(project, typ.MapKey, visiting) {
		return true
	}
	return false
}
