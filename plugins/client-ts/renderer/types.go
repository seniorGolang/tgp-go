// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import "fmt"

type typeDefTs struct {
	name       string
	kind       string
	typeName   string
	nullable   bool
	value      any
	properties map[string]typeDefTs
	// Для импортированных типов храним информацию о пакете и имени типа
	importPkg  string // Пакет импорта (например, "dto")
	importName string // Имя типа в пакете (например, "SomeStruct")
}

func (def typeDefTs) def() (prop string) {

	switch def.kind {
	case "map":
		if def.properties == nil {
			return "Record<string, any>"
		}
		key := def.properties["key"]
		value := def.properties["value"]
		keyType := "string"
		valueType := "any"
		if key.kind != "" {
			keyType = castTypeTs(key.typeLink())
		}
		if value.kind != "" {
			valueType = castTypeTs(value.typeLink())
		}
		return fmt.Sprintf("Record<%s, %s>", keyType, valueType)
	case "array":
		if def.properties == nil {
			return "any[]"
		}
		item := def.properties["item"]
		itemType := "any"
		if item.kind != "" {
			itemType = item.typeLink()
		}
		return fmt.Sprintf("%s[]", itemType)
	case "struct":
		return def.name
	case "scalar":
		// Для скалярных типов (алиасов) используем базовый TypeScript тип
		// Если тип импортирован, используем namespace (например, "dto.UserID")
		if def.importPkg != "" && def.importName != "" {
			return fmt.Sprintf("%s.%s", def.importPkg, def.importName)
		}
		return def.typeName
	default:
		return castTypeTs(def.kind)
	}
}

func (def typeDefTs) typeLink() (link string) {

	switch def.kind {
	case "map":
		if def.importPkg != "" && def.importName != "" {
			link = fmt.Sprintf("%s.%s", def.importPkg, def.importName)
			break
		}
		if def.properties == nil {
			link = "Record<string, any>"
			break
		}
		keyType := "string"
		valueType := "any"
		if key, ok := def.properties["key"]; ok {
			keyType = castTypeTs(key.typeLink())
		}
		if value, ok := def.properties["value"]; ok {
			valueType = castTypeTs(value.typeLink())
		}
		link = fmt.Sprintf("Record<%s, %s>", keyType, valueType)
	case "array":
		if def.properties == nil {
			link = "any[]"
			break
		}
		itemType := "any"
		if item, ok := def.properties["item"]; ok {
			itemType = castTypeTs(item.typeLink())
		}
		link = fmt.Sprintf("%s[]", itemType)
	case "scalar":
		if def.importPkg != "" && def.importName != "" {
			link = fmt.Sprintf("%s.%s", def.importPkg, def.importName)
			break
		}
		link = def.typeName
	case "struct":
		if def.importPkg != "" && def.importName != "" {
			link = fmt.Sprintf("%s.%s", def.importPkg, def.importName)
			break
		}
		if def.name == "" && def.importName != "" {
			link = def.importName
			break
		}
		if def.name == "" {
			link = "any"
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
			link = "any"
		}
	default:
		link = castTypeTs(def.name)
	}
	if def.nullable && link != "any" {
		return link + " | null"
	}
	return link
}
