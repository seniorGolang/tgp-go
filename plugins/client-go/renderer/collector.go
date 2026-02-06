// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"log/slog"
	"strings"

	"tgp/core/i18n"
	"tgp/internal/model"
)

func (r *ClientRenderer) CollectTypeIDsForExchange(contract *model.Contract) map[string]bool {

	collectedTypeIDs := make(map[string]bool)
	processedTypes := make(map[string]bool)
	r.collectTypeIDsFromContract(contract, collectedTypeIDs, processedTypes)
	return collectedTypeIDs
}

func (r *ClientRenderer) CollectTypeIDsForExchangeFromContracts(contracts []*model.Contract) map[string]bool {

	collectedTypeIDs := make(map[string]bool)
	processedTypes := make(map[string]bool)
	for _, contract := range contracts {
		r.collectTypeIDsFromContract(contract, collectedTypeIDs, processedTypes)
	}
	slog.Debug(i18n.Msg("CollectTypeIDsForExchange: completed"), slog.Int("contracts", len(contracts)), slog.Int("collectedTypeIDs", len(collectedTypeIDs)))
	return collectedTypeIDs
}

func (r *ClientRenderer) collectTypeIDsFromContract(contract *model.Contract, collectedTypeIDs map[string]bool, processedTypes map[string]bool) {

	for _, method := range contract.Methods {
		argsWithoutCtx := r.argsWithoutContext(method)
		for _, arg := range argsWithoutCtx {
			r.collectTypeIDFromVariable(arg, collectedTypeIDs, processedTypes)
		}

		resultsWithoutErr := r.resultsWithoutError(method)
		for _, result := range resultsWithoutErr {
			r.collectTypeIDFromVariable(result, collectedTypeIDs, processedTypes)
		}

		for _, errInfo := range method.Errors {
			if errInfo.TypeID != "" {
				r.collectTypeIDRecursive(errInfo.TypeID, collectedTypeIDs, processedTypes)
			}
		}

		if defaultError := model.GetAnnotationValue(r.project, contract, method, nil, "defaultError", ""); defaultError != "" && defaultError != "skip" {
			if tokens := strings.Split(defaultError, ":"); len(tokens) == 2 {
				typeID := fmt.Sprintf("%s:%s", tokens[0], tokens[1])
				r.collectTypeIDRecursive(typeID, collectedTypeIDs, processedTypes)
			}
		}
	}
}

func (r *ClientRenderer) collectTypeIDFromVariable(variable *model.Variable, collectedTypeIDs map[string]bool, processedTypes map[string]bool) {
	if variable == nil {
		return
	}
	r.collectTypeIDFromTypeRef(&variable.TypeRef, collectedTypeIDs, processedTypes)
}

func (r *ClientRenderer) collectTypeIDFromTypeRef(typeRef *model.TypeRef, collectedTypeIDs map[string]bool, processedTypes map[string]bool) {
	if typeRef == nil {
		return
	}
	if typeRef.TypeID != "" {
		r.collectTypeIDRecursive(typeRef.TypeID, collectedTypeIDs, processedTypes)
	}
	if typeRef.MapKey != nil {
		r.collectTypeIDFromTypeRef(typeRef.MapKey, collectedTypeIDs, processedTypes)
	}
	if typeRef.MapValue != nil {
		r.collectTypeIDFromTypeRef(typeRef.MapValue, collectedTypeIDs, processedTypes)
	}
}

func (r *ClientRenderer) collectTypeIDFromStructField(field *model.StructField, collectedTypeIDs map[string]bool, processedTypes map[string]bool) {
	if field == nil {
		return
	}
	r.collectTypeIDFromTypeRef(&field.TypeRef, collectedTypeIDs, processedTypes)
}

func (r *ClientRenderer) collectTypeIDRecursive(typeID string, collectedTypeIDs map[string]bool, processedTypes map[string]bool) {

	if processedTypes[typeID] {
		return
	}
	processedTypes[typeID] = true

	if r.isBuiltinType(typeID) {
		return
	}

	if r.isExcludedTypeID(typeID) {
		return
	}

	typ, ok := r.project.Types[typeID]
	if !ok {
		// Логируем только первые несколько пропущенных типов
		if len(processedTypes) < 10 {
			// Показываем примеры доступных типов для сравнения
			exampleTypes := make([]string, 0, 5)
			for tID := range r.project.Types {
				if len(exampleTypes) < 5 {
					exampleTypes = append(exampleTypes, tID)
				}
			}
			slog.Debug(i18n.Msg("collectTypeIDRecursive: type not found"),
				slog.String("typeID", typeID),
				slog.Int("totalTypes", len(r.project.Types)),
				slog.Any("exampleTypes", exampleTypes))
		}
		return
	}

	if r.isExplicitlyExcludedType(typ) {
		return
	}

	// При генерации будем проверять, нужно ли генерировать локально
	collectedTypeIDs[typeID] = true
	if len(collectedTypeIDs) <= 10 {
		slog.Debug(i18n.Msg("collectTypeIDRecursive: added typeID"), slog.String("typeID", typeID), slog.String("kind", string(typ.Kind)), slog.String("typeName", typ.TypeName), slog.String("importPkgPath", typ.ImportPkgPath))
	}

	// Рекурсивно обходим зависимые типы (используя уже собранные данные Core)
	switch typ.Kind {
	case model.TypeKindArray:
		if typ.ArrayOfID != "" {
			r.collectTypeIDRecursive(typ.ArrayOfID, collectedTypeIDs, processedTypes)
		}
	case model.TypeKindMap:
		if typ.MapKey != nil {
			r.collectTypeIDFromTypeRef(typ.MapKey, collectedTypeIDs, processedTypes)
		}
		if typ.MapValue != nil {
			r.collectTypeIDFromTypeRef(typ.MapValue, collectedTypeIDs, processedTypes)
		}
	case model.TypeKindAlias:
		if typ.AliasOf != "" {
			r.collectTypeIDRecursive(typ.AliasOf, collectedTypeIDs, processedTypes)
		}
		if typ.UnderlyingTypeID != "" {
			r.collectTypeIDRecursive(typ.UnderlyingTypeID, collectedTypeIDs, processedTypes)
		}
	case model.TypeKindStruct:
		for _, field := range typ.StructFields {
			r.collectTypeIDFromStructField(field, collectedTypeIDs, processedTypes)
		}
	case model.TypeKindInterface:
		for _, embedded := range typ.EmbeddedInterfaces {
			if embedded.TypeID != "" {
				r.collectTypeIDRecursive(embedded.TypeID, collectedTypeIDs, processedTypes)
			}
		}
		for _, method := range typ.InterfaceMethods {
			for _, arg := range method.Args {
				r.collectTypeIDFromVariable(arg, collectedTypeIDs, processedTypes)
			}
			for _, result := range method.Results {
				r.collectTypeIDFromVariable(result, collectedTypeIDs, processedTypes)
			}
		}
	}
}

func (r *ClientRenderer) isExcludedTypeID(typeID string) bool {
	if typeID == "" {
		return false
	}

	if r.isBuiltinType(typeID) {
		return true
	}

	if typ, exists := r.project.Types[typeID]; exists {
		return r.isExcludedType(typ)
	}

	parts := strings.SplitN(typeID, ":", 2)
	if len(parts) != 2 {
		return false
	}

	importPkgPath := parts[0]
	typeName := parts[1]

	// time.Time
	if importPkgPath == "time" && typeName == "Time" {
		return true
	}

	// time.Duration
	if importPkgPath == "time" && typeName == "Duration" {
		return true
	}

	// UUID типы
	if strings.HasSuffix(typeName, "UUID") || typeName == "UUID" {
		if importPkgPath == "" {
			return true
		}
		uuidPackages := []string{
			"github.com/google/uuid",
			"github.com/satori/go.uuid",
			"gopkg.in/guregu/null.v4",
		}
		for _, pkg := range uuidPackages {
			if strings.HasPrefix(importPkgPath, pkg) && typeName != "Time" {
				return true
			}
		}
		if strings.Contains(importPkgPath, "uuid") && typeName != "Time" {
			return true
		}
	}

	// Decimal типы
	if strings.HasSuffix(typeName, "Decimal") {
		return true
	}

	// big.Int, big.Float, big.Rat
	if importPkgPath == "math/big" {
		if typeName == "Int" || typeName == "Float" || typeName == "Rat" {
			return true
		}
	}

	// sql.Null типы
	if importPkgPath == "database/sql" {
		if strings.HasPrefix(typeName, "Null") {
			return true
		}
	}

	// guregu/null типы
	if strings.Contains(importPkgPath, "guregu/null") {
		return true
	}

	return false
}

func (r *ClientRenderer) isExcludedType(typ *model.Type) bool {
	if typ == nil {
		return false
	}

	// Встроенные типы
	if r.isBuiltinType(typ.TypeName) {
		return true
	}

	// Явные исключения
	if r.isExplicitlyExcludedType(typ) {
		return true
	}

	if typ.ImportPkgPath != "" && typ.TypeName != "" {
		for _, iface := range typ.ImplementsInterfaces {
			if iface == "encoding/json.Marshaler" {
				return true
			}
		}
	}

	return false
}

func (r *ClientRenderer) isExplicitlyExcludedType(typ *model.Type) bool {
	if typ == nil {
		return false
	}

	// time.Time
	if typ.ImportPkgPath == "time" && typ.TypeName == "Time" {
		return true
	}
	if typ.ImportPkgPath == "" && typ.TypeName == "Time" {
		return true
	}

	// time.Duration
	if typ.ImportPkgPath == "time" && typ.TypeName == "Duration" {
		return true
	}

	// UUID типы
	if strings.HasSuffix(typ.TypeName, "UUID") || typ.TypeName == "UUID" {
		if typ.ImportPkgPath == "" {
			return true
		}
		uuidPackages := []string{
			"github.com/google/uuid",
			"github.com/satori/go.uuid",
			"gopkg.in/guregu/null.v4",
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

	// Decimal типы
	if strings.HasSuffix(typ.TypeName, "Decimal") {
		return true
	}

	// big.Int, big.Float, big.Rat
	if typ.ImportPkgPath == "math/big" {
		if typ.TypeName == "Int" || typ.TypeName == "Float" || typ.TypeName == "Rat" {
			return true
		}
	}

	// sql.Null типы
	if typ.ImportPkgPath == "database/sql" {
		if strings.HasPrefix(typ.TypeName, "Null") {
			return true
		}
	}

	// guregu/null типы
	if strings.Contains(typ.ImportPkgPath, "guregu/null") {
		return true
	}

	return false
}
