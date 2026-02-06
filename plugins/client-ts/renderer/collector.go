// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"strings"

	"tgp/internal/model"
)

func (r *ClientRenderer) CollectTypeIDsForExchange(contract *model.Contract) map[string]bool {

	collectedTypeIDs := make(map[string]bool)
	processedTypes := make(map[string]bool)

	for _, method := range contract.Methods {
		for _, arg := range r.argsWithoutContext(method) {
			r.collectTypeIDFromVariable(arg, collectedTypeIDs, processedTypes)
		}

		for _, result := range r.resultsWithoutError(method) {
			r.collectTypeIDFromVariable(result, collectedTypeIDs, processedTypes)
		}

		// Ошибки могут быть указаны через @tg 400=, @tg 401= и т.д., а также через defaultError
		for _, errInfo := range method.Errors {
			if errInfo.TypeID != "" {
				r.collectTypeIDRecursive(errInfo.TypeID, collectedTypeIDs, processedTypes)
			}
		}

		if defaultError := model.GetAnnotationValue(r.project, contract, method, nil, "defaultError", ""); defaultError != "" && defaultError != "skip" {
			// Парсим формат "pkgPath:TypeName"
			if tokens := strings.Split(defaultError, ":"); len(tokens) == 2 {
				pkgPath := tokens[0]
				typeName := tokens[1]
				typeID := fmt.Sprintf("%s:%s", pkgPath, typeName)
				r.collectTypeIDRecursive(typeID, collectedTypeIDs, processedTypes)
			}
		}
	}

	return collectedTypeIDs
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

	// ВАЖНО: для TS НЕ пропускаем исключаемые типы на этапе сбора,
	// так как они должны быть обработаны при генерации (например, time.Time -> Date)

	typ, ok := r.project.Types[typeID]
	if !ok {
		// Тип не найден - возможно это встроенный тип или исключаемый тип
		// Но для TS мы все равно можем его использовать (например, time.Time -> Date)
		return
	}

	// ВАЖНО: для TS добавляем ВСЕ типы, включая внешние либы
	// При генерации будем проверять, нужно ли генерировать локально или использовать через namespace
	collectedTypeIDs[typeID] = true

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

func (r *ClientRenderer) isBuiltinType(typeID string) bool {
	builtinTypes := map[string]bool{
		"string":  true,
		"int":     true,
		"int8":    true,
		"int16":   true,
		"int32":   true,
		"int64":   true,
		"uint":    true,
		"uint8":   true,
		"uint16":  true,
		"uint32":  true,
		"uint64":  true,
		"float32": true,
		"float64": true,
		"bool":    true,
		"byte":    true,
		"rune":    true,
		"error":   true,
		"any":     true,
	}
	return builtinTypes[typeID]
}

func (r *ClientRenderer) isExplicitlyExcludedType(typ *model.Type) bool {
	if typ == nil {
		return false
	}

	// time.Time - преобразуется в Date
	if typ.ImportPkgPath == "time" && typ.TypeName == "Time" {
		return false // НЕ исключаем, преобразуем
	}
	if typ.ImportPkgPath == "" && typ.TypeName == "Time" {
		return false // НЕ исключаем, преобразуем
	}

	// time.Duration - преобразуется в number
	if typ.ImportPkgPath == "time" && typ.TypeName == "Duration" {
		return false // НЕ исключаем, преобразуем
	}

	// UUID типы - преобразуются в string
	if strings.HasSuffix(typ.TypeName, "UUID") || typ.TypeName == "UUID" {
		return false // НЕ исключаем, преобразуем
	}

	// Decimal типы - преобразуются в number
	if strings.HasSuffix(typ.TypeName, "Decimal") {
		return false // НЕ исключаем, преобразуем
	}

	// big.Int, big.Float, big.Rat - преобразуются в number
	if typ.ImportPkgPath == "math/big" {
		if typ.TypeName == "Int" || typ.TypeName == "Float" || typ.TypeName == "Rat" {
			return false // НЕ исключаем, преобразуем
		}
	}

	// sql.Null типы - преобразуются в соответствующие типы
	if typ.ImportPkgPath == "database/sql" {
		if strings.HasPrefix(typ.TypeName, "Null") {
			return false // НЕ исключаем, преобразуем
		}
	}

	// guregu/null типы - преобразуются в соответствующие типы
	if strings.Contains(typ.ImportPkgPath, "guregu/null") {
		return false // НЕ исключаем, преобразуем
	}

	return false
}
