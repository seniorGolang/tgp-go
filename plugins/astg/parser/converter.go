// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/types"
	"log/slog"

	"tgp/core/i18n"
	"tgp/internal/model"
)

func convertTypeFromGoTypes(typ types.Type, pkgPath string, imports map[string]string, project *model.Project, loader *AutonomousPackageLoader, processingTypes ...map[string]bool) (coreType *model.Type) {

	if typ == nil {
		return
	}

	var processingSet map[string]bool
	if len(processingTypes) > 0 && processingTypes[0] != nil {
		processingSet = processingTypes[0]
	} else {
		processingSet = make(map[string]bool)
	}

	typeID := generateTypeIDFromGoTypes(typ)
	if typeID == "" {
		if basic, ok := typ.(*types.Basic); ok {
			typeID = basic.Name()
		}
	}

	// Если тип уже существует в project.Types, возвращаем его
	if typeID != "" && !isBuiltinTypeName(typeID) {
		var existingType *model.Type
		var exists bool
		if existingType, exists = project.Types[typeID]; exists {
			if len(existingType.ImplementsInterfaces) == 0 && existingType.ImportPkgPath != "" && existingType.TypeName != "" && loader != nil {
				detectInterfaces(typ, existingType, project, loader)
				project.Types[typeID] = existingType
			}
			coreType = existingType
			return
		}
	}

	if typeID != "" && !isBuiltinTypeName(typeID) {
		if processingSet[typeID] {
			var existingType *model.Type
			var exists bool
			if existingType, exists = project.Types[typeID]; exists {
				if len(existingType.ImplementsInterfaces) == 0 && existingType.ImportPkgPath != "" && existingType.TypeName != "" && loader != nil {
					detectInterfaces(typ, existingType, project, loader)
					project.Types[typeID] = existingType
				}
				coreType = existingType
				return
			}
			// Создаем минимальный тип для рекурсивного случая
			coreType = &model.Type{}
			var named *types.Named
			var ok bool
			if named, ok = typ.(*types.Named); ok {
				if named.Obj() != nil {
					coreType.TypeName = named.Obj().Name()
					if named.Obj().Pkg() != nil {
						coreType.ImportPkgPath = named.Obj().Pkg().Path()
						coreType.PkgName = named.Obj().Pkg().Name()
					}
				}
			} else {
				var alias *types.Alias
				if alias, ok = typ.(*types.Alias); ok {
					if alias.Obj() != nil {
						coreType.TypeName = alias.Obj().Name()
						if alias.Obj().Pkg() != nil {
							coreType.ImportPkgPath = alias.Obj().Pkg().Path()
							coreType.PkgName = alias.Obj().Pkg().Name()
						}
					}
				}
			}
			project.Types[typeID] = coreType
			return
		}

		// Помечаем тип как обрабатываемый
		processingSet[typeID] = true
	}

	coreType = &model.Type{}

	// Для именованных типов создаем базовую структуру сразу
	if typeID != "" && !isBuiltinTypeName(typeID) {
		if named, ok := typ.(*types.Named); ok {
			if named.Obj() != nil {
				coreType.TypeName = named.Obj().Name()
				if named.Obj().Pkg() != nil {
					coreType.ImportPkgPath = named.Obj().Pkg().Path()
					coreType.PkgName = named.Obj().Pkg().Name()
				}
			}
		} else if alias, ok := typ.(*types.Alias); ok {
			if alias.Obj() != nil {
				coreType.TypeName = alias.Obj().Name()
				if alias.Obj().Pkg() != nil {
					coreType.ImportPkgPath = alias.Obj().Pkg().Path()
					coreType.PkgName = alias.Obj().Pkg().Name()
				}
			}
		}

		// Создаем тип в project.Types ДО обработки полей - это позволяет правильно обработать рекурсивные типы
		project.Types[typeID] = coreType
	}

	switch t := typ.(type) {
	case *types.Basic:
		coreType.Kind = convertBasicKind(t.Kind())
		coreType.TypeName = t.Name()

	case *types.Named:
		if t.Obj() != nil {
			coreType.TypeName = t.Obj().Name()
			if t.Obj().Pkg() != nil {
				coreType.ImportPkgPath = t.Obj().Pkg().Path()
				coreType.PkgName = t.Obj().Pkg().Name()
				for alias, path := range imports {
					if path == coreType.ImportPkgPath {
						coreType.ImportAlias = alias
						break
					}
				}
			}
		}

		underlying := t.Underlying()

		// Для именованных типов, которые являются массивами/слайсами (например, UUID = [16]byte),
		// сохраняем информацию о типе, а не о массиве
		// Это позволяет правильно использовать тип в генерации кода
		if _, isArray := underlying.(*types.Array); isArray {
			// Именованный тип, который является массивом - сохраняем как именованный тип
			// Kind будет установлен на основе underlying, но ArrayOfID не заполняем
			coreType.Kind = model.TypeKindArray
			// Для массива сохраняем длину и тип элемента
			if arrayType, ok := underlying.(*types.Array); ok {
				coreType.ArrayLen = int(arrayType.Len())
				if arrayType.Elem() != nil {
					elemInfo := convertTypeFromGoTypesToInfo(arrayType.Elem(), coreType.ImportPkgPath, imports, project, loader)
					coreType.ArrayOfID = elemInfo.TypeID
				}
			}
		} else if _, isSlice := underlying.(*types.Slice); isSlice {
			// Именованный тип, который является слайсом - сохраняем как именованный тип
			coreType.Kind = model.TypeKindArray
			coreType.IsSlice = true
			if sliceType, ok := underlying.(*types.Slice); ok {
				if sliceType.Elem() != nil {
					elemInfo := convertTypeFromGoTypesToInfo(sliceType.Elem(), coreType.ImportPkgPath, imports, project, loader)
					coreType.ArrayOfID = elemInfo.TypeID
				}
			}
		} else {
			coreType.Kind = resolveKindFromUnderlying(underlying)

			if structType, ok := underlying.(*types.Struct); ok {
				coreType.Kind = model.TypeKindStruct
				fillStructFields(structType, coreType.ImportPkgPath, imports, project, coreType, loader, processingSet)
			} else if mapType, ok := underlying.(*types.Map); ok {
				if mapType.Key() != nil {
					keyInfo := convertFieldType(mapType.Key(), coreType.ImportPkgPath, imports, project, loader, processingSet)
					if keyInfo.TypeID != "" && keyInfo.TypeID != "invalid type" {
						coreType.MapKey = fieldTypeInfoToTypeRef(keyInfo)
					}
				}
				if mapType.Elem() != nil {
					valueInfo := convertFieldType(mapType.Elem(), coreType.ImportPkgPath, imports, project, loader, processingSet)
					if valueInfo.TypeID != "" && valueInfo.TypeID != "invalid type" {
						coreType.MapValue = fieldTypeInfoToTypeRef(valueInfo)
					}
				}
			}
		}

	case *types.Alias:
		if t.Obj() != nil {
			coreType.TypeName = t.Obj().Name()
			if t.Obj().Pkg() != nil {
				coreType.ImportPkgPath = t.Obj().Pkg().Path()
				coreType.PkgName = t.Obj().Pkg().Name()
				for alias, path := range imports {
					if path == coreType.ImportPkgPath {
						coreType.ImportAlias = alias
						break
					}
				}
			}
		}

		underlying := types.Unalias(t)
		// Устанавливаем Kind на основе underlying типа, но сохраняем информацию об алиасе
		coreType.Kind = resolveKindFromUnderlying(underlying)
		// Если underlying - именованный тип, это алиас
		if _, ok := underlying.(*types.Named); ok {
			coreType.Kind = model.TypeKindAlias
		}

		if named, ok := underlying.(*types.Named); ok {
			if named.Obj() != nil {
				coreType.AliasOf = fmt.Sprintf("%s:%s", named.Obj().Pkg().Path(), named.Obj().Name())
				// Сохраняем базовый тип, если его еще нет
				baseTypeID := coreType.AliasOf
				if _, exists := project.Types[baseTypeID]; !exists {
					basePkgPath := named.Obj().Pkg().Path()
					// basePkgPath должен быть загружен через loader
					// Пока пропускаем, так как loader не передается в эту функцию
					basePkgInfo, ok := loader.GetPackage(basePkgPath)
					if ok && basePkgInfo != nil {
						// Важно: используем тот же processingSet для защиты от рекурсии
						baseCoreType := convertTypeFromGoTypes(named, basePkgPath, basePkgInfo.Imports, project, loader, processingSet)
						if baseCoreType != nil {
							// ВАЖНО: сохраняем базовый тип БЕЗ проверки isExcludedType,
							// так как он нужен для правильной обработки алиаса
							// Проверка isExcludedType применяется только при сохранении через saveTypeFromGoTypes
							project.Types[baseTypeID] = baseCoreType
						} else {
							slog.Debug(i18n.Msg("Failed to convert base type"), slog.String("baseType", baseTypeID))
						}
					} else {
						slog.Debug(i18n.Msg("Package info is nil for base type"), slog.String("baseType", baseTypeID))
					}
				}
				// Это произойдет в expandTypesRecursively через collectTypeFromID
			}
		} else if basic, ok := underlying.(*types.Basic); ok {
			coreType.UnderlyingKind = convertBasicKind(basic.Kind())
			coreType.UnderlyingTypeID = basic.Name()
		} else if structType, ok := underlying.(*types.Struct); ok {
			coreType.Kind = model.TypeKindStruct
			fillStructFields(structType, coreType.ImportPkgPath, imports, project, coreType, loader, processingSet)
		} else if mapType, ok := underlying.(*types.Map); ok {
			if mapType.Key() != nil {
				keyInfo := convertFieldType(mapType.Key(), coreType.ImportPkgPath, imports, project, loader, processingSet)
				if keyInfo.TypeID != "" && keyInfo.TypeID != "invalid type" {
					coreType.MapKey = fieldTypeInfoToTypeRef(keyInfo)
				}
			}
			if mapType.Elem() != nil {
				valueInfo := convertFieldType(mapType.Elem(), coreType.ImportPkgPath, imports, project, loader, processingSet)
				if valueInfo.TypeID != "" && valueInfo.TypeID != "invalid type" {
					coreType.MapValue = fieldTypeInfoToTypeRef(valueInfo)
					coreType.ElementPointers = valueInfo.NumberOfPointers
				}
			}
		}

	case *types.Struct:
		coreType.Kind = model.TypeKindStruct
		fillStructFields(t, pkgPath, imports, project, coreType, loader, processingSet)

	case *types.Interface:
		coreType.Kind = model.TypeKindInterface

	default:
		return
	}

	// detectInterfaces будет вызвана позже, когда loader будет доступен
	// Пока пропускаем

	return
}

func convertBasicKind(kind types.BasicKind) (typeKind model.TypeKind) {

	switch kind {
	case types.String:
		typeKind = model.TypeKindString
		return
	case types.Int:
		typeKind = model.TypeKindInt
		return
	case types.Int8:
		typeKind = model.TypeKindInt8
		return
	case types.Int16:
		typeKind = model.TypeKindInt16
		return
	case types.Int32:
		typeKind = model.TypeKindInt32
		return
	case types.Int64:
		typeKind = model.TypeKindInt64
		return
	case types.Uint:
		typeKind = model.TypeKindUint
		return
	case types.Uint8:
		typeKind = model.TypeKindUint8
		return
	case types.Uint16:
		typeKind = model.TypeKindUint16
		return
	case types.Uint32:
		typeKind = model.TypeKindUint32
		return
	case types.Uint64:
		typeKind = model.TypeKindUint64
		return
	case types.Float32:
		typeKind = model.TypeKindFloat32
		return
	case types.Float64:
		typeKind = model.TypeKindFloat64
		return
	case types.Bool:
		typeKind = model.TypeKindBool
		return
	case types.UntypedNil:
		typeKind = model.TypeKindAny
		return
	default:
		// Handle Byte and Rune which are aliases for Uint8 and Int32
		if kind == types.Byte {
			typeKind = model.TypeKindByte
			return
		}
		if kind == types.Rune {
			typeKind = model.TypeKindRune
			return
		}
		typeKind = model.TypeKindAny
		return
	}
}

func resolveKindFromUnderlying(underlying types.Type) (kind model.TypeKind) {

	switch t := underlying.(type) {
	case *types.Basic:
		kind = convertBasicKind(t.Kind())
		return
	case *types.Struct:
		kind = model.TypeKindStruct
		return
	case *types.Interface:
		kind = model.TypeKindInterface
		return
	case *types.Slice, *types.Array:
		kind = model.TypeKindArray
		return
	case *types.Map:
		kind = model.TypeKindMap
		return
	case *types.Chan:
		kind = model.TypeKindChan
		return
	case *types.Signature:
		kind = model.TypeKindFunction
		return
	case *types.Named, *types.Alias:
		kind = model.TypeKindAlias
		return
	default:
		kind = model.TypeKindAny
		return
	}
}
