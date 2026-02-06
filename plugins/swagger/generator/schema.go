// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"path/filepath"
	"strings"

	"tgp/internal/common"
	"tgp/internal/model"
	"tgp/internal/tags"
	"tgp/plugins/swagger/types"
)

func (g *generator) registerStruct(name string, pkgPath string, methodTags tags.DocTags, variables []*model.Variable) {

	if len(variables) == 0 {
		g.schemas[name] = types.Schema{Type: "object"}
		return
	}

	var required []string
	properties := make(types.Properties)

	isArgument := !strings.Contains(name, "Response") && !strings.Contains(name, "response")

	for _, variable := range variables {
		if g.isContextType(variable.TypeID) {
			continue
		}
		if variable.TypeID == "error" {
			continue
		}

		jsonName := g.getJSONFieldName(variable)
		if jsonName == "" || jsonName == "-" {
			jsonName = types.ToLowerCamel(variable.Name)
		}

		schema := g.variableToSchema(variable, pkgPath, isArgument)
		if schema != nil {
			properties[jsonName] = *schema
			if variable.Annotations != nil && variable.Annotations.IsSet("required") {
				required = append(required, jsonName)
			}
		}
	}

	schema := types.Schema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}

	if desc := methodTags.Value("desc", ""); desc != "" {
		schema.Description = desc
	}

	g.schemas[name] = schema
}

func (g *generator) variableToSchema(variable *model.Variable, pkgPath string, isArgument bool) (schema *types.Schema) {
	return g.typeRefToSchema(&variable.TypeRef, variable.Annotations, pkgPath, isArgument)
}

func (g *generator) typeRefToSchema(typeRef *model.TypeRef, annotations tags.DocTags, pkgPath string, isArgument bool) (schema *types.Schema) {
	if typeRef == nil {
		return nil
	}

	if typeRef.IsSlice || typeRef.ArrayLen > 0 {
		itemTypeRef := &model.TypeRef{
			TypeID:           typeRef.TypeID,
			NumberOfPointers: typeRef.NumberOfPointers,
		}
		itemSchema := g.typeRefToSchema(itemTypeRef, annotations, pkgPath, isArgument)
		if itemSchema == nil {
			return nil
		}
		return &types.Schema{
			Type:  "array",
			Items: itemSchema,
		}
	}

	if typeRef.MapKey != nil && typeRef.MapValue != nil {
		valueSchema := g.typeRefToSchema(typeRef.MapValue, annotations, pkgPath, isArgument)
		if valueSchema == nil {
			return nil
		}
		return &types.Schema{
			Type:                 "object",
			AdditionalProperties: valueSchema,
		}
	}

	if typeRef.TypeID == typeIDIOReader || typeRef.TypeID == typeIDIOReadCloser {
		return &types.Schema{Type: "string", Format: "binary"}
	}

	typeInfo, found := g.project.Types[typeRef.TypeID]

	if !found {
		if types.IsExcludedTypeID(typeRef.TypeID, g.project) {
			if schema = g.basicTypeToSchema(typeRef.TypeID, annotations); schema == nil {
				return
			}
			if typeRef.NumberOfPointers > 0 {
				return &types.Schema{
					OneOf: []types.Schema{
						*schema,
						{Nullable: true},
					},
				}
			}
			return
		}
		if schema = g.basicTypeToSchema(typeRef.TypeID, annotations); schema == nil {
			return
		}
		if typeRef.NumberOfPointers > 0 {
			return &types.Schema{
				OneOf: []types.Schema{
					*schema,
					{Nullable: true},
				},
			}
		}
		return
	}

	hasCustomMarshaler := g.hasMarshaler(typeInfo, isArgument)
	isExcluded := types.IsExplicitlyExcludedType(typeInfo)

	if hasCustomMarshaler && !isExcluded {
		typeName := g.normalizeTypeName(typeInfo, typeRef.TypeID, pkgPath)

		if _, found := g.schemas[typeName]; found {
			result := &types.Schema{
				Ref: fmt.Sprintf(componentsSchemasPrefix+"%s", typeName),
			}
			if typeRef.NumberOfPointers > 0 {
				return &types.Schema{
					OneOf: []types.Schema{
						*result,
						{Nullable: true},
					},
				}
			}
			return result
		}

		anySchema := types.Schema{
			Type: "object",
		}
		anySchema.AdditionalProperties = &types.Schema{Type: "object"}

		g.schemas[typeName] = anySchema

		result := &types.Schema{
			Ref: fmt.Sprintf(componentsSchemasPrefix+"%s", typeName),
		}
		if typeRef.NumberOfPointers > 0 {
			return &types.Schema{
				OneOf: []types.Schema{
					*result,
					{Nullable: true},
				},
			}
		}
		return result
	}

	if types.IsExplicitlyExcludedType(typeInfo) {
		if schema = g.basicTypeToSchema(typeRef.TypeID, annotations); schema == nil {
			return
		}
		if typeRef.NumberOfPointers > 0 {
			return &types.Schema{
				OneOf: []types.Schema{
					*schema,
					{Nullable: true},
				},
			}
		}
		return
	}

	if typeInfo.ArrayOfID != "" && types.IsExcludedTypeID(typeInfo.ArrayOfID, g.project) {
		if schema = g.basicTypeToSchema(typeInfo.ArrayOfID, annotations); schema == nil {
			return
		}
		if typeRef.NumberOfPointers > 0 {
			return &types.Schema{
				OneOf: []types.Schema{
					*schema,
					{Nullable: true},
				},
			}
		}
		return
	}

	if types.IsExcludedTypeID(typeRef.TypeID, g.project) {
		if schema = g.basicTypeToSchema(typeRef.TypeID, annotations); schema == nil {
			return
		}
		if typeRef.NumberOfPointers > 0 {
			return &types.Schema{
				OneOf: []types.Schema{
					*schema,
					{Nullable: true},
				},
			}
		}
		return
	}

	switch typeInfo.Kind {
	case model.TypeKindAlias:
		if typeInfo.AliasOf != "" {
			baseTypeRef := &model.TypeRef{TypeID: typeInfo.AliasOf}
			baseSchema := g.typeRefToSchema(baseTypeRef, annotations, pkgPath, isArgument)
			if baseSchema == nil {
				return
			}

			typeName := g.normalizeTypeName(typeInfo, typeRef.TypeID, pkgPath)

			if _, found := g.schemas[typeName]; found {
				result := &types.Schema{
					Ref: fmt.Sprintf(componentsSchemasPrefix+"%s", typeName),
				}
				if typeRef.NumberOfPointers > 0 {
					return &types.Schema{
						OneOf: []types.Schema{
							*result,
							{Nullable: true},
						},
					}
				}
				return result
			}

			var aliasSchema types.Schema
			if baseSchema.Ref != "" {
				aliasSchema = types.Schema{
					AllOf: []types.Schema{
						{
							Ref: baseSchema.Ref,
						},
					},
				}
			} else {
				aliasSchema = *baseSchema
			}

			g.schemas[typeName] = aliasSchema

			result := &types.Schema{
				Ref: fmt.Sprintf(componentsSchemasPrefix+"%s", typeName),
			}
			if typeRef.NumberOfPointers > 0 {
				return &types.Schema{
					OneOf: []types.Schema{
						*result,
						{Nullable: true},
					},
				}
			}
			return result
		}
		return g.basicTypeToSchema(typeRef.TypeID, annotations)

	case model.TypeKindStruct:
		if schema = g.structTypeToSchema(typeInfo, annotations); schema == nil {
			return
		}
		if typeRef.NumberOfPointers > 0 {
			return &types.Schema{
				OneOf: []types.Schema{
					*schema,
					{Nullable: true},
				},
			}
		}
		return

	case model.TypeKindArray:
		if typeInfo.ArrayOfID != "" {
			itemTypeRef := &model.TypeRef{TypeID: typeInfo.ArrayOfID}
			itemSchema := g.typeRefToSchema(itemTypeRef, annotations, pkgPath, isArgument)
			if itemSchema == nil {
				return
			}
			arraySchema := &types.Schema{
				Type:  "array",
				Items: itemSchema,
			}
			if typeRef.NumberOfPointers > 0 {
				return &types.Schema{
					OneOf: []types.Schema{
						*arraySchema,
						{Nullable: true},
					},
				}
			}
			return arraySchema
		}

	case model.TypeKindMap:
		if typeInfo.MapValue != nil {
			valueSchema := g.typeRefToSchema(typeInfo.MapValue, annotations, pkgPath, isArgument)
			if valueSchema == nil {
				return
			}
			mapSchema := &types.Schema{
				Type:                 "object",
				AdditionalProperties: valueSchema,
			}
			if typeRef.NumberOfPointers > 0 {
				return &types.Schema{
					OneOf: []types.Schema{
						*mapSchema,
						{Nullable: true},
					},
				}
			}
			return mapSchema
		}

	case model.TypeKindString, model.TypeKindInt, model.TypeKindInt8, model.TypeKindInt16,
		model.TypeKindInt32, model.TypeKindInt64, model.TypeKindUint, model.TypeKindUint8,
		model.TypeKindUint16, model.TypeKindUint32, model.TypeKindUint64,
		model.TypeKindFloat32, model.TypeKindFloat64, model.TypeKindBool,
		model.TypeKindByte, model.TypeKindRune, model.TypeKindError, model.TypeKindAny:
		if schema = g.basicTypeToSchema(typeRef.TypeID, annotations); schema == nil {
			return
		}
		if typeRef.NumberOfPointers > 0 {
			return &types.Schema{
				OneOf: []types.Schema{
					*schema,
					{Nullable: true},
				},
			}
		}
		return

	default:
		if typeInfo.TypeName != "" {
			typeName := g.normalizeTypeName(typeInfo, typeRef.TypeID, pkgPath)
			refSchema := &types.Schema{
				Ref: fmt.Sprintf(componentsSchemasPrefix+"%s", typeName),
			}
			if typeRef.NumberOfPointers > 0 {
				return &types.Schema{
					OneOf: []types.Schema{
						*refSchema,
						{Nullable: true},
					},
				}
			}
			return refSchema
		}
	}

	if schema = g.basicTypeToSchema(typeRef.TypeID, annotations); schema == nil {
		return
	}
	if typeRef.NumberOfPointers > 0 {
		return &types.Schema{
			OneOf: []types.Schema{
				*schema,
				{Nullable: true},
			},
		}
	}
	return
}

func (g *generator) structTypeToSchema(typeInfo *model.Type, varTags tags.DocTags) (schema *types.Schema) {

	var typeID string
	for id, typ := range common.SortedPairs(g.project.Types) {
		if typ == typeInfo {
			typeID = id
			break
		}
	}
	if typeID == "" {
		if typeInfo.ImportPkgPath != "" && typeInfo.TypeName != "" {
			typeID = fmt.Sprintf("%s:%s", typeInfo.ImportPkgPath, typeInfo.TypeName)
		} else {
			typeID = typeInfo.TypeName
		}
	}
	typeName := g.normalizeTypeName(typeInfo, typeID, typeInfo.ImportPkgPath)

	if _, found := g.schemas[typeName]; found {
		return &types.Schema{
			Ref: fmt.Sprintf(componentsSchemasPrefix+"%s", typeName),
		}
	}

	g.schemas[typeName] = types.Schema{
		Type:       "object",
		Properties: make(types.Properties),
	}

	var required []string
	properties := make(types.Properties)

	for _, field := range typeInfo.StructFields {
		jsonName := g.getJSONFieldNameFromStructField(field)
		if jsonName == "" || jsonName == "-" {
			continue
		}

		fieldVar := &model.Variable{
			TypeRef:     field.TypeRef,
			Name:        field.Name,
			Annotations: tags.ParseTags(field.Docs),
		}

		fieldSchema := g.variableToSchema(fieldVar, typeInfo.ImportPkgPath, true)
		if fieldSchema != nil && !fieldSchema.IsEmpty() {
			properties[jsonName] = *fieldSchema
			if field.NumberOfPointers > 0 {
				continue
			}
			if jsonTag, ok := field.Tags["json"]; ok && len(jsonTag) > 0 {
				hasOmitempty := false
				for _, tag := range jsonTag {
					if strings.TrimSpace(tag) == "omitempty" {
						hasOmitempty = true
						break
					}
				}
				if !hasOmitempty {
					required = append(required, jsonName)
				}
			} else {
				required = append(required, jsonName)
			}
		}
	}

	schemaObj := types.Schema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}

	g.schemas[typeName] = schemaObj

	return &types.Schema{
		Ref: fmt.Sprintf(componentsSchemasPrefix+"%s", typeName),
	}
}

func (g *generator) basicTypeToSchema(typeID string, varTags tags.DocTags) (result *types.Schema) {

	schema := &types.Schema{}

	if varTags != nil {
		if desc := varTags.Value("desc", ""); desc != "" {
			schema.Description = desc
		}
		if example := varTags.Value("example", ""); example != "" {
			schema.Example = example
		}
		if format := varTags.Value("format", ""); format != "" {
			schema.Format = format
		}
		if enums := varTags.Value("enums", ""); enums != "" {
			schema.Enum = strings.Split(enums, ",")
		}
		if newType := varTags.Value("type", ""); newType != "" {
			schema.Type = newType
			return schema
		}
	}

	openAPIType, format := types.GetSerializationFormatForTypeID(typeID, g.project)
	schema.Type = openAPIType
	if format != "" {
		schema.Format = format
	}

	return schema
}

func (g *generator) normalizeTypeName(typeInfo *model.Type, typeID string, defaultPkgPath string) (typeName string) {

	var pkgPath string

	switch {
	case typeInfo.TypeName != "":
		typeName = typeInfo.TypeName
		pkgPath = typeInfo.ImportPkgPath
		if pkgPath == "" {
			if strings.Contains(typeID, ":") {
				parts := strings.Split(typeID, ":")
				if len(parts) == 2 {
					pkgPath = parts[0]
				}
			}
		}
	default:
		if strings.Contains(typeID, ":") {
			parts := strings.Split(typeID, ":")
			if len(parts) == 2 {
				pkgPath = parts[0]
				typeName = parts[1]
			} else {
				typeName = typeID
			}
		} else {
			typeName = typeID
		}
	}

	if pkgPath == "" {
		pkgPath = defaultPkgPath
	}

	if pkgPath != "" && !strings.Contains(typeName, ".") {
		var pkgName string
		for _, typ := range common.SortedPairs(g.project.Types) {
			if typ.ImportPkgPath == pkgPath && typ.TypeName == typeName {
				if typ.PkgName != "" {
					pkgName = typ.PkgName
					break
				}
			}
		}
		if pkgName == "" {
			basePkg := filepath.Base(pkgPath)
			if strings.Contains(pkgPath, "/") {
				parts := strings.Split(pkgPath, "/")
				if len(parts) > 0 {
					basePkg = parts[len(parts)-1]
				}
			}
			pkgName = basePkg
		}
		typeName = fmt.Sprintf("%s.%s", pkgName, typeName)
	}

	return typeName
}

func (g *generator) getJSONFieldName(variable *model.Variable) (jsonName string) {

	fieldName := types.ToLowerCamel(variable.Name)

	if variable.Annotations != nil {
		if jsonTag := variable.Annotations.Value("json", ""); jsonTag != "" {
			jsonParts := strings.Split(jsonTag, ",")
			return jsonParts[0]
		}
	}

	if strings.HasPrefix(fieldName, strings.ToLower(string(fieldName[0]))) {
		return ""
	}

	return fieldName
}

func (g *generator) getJSONFieldNameFromStructField(field *model.StructField) (jsonName string) {

	fieldName := types.ToLowerCamel(field.Name)

	if jsonTag, ok := field.Tags["json"]; ok && len(jsonTag) > 0 {
		jsonName = strings.TrimSpace(jsonTag[0])
		if jsonName != "" && jsonName != "-" {
			return
		}
		return ""
	}

	if strings.HasPrefix(field.Name, strings.ToLower(string(field.Name[0]))) {
		return ""
	}

	return fieldName
}

func (g *generator) hasMarshaler(typ *model.Type, isArgument bool) (hasMarshaler bool) {

	if typ == nil {
		return
	}

	if len(typ.ImplementsInterfaces) > 0 {
		for _, iface := range typ.ImplementsInterfaces {
			if isArgument {
				if iface == "encoding/json:Marshaler" || iface == "encoding/text:Marshaler" {
					return true
				}
			} else {
				if iface == "encoding/json:Unmarshaler" || iface == "encoding/text:Unmarshaler" {
					return true
				}
			}
		}
	}

	if typ.Kind == model.TypeKindAlias && typ.AliasOf != "" {
		if baseType, exists := g.project.Types[typ.AliasOf]; exists {
			return g.hasMarshaler(baseType, isArgument)
		}
	}

	return false
}

func (g *generator) isContextType(typeID string) (isContext bool) {
	return typeID == "context:Context" || typeID == "Context"
}

func (g *generator) toSchema(typeName string) (schema types.Schema) {

	typeName = strings.TrimPrefix(typeName, "*")
	return types.Schema{
		Ref: fmt.Sprintf(componentsSchemasPrefix+"%s", typeName),
	}
}
