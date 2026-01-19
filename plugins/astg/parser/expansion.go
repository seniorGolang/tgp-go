// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/types"
	"log/slog"

	"golang.org/x/tools/go/types/typeutil"

	"tgp/core/i18n"
	"tgp/internal/model"
)

// expandTypesRecursively рекурсивно разбирает все типы из контрактов.
func expandTypesRecursively(project *model.Project, loader *AutonomousPackageLoader) (err error) {

	seenTypes := &typeutil.Map{}
	msets := &typeutil.MethodSetCache{}

	// Собираем все типы из контрактов
	for _, contract := range project.Contracts {
		for _, method := range contract.Methods {
			// Обрабатываем аргументы
			for _, arg := range method.Args {
				if err = collectTypeFromID(arg.TypeID, project, seenTypes, msets, loader); err != nil {
					return
				}
			}

			// Обрабатываем результаты
			for _, result := range method.Results {
				if err = collectTypeFromID(result.TypeID, project, seenTypes, msets, loader); err != nil {
					return
				}
			}
		}
	}

	return
}

// collectTypeFromID получает types.Type по typeID и рекурсивно обходит все его зависимости.
func collectTypeFromID(typeID string, project *model.Project, seenTypes *typeutil.Map, msets *typeutil.MethodSetCache, loader *AutonomousPackageLoader) (err error) {

	// Получаем тип из project.Types (используется как кэш)
	var typ *model.Type
	var exists bool
	if typ, exists = project.Types[typeID]; !exists {
		// Тип не найден - пытаемся загрузить его из пакета
		if err = ensureTypeLoaded(typeID, project, loader); err != nil {
			slog.Debug(i18n.Msg("Failed to load type"), slog.String("type", typeID), slog.Any("error", err))
			return
		}
		if typ, exists = project.Types[typeID]; !exists {
			// Тип не был сохранен - возможно, это встроенный тип или ошибка загрузки
			return
		}
	}

	// НЕ пропускаем исключенные типы - они нужны для анализа интерфейсов
	// isExcludedType используется только для фильтрации при сохранении в финальный результат

	// Если это алиас, обрабатываем базовый тип рекурсивно
	if typ.Kind == model.TypeKindAlias && typ.AliasOf != "" {
		// Обрабатываем базовый тип рекурсивно
		// collectTypeFromID загрузит базовый тип через ensureTypeLoaded, если его еще нет
		if err = collectTypeFromID(typ.AliasOf, project, seenTypes, msets, loader); err != nil {
			slog.Debug(i18n.Msg("Failed to collect base type for alias"), slog.String("baseType", typ.AliasOf), slog.String("alias", typeID), slog.Any("error", err))
			// Продолжаем обработку, даже если базовый тип не загружен
		}
		// Продолжаем обработку текущего типа для обработки его полей (если есть)
		// Но если это просто алиас без полей, можно вернуться
		if len(typ.StructFields) == 0 {
			// Это простой алиас - базовый тип уже обработан (или будет обработан), можно вернуться
			return
		}
	}

	// Получаем types.Type из пакета для рекурсивного обхода
	if typ.ImportPkgPath == "" || typ.TypeName == "" {
		return
	}

	var pkgInfo *PackageInfo
	var ok bool
	if pkgInfo, ok = loader.GetPackage(typ.ImportPkgPath); !ok || pkgInfo == nil || pkgInfo.Types == nil {
		return
	}

	obj := pkgInfo.Types.Scope().Lookup(typ.TypeName)
	if obj == nil {
		return
	}

	var typeNameObj *types.TypeName
	if typeNameObj, ok = obj.(*types.TypeName); !ok {
		return
	}

	// Рекурсивно обходим все типы, начиная с этого типа
	forEachReachableType(typeNameObj.Type(), project, seenTypes, msets, loader)

	// Также обрабатываем базовый тип алиаса, если он еще не обработан
	if typ.Kind == model.TypeKindAlias && typ.AliasOf != "" {
		// Убеждаемся, что базовый тип обработан рекурсивно
		var baseType *model.Type
		if baseType, exists = project.Types[typ.AliasOf]; exists {
			// Обрабатываем базовый тип рекурсивно через forEachReachableType
			if baseType.ImportPkgPath != "" && baseType.TypeName != "" {
				var basePkgInfo *PackageInfo
				if basePkgInfo, ok = loader.GetPackage(baseType.ImportPkgPath); ok && basePkgInfo != nil && basePkgInfo.Types != nil {
					baseObj := basePkgInfo.Types.Scope().Lookup(baseType.TypeName)
					if baseObj != nil {
						var baseTypeNameObj *types.TypeName
						if baseTypeNameObj, ok = baseObj.(*types.TypeName); ok {
							forEachReachableType(baseTypeNameObj.Type(), project, seenTypes, msets, loader)
						}
					}
				}
			}
		}
	}

	return
}

// forEachReachableType рекурсивно обходит все типы, достижимые из данного типа.
// Основано на gopls/internal/typesinternal/element.go:ForEachElement
func forEachReachableType(t types.Type, project *model.Project, seenTypes *typeutil.Map, msets *typeutil.MethodSetCache, loader *AutonomousPackageLoader) {
	var visit func(t types.Type, skip bool)
	visit = func(t types.Type, skip bool) {
		if !skip {
			if seen, _ := seenTypes.At(t).(bool); seen {
				return
			}
			seenTypes.Set(t, true)
			saveTypeFromGoTypes(t, project, loader)
		}
		tmset := msets.MethodSet(t)
		for method := range tmset.Methods() {
			sig := method.Type().(*types.Signature)
			visit(sig.Params(), true)
			visit(sig.Results(), true)
		}

		switch t := t.(type) {
		case *types.Alias:
			visit(types.Unalias(t), skip)
			if !skip {
				saveTypeFromGoTypes(t, project, loader)
			}

		case *types.Basic:
			// nop

		case *types.Interface:
			// nop

		case *types.Pointer:
			visit(t.Elem(), false)

		case *types.Slice:
			visit(t.Elem(), false)

		case *types.Chan:
			visit(t.Elem(), false)

		case *types.Map:
			visit(t.Key(), false)
			visit(t.Elem(), false)

		case *types.Signature:
			if t.Recv() != nil {
				return
			}
			visit(t.Params(), true)
			visit(t.Results(), true)

		case *types.Named:
			ptrType := types.NewPointer(t)
			visit(ptrType, false)
			visit(t.Underlying(), true)

		case *types.Array:
			visit(t.Elem(), false)

		case *types.Struct:
			for i, n := 0, t.NumFields(); i < n; i++ {
				visit(t.Field(i).Type(), false)
			}

		case *types.Tuple:
			for i, n := 0, t.Len(); i < n; i++ {
				visit(t.At(i).Type(), false)
			}

		case *types.TypeParam, *types.Union:

		default:
		}
	}
	visit(t, false)
}

// saveTypeFromGoTypes сохраняет тип из go/types в project.Types.
func saveTypeFromGoTypes(t types.Type, project *model.Project, loader *AutonomousPackageLoader) {
	typeID := generateTypeIDFromGoTypes(t)
	if typeID == "" {
		return
	}

	// Проверяем, не является ли это встроенным типом
	if basic, ok := t.(*types.Basic); ok {
		if isBuiltinTypeName(basic.Name()) {
			return
		}
	}

	// Проверяем, не сохранен ли уже тип
	if _, exists := project.Types[typeID]; exists {
		return
	}

	// Получаем информацию о пакете
	var importPkgPath string
	var typeName string

	switch t := t.(type) {
	case *types.Named:
		if t.Obj() != nil {
			typeName = t.Obj().Name()
			if t.Obj().Pkg() != nil {
				importPkgPath = t.Obj().Pkg().Path()
			}
		}
	case *types.Alias:
		if t.Obj() != nil {
			typeName = t.Obj().Name()
			if t.Obj().Pkg() != nil {
				importPkgPath = t.Obj().Pkg().Path()
			}
		}
	}

	if importPkgPath == "" || typeName == "" {
		return
	}

	pkgInfo, ok := loader.GetPackage(importPkgPath)
	if !ok || pkgInfo == nil {
		return
	}

	// Конвертируем тип через convertTypeFromGoTypes
	// Создаем processingSet для защиты от рекурсии
	processingSet := make(map[string]bool)
	coreType := convertTypeFromGoTypes(t, importPkgPath, pkgInfo.Imports, project, loader, processingSet)
	if coreType == nil {
		return
	}

	// Определяем интерфейсы, которые реализует тип
	detectInterfaces(t, coreType, project, loader)

	// Сохраняем тип в project.Types
	// Все типы сохраняются для полного анализа, project.Types используется как кэш
	project.Types[typeID] = coreType
}

// generateTypeIDFromGoTypes генерирует typeID для types.Type.
func generateTypeIDFromGoTypes(t types.Type) (typeID string) {

	if t == nil {
		return
	}

	switch t := t.(type) {
	case *types.Basic:
		result := t.Name()
		//nolint:staticcheck // QF1003: проверка пустой строки более читаема через if
		if result == "" || result == "invalid type" {
			// Это не должно происходить - "invalid type" не является валидным именем типа
			// Возвращаем пустую строку, чтобы fallback логика сработала
			return
		}
		typeID = result
		return

	case *types.Named:
		if t.Obj() != nil {
			typeName := t.Obj().Name()
			if t.Obj().Pkg() != nil {
				importPkgPath := t.Obj().Pkg().Path()
				typeID = fmt.Sprintf("%s:%s", importPkgPath, typeName)
				return
			}
			typeID = typeName
			return
		}
		// Если Obj() == nil, пытаемся получить информацию из underlying типа
		underlying := t.Underlying()
		if underlying != nil {
			// Рекурсивно пытаемся получить typeID из underlying типа
			var underlyingID string
			if underlyingID = generateTypeIDFromGoTypes(underlying); underlyingID != "" {
				typeID = underlyingID
				return
			}
		}
		return

	case *types.Alias:
		if t.Obj() != nil {
			typeName := t.Obj().Name()
			if t.Obj().Pkg() != nil {
				importPkgPath := t.Obj().Pkg().Path()
				typeID = fmt.Sprintf("%s:%s", importPkgPath, typeName)
				return
			}
			typeID = typeName
			return
		}
		// Если Obj() == nil, пытаемся получить информацию из underlying типа
		underlying := types.Unalias(t)
		if underlying != nil {
			// Рекурсивно пытаемся получить typeID из underlying типа
			var underlyingID string
			if underlyingID = generateTypeIDFromGoTypes(underlying); underlyingID != "" {
				typeID = underlyingID
				return
			}
		}
		return

	case *types.Interface:
		// Интерфейсы без имени не могут быть идентифицированы однозначно
		// Но если это underlying тип именованного типа, он должен быть обработан через *types.Named
		// Для анонимных интерфейсов возвращаем пустую строку
		return

	default:
		return
	}
}

// ensureTypeLoaded загружает тип из пакета, если его еще нет в project.Types.
func ensureTypeLoaded(typeID string, project *model.Project, loader *AutonomousPackageLoader) (err error) {

	var existingType *model.Type
	var exists bool
	if existingType, exists = project.Types[typeID]; exists {
		if existingType.ImportPkgPath != "" && existingType.TypeName != "" {
			return
		}
	}

	parts := splitTypeID(typeID)
	if len(parts) != 2 {
		return
	}

	importPkgPath := parts[0]
	typeName := parts[1]

	if isBuiltinTypeName(typeName) {
		return
	}

	var pkgInfo *PackageInfo
	var ok bool
	if pkgInfo, ok = loader.GetPackage(importPkgPath); !ok || pkgInfo == nil || pkgInfo.Types == nil {
		if _, err = loader.LoadPackageForType(importPkgPath, typeName); err != nil {
			err = fmt.Errorf("package %s not found: %w", importPkgPath, err)
			return
		}
		if pkgInfo, ok = loader.GetPackage(importPkgPath); !ok || pkgInfo == nil || pkgInfo.Types == nil {
			err = fmt.Errorf("package %s not found after load", importPkgPath)
			return
		}
	}

	var obj types.Object
	if obj = pkgInfo.Types.Scope().Lookup(typeName); obj == nil {
		loader.mu.Lock()
		delete(loader.cache, importPkgPath)
		loader.mu.Unlock()

		if _, err = loader.LoadPackageForType(importPkgPath, typeName); err != nil {
			err = fmt.Errorf("package %s not found after reload for type %s: %w", importPkgPath, typeName, err)
			return
		}
		if pkgInfo, ok = loader.GetPackage(importPkgPath); !ok || pkgInfo == nil || pkgInfo.Types == nil {
			err = fmt.Errorf("package %s not found after reload", importPkgPath)
			return
		}

		if obj = pkgInfo.Types.Scope().Lookup(typeName); obj == nil {
			allNames := pkgInfo.Types.Scope().Names()
			err = fmt.Errorf("type %s not found in package %s after reload (available types: %v)", typeName, importPkgPath, allNames)
			return
		}
	}

	var typeNameObj *types.TypeName
	if typeNameObj, ok = obj.(*types.TypeName); !ok {
		err = fmt.Errorf("object %s is not a type name", typeName)
		return
	}

	processingSet := make(map[string]bool)
	var coreType *model.Type
	if coreType = convertTypeFromGoTypes(typeNameObj.Type(), importPkgPath, pkgInfo.Imports, project, loader, processingSet); coreType == nil {
		err = fmt.Errorf("failed to convert type %s", typeID)
		return
	}

	detectInterfaces(typeNameObj.Type(), coreType, project, loader)
	project.Types[typeID] = coreType

	if coreType.Kind == model.TypeKindAlias && coreType.AliasOf != "" {
		if _, exists = project.Types[coreType.AliasOf]; !exists {
			if err = ensureTypeLoaded(coreType.AliasOf, project, loader); err != nil {
				slog.Debug(i18n.Msg("Failed to load base type"), slog.String("baseType", coreType.AliasOf), slog.Any("error", err))
			}
		}
	}

	return
}

// splitTypeID объявлена в utils.go
