// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package types

import (
	"strings"

	"tgp/internal/model"
)

const (
	pkgTime            = "time"
	typeTime           = "Time"
	typeDuration       = "Duration"
	pkgMathBig         = "math/big"
	typeInt            = "Int"
	typeFloat          = "Float"
	typeRat            = "Rat"
	pkgDatabaseSQL     = "database/sql"
	pkgGureguNull      = "guregu/null"
	interfaceMarshaler = "encoding/json:Marshaler"
)

var uuidPackages = []string{
	"github.com/google/uuid",
	"github.com/satori/go.uuid",
	"gopkg.in/guregu/null.v4",
}

func IsExcludedTypeID(typeID string, project *model.Project) (excluded bool) {
	if typeID == "" {
		return
	}

	if IsBuiltinTypeID(typeID) {
		return true
	}

	var typ *model.Type
	var exists bool
	if typ, exists = project.Types[typeID]; exists {
		return IsExplicitlyExcludedType(typ)
	}

	parts := strings.SplitN(typeID, ":", 2)
	if len(parts) != 2 {
		return
	}

	importPkgPath := parts[0]
	typeName := parts[1]

	if importPkgPath == pkgTime && typeName == typeTime {
		return true
	}

	if importPkgPath == pkgTime && typeName == typeDuration {
		return true
	}

	if strings.HasSuffix(typeName, "UUID") || typeName == "UUID" {
		if importPkgPath == "" {
			return true
		}
		for _, pkg := range uuidPackages {
			if strings.HasPrefix(importPkgPath, pkg) && typeName != typeTime {
				return true
			}
		}
		if strings.Contains(importPkgPath, "uuid") && typeName != typeTime {
			return true
		}
	}

	if strings.HasSuffix(typeName, "Decimal") {
		return true
	}

	if importPkgPath == pkgMathBig {
		if typeName == typeInt || typeName == typeFloat || typeName == typeRat {
			return true
		}
	}

	if importPkgPath == pkgDatabaseSQL {
		if strings.HasPrefix(typeName, "Null") {
			return true
		}
	}

	if strings.Contains(importPkgPath, pkgGureguNull) {
		return true
	}

	return
}

func IsBuiltinTypeID(typeID string) (isBuiltin bool) {
	switch typeID {
	case "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128",
		"bool", "byte", "rune", "error", "any":
		return true
	}
	return
}

func IsExplicitlyExcludedType(typ *model.Type) (excluded bool) {
	if typ == nil {
		return
	}

	if typ.ImportPkgPath == pkgTime && typ.TypeName == typeTime {
		return true
	}
	if typ.ImportPkgPath == "" && typ.TypeName == typeTime {
		return true
	}

	if typ.ImportPkgPath == pkgTime && typ.TypeName == typeDuration {
		return true
	}

	if strings.HasSuffix(typ.TypeName, "UUID") || typ.TypeName == "UUID" {
		if typ.ImportPkgPath == "" {
			return true
		}
		for _, pkg := range uuidPackages {
			if strings.HasPrefix(typ.ImportPkgPath, pkg) && typ.TypeName != typeTime {
				return true
			}
		}
		if strings.Contains(typ.ImportPkgPath, "uuid") && typ.TypeName != typeTime {
			return true
		}
	}

	if strings.HasSuffix(typ.TypeName, "Decimal") {
		return true
	}

	if typ.ImportPkgPath == pkgMathBig {
		if typ.TypeName == typeInt || typ.TypeName == typeFloat || typ.TypeName == typeRat {
			return true
		}
	}

	if typ.ImportPkgPath == pkgDatabaseSQL {
		if strings.HasPrefix(typ.TypeName, "Null") {
			return true
		}
	}

	if strings.Contains(typ.ImportPkgPath, pkgGureguNull) {
		return true
	}

	return
}

func GetSerializationFormatForTypeID(typeID string, project *model.Project) (openAPIType string, format string) {
	if IsBuiltinTypeID(typeID) {
		switch typeID {
		case "string":
			return "string", ""
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
			return "integer", ""
		case "float32", "float64":
			return "number", ""
		case "bool":
			return "boolean", ""
		case "byte":
			return "string", "byte"
		case "rune":
			return "string", ""
		case "error", "any":
			return "object", ""
		}
	}

	var typ *model.Type
	var exists bool
	if typ, exists = project.Types[typeID]; exists {
		return GetSerializationFormat(typ, project)
	}

	parts := strings.SplitN(typeID, ":", 2)
	if len(parts) == 2 {
		importPkgPath := parts[0]
		typeName := parts[1]

		if importPkgPath == pkgTime && typeName == typeTime {
			return "string", "date-time"
		}

		if importPkgPath == pkgTime && typeName == typeDuration {
			return "string", "duration"
		}

		if strings.HasSuffix(typeName, "UUID") || typeName == "UUID" {
			return "string", "uuid"
		}
	}

	return
}

func GetSerializationFormat(typ *model.Type, project *model.Project) (openAPIType string, format string) {
	if typ.ImportPkgPath == pkgTime && typ.TypeName == typeTime {
		return "string", "date-time"
	}

	if typ.ImportPkgPath == pkgTime && typ.TypeName == typeDuration {
		return "string", "duration"
	}

	if strings.Contains(typ.TypeName, "UUID") || strings.Contains(typ.ImportPkgPath, "uuid") {
		return "string", "uuid"
	}

	if containsString(typ.ImplementsInterfaces, interfaceMarshaler) {
		return "string", ""
	}

	return
}

func containsString(slice []string, str string) (found bool) {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return
}

func ToCamel(s string) (result string) {
	if s == "" {
		return s
	}

	if isAllUpper(s) {
		return s
	}

	result = strings.ToUpper(string(s[0])) + s[1:]

	parts := strings.FieldsFunc(result, func(r rune) bool {
		return r == '_' || r == '-'
	})
	if len(parts) > 1 {
		result = ""
		for _, part := range parts {
			if part != "" {
				result += strings.ToUpper(string(part[0])) + part[1:]
			}
		}
	}

	return
}

func ToLowerCamel(s string) (result string) {
	if s == "" {
		return s
	}

	if isAllUpper(s) {
		return s
	}

	result = strings.ToLower(string(s[0])) + s[1:]

	parts := strings.FieldsFunc(result, func(r rune) bool {
		return r == '_' || r == '-'
	})
	if len(parts) > 1 {
		result = parts[0]
		for i := 1; i < len(parts); i++ {
			if parts[i] != "" {
				result += strings.ToUpper(string(parts[i][0])) + parts[i][1:]
			}
		}
	}

	return
}

func isAllUpper(s string) (allUpper bool) {
	for _, v := range s {
		if v >= 'a' && v <= 'z' {
			return
		}
	}
	return true
}
