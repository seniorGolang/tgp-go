// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"tgp/internal/model"
)

// fillStructFields заполняет поля структуры из go/types.
func fillStructFields(structType *types.Struct, pkgPath string, imports map[string]string, project *model.Project, coreType *model.Type, loader *AutonomousPackageLoader, processingTypes ...map[string]bool) {

	if structType == nil {
		return
	}

	coreType.StructFields = make([]*model.StructField, 0)

	// Создаем или используем существующий set обрабатываемых типов
	var processingSet map[string]bool
	if len(processingTypes) > 0 && processingTypes[0] != nil {
		processingSet = processingTypes[0]
	} else {
		processingSet = make(map[string]bool)
	}

	// Получаем AST структуру для извлечения тегов
	var astStructType *ast.StructType
	if pkgInfo, ok := loader.GetPackage(pkgPath); ok && pkgInfo != nil {
		// Ищем структуру в файлах пакета
		for _, file := range pkgInfo.Files {
			astStructType = findASTStructType(file, coreType.TypeName, pkgInfo.TypeInfo)
			if astStructType != nil {
				break
			}
		}
	}

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		if field == nil {
			continue
		}

		fieldName := field.Name()
		fieldType := field.Type()

		typeInfo := convertFieldType(fieldType, pkgPath, imports, project, loader, processingSet)

		// Извлекаем теги из go/types.Struct.Tag или из AST
		tags := make(map[string][]string)
		// Сначала пробуем получить из go/types
		if fieldTag := structType.Tag(i); fieldTag != "" {
			// Парсим теги в формате `json:"name,omitempty" xml:"name"`
			parsedTags := parseStructTag(fieldTag)
			tags = parsedTags
		} else if astStructType != nil {
			// Если в go/types нет тегов, пытаемся получить из AST
			astTags := extractTagsFromASTStruct(astStructType, fieldName)
			if len(astTags) > 0 {
				tags = astTags
			}
		}

		// Извлекаем комментарии из AST
		docs := []string{}
		if astStructType != nil {
			astDocs := extractDocsFromASTStruct(astStructType, fieldName)
			if len(astDocs) > 0 {
				docs = astDocs
			}
		}

		structField := &model.StructField{
			Name:             fieldName,
			TypeID:           typeInfo.TypeID,
			NumberOfPointers: typeInfo.NumberOfPointers,
			IsSlice:          typeInfo.IsSlice,
			ArrayLen:         typeInfo.ArrayLen,
			IsEllipsis:       typeInfo.IsEllipsis,
			ElementPointers:  typeInfo.ElementPointers,
			MapKeyID:         typeInfo.MapKeyID,
			MapValueID:       typeInfo.MapValueID,
			MapKeyPointers:   typeInfo.MapKeyPointers,
			Tags:             tags,
			Docs:             docs,
		}

		coreType.StructFields = append(coreType.StructFields, structField)
	}
}

// fieldTypeInfo содержит информацию о типе поля.
type fieldTypeInfo struct {
	TypeID           string
	NumberOfPointers int
	IsSlice          bool
	ArrayLen         int
	IsEllipsis       bool
	ElementPointers  int // Для элементов массивов/слайсов и значений map
	MapKeyID         string
	MapValueID       string
	MapKeyPointers   int
}

// convertFieldType конвертирует тип поля в fieldTypeInfo.
// processingTypes используется для защиты от рекурсии при циклических зависимостях.
func convertFieldType(typ types.Type, pkgPath string, imports map[string]string, project *model.Project, loader *AutonomousPackageLoader, processingTypes ...map[string]bool) (info fieldTypeInfo) {

	// Создаем или используем существующий set обрабатываемых типов
	var processingSet map[string]bool
	if len(processingTypes) > 0 && processingTypes[0] != nil {
		processingSet = processingTypes[0]
	} else {
		processingSet = make(map[string]bool)
	}

	// Убираем указатели
	for {
		if ptr, ok := typ.(*types.Pointer); ok {
			info.NumberOfPointers++
			typ = ptr.Elem()
			continue
		}
		break
	}

	switch t := typ.(type) {
	case *types.Array:
		info.IsSlice = false
		info.ArrayLen = int(t.Len())
		if t.Elem() != nil {
			eltInfo := convertFieldType(t.Elem(), pkgPath, imports, project, loader, processingSet)
			if eltInfo.TypeID != "" && eltInfo.TypeID != "invalid type" {
				info.TypeID = eltInfo.TypeID
				info.ElementPointers = eltInfo.NumberOfPointers
			}
		}

	case *types.Slice:
		info.IsSlice = true
		if t.Elem() != nil {
			eltInfo := convertFieldType(t.Elem(), pkgPath, imports, project, loader, processingSet)
			if eltInfo.TypeID != "" && eltInfo.TypeID != "invalid type" {
				info.TypeID = eltInfo.TypeID
				info.ElementPointers = eltInfo.NumberOfPointers
			} else {
				// Пытаемся получить typeID напрямую через generateTypeIDFromGoTypes
				directTypeID := generateTypeIDFromGoTypes(t.Elem())
				if directTypeID != "" && directTypeID != "invalid type" {
					info.TypeID = directTypeID
				}
			}
		}

	case *types.Map:
		if t.Key() != nil {
			keyInfo := convertFieldType(t.Key(), pkgPath, imports, project, loader, processingSet)
			if keyInfo.TypeID != "" && keyInfo.TypeID != "invalid type" {
				info.MapKeyID = keyInfo.TypeID
				info.MapKeyPointers = keyInfo.NumberOfPointers
			}
		}
		if t.Elem() != nil {
			valueInfo := convertFieldType(t.Elem(), pkgPath, imports, project, loader, processingSet)
			if valueInfo.TypeID != "" && valueInfo.TypeID != "invalid type" {
				info.MapValueID = valueInfo.TypeID
				info.ElementPointers = valueInfo.NumberOfPointers
			}
		}

	default:
		// Генерируем typeID для типа
		typeID := generateTypeIDFromGoTypes(typ)
		//nolint:staticcheck // QF1003: проверка пустой строки более читаема через if
		if typeID == "" {
			if basic, ok := typ.(*types.Basic); ok {
				typeID = basic.Name()
				// Проверяем, не является ли это "invalid type"
				if typeID == "invalid type" {
					typeID = ""
				}
			} else if named, ok := typ.(*types.Named); ok && named.Obj() != nil {
				// Fallback для именованных типов
				typeName := named.Obj().Name()
				if named.Obj().Pkg() != nil {
					importPkgPath := named.Obj().Pkg().Path()
					typeID = fmt.Sprintf("%s:%s", importPkgPath, typeName)
				} else {
					typeID = typeName
				}
			} else if alias, ok := typ.(*types.Alias); ok && alias.Obj() != nil {
				// Fallback для алиасов
				typeName := alias.Obj().Name()
				if alias.Obj().Pkg() != nil {
					importPkgPath := alias.Obj().Pkg().Path()
					typeID = fmt.Sprintf("%s:%s", importPkgPath, typeName)
				} else {
					typeID = typeName
				}
			}
		} else if typeID == "invalid type" {
			// Пытаемся получить typeID через fallback
			if named, ok := typ.(*types.Named); ok && named.Obj() != nil {
				typeName := named.Obj().Name()
				if named.Obj().Pkg() != nil {
					importPkgPath := named.Obj().Pkg().Path()
					typeID = fmt.Sprintf("%s:%s", importPkgPath, typeName)
				} else {
					typeID = typeName
				}
			} else if alias, ok := typ.(*types.Alias); ok && alias.Obj() != nil {
				typeName := alias.Obj().Name()
				if alias.Obj().Pkg() != nil {
					importPkgPath := alias.Obj().Pkg().Path()
					typeID = fmt.Sprintf("%s:%s", importPkgPath, typeName)
				} else {
					typeID = typeName
				}
			} else {
				typeID = ""
			}
		}
		info.TypeID = typeID

		// Обрабатываем именованные типы и алиасы
		if typeID != "" && !isBuiltinTypeName(typeID) {
			// Если тип уже существует в project.Types, просто возвращаем его
			if _, exists := project.Types[typeID]; exists {
				return info
			}

			// Обрабатываем алиасы
			if alias, ok := typ.(*types.Alias); ok {
				underlying := types.Unalias(alias)
				underlyingInfo := convertFieldType(underlying, pkgPath, imports, project, loader, processingSet)
				info = underlyingInfo
				// Генерируем typeID для алиаса
				if alias.Obj() != nil && alias.Obj().Pkg() != nil {
					typeID := fmt.Sprintf("%s:%s", alias.Obj().Pkg().Path(), alias.Obj().Name())
					info.TypeID = typeID
					// Сохраняем алиас в project.Types через convertTypeFromGoTypes
					if _, exists := project.Types[typeID]; !exists {
						pkgInfo, ok := loader.GetPackage(alias.Obj().Pkg().Path())
						if ok && pkgInfo != nil {
							coreType := convertTypeFromGoTypes(typ, alias.Obj().Pkg().Path(), pkgInfo.Imports, project, loader, processingSet)
							if coreType != nil {
								// Определяем интерфейсы для типа
								detectInterfaces(typ, coreType, project, loader)
								// Обновляем тип в project.Types, чтобы сохранить интерфейсы
								project.Types[typeID] = coreType
							}
						}
					}
				}
			} else {
				// Сохраняем тип в project.Types, если это именованный тип
				if named, ok := typ.(*types.Named); ok {
					if named.Obj() != nil && named.Obj().Pkg() != nil {
						importPkgPath := named.Obj().Pkg().Path()
						pkgInfo, ok := loader.GetPackage(importPkgPath)
						if ok && pkgInfo != nil {
							coreType := convertTypeFromGoTypes(typ, importPkgPath, pkgInfo.Imports, project, loader, processingSet)
							if coreType != nil {
								// Определяем интерфейсы для типа
								detectInterfaces(typ, coreType, project, loader)
								// Обновляем тип в project.Types, чтобы сохранить интерфейсы
								project.Types[typeID] = coreType
							}
						} else {
							// Пакет не загружен - пытаемся загрузить
							if pkgInfo, err := loader.LoadPackageForType(importPkgPath, named.Obj().Name()); err == nil {
								coreType := convertTypeFromGoTypes(typ, importPkgPath, pkgInfo.Imports, project, loader, processingSet)
								if coreType != nil {
									detectInterfaces(typ, coreType, project, loader)
									project.Types[typeID] = coreType
								}
							}
						}
					} else if named.Obj() != nil {
						// Тип без пакета - возможно, это встроенный тип или тип из текущего пакета
						// Используем имя типа как typeID, но только если typeID пустой
						if typeID == "" {
							typeID = named.Obj().Name()
							info.TypeID = typeID
						}
					}
				}
			}
		}
	}

	return
}

// findASTStructType находит AST структуру по имени типа.
func findASTStructType(file *ast.File, typeName string, typeInfo *types.Info) (foundStruct *ast.StructType) {

	if file == nil {
		return
	}

	// Ищем объявление типа в AST
	ast.Inspect(file, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok {
			if genDecl.Tok == token.TYPE {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if typeSpec.Name != nil && typeSpec.Name.Name == typeName {
							if structType, ok := typeSpec.Type.(*ast.StructType); ok {
								foundStruct = structType
								return false // Найдено, прекращаем поиск
							}
						}
					}
				}
			}
		}
		return true
	})

	return
}

// extractTagsFromASTStruct извлекает теги полей из AST структуры.
func extractTagsFromASTStruct(structType *ast.StructType, fieldName string) (tags map[string][]string) {

	if structType == nil || structType.Fields == nil {
		tags = make(map[string][]string)
		return
	}

	for _, field := range structType.Fields.List {
		// Проверяем, совпадает ли имя поля
		for _, name := range field.Names {
			if name.Name == fieldName {
				if field.Tag != nil {
					// Убираем обратные кавычки из тега
					tagValue := field.Tag.Value
					if len(tagValue) >= 2 && tagValue[0] == '`' && tagValue[len(tagValue)-1] == '`' {
						tagValue = tagValue[1 : len(tagValue)-1]
					}
					return parseStructTag(tagValue)
				}
			}
		}
	}

	tags = make(map[string][]string)
	return
}

// extractDocsFromASTStruct извлекает комментарии полей из AST структуры.
func extractDocsFromASTStruct(structType *ast.StructType, fieldName string) (docs []string) {

	if structType == nil || structType.Fields == nil {
		return
	}

	for _, field := range structType.Fields.List {
		// Проверяем, совпадает ли имя поля
		for _, name := range field.Names {
			if name.Name == fieldName {
				// Извлекаем комментарии из Doc и Comment
				docs = extractComments(field.Doc, field.Comment)
				return
			}
		}
	}

	return
}

// parseStructTag парсит теги структуры в формате `json:"name,omitempty" xml:"name"`.
// Возвращает map, где ключ - имя тега (например, "json"), значение - массив значений тега.
func parseStructTag(tag string) (result map[string][]string) {

	result = make(map[string][]string)
	if tag == "" {
		return
	}

	// Используем reflect.StructTag для парсинга
	// Но так как мы не можем использовать reflect в core, парсим вручную
	// Формат: `key1:"value1" key2:"value2" key3:"value3,option1,option2"`
	for tag != "" {
		// Пропускаем пробелы
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// Ищем ключ (до первого ':')
		keyEnd := 0
		for keyEnd < len(tag) && tag[keyEnd] != ':' {
			keyEnd++
		}
		if keyEnd == 0 || keyEnd == len(tag) {
			break
		}
		key := tag[:keyEnd]
		tag = tag[keyEnd+1:]

		// Пропускаем пробелы после ':'
		i = 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" || tag[0] != '"' {
			break
		}

		// Ищем значение в кавычках
		tag = tag[1:] // Пропускаем открывающую кавычку
		valueEnd := 0
		for valueEnd < len(tag) && tag[valueEnd] != '"' {
			if tag[valueEnd] == '\\' && valueEnd+1 < len(tag) {
				valueEnd += 2 // Пропускаем экранированный символ
			} else {
				valueEnd++
			}
		}
		if valueEnd == len(tag) {
			break
		}
		value := tag[:valueEnd]
		tag = tag[valueEnd+1:] // Пропускаем закрывающую кавычку

		// Разбиваем значение по запятым (для опций типа "name,omitempty")
		values := strings.Split(value, ",")
		result[key] = values
	}

	return
}
