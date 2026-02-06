// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	. "github.com/dave/jennifer/jen" // nolint:staticcheck

	"tgp/core/i18n"
	"tgp/internal/common"
	"tgp/internal/model"
)

func (r *ClientRenderer) RenderClientTypes(collectedTypeIDs map[string]bool) error {

	if len(collectedTypeIDs) == 0 {
		// Нет типов для генерации
		return nil
	}

	// Отладка: логируем количество собранных типов
	slog.Debug(i18n.Msg("RenderClientTypes: starting"), slog.Int("collectedTypeIDs", len(collectedTypeIDs)), slog.Int("projectTypes", len(r.project.Types)))

	// Создаем директорию dto, если её нет
	dtoDir := path.Join(r.outDir, "dto")
	if err := os.MkdirAll(dtoDir, 0755); err != nil {
		return fmt.Errorf("%s: %w", i18n.Msg("failed to create dto directory"), err)
	}

	srcFile := NewSrcFile("dto")
	srcFile.PackageComment(DoNotEdit)

	ctx := context.WithValue(context.Background(), keyCode, srcFile) // nolint
	ctx = context.WithValue(ctx, keyPackage, "dto")                  // nolint

	notFoundCount := 0
	skippedCount := 0
	generatedCount := 0
	for _, typeID := range common.SortedKeys(collectedTypeIDs) {
		typ, ok := r.project.Types[typeID]
		if !ok {
			notFoundCount++
			slog.Debug(i18n.Msg("RenderClientTypes: typeID not found in project.Types"), slog.String("typeID", typeID))
			continue
		}

		if r.isBuiltinType(typeID) {
			skippedCount++
			slog.Debug(i18n.Msg("RenderClientTypes: skipping builtin type"), slog.String("typeID", typeID))
			continue
		}

		isFromCurrentProject := r.isTypeFromCurrentProject(typ.ImportPkgPath)

		// ВАЖНО: алиасы на внешние типы тоже генерируем, но как алиасы (type Alias = ExternalType)
		// Это позволяет сохранить семантику алиаса в клиенте
		if !isFromCurrentProject {
			// Для внешних типов генерируем только алиасы
			if typ.Kind == model.TypeKindAlias && typ.AliasOf != "" {
				// Алиас на внешний тип - генерируем как алиас
			} else {
				// Остальные внешние типы не генерируем
				continue
			}
		}

		// ВАЖНО: пропускаем анонимные типы (interface:anonymous, struct:anonymous и т.д.)
		// так как они не могут быть сгенерированы как отдельные типы
		if strings.Contains(typeID, ":interface:anonymous") ||
			strings.Contains(typeID, ":struct:anonymous") ||
			strings.Contains(typeID, ":func:anonymous") {
			continue
		}

		typeName := typ.TypeName
		if typeName == "" {
			if strings.Contains(typeID, ":") {
				parts := strings.SplitN(typeID, ":", 2)
				if len(parts) == 2 {
					typeName = parts[1]
					// ВАЖНО: для анонимных типов (interface:anonymous, struct:anonymous и т.д.)
					// не генерируем отдельный файл, так как это невалидное имя типа
					if strings.Contains(typeName, ":") || typeName == "anonymous" || typeName == "interface" || typeName == "struct" || typeName == "func" {
						continue
					}
				} else {
					typeName = typeID
				}
			} else {
				typeName = typeID
			}
		}

		var typeCode Code
		switch {
		case typ.Kind == model.TypeKindStruct:
			typeCode = r.generateClientStruct(ctx, typeName, typ)
		case typ.Kind == model.TypeKindInterface:
			typeCode = r.generateClientInterface(ctx, typeName, typ)
		case typ.Kind == model.TypeKindAlias:
			// Алиасы (type ID = string)
			// ВАЖНО: алиасы всегда генерируем, чтобы сохранить семантику (type Alias = BaseType)
			typeCode = r.generateClientAlias(ctx, typeName, typ)
		case typ.ImportPkgPath != "" && typ.TypeName != "":
			// Именованные типы с базовым типом (type UserID int64, type Email string)
			// Имеют Kind как базовый тип, но ImportPkgPath и TypeName указывают на именованный тип
			typeCode = r.generateClientAlias(ctx, typeName, typ)
		default:
			// Для остальных типов (массивы, мапы, встроенные базовые типы без имени) не генерируем
			continue
		}

		if typeCode != nil {
			// Создаем отдельный файл для каждого типа
			typeFile := NewSrcFile("dto")
			typeFile.PackageComment(DoNotEdit)
			typeCtx := context.WithValue(context.Background(), keyCode, typeFile) // nolint
			typeCtx = context.WithValue(typeCtx, keyPackage, "dto")               // nolint

			// ВАЖНО: перегенерируем typeCode с правильным контекстом для алиасов
			// Это нужно для правильной установки ImportName в правильном файле
			// Согласно JENNIFER_IMPORTS_GUIDE.md: ImportName должен вызываться с правильным srcFile в контексте
			if typ.Kind == model.TypeKindAlias && typ.AliasOf != "" {
				typeCode = r.generateClientAlias(typeCtx, typeName, typ)
			}

			typeFile.Add(typeCode)

			// Сохраняем файл с именем типа (в нижнем регистре)
			fileName := strings.ToLower(typeName) + ".go"
			if err := typeFile.Save(path.Join(dtoDir, fileName)); err != nil {
				return fmt.Errorf("%s: %w", fmt.Sprintf(i18n.Msg("failed to save type file %s"), typeName), err)
			}
			generatedCount++
			slog.Debug(i18n.Msg("RenderClientTypes: generated type"), slog.String("typeName", typeName), slog.String("typeID", typeID), slog.String("kind", string(typ.Kind)), slog.String("fileName", fileName))
		} else {
			skippedCount++
			slog.Debug(i18n.Msg("RenderClientTypes: typeCode is nil"), slog.String("typeID", typeID), slog.String("typeName", typeName), slog.String("kind", string(typ.Kind)))
		}
	}

	slog.Debug(i18n.Msg("RenderClientTypes: summary"), slog.Int("total", len(collectedTypeIDs)), slog.Int("notFound", notFoundCount), slog.Int("skipped", skippedCount), slog.Int("generated", generatedCount))
	return nil
}

func (r *ClientRenderer) generateClientStruct(ctx context.Context, typeName string, typ *model.Type) Code {

	sortedFields := slices.Clone(typ.StructFields)
	slices.SortFunc(sortedFields, func(a, b *model.StructField) int {
		nameA := a.Name
		nameB := b.Name
		// Встроенные поля (с пустым именем) идут первыми
		if nameA == "" && nameB != "" {
			return -1
		}
		if nameA != "" && nameB == "" {
			return 1
		}
		if nameA < nameB {
			return -1
		}
		if nameA > nameB {
			return 1
		}
		return 0
	})

	return Type().Id(typeName).StructFunc(func(gr *Group) {
		for _, field := range sortedFields {
			fieldCode := r.generateClientStructField(ctx, field)
			gr.Add(fieldCode)
		}
	})
}

func (r *ClientRenderer) generateClientStructField(ctx context.Context, field *model.StructField) *Statement {

	var s *Statement
	if field.Name == "" {
		// Встроенное поле - генерируем только тип без имени
		s = &Statement{}
	} else {
		// Обычное поле - генерируем с именем
		fieldName := ToCamel(field.Name)
		s = Id(fieldName)
	}

	switch {
	case field.IsSlice || field.ArrayLen > 0:
		if field.IsSlice {
			s = s.Index()
		} else {
			s = s.Index(Lit(field.ArrayLen))
		}
		if field.TypeID != "" {
			// ВАЖНО: используем ElementPointers, а не NumberOfPointers
			s = s.Add(r.fieldTypeForClient(ctx, field.TypeID, field.ElementPointers, false))
		} else {
			// Пустой TypeID элемента (например, не удалось разрешить тип) — генерируем []any
			s = s.Id("any")
		}

	case field.MapKey != nil && field.MapValue != nil:
		keyType := r.fieldTypeFromTypeRefForClient(ctx, field.MapKey, false)
		valueType := r.fieldTypeFromTypeRefForClient(ctx, field.MapValue, false)
		s = s.Map(keyType).Add(valueType)

	default:
		// Обычное поле - указатели применяются к типу
		// ВАЖНО: проверяем анонимные интерфейсы и пустые typeID перед вызовом fieldTypeForClient
		switch {
		case field.TypeID == "":
			// Пустой typeID обычно означает any
			s = s.Id("any")
		case strings.Contains(field.TypeID, ":interface:anonymous"):
			s = s.Id("any")
		default:
			s = s.Add(r.fieldTypeForClient(ctx, field.TypeID, field.NumberOfPointers, false))
		}
	}

	tags := make(map[string]string)
	for tagName, tagValues := range common.SortedPairs(field.Tags) {
		if len(tagValues) > 0 {
			tags[tagName] = strings.Join(tagValues, ",")
		}
	}
	if len(tags) > 0 {
		s = s.Tag(tags)
	}

	if len(field.Docs) > 0 {
		for _, doc := range field.Docs {
			s = s.Comment(doc)
		}
	}

	return s
}

func (r *ClientRenderer) generateClientInterface(ctx context.Context, typeName string, typ *model.Type) Code {

	sortedEmbedded := slices.Clone(typ.EmbeddedInterfaces)
	slices.SortFunc(sortedEmbedded, func(a, b *model.Variable) int {
		if a.TypeID < b.TypeID {
			return -1
		}
		if a.TypeID > b.TypeID {
			return 1
		}
		return 0
	})

	sortedMethods := slices.Clone(typ.InterfaceMethods)
	slices.SortFunc(sortedMethods, func(a, b *model.Function) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})

	return Type().Id(typeName).InterfaceFunc(func(gr *Group) {
		for _, embedded := range sortedEmbedded {
			embeddedType := r.fieldTypeForClient(ctx, embedded.TypeID, 0, false)
			gr.Add(embeddedType)
		}

		for _, method := range sortedMethods {
			methodCode := r.generateClientMethod(ctx, method)
			gr.Add(methodCode)
		}
	})
}

func (r *ClientRenderer) generateClientMethod(ctx context.Context, method *model.Function) *Statement {

	s := Id(method.Name)

	args := make([]Code, 0, len(method.Args))
	for _, arg := range method.Args {
		argType := r.fieldTypeFromVariableForClient(ctx, arg, false)
		args = append(args, Id(ToLowerCamel(arg.Name)).Add(argType))
	}

	// Результаты
	results := make([]Code, 0, len(method.Results))
	for _, result := range method.Results {
		resultType := r.fieldTypeFromVariableForClient(ctx, result, false)
		results = append(results, resultType)
	}

	s = s.Params(args...)
	if len(results) > 0 {
		s = s.Params(results...)
	}

	return s
}

func (r *ClientRenderer) generateClientAlias(ctx context.Context, typeName string, typ *model.Type) Code {

	// Для алиасов всегда используем базовый тип через AliasOf
	// Это позволяет сохранить семантику алиаса в клиенте (type Alias = BaseType)
	if typ.AliasOf != "" {
		baseType := r.fieldTypeForClient(ctx, typ.AliasOf, 0, false)
		return Type().Id(typeName).Op("=").Add(baseType)
	}
	if typ.UnderlyingTypeID != "" {
		baseType := r.fieldTypeForClient(ctx, typ.UnderlyingTypeID, 0, false)
		return Type().Id(typeName).Op("=").Add(baseType)
	}

	if typ.Kind == model.TypeKindMap {
		if typ.MapKey != nil && typ.MapValue != nil {
			keyType := r.fieldTypeFromTypeRefForClient(ctx, typ.MapKey, false)
			valueType := r.fieldTypeFromTypeRefForClient(ctx, typ.MapValue, false)
			return Type().Id(typeName).Op("=").Map(keyType).Add(valueType)
		}
		return Type().Id(typeName).Op("=").Map(Id("string")).Id("any")
	}

	// Для именованных типов с базовым типом (type UserID int64) используем UnderlyingKind
	if typ.UnderlyingKind != "" {
		return Type().Id(typeName).Id(string(typ.UnderlyingKind))
	}

	// Fallback на базовый Kind (если UnderlyingKind не установлен, используем Kind)
	// Но не для map типов - они уже обработаны выше
	if typ.Kind != model.TypeKindMap {
		return Type().Id(typeName).Id(string(typ.Kind))
	}

	// Если дошли сюда и это map, используем string для ключа и any для значения
	return Type().Id(typeName).Op("=").Map(Id("string")).Id("any")
}

func (r *ClientRenderer) fieldTypeForClient(ctx context.Context, typeID string, numberOfPointers int, allowEllipsis bool) *Statement {

	c := &Statement{}

	for i := 0; i < numberOfPointers; i++ {
		c.Op("*")
	}

	typ, ok := r.project.Types[typeID]
	if !ok {
		// Тип не найден - проверяем, является ли он встроенным
		if r.isBuiltinType(typeID) {
			return c.Id(typeID)
		}
		// ВАЖНО: проверяем анонимные типы (interface:anonymous, struct:anonymous и т.д.)
		if strings.Contains(typeID, ":interface:anonymous") {
			return c.Id("any")
		}
		// Если тип не найден, но это внешний тип (содержит ":"), используем его как именованный тип
		if strings.Contains(typeID, ":") {
			parts := strings.SplitN(typeID, ":", 2)
			if len(parts) == 2 {
				importPkgPath := parts[0]
				typeName := parts[1]
				// Если это внешний тип (не из текущего проекта), используем его как именованный тип
				if !r.isTypeFromCurrentProject(importPkgPath) {
					if srcFile, ok := ctx.Value(keyCode).(GoFile); ok {
						packageName := filepath.Base(importPkgPath)
						srcFile.ImportName(importPkgPath, packageName)
						return c.Qual(importPkgPath, typeName)
					}
					return c.Qual(importPkgPath, typeName)
				}
			}
		}
		// Если тип не найден в project.Types, это ошибка - Core должен был обработать все типы
		// Возвращаем typeID как fallback
		return c.Id(typeID)
	}

	// ВАЖНО: для типов из внешних пакетов (не из текущего проекта) используем их как именованные типы,
	// независимо от Kind. Например, uuid.UUID имеет Kind == TypeKindArray, но это именованный тип
	// из внешнего пакета, и его нужно использовать как uuid.UUID, а не как [16]byte
	// Эта проверка должна быть ПЕРВОЙ, до всех остальных обработок (ellipsis, switch по Kind и т.д.)
	if typ.ImportPkgPath != "" && typ.TypeName != "" {
		if !r.isTypeFromCurrentProject(typ.ImportPkgPath) {
			// Тип из внешнего пакета - используем информацию из shared напрямую
			// Согласно JENNIFER_IMPORTS_GUIDE.md: используем ImportName с PkgName из shared
			if srcFile, ok := ctx.Value(keyCode).(GoFile); ok {
				// PkgName содержит реальное имя пакета из package декларации (например, "uuid")
				packageName := typ.PkgName
				if packageName == "" {
					// Fallback на последнюю часть пути, если PkgName не установлен
					packageName = filepath.Base(typ.ImportPkgPath)
				}
				srcFile.ImportName(typ.ImportPkgPath, packageName)
				return c.Qual(typ.ImportPkgPath, typ.TypeName)
			}
			return c.Qual(typ.ImportPkgPath, typ.TypeName)
		}
	}

	if typ.IsEllipsis && allowEllipsis {
		c.Op("...")
		if typ.ArrayOfID != "" {
			return c.Add(r.fieldTypeForClient(ctx, typ.ArrayOfID, 0, false))
		}
		return c
	}

	switch typ.Kind {
	case model.TypeKindArray:
		switch {
		case typ.IsSlice:
			c.Index()
		case typ.ArrayLen > 0:
			c.Index(Lit(typ.ArrayLen))
		default:
			c.Index()
		}
		if typ.ArrayOfID != "" {
			return c.Add(r.fieldTypeForClient(ctx, typ.ArrayOfID, 0, false))
		}
		return c

	case model.TypeKindMap:
		// ВАЖНО: если это именованный map тип из текущего проекта, используем имя типа
		if typ.TypeName != "" && typ.ImportPkgPath != "" {
			if r.isTypeFromCurrentProject(typ.ImportPkgPath) {
				// Тип из текущего проекта - используем имя типа
				if currentPkg, ok := ctx.Value(keyPackage).(string); ok && currentPkg == "dto" {
					return c.Id(typ.TypeName)
				}
				if srcFile, ok := ctx.Value(keyCode).(GoFile); ok {
					dtoPkgPath := fmt.Sprintf("%s/dto", r.pkgPath(r.outDir))
					srcFile.ImportName(dtoPkgPath, "dto")
					return c.Qual(dtoPkgPath, typ.TypeName)
				}
				return c.Id(typ.TypeName)
			}
		}
		// Для неименованных map типов генерируем map напрямую
		if typ.MapKey != nil && typ.MapValue != nil {
			keyType := r.fieldTypeFromTypeRefForClient(ctx, typ.MapKey, false)
			valueType := r.fieldTypeFromTypeRefForClient(ctx, typ.MapValue, false)
			return c.Map(keyType).Add(valueType)
		}
		return c.Map(Id("string")).Id("any")

	case model.TypeKindChan:
		chanType := c
		switch typ.ChanDirection {
		case 1: // send only
			chanType = chanType.Chan().Op("<-")
		case 2: // receive only
			chanType = chanType.Op("<-").Chan()
		default: // both
			chanType = chanType.Chan()
		}
		if typ.ChanOfID != "" {
			return chanType.Add(r.fieldTypeForClient(ctx, typ.ChanOfID, 0, false))
		}
		return chanType

	case model.TypeKindStruct, model.TypeKindInterface:
		// ВАЖНО: все типы из текущего проекта должны генерироваться локально и использоваться из dto пакета
		if typ.ImportPkgPath != "" && typ.TypeName != "" {
			if r.isTypeFromCurrentProject(typ.ImportPkgPath) {
				// Тип из текущего проекта
				// ВАЖНО: если мы генерируем код в пакете dto, то типы из того же пакета
				// используем без импорта (просто по имени), чтобы избежать циклических импортов
				if currentPkg, ok := ctx.Value(keyPackage).(string); ok && currentPkg == "dto" {
					return c.Id(typ.TypeName)
				}
				// Тип из текущего проекта, но не генерируется в пакете dto - используем dto пакет
				if srcFile, ok := ctx.Value(keyCode).(GoFile); ok {
					dtoPkgPath := fmt.Sprintf("%s/dto", r.pkgPath(r.outDir))
					srcFile.ImportName(dtoPkgPath, "dto")
					return c.Qual(dtoPkgPath, typ.TypeName)
				}
				return c.Id(typ.TypeName)
			}
			// Тип из внешнего пакета - используем информацию из shared напрямую
			// Согласно JENNIFER_IMPORTS_GUIDE.md: используем ImportName с PkgName из shared
			// ImportName устанавливает имя пакета для использования в Qual
			// Если имя пакета совпадает с используемым в коде, jennifer НЕ добавляет псевдоним
			if srcFile, ok := ctx.Value(keyCode).(GoFile); ok {
				// PkgName содержит реальное имя пакета из package декларации (например, "jose")
				packageName := typ.PkgName
				if packageName == "" {
					// Fallback на последнюю часть пути, если PkgName не установлен
					packageName = filepath.Base(typ.ImportPkgPath)
				}
				srcFile.ImportName(typ.ImportPkgPath, packageName)
				return c.Qual(typ.ImportPkgPath, typ.TypeName)
			}
			return c.Qual(typ.ImportPkgPath, typ.TypeName)
		}
		// Если TypeName пустой, это может быть анонимный интерфейс (any)
		if strings.Contains(typeID, ":interface:anonymous") || typ.Kind == model.TypeKindAny {
			return c.Id("any")
		}
		// Если нет ImportPkgPath, используем TypeName напрямую
		if typ.TypeName != "" {
			return c.Id(typ.TypeName)
		}
		// Fallback на any для пустых интерфейсов
		return c.Id("any")

	case model.TypeKindFunction:
		args := make([]Code, 0, len(typ.FunctionArgs))
		for _, arg := range typ.FunctionArgs {
			argType := r.fieldTypeFromVariableForClient(ctx, arg, false)
			args = append(args, argType)
		}
		results := make([]Code, 0, len(typ.FunctionResults))
		for _, res := range typ.FunctionResults {
			resType := r.fieldTypeFromVariableForClient(ctx, res, false)
			results = append(results, resType)
		}
		return c.Func().Params(args...).Params(results...)

	case model.TypeKindAlias:
		// ВАЖНО: для алиасов из текущего проекта используем локальный тип из dto пакета
		// Это позволяет сохранить семантику алиаса в клиенте
		if typ.ImportPkgPath != "" && typ.TypeName != "" {
			if r.isTypeFromCurrentProject(typ.ImportPkgPath) {
				// Тип из текущего проекта - используем локальный тип из dto пакета
				if currentPkg, ok := ctx.Value(keyPackage).(string); ok && currentPkg == "dto" {
					return c.Id(typ.TypeName)
				}
				// Тип из текущего проекта, но не генерируется в пакете dto - используем dto пакет
				if srcFile, ok := ctx.Value(keyCode).(GoFile); ok {
					dtoPkgPath := fmt.Sprintf("%s/dto", r.pkgPath(r.outDir))
					srcFile.ImportName(dtoPkgPath, "dto")
					return c.Qual(dtoPkgPath, typ.TypeName)
				}
				return c.Id(typ.TypeName)
			}
		}
		// Для внешних алиасов или если нет ImportPkgPath - используем базовый тип
		// НО: если это алиас из текущего проекта, но ImportPkgPath не установлен,
		// проверяем, есть ли TypeName и используем его
		if typ.AliasOf != "" {
			// ВАЖНО: если алиас имеет TypeName и ImportPkgPath из текущего проекта,
			// используем имя алиаса, а не базовый тип, независимо от того, откуда базовый тип
			if typ.TypeName != "" {
				isAliasFromCurrentProject := false
				if typ.ImportPkgPath != "" {
					isAliasFromCurrentProject = r.isTypeFromCurrentProject(typ.ImportPkgPath)
				} else {
					// Если ImportPkgPath не установлен, проверяем по базовому типу
					if baseType, exists := r.project.Types[typ.AliasOf]; exists {
						isAliasFromCurrentProject = r.isTypeFromCurrentProject(baseType.ImportPkgPath)
					}
				}

				if isAliasFromCurrentProject {
					// Алиас из текущего проекта - используем имя алиаса
					if currentPkg, ok := ctx.Value(keyPackage).(string); ok && currentPkg == "dto" {
						return c.Id(typ.TypeName)
					}
					if srcFile, ok := ctx.Value(keyCode).(GoFile); ok {
						dtoPkgPath := fmt.Sprintf("%s/dto", r.pkgPath(r.outDir))
						srcFile.ImportName(dtoPkgPath, "dto")
						return c.Qual(dtoPkgPath, typ.TypeName)
					}
					return c.Id(typ.TypeName)
				}
			}
			// Для внешних алиасов используем базовый тип
			if baseType, exists := r.project.Types[typ.AliasOf]; exists {
				importPkgPath := baseType.ImportPkgPath
				typeName := baseType.TypeName
				if srcFile, ok := ctx.Value(keyCode).(GoFile); ok {
					// PkgName содержит реальное имя пакета из package декларации
					packageName := baseType.PkgName
					if packageName == "" {
						packageName = filepath.Base(importPkgPath)
					}
					srcFile.ImportName(importPkgPath, packageName)
				}
				return c.Qual(importPkgPath, typeName)
			}
			return r.fieldTypeForClient(ctx, typ.AliasOf, numberOfPointers, allowEllipsis)
		}
		// Если TypeName есть, но нет AliasOf, используем TypeName напрямую
		if typ.TypeName != "" {
			if currentPkg, ok := ctx.Value(keyPackage).(string); ok && currentPkg == "dto" {
				return c.Id(typ.TypeName)
			}
		}
		if typ.UnderlyingTypeID != "" {
			return r.fieldTypeForClient(ctx, typ.UnderlyingTypeID, numberOfPointers, allowEllipsis)
		}
		// Fallback на UnderlyingKind
		if typ.UnderlyingKind != "" {
			return c.Id(string(typ.UnderlyingKind))
		}
		// Если ничего не найдено, используем Kind
		return c.Id(string(typ.Kind))

	case model.TypeKindString, model.TypeKindInt, model.TypeKindInt8, model.TypeKindInt16,
		model.TypeKindInt32, model.TypeKindInt64, model.TypeKindUint, model.TypeKindUint8,
		model.TypeKindUint16, model.TypeKindUint32, model.TypeKindUint64,
		model.TypeKindFloat32, model.TypeKindFloat64, model.TypeKindBool,
		model.TypeKindByte, model.TypeKindRune, model.TypeKindError, model.TypeKindAny:
		// ВАЖНО: все типы из текущего проекта должны генерироваться локально и использоваться из dto пакета
		// Если у типа есть ImportPkgPath и TypeName, это именованный тип (например, UserID int64, Email string)
		if typ.ImportPkgPath != "" && typ.TypeName != "" {
			if r.isTypeFromCurrentProject(typ.ImportPkgPath) {
				// Тип из текущего проекта
				// ВАЖНО: если мы генерируем код в пакете dto, то типы из того же пакета
				// используем без импорта (просто по имени), чтобы избежать циклических импортов
				if currentPkg, ok := ctx.Value(keyPackage).(string); ok && currentPkg == "dto" {
					return c.Id(typ.TypeName)
				}
				// Тип из текущего проекта, но не генерируется в пакете dto - используем dto пакет
				if srcFile, ok := ctx.Value(keyCode).(GoFile); ok {
					dtoPkgPath := fmt.Sprintf("%s/dto", r.pkgPath(r.outDir))
					srcFile.ImportName(dtoPkgPath, "dto")
					return c.Qual(dtoPkgPath, typ.TypeName)
				}
				return c.Id(typ.TypeName)
			}
			// Тип из внешнего пакета (например, time.Time) - используем информацию из shared напрямую
			// Согласно JENNIFER_IMPORTS_GUIDE.md: используем ImportName с PkgName из shared
			if srcFile, ok := ctx.Value(keyCode).(GoFile); ok {
				// PkgName содержит реальное имя пакета из package декларации
				packageName := typ.PkgName
				if packageName == "" {
					packageName = filepath.Base(typ.ImportPkgPath)
				}
				srcFile.ImportName(typ.ImportPkgPath, packageName)
			}
			return c.Qual(typ.ImportPkgPath, typ.TypeName)
		}
		// Встроенный базовый тип - используем Kind как имя типа
		return c.Id(string(typ.Kind))

	default:
		return c
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

func (r *ClientRenderer) fieldTypeFromVariableForClient(ctx context.Context, variable *model.Variable, allowEllipsis bool) *Statement {
	return r.fieldTypeFromTypeRefForClient(ctx, &variable.TypeRef, allowEllipsis)
}

func (r *ClientRenderer) fieldTypeFromTypeRefForClient(ctx context.Context, typeRef *model.TypeRef, allowEllipsis bool) *Statement {
	c := &Statement{}

	if typeRef.IsEllipsis && allowEllipsis {
		for i := 0; i < typeRef.NumberOfPointers; i++ {
			c.Op("*")
		}
		c.Op("...")
		if typeRef.TypeID != "" {
			return c.Add(r.fieldTypeForClient(ctx, typeRef.TypeID, 0, false))
		}
		return c
	}

	if typeRef.IsSlice || typeRef.ArrayLen > 0 {
		if typeRef.TypeID != "" {
			typ, ok := r.project.Types[typeRef.TypeID]
			if ok && typ.ImportPkgPath != "" && typ.TypeName != "" {
				if !r.isTypeFromCurrentProject(typ.ImportPkgPath) {
					for i := 0; i < typeRef.NumberOfPointers; i++ {
						c.Op("*")
					}
					if srcFile, ok := ctx.Value(keyCode).(GoFile); ok {
						packageName := typ.PkgName
						if packageName == "" {
							packageName = filepath.Base(typ.ImportPkgPath)
						}
						srcFile.ImportName(typ.ImportPkgPath, packageName)
						return c.Qual(typ.ImportPkgPath, typ.TypeName)
					}
					return c.Qual(typ.ImportPkgPath, typ.TypeName)
				}
			}
		}
		for i := 0; i < typeRef.NumberOfPointers; i++ {
			c.Op("*")
		}
		if typeRef.IsSlice {
			c.Index()
		} else {
			c.Index(Lit(typeRef.ArrayLen))
		}
		if typeRef.TypeID != "" {
			return c.Add(r.fieldTypeForClient(ctx, typeRef.TypeID, typeRef.ElementPointers, false))
		}
		return c.Add(Id("any"))
	}

	if typeRef.MapKey != nil && typeRef.MapValue != nil {
		for i := 0; i < typeRef.NumberOfPointers; i++ {
			c.Op("*")
		}
		keyType := r.fieldTypeFromTypeRefForClient(ctx, typeRef.MapKey, false)
		valueType := r.fieldTypeFromTypeRefForClient(ctx, typeRef.MapValue, false)
		return c.Map(keyType).Add(valueType)
	}

	return c.Add(r.fieldTypeForClient(ctx, typeRef.TypeID, 0, false))
}
