// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/ast"
	"go/types"
	"log/slog"

	"tgp/core/i18n"
	"tgp/internal/model"
	"tgp/internal/tags"
)

func convertMethod(methodName string, funcType *ast.FuncType, docs []string, contractID string, pkgPath string, imports map[string]string, typeInfo *types.Info, project *model.Project, loader *AutonomousPackageLoader) (method *model.Method) {

	methodDocs, methodDirectives := splitDocsAndDirectives(docs)
	methodAnnotations := tags.ParseTags(docs)
	method = &model.Method{
		Name:        methodName,
		ContractID:  contractID,
		Docs:        methodDocs,
		Directives:  methodDirectives,
		Annotations: methodAnnotations,
		Args:        make([]*model.Variable, 0),
		Results:     make([]*model.Variable, 0),
	}

	if funcType.Params != nil {
		for _, param := range funcType.Params.List {
			var convertedTypeInfo typeConversionInfo
			var typeOK bool
			if convertedTypeInfo, typeOK = typeRefFromExpr(param.Type, pkgPath, imports, project, loader, typeInfo); !typeOK {
				slog.Error(i18n.Msg("failed to convert type for parameter in method"), slog.String("method", methodName), slog.String("package", pkgPath), slog.String("expr", types.ExprString(param.Type)))
				continue
			}

			paramLines := extractComments(param.Doc, param.Comment)
			paramDocs, paramDirectives := splitDocsAndDirectives(paramLines)
			paramAnnotations := tags.ParseTags(paramLines)

			if len(param.Names) > 0 {
				for _, name := range param.Names {
					method.Args = append(method.Args, &model.Variable{
						TypeRef:     *conversionInfoToTypeRef(convertedTypeInfo),
						Name:        name.Name,
						Docs:        paramDocs,
						Directives:  paramDirectives,
						Annotations: paramAnnotations,
					})
				}
			} else {
				argName := fmt.Sprintf("arg%d", len(method.Args)+1)
				method.Args = append(method.Args, &model.Variable{
					TypeRef:     *conversionInfoToTypeRef(convertedTypeInfo),
					Name:        argName,
					Docs:        paramDocs,
					Directives:  paramDirectives,
					Annotations: paramAnnotations,
				})
			}
		}
	}

	if funcType.Results != nil {
		for _, result := range funcType.Results.List {
			var resultTypeInfo typeConversionInfo
			var typeOK bool
			if resultTypeInfo, typeOK = typeRefFromExpr(result.Type, pkgPath, imports, project, loader, typeInfo); !typeOK {
				slog.Error(i18n.Msg("failed to convert type for result in method"), slog.String("method", methodName), slog.String("package", pkgPath), slog.String("expr", types.ExprString(result.Type)))
				continue
			}

			resultLines := extractComments(result.Doc, result.Comment)
			resultDocs, resultDirectives := splitDocsAndDirectives(resultLines)
			resultAnnotations := tags.ParseTags(resultLines)

			if len(result.Names) > 0 {
				for _, name := range result.Names {
					method.Results = append(method.Results, &model.Variable{
						TypeRef:     *conversionInfoToTypeRef(resultTypeInfo),
						Name:        name.Name,
						Docs:        resultDocs,
						Directives:  resultDirectives,
						Annotations: resultAnnotations,
					})
				}
			} else {
				resultName := fmt.Sprintf("result%d", len(method.Results)+1)
				method.Results = append(method.Results, &model.Variable{
					TypeRef:     *conversionInfoToTypeRef(resultTypeInfo),
					Name:        resultName,
					Docs:        resultDocs,
					Directives:  resultDirectives,
					Annotations: resultAnnotations,
				})
			}
		}
	}

	method.Handler = model.HandlerInfoFromAnnotations(method.Annotations)

	return
}

func ensureTypeInProject(typeID string, typ types.Type, pkgPath string, imports map[string]string, project *model.Project, loader *AutonomousPackageLoader) (coreType *model.Type, err error) {

	if existing, exists := project.Types[typeID]; exists {
		return existing, nil
	}
	if loader == nil || loader.resolver == nil {
		return nil, fmt.Errorf("package loader is not initialized")
	}
	parts := splitTypeID(typeID)
	if len(parts) != 2 {
		return nil, nil
	}
	typeName := parts[1]
	if isBuiltinTypeName(typeName) {
		return nil, nil
	}
	if typ == nil {
		pkgPath = parts[0]
		var pkgInfo *PackageInfo
		if pkgInfo, err = loader.LoadPackageForType(pkgPath, typeName); err != nil {
			return nil, err
		}
		obj := pkgInfo.Types.Scope().Lookup(typeName)
		if obj == nil {
			return nil, nil
		}
		typeNameObj, ok := obj.(*types.TypeName)
		if !ok {
			return nil, nil
		}
		typ = typeNameObj.Type()
		imports = pkgInfo.Imports
	} else {
		if pkgPath == "" {
			switch t := typ.(type) {
			case *types.Named:
				if t.Obj() != nil && t.Obj().Pkg() != nil {
					pkgPath = t.Obj().Pkg().Path()
				}
			case *types.Alias:
				if t.Obj() != nil && t.Obj().Pkg() != nil {
					pkgPath = t.Obj().Pkg().Path()
				}
			}
		}
		if imports == nil && pkgPath != "" {
			pkgInfo, ok := loader.GetPackage(pkgPath)
			if !ok {
				typeNameForLoad := typeName
				if n, namedOK := typ.(*types.Named); namedOK && n.Obj() != nil {
					typeNameForLoad = n.Obj().Name()
				} else if a, aliasOK := typ.(*types.Alias); aliasOK && a.Obj() != nil {
					typeNameForLoad = a.Obj().Name()
				}
				var loadErr error
				if pkgInfo, loadErr = loader.LoadPackageForType(pkgPath, typeNameForLoad); loadErr != nil {
					return nil, loadErr
				}
				imports = pkgInfo.Imports
			} else if pkgInfo != nil {
				imports = pkgInfo.Imports
			}
		}
	}
	if typ == nil || pkgPath == "" {
		return nil, nil
	}
	if imports == nil {
		imports = make(map[string]string)
	}
	processingSet := make(map[string]bool)
	if coreType, err = convertTypeFromGoTypes(typ, pkgPath, imports, project, loader, processingSet); err != nil {
		return nil, err
	}
	if coreType == nil {
		return nil, nil
	}
	detectInterfaces(typ, coreType, project, loader)
	detectParseFromString(typ, coreType, project, loader)
	project.Types[typeID] = coreType
	return coreType, nil
}
