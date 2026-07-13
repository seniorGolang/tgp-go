// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"strings"

	"tgp/internal/model"
)

var uuidPackages = []string{
	"github.com/google/uuid",
	"github.com/satori/go.uuid",
	"gopkg.in/guregu/null.v4",
}

func (r *ClientRenderer) followAliasChain(typ *model.Type) (base *model.Type) {

	if typ == nil {
		return nil
	}
	visited := make(map[string]bool)
	current := typ
	for current != nil {
		if current.Kind != model.TypeKindAlias || current.AliasOf == "" {
			return current
		}
		if visited[current.AliasOf] {
			return current
		}
		visited[current.AliasOf] = true
		next, exists := r.project.Types[current.AliasOf]
		if !exists {
			return current
		}
		current = next
	}
	return current
}

func (r *ClientRenderer) isExplicitlyExcludedType(typ *model.Type) (excluded bool) {

	if typ == nil {
		return false
	}
	if r.isExplicitlyExcludedBaseType(typ) {
		return true
	}
	if typ.Kind == model.TypeKindAlias && typ.AliasOf != "" {
		if base, exists := r.project.Types[typ.AliasOf]; exists {
			return r.isExplicitlyExcludedType(base)
		}
	}
	return false
}

func (r *ClientRenderer) isExplicitlyExcludedBaseType(typ *model.Type) (excluded bool) {

	if typ.ImportPkgPath == "time" && typ.TypeName == "Time" {
		return true
	}
	if typ.ImportPkgPath == "" && typ.TypeName == "Time" {
		return true
	}
	if typ.ImportPkgPath == "time" && typ.TypeName == "Duration" {
		return true
	}
	if strings.HasSuffix(typ.TypeName, "UUID") || typ.TypeName == "UUID" {
		if typ.ImportPkgPath == "" {
			return true
		}
		for _, pkg := range uuidPackages {
			if strings.HasPrefix(typ.ImportPkgPath, pkg) && typ.TypeName != "Time" {
				return true
			}
		}
		if strings.Contains(typ.ImportPkgPath, "uuid") && typ.TypeName != "Time" {
			return true
		}
	}
	if strings.HasSuffix(typ.TypeName, "Decimal") {
		return true
	}
	if typ.ImportPkgPath == "math/big" {
		if typ.TypeName == "Int" || typ.TypeName == "Float" || typ.TypeName == "Rat" {
			return true
		}
	}
	if typ.ImportPkgPath == "database/sql" {
		if strings.HasPrefix(typ.TypeName, "Null") {
			return true
		}
	}
	if strings.Contains(typ.ImportPkgPath, "guregu/null") {
		return true
	}
	return false
}

func (r *ClientRenderer) isExcludedTypeID(typeID string) (excluded bool) {

	if typeID == "" {
		return false
	}
	if r.isBuiltinType(typeID) {
		return true
	}
	if typ, exists := r.project.Types[typeID]; exists {
		return r.isExplicitlyExcludedType(typ)
	}
	colon := strings.LastIndex(typeID, ":")
	if colon <= 0 {
		return false
	}
	importPkgPath := typeID[:colon]
	typeName := typeID[colon+1:]
	if importPkgPath == "time" && (typeName == "Time" || typeName == "Duration") {
		return true
	}
	if strings.HasSuffix(typeName, "UUID") || typeName == "UUID" {
		return true
	}
	if strings.HasSuffix(typeName, "Decimal") {
		return true
	}
	if importPkgPath == "math/big" {
		if typeName == "Int" || typeName == "Float" || typeName == "Rat" {
			return true
		}
	}
	if importPkgPath == "database/sql" && strings.HasPrefix(typeName, "Null") {
		return true
	}
	if strings.Contains(importPkgPath, "guregu/null") {
		return true
	}
	if _, okBuiltin := lookupBuiltinTS("/"+lastSegment(importPkgPath), typeName); okBuiltin {
		return true
	}
	return false
}
