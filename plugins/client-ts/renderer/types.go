// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import "fmt"

type typeDefTS struct {
	name       string
	kind       string
	typeName   string
	nullable   bool
	value      any
	properties map[string]typeDefTS
	// Для импортированных типов храним информацию о пакете и имени типа
	importPkg  string // Пакет импорта (например, "dto")
	importName string // Имя типа в пакете (например, "SomeStruct")
}

func (def typeDefTS) def() (prop string) {

	switch def.kind {
	case kindMap:
		if def.properties == nil {
			return "Record<string, any>"
		}
		key := def.properties["key"]
		value := def.properties["value"]
		keyType := typeString
		valueType := typeAny
		if key.kind != "" {
			keyType = castTypeTS(key.typeLink())
		}
		if value.kind != "" {
			valueType = castTypeTS(value.typeLink())
		}
		return fmt.Sprintf("Record<%s, %s>", keyType, valueType)
	case kindArray:
		if def.properties == nil {
			return "any[]"
		}
		item := def.properties["item"]
		itemType := typeAny
		if item.kind != "" {
			itemType = item.typeLink()
		}
		return itemType + "[]"
	case kindStruct:
		return def.name
	case kindScalar:
		// Для скалярных типов (алиасов) используем базовый TypeScript тип
		// Если тип импортирован, используем namespace (например, "dto.UserID")
		if def.importPkg != "" && def.importName != "" {
			return fmt.Sprintf("%s.%s", def.importPkg, def.importName)
		}
		return def.typeName
	default:
		return castTypeTS(def.kind)
	}
}

func (def typeDefTS) typeLink() (link string) {

	switch def.kind {
	case kindMap:
		if def.importPkg != "" && def.importName != "" {
			link = fmt.Sprintf("%s.%s", def.importPkg, def.importName)
			break
		}
		if def.properties == nil {
			link = "Record<string, any>"
			break
		}
		keyType := typeString
		valueType := typeAny
		if key, ok := def.properties["key"]; ok {
			keyType = castTypeTS(key.typeLink())
		}
		if value, ok := def.properties["value"]; ok {
			valueType = castTypeTS(value.typeLink())
		}
		link = fmt.Sprintf("Record<%s, %s>", keyType, valueType)
	case kindArray:
		if def.properties == nil {
			link = "any[]"
			break
		}
		itemType := typeAny
		if item, ok := def.properties["item"]; ok {
			itemType = castTypeTS(item.typeLink())
		}
		link = itemType + "[]"
	case kindScalar:
		if def.importPkg != "" && def.importName != "" {
			link = fmt.Sprintf("%s.%s", def.importPkg, def.importName)
			break
		}
		link = def.typeName
	case kindStruct:
		if def.importPkg != "" && def.importName != "" {
			link = fmt.Sprintf("%s.%s", def.importPkg, def.importName)
			break
		}
		if def.name == "" && def.importName != "" {
			link = def.importName
			break
		}
		if def.name == "" {
			link = typeAny
			break
		}
		link = def.name
	case "":
		switch {
		case def.name != "":
			link = def.name
		case def.typeName != "":
			link = def.typeName
		default:
			link = typeAny
		}
	default:
		link = castTypeTS(def.name)
	}
	if def.nullable && link != typeAny {
		return link + " | null"
	}
	return link
}
