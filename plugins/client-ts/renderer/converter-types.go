// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"strings"

	"tgp/internal/common"
	"tgp/internal/model"
)

func (r *ClientRenderer) hasMarshaler(typ *model.Type, isArgument bool) bool {
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

func (r *ClientRenderer) walkVariable(typeName, pkgPath string, variable *model.Variable, varTags map[string]string, isArgument bool) (schema typeDefTs) {
	return r.walkVariableWithVisited(typeName, pkgPath, variable, varTags, make(map[string]bool), isArgument)
}

func (r *ClientRenderer) walkVariableWithVisited(typeName, pkgPath string, variable *model.Variable, varTags map[string]string, processing map[string]bool, isArgument bool) (schema typeDefTs) {
	return r.walkTypeRefWithVisited(typeName, pkgPath, &variable.TypeRef, varTags, processing, isArgument)
}

func (r *ClientRenderer) walkTypeRefWithVisited(typeName, pkgPath string, typeRef *model.TypeRef, varTags map[string]string, processing map[string]bool, isArgument bool) (schema typeDefTs) {
	if typeRef == nil {
		return
	}

	schema.name = typeName
	schema.properties = make(map[string]typeDefTs)
	if varTags != nil && annotationIsSet(varTags, "nullable") {
		schema.nullable = annotationValue(varTags, "nullable", "") == "true"
	}

	if typeRef.IsSlice || typeRef.ArrayLen > 0 {
		schema.kind = "array"
		schema.typeName = "array"
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
						schema.kind = "scalar"
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
							if existingDef, exists := r.typeDefTs[typeKey]; !exists || len(existingDef.properties) == 0 {
								mapDef := typeDefTs{
									kind:       "map",
									typeName:   "map",
									name:       typ.TypeName,
									importPkg:  schema.importPkg,
									importName: schema.importName,
									properties: map[string]typeDefTs{
										"key":   r.walkTypeRefWithVisited("key", pkgPath, typ.MapKey, nil, processing, isArgument),
										"value": r.walkTypeRefWithVisited("value", pkgPath, typ.MapValue, nil, processing, isArgument),
									},
								}
								r.typeDefTs[typeKey] = mapDef
							}
						}
						return schema
					}
				}
			}
		}
		schema.kind = "map"
		schema.typeName = "map"
		schema.nullable = true
		schema.properties["key"] = r.walkTypeRefWithVisited("key", pkgPath, typeRef.MapKey, nil, processing, isArgument)
		schema.properties["value"] = r.walkTypeRefWithVisited("value", pkgPath, typeRef.MapValue, nil, processing, isArgument)
		return
	}

	typeID := typeRef.TypeID
	typ, ok := r.project.Types[typeID]
	// ВАЖНО: если TypeID указывает на именованный map тип из текущего проекта,
	// используем ссылку на тип, даже если MapKey и MapValue пустые
	if ok && typ.Kind == model.TypeKindMap && typ.TypeName != "" && typ.ImportPkgPath != "" {
		if r.isTypeFromCurrentProject(typ.ImportPkgPath) {
			// Это именованный map тип из текущего проекта - используем ссылку на тип
			schema.kind = "scalar"
			schema.typeName = typ.TypeName
			schema.name = typ.TypeName
			// Сохраняем информацию об импорте
			if typ.PkgName != "" {
				schema.importPkg = typ.PkgName
			} else if typ.ImportAlias != "" {
				schema.importPkg = typ.ImportAlias
			}
			schema.importName = typ.TypeName
			// Сохраняем тип в typeDefTs
			if schema.importPkg != "" && schema.importName != "" {
				typeKey := fmt.Sprintf("%s:%s", schema.importPkg, schema.importName)
				if existingDef, exists := r.typeDefTs[typeKey]; !exists || len(existingDef.properties) == 0 {
					var keyTypeRef, valueTypeRef *model.TypeRef
					if typ.MapKey != nil {
						keyTypeRef = typ.MapKey
					} else {
						keyTypeRef = &model.TypeRef{TypeID: "string"}
					}
					if typ.MapValue != nil {
						valueTypeRef = typ.MapValue
					} else {
						valueTypeRef = &model.TypeRef{TypeID: "any"}
					}
					mapDef := typeDefTs{
						kind:       "map",
						typeName:   "map",
						name:       typ.TypeName,
						importPkg:  schema.importPkg,
						importName: schema.importName,
						properties: map[string]typeDefTs{
							"key":   r.walkTypeRefWithVisited("key", pkgPath, keyTypeRef, nil, processing, isArgument),
							"value": r.walkTypeRefWithVisited("value", pkgPath, valueTypeRef, nil, processing, isArgument),
						},
					}
					r.typeDefTs[typeKey] = mapDef
				}
			}
			return schema
		}
	}
	if !ok {
		// Тип не найден - возможно это встроенный тип или исключаемый тип (time.Time, UUID и т.п.)
		if typeID == TypeIDIOReader || typeID == TypeIDIOReadCloser {
			schema.kind = "scalar"
			schema.typeName = "Blob"
			return
		}
		if strings.Contains(typeID, "time") && strings.Contains(typeID, "Time") {
			schema.kind = "scalar"
			schema.typeName = "Date"
			return
		}
		if strings.Contains(typeID, "UUID") {
			schema.kind = "scalar"
			schema.typeName = "string"
			return
		}
		// Для остальных используем typeIDToTSType
		typeStr := r.typeIDToTSType(typeID)
		schema.kind = "scalar"
		schema.typeName = typeStr
		return
	}

	// ВАЖНО: Проверяем маршалеры ПЕРЕД проверкой исключений
	// Типы с маршалерами должны быть any, независимо от того, являются ли они исключениями
	// (кроме явных исключений, формат которых известен)
	hasCustomMarshaler := r.hasMarshaler(typ, isArgument)
	isExcluded := r.isExplicitlyExcludedType(typ)

	// Если тип имеет кастомный маршалер и НЕ является явным исключением,
	// используем any, так как мы не знаем формат сериализации
	// ВАЖНО: не обрабатываем содержимое типа (поля, элементы и т.д.), просто возвращаем any
	// НО сохраняем тип в typeDefTs как алиас на any, чтобы его можно было использовать в других типах
	if hasCustomMarshaler && !isExcluded {
		schema.kind = "scalar"
		schema.typeName = "any"
		// Сохраняем информацию об импорте для создания type alias
		if typ.ImportPkgPath != "" {
			if typ.PkgName != "" {
				schema.importPkg = typ.PkgName
			} else if typ.ImportAlias != "" {
				schema.importPkg = typ.ImportAlias
			}
			if typ.TypeName != "" {
				schema.importName = typ.TypeName
			}
		}
		// Сохраняем тип в typeDefTs как алиас на any
		if schema.importPkg != "" && schema.importName != "" {
			typeKey := fmt.Sprintf("%s:%s", schema.importPkg, schema.importName)
			r.typeDefTs[typeKey] = schema
		}
		return
	}

	if isExcluded {
		if typ.ImportPkgPath == "time" && typ.TypeName == "Time" {
			schema.kind = "scalar"
			schema.typeName = "Date"
			return
		}
		if strings.Contains(typeID, "time") && strings.Contains(typeID, "Time") {
			schema.kind = "scalar"
			schema.typeName = "Date"
			return
		}
		if strings.Contains(typeID, "UUID") || strings.HasSuffix(typ.TypeName, "UUID") {
			schema.kind = "scalar"
			schema.typeName = "string"
			return
		}
		typeStr := r.typeIDToTSType(typeID)
		schema.kind = "scalar"
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
		if typ.ImportPkgPath == "time" && typ.TypeName == "Time" {
			schema.kind = "scalar"
			schema.typeName = "Date"
			return
		}
		var baseTSType string
		switch typ.Kind {
		case model.TypeKindString:
			baseTSType = "string"
		case model.TypeKindInt, model.TypeKindInt8, model.TypeKindInt16, model.TypeKindInt32, model.TypeKindInt64,
			model.TypeKindUint, model.TypeKindUint8, model.TypeKindUint16, model.TypeKindUint32, model.TypeKindUint64,
			model.TypeKindByte, model.TypeKindRune:
			baseTSType = "number"
		case model.TypeKindFloat32, model.TypeKindFloat64:
			baseTSType = "number"
		case model.TypeKindBool:
			baseTSType = "boolean"
		default:
			// Fallback: используем castTypeTs
			baseTSType = castTypeTs(typ.TypeName)
		}
		schema.kind = "scalar"
		schema.typeName = baseTSType
		// Если тип импортирован (именованный тип или алиас), сохраняем информацию об импорте
		// ВАЖНО: сохраняем типы алиасов даже если они используются только в полях структур
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
			r.typeDefTs[typeKey] = schema
		}
		return

	case model.TypeKindStruct:
		// Проверка на исключаемые типы уже выполнена выше, здесь обрабатываем только обычные структуры
		if typ.TypeName != "" {
			schema.name = typ.TypeName
		} else {
			schema.name = typeName
		}
		schema.kind = "struct"
		schema.typeName = "struct"

		// Защита от циклических зависимостей: проверяем, не обрабатывается ли уже этот тип
		if processing[typeID] {
			// Циклическая зависимость обнаружена - возвращаем ссылку на тип без полей
			schema.properties = make(map[string]typeDefTs)
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
					for eField, def := range common.SortedPairs(embed.properties) {
						schema.properties[eField] = def
					}
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
		// Сохраняем структуру в typeDefTs
		// Если тип уже существует, но новый имеет больше полей, заменяем его
		existing, exists := r.typeDefTs[typeKey]
		switch {
		case !exists:
			r.typeDefTs[typeKey] = schema
		case len(schema.properties) > len(existing.properties):
			// Если новый тип имеет больше полей, заменяем старый
			r.typeDefTs[typeKey] = schema
		case len(schema.properties) == 0 && len(existing.properties) > 0:
			// Если новый тип без полей, а старый с полями - не заменяем
			// (это может быть случай, когда структура была обработана до обработки полей)
		default:
			// В остальных случаях заменяем (новый тип может быть более полным)
			r.typeDefTs[typeKey] = schema
		}
		// Возвращаем схему для использования в typeLink()
		return

	case model.TypeKindAlias:
		// Для алиасов создаем type alias, который ссылается на базовый тип
		// ВАЖНО: не разворачиваем базовый тип, а сохраняем алиас как ссылку
		if typ.AliasOf != "" {
			baseType, baseTypeExists := r.project.Types[typ.AliasOf]
			if !baseTypeExists {
				aliasTypeRef := &model.TypeRef{TypeID: typ.AliasOf}
				return r.walkTypeRefWithVisited(typeName, pkgPath, aliasTypeRef, varTags, processing, isArgument)
			}

			baseTypeRefForWalk := &model.TypeRef{TypeID: typ.AliasOf}
			baseSchema := r.walkTypeRefWithVisited("base", pkgPath, baseTypeRefForWalk, nil, processing, isArgument)

			var baseTypeRefName string
			switch {
			case baseSchema.importPkg != "" && baseSchema.importName != "":
				baseTypeRefName = fmt.Sprintf("%s.%s", baseSchema.importPkg, baseSchema.importName)
			case baseType.ImportPkgPath != "":
				switch {
				case baseType.PkgName != "":
					baseTypeRefName = fmt.Sprintf("%s.%s", baseType.PkgName, baseType.TypeName)
				case baseType.ImportAlias != "":
					baseTypeRefName = fmt.Sprintf("%s.%s", baseType.ImportAlias, baseType.TypeName)
				default:
					baseTypeRefName = baseType.TypeName
				}
			default:
				baseTypeRefName = baseSchema.typeLink()
			}

			// Создаем схему алиаса
			schema.kind = "scalar"
			schema.typeName = baseTypeRefName
			schema.name = typ.TypeName

			// Сохраняем информацию об импорте алиаса
			if typ.ImportPkgPath != "" {
				if typ.PkgName != "" {
					schema.importPkg = typ.PkgName
				} else if typ.ImportAlias != "" {
					schema.importPkg = typ.ImportAlias
				}
				schema.importName = typ.TypeName
			}

			// Сохраняем алиас в typeDefTs
			if schema.importPkg != "" && schema.importName != "" {
				typeKey := fmt.Sprintf("%s:%s", schema.importPkg, schema.importName)
				r.typeDefTs[typeKey] = schema
			}

			return schema
		}

	case model.TypeKindArray:
		schema.kind = "array"
		schema.typeName = "array"
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
				// и сохраняем как именованный тип
				schema.kind = "map"
				schema.typeName = "map"
				schema.nullable = true
				var keyTypeRef, valueTypeRef *model.TypeRef
				if typ.MapKey != nil {
					keyTypeRef = typ.MapKey
				} else {
					keyTypeRef = &model.TypeRef{TypeID: "string"}
				}
				if typ.MapValue != nil {
					valueTypeRef = typ.MapValue
				} else {
					valueTypeRef = &model.TypeRef{TypeID: "any"}
				}
				schema.properties["key"] = r.walkTypeRefWithVisited("key", pkgPath, keyTypeRef, nil, processing, isArgument)
				schema.properties["value"] = r.walkTypeRefWithVisited("value", pkgPath, valueTypeRef, nil, processing, isArgument)
				// Сохраняем имя типа для использования в typeLink
				schema.name = typ.TypeName
				// Сохраняем информацию об импорте
				if typ.PkgName != "" {
					schema.importPkg = typ.PkgName
				} else if typ.ImportAlias != "" {
					schema.importPkg = typ.ImportAlias
				}
				schema.importName = typ.TypeName
				// Сохраняем тип в typeDefTs
				if schema.importPkg != "" && schema.importName != "" {
					typeKey := fmt.Sprintf("%s:%s", schema.importPkg, schema.importName)
					r.typeDefTs[typeKey] = schema
				}
				return schema
			}
		}
		// Для неименованных map типов генерируем map напрямую
		schema.kind = "map"
		schema.typeName = "map"
		schema.nullable = true
		var keyTypeRef, valueTypeRef *model.TypeRef
		if typ.MapKey != nil {
			keyTypeRef = typ.MapKey
		} else {
			keyTypeRef = &model.TypeRef{TypeID: "string"}
		}
		if typ.MapValue != nil {
			valueTypeRef = typ.MapValue
		} else {
			valueTypeRef = &model.TypeRef{TypeID: "any"}
		}
		schema.properties["key"] = r.walkTypeRefWithVisited("key", pkgPath, keyTypeRef, nil, processing, isArgument)
		schema.properties["value"] = r.walkTypeRefWithVisited("value", pkgPath, valueTypeRef, nil, processing, isArgument)

	case model.TypeKindInterface, model.TypeKindAny:
		schema.kind = "scalar"
		schema.name = "interface"
		schema.typeName = "any"
		schema.nullable = true

	default:
		// Fallback - используем имя типа
		schema.kind = "scalar"
		schema.typeName = typ.TypeName
		if schema.typeName == "" {
			schema.typeName = "any"
		}
	}

	return
}

func (r *ClientRenderer) jsonName(field *model.StructField) (value string, inline bool) {
	if field.Name == "" {
		return field.Name, false
	}
	value = field.Name
	if tagValues, ok := field.Tags["json"]; ok && len(tagValues) > 0 {
		value = tagValues[0]
		if len(tagValues) == 2 {
			inline = tagValues[1] == "inline"
		}
	}
	// Если значение из тега равно "-", пропускаем поле
	if value == "-" {
		return value, false
	}
	// НО только если значение не было взято из тега json
	if len(value) > 0 && value[0] >= 'a' && value[0] <= 'z' && !inline {
		// Если значение было взято из тега, не пропускаем (тег явно указан)
		if _, ok := field.Tags["json"]; !ok {
			value = "-"
		}
	}
	return
}

func (r *ClientRenderer) typeIDToTSType(typeID string) string {
	// Базовые типы
	switch typeID {
	case "string", "builtin:string":
		return "string"
	case "int", "int8", "int16", "int32", "int64", "builtin:int", "builtin:int8", "builtin:int16", "builtin:int32", "builtin:int64":
		return "number"
	case "uint", "uint8", "uint16", "uint32", "uint64", "builtin:uint", "builtin:uint8", "builtin:uint16", "builtin:uint32", "builtin:uint64":
		return "number"
	case "byte", "builtin:byte":
		return "number"
	case "rune", "builtin:rune":
		return "number"
	case "float32", "float64", "builtin:float32", "builtin:float64":
		return "number"
	case "bool", "builtin:bool":
		return "boolean"
	case "any", "interface{}", "builtin:any":
		return "any"
	case "error", "builtin:error":
		return "Error"
	case "context:Context", "context.Context":
		return "any" // context не используется в TypeScript
	}

	// Если тип найден в project.Types, используем его имя
	if typ, ok := r.project.Types[typeID]; ok {
		if typ.TypeName != "" {
			return castTypeTs(typ.TypeName)
		}
	}

	// Если typeID содержит ":", извлекаем имя типа
	if strings.Contains(typeID, ":") {
		parts := strings.SplitN(typeID, ":", 2)
		if len(parts) == 2 {
			return castTypeTs(parts[1])
		}
	}

	// По умолчанию используем any
	return "any"
}

func castTypeTs(originName string) (typeName string) {
	typeName = originName
	switch originName {
	case "JSON":
		typeName = "any"
	case "bool":
		typeName = "boolean"
	case "interface":
		typeName = "any"
	case "gorm.DeletedAt":
		typeName = "Date"
	case "time.Time":
		typeName = "Date"
	case "[]byte":
		typeName = "string"
	case "float32", "float64":
		typeName = "number"
	case "byte", "int", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "time.Duration":
		typeName = "number"
	}
	if strings.HasSuffix(originName, "NullTime") {
		typeName = "Date"
	}
	if strings.HasSuffix(originName, "RawMessage") {
		typeName = "any"
	}
	if strings.HasSuffix(originName, "UUID") {
		typeName = "string"
	}
	if strings.HasSuffix(originName, "Decimal") {
		typeName = "number"
	}
	return
}

func annotationIsSet(annotations map[string]string, key string) bool {
	if annotations == nil {
		return false
	}
	_, ok := annotations[key]
	return ok
}

func annotationValue(annotations map[string]string, key, defaultValue string) string {
	if annotations == nil {
		return defaultValue
	}
	if value, ok := annotations[key]; ok {
		return value
	}
	return defaultValue
}

func parseTagsFromDocs(docs []string) map[string]string {
	result := make(map[string]string)
	for _, doc := range docs {
		// Простая реализация - ищем теги в формате @tag value
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
