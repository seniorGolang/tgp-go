// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/ast"
	"go/token"
	"go/types"
	"log/slog"
	"strings"

	"tgp/core/i18n"
	"tgp/internal/model"
	"tgp/internal/tags"
)

func fillStructFields(structType *types.Struct, pkgPath string, imports map[string]string, project *model.Project, coreType *model.Type, loader *AutonomousPackageLoader, _ ...map[string]bool) {

	if structType == nil {
		return
	}

	coreType.StructFields = make([]*model.StructField, 0)

	var astStructType *ast.StructType
	if pkgInfo, ok := loader.GetPackage(pkgPath); ok && pkgInfo != nil {
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

		typeInfo := convertFieldType(fieldType, pkgPath, imports, project, loader)

		fieldTags := make(map[string][]string)
		if fieldTag := structType.Tag(i); fieldTag != "" {
			fieldTags = parseStructTag(fieldTag)
		} else if astStructType != nil {
			astTags := extractTagsFromASTStruct(astStructType, fieldName)
			if len(astTags) > 0 {
				fieldTags = astTags
			}
		}

		var docs, directives []string
		var annotations tags.DocTags
		if astStructType != nil {
			astLines := extractDocsFromASTStruct(astStructType, fieldName)
			if len(astLines) > 0 {
				docs, directives = splitDocsAndDirectives(astLines)
				annotations = tags.ParseTags(astLines)
			}
		}

		structField := &model.StructField{
			TypeRef:     *fieldTypeInfoToTypeRef(typeInfo),
			Name:        fieldName,
			Tags:        fieldTags,
			Docs:        docs,
			Directives:  directives,
			Annotations: annotations,
		}

		coreType.StructFields = append(coreType.StructFields, structField)
	}
}

type fieldTypeInfo = typeConversionInfo

func convertFieldType(typ types.Type, pkgPath string, imports map[string]string, project *model.Project, loader *AutonomousPackageLoader) (info fieldTypeInfo) {

	if typ == nil {
		slog.Error(i18n.Msg("failed to convert field type"), slog.String("package", pkgPath), slog.String("reason", "nil type"))
		return
	}
	var ok bool
	if info, ok = typeRefFromTypes(typ, pkgPath, imports, project, loader); !ok {
		slog.Error(i18n.Msg("failed to convert field type"), slog.String("package", pkgPath), slog.String("goType", typ.String()))
	}
	return
}

func fieldTypeInfoToTypeRef(info fieldTypeInfo) (ref *model.TypeRef) {

	return conversionInfoToTypeRef(info)
}

func findASTStructType(file *ast.File, typeName string, typeInfo *types.Info) (foundStruct *ast.StructType) {

	if file == nil {
		return
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok {
			if genDecl.Tok == token.TYPE {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if typeSpec.Name != nil && typeSpec.Name.Name == typeName {
							if structType, ok := typeSpec.Type.(*ast.StructType); ok {
								foundStruct = structType
								return false
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

func findTypeSpecAndGenDecl(file *ast.File, typeName string) (typeSpec *ast.TypeSpec, genDecl *ast.GenDecl) {

	if file == nil {
		return
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if g, ok := n.(*ast.GenDecl); ok && g.Tok == token.TYPE {
			for _, spec := range g.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name != nil && ts.Name.Name == typeName {
					typeSpec = ts
					genDecl = g
					return false
				}
			}
		}
		return true
	})

	return
}

func getTypeDocs(loader *AutonomousPackageLoader, pkgPath string, typeName string) (docs, directives []string) {

	if loader == nil || pkgPath == "" || typeName == "" {
		return
	}

	pkgInfo, ok := loader.GetPackage(pkgPath)
	if !ok || pkgInfo == nil || len(pkgInfo.Files) == 0 {
		return
	}

	for _, file := range pkgInfo.Files {
		ts, g := findTypeSpecAndGenDecl(file, typeName)
		if ts != nil && g != nil {
			lines := extractComments(g.Doc, ts.Doc, ts.Comment)
			return splitDocsAndDirectives(lines)
		}
	}

	return
}

func extractTagsFromASTStruct(structType *ast.StructType, fieldName string) (tags map[string][]string) {

	if structType == nil || structType.Fields == nil {
		tags = make(map[string][]string)
		return
	}

	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			if name.Name == fieldName {
				if field.Tag != nil {
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

func extractDocsFromASTStruct(structType *ast.StructType, fieldName string) (docs []string) {

	if structType == nil || structType.Fields == nil {
		return
	}

	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			if name.Name == fieldName {
				docs = extractComments(field.Doc, field.Comment)
				return
			}
		}
	}

	return
}

func parseStructTag(tag string) (result map[string][]string) {

	result = make(map[string][]string)
	if tag == "" {
		return
	}

	for tag != "" {
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		keyEnd := 0
		for keyEnd < len(tag) && tag[keyEnd] != ':' {
			keyEnd++
		}
		if keyEnd == 0 || keyEnd == len(tag) {
			break
		}
		key := tag[:keyEnd]
		tag = tag[keyEnd+1:]

		i = 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" || tag[0] != '"' {
			break
		}

		tag = tag[1:]
		valueEnd := 0
		for valueEnd < len(tag) && tag[valueEnd] != '"' {
			if tag[valueEnd] == '\\' && valueEnd+1 < len(tag) {
				valueEnd += 2
			} else {
				valueEnd++
			}
		}
		if valueEnd == len(tag) {
			break
		}
		value := tag[:valueEnd]
		tag = tag[valueEnd+1:]

		result[key] = strings.Split(value, ",")
	}

	return
}
