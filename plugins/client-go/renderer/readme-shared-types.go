// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"strings"

	"tgp/internal/common"
	"tgp/internal/markdown"

	"tgp/internal/model"
)

func (r *ClientRenderer) renderAllTypes(md *markdown.Markdown, allTypes map[string]*typeUsage) {
	sharedTypesAnchor := generateAnchor("Общие типы")
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", sharedTypesAnchor))
	md.LF()
	md.H2("Общие типы")
	md.PlainText("Типы данных, используемые в клиенте. Типы, используемые в нескольких методах, описаны здесь для избежания дублирования.")
	md.LF()

	for key, usage := range common.SortedPairs(allTypes) {
		typ, ok := r.project.Types[key]
		if !ok {
			// Пытаемся найти тип по pkgPath и typeName
			for _, t := range common.SortedPairs(r.project.Types) {
				if t.ImportPkgPath == usage.pkgPath && t.TypeName == usage.typeName {
					typ = t
					break
				}
			}
		}

		isExternal := typ != nil && typ.ImportPkgPath != "" && !r.isTypeFromCurrentProject(typ.ImportPkgPath)

		if typ != nil && typ.Kind == model.TypeKindStruct && !isExternal {
			r.renderStructTypeTable(md, typ, usage.fullTypeName, usage.pkgPath)
		} else {
			// Если тип не найден, не структура или из внешней библиотеки, выводим только заголовок
			typeAnchor := typeAnchorID(usage.fullTypeName)
			md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", typeAnchor))
			md.LF()
			md.H4(usage.fullTypeName)

			// Если тип из внешней библиотеки, добавляем информацию о библиотеке
			if isExternal {
				importPath := typ.ImportPkgPath
				libURL := fmt.Sprintf("https://pkg.go.dev/%s", importPath)
				md.PlainText(fmt.Sprintf("Тип из внешней библиотеки: [%s](%s)", importPath, libURL))
				md.LF()
			}

			md.LF()
		}
	}

	md.HorizontalRule()
}

func (r *ClientRenderer) getTypeLinkFromStructField(field *model.StructField, pkgPath string) string {
	switch {
	case field.IsSlice || field.ArrayLen > 0:
		return r.getTypeLink(field.TypeID, pkgPath)
	case field.MapKey != nil && field.MapValue != nil:
		keyLink := r.getTypeLinkFromTypeRef(field.MapKey, pkgPath)
		valueLink := r.getTypeLinkFromTypeRef(field.MapValue, pkgPath)
		links := []string{}
		if keyLink != "-" {
			links = append(links, keyLink)
		}
		if valueLink != "-" {
			links = append(links, valueLink)
		}
		if len(links) > 0 {
			return strings.Join(links, ", ")
		}
		return "-"
	default:
		return r.getTypeLink(field.TypeID, pkgPath)
	}
}

func (r *ClientRenderer) getTypeLinkFromVariable(variable *model.Variable, pkgPath string) string {
	return r.getTypeLinkFromTypeRef(&variable.TypeRef, pkgPath)
}

func (r *ClientRenderer) getTypeLinkFromTypeRef(typeRef *model.TypeRef, pkgPath string) string {
	if typeRef == nil {
		return "-"
	}
	switch {
	case typeRef.IsSlice || typeRef.ArrayLen > 0:
		return r.getTypeLink(typeRef.TypeID, pkgPath)
	case typeRef.MapKey != nil && typeRef.MapValue != nil:
		keyLink := r.getTypeLinkFromTypeRef(typeRef.MapKey, pkgPath)
		valueLink := r.getTypeLinkFromTypeRef(typeRef.MapValue, pkgPath)
		links := []string{}
		if keyLink != "-" {
			links = append(links, keyLink)
		}
		if valueLink != "-" {
			links = append(links, valueLink)
		}
		if len(links) > 0 {
			return strings.Join(links, ", ")
		}
		return "-"
	default:
		return r.getTypeLink(typeRef.TypeID, pkgPath)
	}
}

func (r *ClientRenderer) getTypeLink(typeID, pkgPath string) string {
	if r.isBuiltinType(typeID) {
		return "-"
	}

	if structType, typeName, _ := r.getStructType(typeID, pkgPath); structType != nil {
		if typeName == "" {
			// Если typeName пустой, используем typeID
			typeName = typeID
		}
		typeAnchor := typeAnchorID(typeName)
		if r.typeAnchorsSet != nil && !r.typeAnchorsSet[typeAnchor] {
			return "-"
		}
		typeStr := r.goTypeString(typeID, pkgPath)
		return fmt.Sprintf("[%s](#%s)", typeStr, typeAnchor)
	}

	typ, ok := r.project.Types[typeID]
	if ok && typ.ImportPkgPath != "" && !r.isTypeFromCurrentProject(typ.ImportPkgPath) {
		importPath := typ.ImportPkgPath
		libURL := fmt.Sprintf("https://pkg.go.dev/%s", importPath)
		typeStr := r.goTypeString(typeID, pkgPath)
		return fmt.Sprintf("[%s](%s)", typeStr, libURL)
	}

	return "-"
}

func (r *ClientRenderer) renderStructTypeTable(md *markdown.Markdown, structType *model.Type, typeName string, pkgPath string) {
	typeAnchor := typeAnchorID(typeName)
	md.PlainText(fmt.Sprintf("<a id=\"%s\"></a>", typeAnchor))
	md.LF()
	md.H4(typeName)

	rows := make([][]string, 0)
	for _, field := range structType.StructFields {
		fieldName, _ := r.jsonName(field)
		if fieldName == "-" {
			continue
		}

		// Тип поля
		typeStr := r.goTypeStringFromStructField(field, pkgPath)

		// Описание
		fieldTags := r.parseTagsFromDocs(strings.Join(field.Docs, "\n"))
		fieldDesc := fieldTags[tagDesc]

		isRequired := fieldTags[tagRequired] != ""
		requiredStr := "Нет"
		if isRequired {
			requiredStr = "Да"
		}

		isNullable := field.NumberOfPointers > 0
		nullableStr := "Нет"
		if isNullable {
			nullableStr = "Да"
		}

		hasOmitempty := false
		if tagValues, ok := field.Tags["json"]; ok {
			for _, val := range tagValues {
				if val == "omitempty" {
					hasOmitempty = true
					break
				}
			}
		}
		omitemptyStr := "Нет"
		if hasOmitempty {
			omitemptyStr = "Да"
		}

		typeLink := r.getTypeLinkFromStructField(field, pkgPath)

		if fieldDesc != "" {
			rows = append(rows, []string{
				markdown.Code(fieldName),
				markdown.Code(typeStr),
				fieldDesc,
				requiredStr,
				nullableStr,
				omitemptyStr,
				typeLink,
			})
		} else {
			rows = append(rows, []string{
				markdown.Code(fieldName),
				markdown.Code(typeStr),
				requiredStr,
				nullableStr,
				omitemptyStr,
				typeLink,
			})
		}
	}

	var headers []string
	hasDescriptions := false
	for _, field := range structType.StructFields {
		fieldTags := r.parseTagsFromDocs(strings.Join(field.Docs, "\n"))
		fieldDesc := fieldTags[tagDesc]
		if fieldDesc != "" {
			hasDescriptions = true
			break
		}
	}

	if hasDescriptions {
		headers = []string{"Поле", "Тип", "Описание", "Обязательное", "Nullable", "Omitempty", "Ссылка на тип"}
	} else {
		headers = []string{"Поле", "Тип", "Обязательное", "Nullable", "Omitempty", "Ссылка на тип"}
	}

	tableSet := markdown.TableSet{
		Header: headers,
		Rows:   rows,
	}
	md.Table(tableSet)
	md.LF()
}
