// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"maps"
	"strings"

	"tgp/internal/common"
	"tgp/internal/model"
)

func (r *ClientRenderer) hasMarshaler(typ *model.Type, isArgument bool) (ok bool) {

	if typ == nil {
		return false
	}

	// ВАЖНО: интерфейсы хранятся в формате "pkgPath:InterfaceName" (с двоеточием), а не "pkgPath.InterfaceName"
	if len(typ.ImplementsInterfaces) > 0 {
		for _, iface := range typ.ImplementsInterfaces {
			if isArgument {
				// Для аргументов проверяем Marshaler или TextMarshaler (сериализация при отправке)
				if iface == "encoding/json:Marshaler" || iface == "encoding/text:Marshaler" {
					return true
				}
			} else {
				// Для возвращаемых значений проверяем Unmarshaler или TextUnmarshaler (десериализация при получении)
				if iface == "encoding/json:Unmarshaler" || iface == "encoding/text:Unmarshaler" {
					return true
				}
			}
		}
	}

	// Для алиасов проверяем базовый тип (но не рекурсивно по его содержимому)
	if typ.Kind == model.TypeKindAlias && typ.AliasOf != "" {
		if baseType, exists := r.project.Types[typ.AliasOf]; exists {
			return r.hasMarshaler(baseType, isArgument)
		}
	}

	return false
}

func (r *ClientRenderer) walkVariable(typeName, pkgPath string, variable *model.Variable, varTags map[string]string, isArgument bool) (schema typeDefTS) {
	return r.walkVariableWithVisited(typeName, pkgPath, variable, varTags, make(map[string]bool), isArgument)
}

func (r *ClientRenderer) walkVariableWithVisited(typeName, pkgPath string, variable *model.Variable, varTags map[string]string, processing map[string]bool, isArgument bool) (schema typeDefTS) {
	return r.walkTypeRefWithVisited(typeName, pkgPath, &variable.TypeRef, varTags, processing, isArgument)
}

func (r *ClientRenderer) walkTypeRefWithVisited(typeName, pkgPath string, typeRef *model.TypeRef, varTags map[string]string, processing map[string]bool, isArgument bool) (schema typeDefTS) {

	if typeRef == nil {
		return
	}

	schema.name = typeName
	schema.properties = make(map[string]typeDefTS)
	if varTags != nil && annotationIsSet(varTags, "nullable") {
		schema.nullable = annotationValue(varTags, "nullable", "") == "true"
	}

	if typeRef.IsSlice || typeRef.ArrayLen > 0 {
		schema.kind = kindArray
		schema.typeName = kindArray
		schema.nullable = true
		if typeRef.TypeID != "" {
			itemTypeRef := &model.TypeRef{TypeID: typeRef.TypeID}
			schema.properties["item"] = r.walkTypeRefWithVisited("item", pkgPath, itemTypeRef, nil, processing, isArgument)
		}
		return
	}

	if typeRef.MapKey != nil && typeRef.MapValue != nil {
		if typeRef.TypeID != "" {
			if typ, ok := r.project.Types[typeRef.TypeID]; ok {
				if typ.Kind == model.TypeKindMap && typ.TypeName != "" && typ.ImportPkgPath != "" {
					if r.isTypeFromCurrentProject(typ.ImportPkgPath) {
						schema.kind = kindScalar
						schema.typeName = typ.TypeName
						schema.name = typ.TypeName
						if typ.PkgName != "" {
							schema.importPkg = typ.PkgName
						} else if typ.ImportAlias != "" {
							schema.importPkg = typ.ImportAlias
						}
						schema.importName = typ.TypeName
						if schema.importPkg != "" && schema.importName != "" {
							typeKey := fmt.Sprintf("%s:%s", schema.importPkg, schema.importName)
							if existingDef, exists := r.typeDefTS[typeKey]; !exists || len(existingDef.properties) == 0 {
								mapDef := typeDefTS{
									kind:       kindMap,
									typeName:   kindMap,
									name:       typ.TypeName,
									importPkg:  schema.importPkg,
									importName: schema.importName,
									properties: map[string]typeDefTS{
										"key":   r.walkTypeRefWithVisited("key", pkgPath, typ.MapKey, nil, processing, isArgument),
										"value": r.walkTypeRefWithVisited("value", pkgPath, typ.MapValue, nil, processing, isArgument),
									},
								}
								r.typeDefTS[typeKey] = mapDef
							}
						}
						return schema
					}
				}
			}
		}
		schema.kind = kindMap
		schema.typeName = kindMap
		schema.nullable = true
		schema.properties["key"] = r.walkTypeRefWithVisited("key", pkgPath, typeRef.MapKey, nil, processing, isArgument)
		schema.properties["value"] = r.walkTypeRefWithVisited("value", pkgPath, typeRef.MapValue, nil, processing, isArgument)
		return
	}

	typeID := typeRef.TypeID
	typ, ok := r.project.Types[typeID]
	if !ok {
		if typeID == TypeIDIOReader || typeID == TypeIDIOReadCloser {
			schema.kind = kindScalar
			schema.typeName = "Blob"
			return
		}
		if tsType, okBuiltin := r.goBuiltinTSType(typeID, nil); okBuiltin {
			schema.kind = kindScalar
			schema.typeName = tsType
			return
		}
		if r.isExcludedTypeID(typeID) {
			typeStr := r.typeIDToTSType(typeID)
			schema.kind = kindScalar
			schema.typeName = typeStr
			return
		}
		typeStr := r.typeIDToTSType(typeID)
		schema.kind = kindScalar
		schema.typeName = typeStr
		return
	}

	if typ.Kind == model.TypeKindAlias && typ.AliasOf != "" && typ.TypeName != "" && r.isTypeFromCurrentProject(typ.ImportPkgPath) {
		return r.walkNamedAliasType(schema, typeName, pkgPath, typ, typeRef, varTags, processing, isArgument)
	}

	if len(typ.Enums) > 0 && typ.TypeName != "" {
		return r.walkNamedEnumType(schema, typ, typeRef)
	}

	if tsType, okBuiltin := r.goBuiltinTSType(typeID, typ); okBuiltin {
		schema.kind = kindScalar
		schema.typeName = tsType
		return
	}
	// ВАЖНО: если TypeID указывает на именованный map тип из текущего проекта,
	// используем ссылку на тип, даже если MapKey и MapValue пустые
	if ok && typ.Kind == model.TypeKindMap && typ.TypeName != "" && typ.ImportPkgPath != "" {
		if r.isTypeFromCurrentProject(typ.ImportPkgPath) {
			// Это именованный map тип из текущего проекта - используем ссылку на тип
			schema.kind = kindScalar
			schema.typeName = typ.TypeName
			schema.name = typ.TypeName
			if typ.PkgName != "" {
				schema.importPkg = typ.PkgName
			} else if typ.ImportAlias != "" {
				schema.importPkg = typ.ImportAlias
			}
			schema.importName = typ.TypeName
			if schema.importPkg != "" && schema.importName != "" {
				typeKey := fmt.Sprintf("%s:%s", schema.importPkg, schema.importName)
				if existingDef, exists := r.typeDefTS[typeKey]; !exists || len(existingDef.properties) == 0 {
					var keyTypeRef, valueTypeRef *model.TypeRef
					if typ.MapKey != nil {
						keyTypeRef = typ.MapKey
					} else {
						keyTypeRef = &model.TypeRef{TypeID: typeString}
					}
					if typ.MapValue != nil {
						valueTypeRef = typ.MapValue
					} else {
						valueTypeRef = &model.TypeRef{TypeID: typeAny}
					}
					mapDef := typeDefTS{
						kind:       kindMap,
						typeName:   kindMap,
						name:       typ.TypeName,
						importPkg:  schema.importPkg,
						importName: schema.importName,
						properties: map[string]typeDefTS{
							"key":   r.walkTypeRefWithVisited("key", pkgPath, keyTypeRef, nil, processing, isArgument),
							"value": r.walkTypeRefWithVisited("value", pkgPath, valueTypeRef, nil, processing, isArgument),
						},
					}
					r.typeDefTS[typeKey] = mapDef
				}
			}
			return schema
		}
	}

	hasCustomMarshaler := r.hasMarshaler(typ, isArgument)
	isExcluded := r.isExplicitlyExcludedType(typ)

	// Если тип имеет кастомный маршалер и НЕ является явным исключением,
	// используем any, так как мы не знаем формат сериализации
	// ВАЖНО: не обрабатываем содержимое типа (поля, элементы и т.д.), просто возвращаем any
	// НО сохраняем тип в typeDefTS как алиас на any, чтобы его можно было использовать в других типах
	if hasCustomMarshaler && !isExcluded {
		schema.kind = kindScalar
		schema.typeName = typeAny
		return
	}

	if isExcluded {
		if tsType, okBuiltin := r.goBuiltinTSType(typeID, typ); okBuiltin {
			schema.kind = kindScalar
			schema.typeName = tsType
			return
		}
		typeStr := r.typeIDToTSType(typeID)
		schema.kind = kindScalar
		schema.typeName = typeStr
		return
	}

	if typeRef.NumberOfPointers > 0 {
		schema.nullable = true
	}

	switch typ.Kind {
	case model.TypeKindString, model.TypeKindInt, model.TypeKindInt8, model.TypeKindInt16, model.TypeKindInt32, model.TypeKindInt64,
		model.TypeKindUint, model.TypeKindUint8, model.TypeKindUint16, model.TypeKindUint32, model.TypeKindUint64,
		model.TypeKindFloat32, model.TypeKindFloat64, model.TypeKindBool, model.TypeKindByte, model.TypeKindRune:
		if typ.ImportPkgPath == pkgTime && typ.TypeName == nameTime {
			schema.kind = kindScalar
			schema.typeName = nameDate
			return
		}
		var baseTSType string
		switch typ.Kind {
		case model.TypeKindString:
			baseTSType = typeString
		case model.TypeKindInt, model.TypeKindInt8, model.TypeKindInt16, model.TypeKindInt32, model.TypeKindInt64,
			model.TypeKindUint, model.TypeKindUint8, model.TypeKindUint16, model.TypeKindUint32, model.TypeKindUint64,
			model.TypeKindByte, model.TypeKindRune:
			baseTSType = typeNumber
		case model.TypeKindFloat32, model.TypeKindFloat64:
			baseTSType = typeNumber
		case model.TypeKindBool:
			baseTSType = typeBoolean
		default:
			// Fallback: используем castTypeTS
			baseTSType = castTypeTS(typ.TypeName)
		}
		schema.kind = kindScalar
		schema.typeName = baseTSType
		if typ.TypeName != "" && typ.ImportPkgPath != "" {
			if typ.PkgName != "" {
				schema.importPkg = typ.PkgName
			} else if typ.ImportAlias != "" {
				schema.importPkg = typ.ImportAlias
			}
			schema.importName = typ.TypeName
			var typeKey string
			if schema.importPkg != "" && schema.importName != "" {
				typeKey = fmt.Sprintf("%s:%s", schema.importPkg, schema.importName)
			} else {
				typeKey = typeID
			}
			// Всегда сохраняем типы алиасов из импортированных пакетов
			r.typeDefTS[typeKey] = schema
		}
		return

	case model.TypeKindStruct:
		// Проверка на исключаемые типы уже выполнена выше, здесь обрабатываем только обычные структуры
		if typ.TypeName != "" {
			schema.name = typ.TypeName
		} else {
			schema.name = typeName
		}
		schema.kind = kindStruct
		schema.typeName = kindStruct

		// Защита от циклических зависимостей: проверяем, не обрабатывается ли уже этот тип
		if processing[typeID] {
			// Циклическая зависимость обнаружена - возвращаем ссылку на тип без полей
			schema.properties = make(map[string]typeDefTS)
			return
		}

		// Помечаем тип как обрабатываемый
		processing[typeID] = true
		defer delete(processing, typeID)

		if len(typ.StructFields) > 0 {
			for _, field := range typ.StructFields {
				fieldName, inline := r.jsonName(field)
				if fieldName != "-" {
					fieldTags := parseTagsFromDocs(field.Docs)
					embed := r.walkTypeRefWithVisited(field.Name, typ.ImportPkgPath, &field.TypeRef, fieldTags, processing, isArgument)
					if !inline {
						schema.properties[fieldName] = embed
						continue
					}
					// Inline поля - добавляем их свойства напрямую
					maps.Insert(schema.properties, common.SortedPairs(embed.properties))
				}
			}
		}
		// Если тип импортирован, сохраняем информацию об импорте
		if typ.ImportPkgPath != "" {
			if typ.PkgName != "" {
				schema.importPkg = typ.PkgName
			} else if typ.ImportAlias != "" {
				schema.importPkg = typ.ImportAlias
			}
			if typ.TypeName != "" {
				schema.importName = typ.TypeName
			} else {
				schema.importName = schema.name
			}
		}
		var typeKey string
		if schema.importPkg != "" && schema.importName != "" {
			typeKey = fmt.Sprintf("%s:%s", schema.importPkg, schema.importName)
		} else {
			typeKey = typeID
		}
		// Если тип уже существует, но новый имеет больше полей, заменяем его
		existing, exists := r.typeDefTS[typeKey]
		switch {
		case !exists:
			r.typeDefTS[typeKey] = schema
		case len(schema.properties) > len(existing.properties):
			// Если новый тип имеет больше полей, заменяем старый
			r.typeDefTS[typeKey] = schema
		case len(schema.properties) == 0 && len(existing.properties) > 0:
			// Если новый тип без полей, а старый с полями - не заменяем
			// (это может быть случай, когда структура была обработана до обработки полей)
		default:
			// В остальных случаях заменяем (новый тип может быть более полным)
			r.typeDefTS[typeKey] = schema
		}
		return

	case model.TypeKindArray:
		schema.kind = kindArray
		schema.typeName = kindArray
		schema.nullable = true
		if typ.ArrayOfID != "" {
			itemTypeRef := &model.TypeRef{TypeID: typ.ArrayOfID}
			schema.properties["item"] = r.walkTypeRefWithVisited("item", pkgPath, itemTypeRef, nil, processing, isArgument)
		}

	case model.TypeKindMap:
		// ВАЖНО: если это именованный map тип из текущего проекта, используем имя типа
		if typ.TypeName != "" && typ.ImportPkgPath != "" {
			if r.isTypeFromCurrentProject(typ.ImportPkgPath) {
				// Тип из текущего проекта - генерируем map с правильными типами ключа и значения
				schema.kind = kindMap
				schema.typeName = kindMap
				schema.nullable = true
				var keyTypeRef, valueTypeRef *model.TypeRef
				if typ.MapKey != nil {
					keyTypeRef = typ.MapKey
				} else {
					keyTypeRef = &model.TypeRef{TypeID: typeString}
				}
				if typ.MapValue != nil {
					valueTypeRef = typ.MapValue
				} else {
					valueTypeRef = &model.TypeRef{TypeID: typeAny}
				}
				schema.properties["key"] = r.walkTypeRefWithVisited("key", pkgPath, keyTypeRef, nil, processing, isArgument)
				schema.properties["value"] = r.walkTypeRefWithVisited("value", pkgPath, valueTypeRef, nil, processing, isArgument)
				schema.name = typ.TypeName
				if typ.PkgName != "" {
					schema.importPkg = typ.PkgName
				} else if typ.ImportAlias != "" {
					schema.importPkg = typ.ImportAlias
				}
				schema.importName = typ.TypeName
				if schema.importPkg != "" && schema.importName != "" {
					typeKey := fmt.Sprintf("%s:%s", schema.importPkg, schema.importName)
					r.typeDefTS[typeKey] = schema
				}
				return schema
			}
		}
		// Для неименованных map типов генерируем map напрямую
		schema.kind = kindMap
		schema.typeName = kindMap
		schema.nullable = true
		var keyTypeRef, valueTypeRef *model.TypeRef
		if typ.MapKey != nil {
			keyTypeRef = typ.MapKey
		} else {
			keyTypeRef = &model.TypeRef{TypeID: typeString}
		}
		if typ.MapValue != nil {
			valueTypeRef = typ.MapValue
		} else {
			valueTypeRef = &model.TypeRef{TypeID: typeAny}
		}
		schema.properties["key"] = r.walkTypeRefWithVisited("key", pkgPath, keyTypeRef, nil, processing, isArgument)
		schema.properties["value"] = r.walkTypeRefWithVisited("value", pkgPath, valueTypeRef, nil, processing, isArgument)

	case model.TypeKindInterface, model.TypeKindAny:
		schema.kind = kindScalar
		schema.name = kindInterface
		schema.typeName = typeAny
		schema.nullable = true

	default:
		// Fallback - используем имя типа
		schema.kind = kindScalar
		schema.typeName = typ.TypeName
		if schema.typeName == "" {
			schema.typeName = typeAny
		}
	}

	return
}

func (r *ClientRenderer) walkNamedAliasType(schema typeDefTS, _typeName, pkgPath string, typ *model.Type, typeRef *model.TypeRef, _varTags map[string]string, processing map[string]bool, isArgument bool) (result typeDefTS) {

	if len(typ.Enums) > 0 {
		return r.walkNamedEnumType(schema, typ, typeRef)
	}
	if typeRef.NumberOfPointers > 0 {
		schema.nullable = true
	}
	baseTypeRefForWalk := &model.TypeRef{TypeID: typ.AliasOf}
	baseSchema := r.walkTypeRefWithVisited("base", pkgPath, baseTypeRefForWalk, nil, processing, isArgument)
	var underlyingType string
	if baseSchema.importPkg != "" && baseSchema.importName != "" {
		underlyingType = baseSchema.typeLink()
	} else {
		underlyingType = baseSchema.typeName
		if underlyingType == "" {
			underlyingType = baseSchema.def()
		}
	}
	schema.kind = kindScalar
	schema.typeName = underlyingType
	schema.name = typ.TypeName
	if typ.ImportPkgPath != "" {
		if typ.PkgName != "" {
			schema.importPkg = typ.PkgName
		} else if typ.ImportAlias != "" {
			schema.importPkg = typ.ImportAlias
		}
		schema.importName = typ.TypeName
	}
	if schema.importPkg != "" && schema.importName != "" {
		typeKey := fmt.Sprintf("%s:%s", schema.importPkg, schema.importName)
		r.typeDefTS[typeKey] = schema
	}
	return schema
}

func (r *ClientRenderer) walkNamedEnumType(schema typeDefTS, typ *model.Type, typeRef *model.TypeRef) (result typeDefTS) {

	if typeRef != nil && typeRef.NumberOfPointers > 0 {
		schema.nullable = true
	}
	schema.kind = kindScalar
	schema.typeName = tsEnumUnion(typ.Enums, typ)
	schema.name = typ.TypeName
	switch {
	case typ.PkgName != "":
		schema.importPkg = typ.PkgName
	case typ.ImportAlias != "":
		schema.importPkg = typ.ImportAlias
	case typ.ImportPkgPath != "":
		parts := strings.Split(typ.ImportPkgPath, "/")
		schema.importPkg = parts[len(parts)-1]
	}
	schema.importName = typ.TypeName
	if schema.importPkg != "" && schema.importName != "" {
		typeKey := fmt.Sprintf("%s:%s", schema.importPkg, schema.importName)
		r.typeDefTS[typeKey] = schema
	}
	return schema
}

func tsEnumUnion(enums []*model.EnumValue, typ *model.Type) (union string) {

	parts := make([]string, 0, len(enums))
	numeric := isTSEnumNumeric(typ)
	for _, item := range enums {
		if item == nil {
			continue
		}
		if numeric {
			parts = append(parts, item.Value)
			continue
		}
		parts = append(parts, fmt.Sprintf("%q", item.Value))
	}
	if len(parts) == 0 {
		return typeString
	}
	return strings.Join(parts, " | ")
}

func isTSEnumNumeric(typ *model.Type) (ok bool) {

	if typ == nil {
		return false
	}
	kind := typ.Kind
	if typ.UnderlyingKind != "" {
		kind = typ.UnderlyingKind
	}
	switch kind {
	case model.TypeKindInt, model.TypeKindInt8, model.TypeKindInt16, model.TypeKindInt32, model.TypeKindInt64,
		model.TypeKindUint, model.TypeKindUint8, model.TypeKindUint16, model.TypeKindUint32, model.TypeKindUint64,
		model.TypeKindByte, model.TypeKindRune, model.TypeKindFloat32, model.TypeKindFloat64:
		return true
	default:
		return false
	}
}

func (r *ClientRenderer) typeIDToTSType(typeID string) (s string) {

	// Базовые типы
	switch typeID {
	case typeString, "builtin:string":
		return typeString
	case typeInt, typeInt8, typeInt16, typeInt32, typeInt64, "builtin:int", "builtin:int8", "builtin:int16", "builtin:int32", "builtin:int64":
		return typeNumber
	case typeUint, typeUint8, typeUint16, typeUint32, typeUint64, "builtin:uint", "builtin:uint8", "builtin:uint16", "builtin:uint32", "builtin:uint64":
		return typeNumber
	case typeByte, "builtin:byte":
		return typeNumber
	case "rune", "builtin:rune":
		return typeNumber
	case typeFloat32, typeFloat64, "builtin:float32", "builtin:float64":
		return typeNumber
	case typeBool, "builtin:bool":
		return typeBoolean
	case typeAny, "interface{}", "builtin:any":
		return typeAny
	case typeError, "builtin:error":
		return "Error"
	case "context:Context", "context.Context":
		return typeAny // context не используется в TypeScript
	}

	// Если тип найден в project.Types, используем его имя
	if typ, ok := r.project.Types[typeID]; ok {
		if typ.TypeName != "" {
			return castTypeTS(typ.TypeName)
		}
	}

	// Если typeID содержит ":", извлекаем имя типа
	if strings.Contains(typeID, ":") {
		parts := strings.SplitN(typeID, ":", 2)
		if len(parts) == 2 {
			return castTypeTS(parts[1])
		}
	}

	// По умолчанию используем any
	return typeAny
}

func castTypeTS(originName string) (typeName string) {

	typeName = originName
	switch originName {
	case "JSON":
		typeName = typeAny
	case typeBool:
		typeName = typeBoolean
	case kindInterface:
		typeName = typeAny
	case "gorm.DeletedAt":
		typeName = nameDate
	case "time.Time":
		typeName = nameDate
	case "[]byte":
		typeName = typeString
	case typeFloat32, typeFloat64:
		typeName = typeNumber
	case typeByte, typeInt, typeInt32, typeInt64, typeUint, typeUint8, typeUint16, typeUint32, typeUint64, "time.Duration":
		typeName = typeNumber
	}
	if strings.HasSuffix(originName, "NullTime") {
		typeName = nameDate
	}
	if strings.HasSuffix(originName, "RawMessage") {
		typeName = typeAny
	}
	if strings.HasSuffix(originName, "UUID") {
		typeName = typeString
	}
	if strings.HasSuffix(originName, "Decimal") {
		typeName = typeNumber
	}
	return
}

func annotationIsSet(annotations map[string]string, key string) (ok bool) {

	if annotations == nil {
		return false
	}
	_, ok = annotations[key]
	return
}

func annotationValue(annotations map[string]string, key, defaultValue string) (s string) {

	if annotations == nil {
		return defaultValue
	}
	if value, ok := annotations[key]; ok {
		return value
	}
	return defaultValue
}

func parseTagsFromDocs(docs []string) (out map[string]string) {

	result := make(map[string]string)
	for _, doc := range docs {
		if strings.HasPrefix(doc, "@") {
			parts := strings.Fields(doc)
			if len(parts) >= 2 {
				tagName := strings.TrimPrefix(parts[0], "@")
				result[tagName] = parts[1]
			}
		}
	}
	return result
}
