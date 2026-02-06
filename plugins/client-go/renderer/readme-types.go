// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"path/filepath"
	"strings"

	"tgp/internal/model"
)

type typeUsage struct {
	typeName     string
	pkgPath      string
	fullTypeName string
	locations    []string
}

func (r *ClientRenderer) collectStructTypes() map[string]*typeUsage {
	typeUsages := make(map[string]*typeUsage)

	for _, contractName := range r.ContractKeys() {
		contract := r.FindContract(contractName)
		if contract == nil {
			continue
		}
		for _, method := range contract.Methods {
			args := r.argsWithoutContext(method)
			for _, arg := range args {
				if structType, typeName, pkg := r.getStructType(arg.TypeID, contract.PkgPath); structType != nil {
					keyTypeName := typeName
					if typeName == "" {
						typeName = arg.Name
						keyTypeName = arg.Name
					}
					if strings.Contains(keyTypeName, ".") {
						parts := strings.Split(keyTypeName, ".")
						keyTypeName = parts[len(parts)-1]
					}
					key := fmt.Sprintf("%s.%s", pkg, keyTypeName)

					fullTypeNameForKey := typeName
					if fullTypeNameForKey == "" {
						fullTypeNameForKey = arg.Name
					}

					if _, ok := typeUsages[key]; !ok {
						typeUsages[key] = &typeUsage{
							typeName:     keyTypeName,
							pkgPath:      pkg,
							fullTypeName: fullTypeNameForKey,
							locations:    make([]string, 0),
						}
					}
					location := fmt.Sprintf("%s.%s.%s", contract.Name, method.Name, arg.Name)
					typeUsages[key].locations = append(typeUsages[key].locations, location)
				}
			}

			// Результаты
			results := r.resultsWithoutError(method)
			for _, result := range results {
				if structType, typeName, pkg := r.getStructType(result.TypeID, contract.PkgPath); structType != nil {
					keyTypeName := typeName
					if typeName == "" {
						typeName = result.Name
						keyTypeName = result.Name
					}
					if strings.Contains(keyTypeName, ".") {
						parts := strings.Split(keyTypeName, ".")
						keyTypeName = parts[len(parts)-1]
					}
					key := fmt.Sprintf("%s.%s", pkg, keyTypeName)

					fullTypeNameForKey := typeName
					if fullTypeNameForKey == "" {
						fullTypeNameForKey = result.Name
					}

					if _, ok := typeUsages[key]; !ok {
						typeUsages[key] = &typeUsage{
							typeName:     keyTypeName,
							pkgPath:      pkg,
							fullTypeName: fullTypeNameForKey,
							locations:    make([]string, 0),
						}
					}
					location := fmt.Sprintf("%s.%s.%s", contract.Name, method.Name, result.Name)
					typeUsages[key].locations = append(typeUsages[key].locations, location)
				}
			}
		}
	}

	return typeUsages
}

func (r *ClientRenderer) getStructType(typeID, pkgPath string) (structType *model.Type, typeName string, pkg string) {
	typ, ok := r.project.Types[typeID]
	if !ok {
		return nil, "", ""
	}

	if typ.Kind != model.TypeKindStruct || typ.TypeName == "" {
		return nil, "", ""
	}

	typeName = typ.TypeName
	pkg = typ.ImportPkgPath
	if pkg == "" {
		pkg = pkgPath
	}

	return typ, typeName, pkg
}

func (r *ClientRenderer) goTypeStringFromVariable(variable *model.Variable, pkgPath string) string {
	return r.variableToGoTypeString(variable, pkgPath)
}

func (r *ClientRenderer) goTypeStringFromStructField(field *model.StructField, pkgPath string) string {
	return r.typeRefToGoTypeString(&field.TypeRef, pkgPath)
}

func (r *ClientRenderer) goTypeString(typeID, pkgPath string) string {
	typ, ok := r.project.Types[typeID]
	if !ok {
		// Тип не найден - возможно, это встроенный тип
		if r.isBuiltinType(typeID) {
			return typeID
		}

		// Если typeID содержит ":", это импортированный тип
		if strings.Contains(typeID, ":") {
			parts := strings.SplitN(typeID, ":", 2)
			if len(parts) == 2 {
				importPkg := parts[0]
				typeName := parts[1]
				baseName := filepath.Base(importPkg)
				if baseName == "" {
					baseName = importPkg
				}
				return fmt.Sprintf("%s.%s", baseName, typeName)
			}
		}

		return typeID
	}

	// Тип найден в project.Types

	// Сначала проверяем импортированные типы (имеют ImportPkgPath)
	if typ.ImportPkgPath != "" {
		alias := typ.ImportAlias
		if alias == "" {
			alias = filepath.Base(typ.ImportPkgPath)
		}

		typeName := typ.TypeName
		if typeName == "" {
			if strings.Contains(typeID, ":") {
				parts := strings.SplitN(typeID, ":", 2)
				if len(parts) == 2 {
					typeName = parts[1]
				}
			}
		}

		if typeName != "" {
			return fmt.Sprintf("%s.%s", alias, typeName)
		}
	}

	switch typ.Kind {
	case model.TypeKindString, model.TypeKindInt, model.TypeKindInt8, model.TypeKindInt16, model.TypeKindInt32, model.TypeKindInt64,
		model.TypeKindUint, model.TypeKindUint8, model.TypeKindUint16, model.TypeKindUint32, model.TypeKindUint64,
		model.TypeKindFloat32, model.TypeKindFloat64, model.TypeKindBool, model.TypeKindByte, model.TypeKindRune, model.TypeKindError, model.TypeKindAny:
		if typ.ImportPkgPath != "" && typ.TypeName != "" {
			alias := typ.ImportAlias
			if alias == "" {
				alias = filepath.Base(typ.ImportPkgPath)
			}
			return fmt.Sprintf("%s.%s", alias, typ.TypeName)
		}
		return string(typ.Kind)
	case model.TypeKindStruct:
		structName := typ.TypeName
		if structName == "" {
			if strings.Contains(typeID, ":") {
				parts := strings.SplitN(typeID, ":", 2)
				if len(parts) == 2 {
					structName = parts[1]
				}
			}
		}

		if typ.ImportPkgPath != "" {
			alias := typ.ImportAlias
			if alias == "" {
				alias = filepath.Base(typ.ImportPkgPath)
			}
			if structName != "" {
				return fmt.Sprintf("%s.%s", alias, structName)
			}
		}

		if structName != "" {
			return structName
		}
	case model.TypeKindArray:
		if typ.IsSlice {
			elemType := r.goTypeString(typ.ArrayOfID, pkgPath)
			return fmt.Sprintf("[]%s", elemType)
		}
		elemType := r.goTypeString(typ.ArrayOfID, pkgPath)
		return fmt.Sprintf("[%d]%s", typ.ArrayLen, elemType)
	case model.TypeKindMap:
		keyType := r.typeRefToGoTypeString(typ.MapKey, pkgPath)
		valueType := r.typeRefToGoTypeString(typ.MapValue, pkgPath)
		return fmt.Sprintf("map[%s]%s", keyType, valueType)
	}

	// Если TypeName задан, используем его
	if typ.TypeName != "" {
		if typ.ImportPkgPath != "" {
			alias := typ.ImportAlias
			if alias == "" {
				alias = filepath.Base(typ.ImportPkgPath)
			}
			return fmt.Sprintf("%s.%s", alias, typ.TypeName)
		}
		return typ.TypeName
	}

	// Если ничего не помогло, пытаемся извлечь из typeID
	if strings.Contains(typeID, ":") {
		parts := strings.SplitN(typeID, ":", 2)
		if len(parts) == 2 {
			baseName := filepath.Base(parts[0])
			if baseName == "" {
				baseName = parts[0]
			}
			return fmt.Sprintf("%s.%s", baseName, parts[1])
		}
	}

	return typeID
}

func (r *ClientRenderer) jsonName(field *model.StructField) (value string, inline bool) {
	if tagValues, ok := field.Tags["json"]; ok {
		for _, val := range tagValues {
			if val == "inline" {
				inline = true
				continue
			}
			if val != "omitempty" && val != "-" {
				value = val
			}
		}
	}
	if value == "" {
		value = field.Name
	}
	return
}
