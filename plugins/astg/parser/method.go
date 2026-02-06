// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log/slog"
	"strconv"
	"strings"

	"tgp/core/i18n"
	"tgp/internal/model"
	"tgp/internal/tags"
)

func convertMethod(methodName string, funcType *ast.FuncType, docs []string, contractID string, pkgPath string, imports map[string]string, typeInfo *types.Info, project *model.Project, loader *AutonomousPackageLoader) (method *model.Method) {

	methodAnnotations := tags.ParseTags(docs)
	method = &model.Method{
		Name:        methodName,
		ContractID:  contractID,
		Docs:        removeAnnotationsFromDocs(docs),
		Annotations: methodAnnotations,
		Args:        make([]*model.Variable, 0),
		Results:     make([]*model.Variable, 0),
	}

	if funcType.Params != nil {
		for _, param := range funcType.Params.List {
			convertedTypeInfo := convertTypeFromAST(param.Type, pkgPath, imports, project, loader, typeInfo)
			if convertedTypeInfo.TypeID == "" && convertedTypeInfo.MapKey == nil {
				if typeInfo != nil {
					typ := typeInfo.TypeOf(param.Type)
					if typ != nil {
						baseTyp := typ
						pointers := 0
						for {
							if ptr, ok := baseTyp.(*types.Pointer); ok {
								pointers++
								baseTyp = ptr.Elem()
								continue
							}
							break
						}
						typeID := generateTypeIDFromGoTypes(baseTyp)
						if typeID != "" {
							convertedTypeInfo.TypeID = typeID
							convertedTypeInfo.NumberOfPointers = pointers
							_, _ = ensureTypeInProject(typeID, baseTyp, "", nil, project, loader)
						} else {
							typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
							if typeInfoResult.TypeID != "" {
								convertedTypeInfo = typeInfoResult
							} else {
								if named, ok := baseTyp.(*types.Named); ok && named.Obj() != nil {
									typeName := named.Obj().Name()
									if named.Obj().Pkg() != nil {
										importPkgPath := named.Obj().Pkg().Path()
										typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
										convertedTypeInfo.TypeID = typeID
										convertedTypeInfo.NumberOfPointers = pointers
										_, _ = ensureTypeInProject(typeID, typ, importPkgPath, nil, project, loader)
									} else {
										convertedTypeInfo.TypeID = typeName
										convertedTypeInfo.NumberOfPointers = pointers
									}
								} else if alias, ok := baseTyp.(*types.Alias); ok && alias.Obj() != nil {
									typeName := alias.Obj().Name()
									if alias.Obj().Pkg() != nil {
										importPkgPath := alias.Obj().Pkg().Path()
										typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
										convertedTypeInfo.TypeID = typeID
										convertedTypeInfo.NumberOfPointers = pointers
										_, _ = ensureTypeInProject(typeID, typ, importPkgPath, nil, project, loader)
									} else {
										convertedTypeInfo.TypeID = typeName
										convertedTypeInfo.NumberOfPointers = pointers
									}
								}
							}
						}
					}
				}
				if convertedTypeInfo.TypeID == "" && convertedTypeInfo.MapKey == nil {
					if typeInfo != nil {
						typ := typeInfo.TypeOf(param.Type)
						if typ != nil {
							typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
							if typeInfoResult.TypeID != "" {
								convertedTypeInfo = typeInfoResult
							} else {
								if named, ok := typ.(*types.Named); ok && named.Obj() != nil {
									typeName := named.Obj().Name()
									if named.Obj().Pkg() != nil {
										importPkgPath := named.Obj().Pkg().Path()
										typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
										convertedTypeInfo.TypeID = typeID
										_, _ = ensureTypeInProject(typeID, typ, importPkgPath, imports, project, loader)
									} else {
										convertedTypeInfo.TypeID = typeName
									}
								} else if alias, ok := typ.(*types.Alias); ok && alias.Obj() != nil {
									typeName := alias.Obj().Name()
									if alias.Obj().Pkg() != nil {
										importPkgPath := alias.Obj().Pkg().Path()
										typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
										convertedTypeInfo.TypeID = typeID
										_, _ = ensureTypeInProject(typeID, typ, importPkgPath, imports, project, loader)
									} else {
										convertedTypeInfo.TypeID = typeName
									}
								}
							}
						}
					}
					if convertedTypeInfo.TypeID == "" && convertedTypeInfo.MapKey == nil {
						if ident, ok := param.Type.(*ast.Ident); ok {
							if isBuiltinTypeName(ident.Name) {
								convertedTypeInfo.TypeID = ident.Name
							} else {
								slog.Debug(i18n.Msg("Failed to convert type for parameter in method"), slog.String("method", methodName), slog.String("type", ident.Name))
								continue
							}
						} else if selExpr, ok := param.Type.(*ast.SelectorExpr); ok {
							if x, ok := selExpr.X.(*ast.Ident); ok {
								importAlias := x.Name
								typeName := selExpr.Sel.Name
								importPkgPath, ok := imports[importAlias]
								if ok {
									typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
									convertedTypeInfo.TypeID = typeID
									_, _ = ensureTypeInProject(typeID, nil, "", nil, project, loader)
								} else {
									slog.Debug(i18n.Msg("Failed to convert type for parameter in method"), slog.String("method", methodName), slog.String("importAlias", importAlias))
									continue
								}
							} else {
								slog.Debug(i18n.Msg("Failed to convert type for parameter in method"), slog.String("method", methodName), slog.Any("type", param.Type))
								continue
							}
						} else {
							slog.Debug(i18n.Msg("Failed to convert type for parameter in method"), slog.String("method", methodName), slog.Any("type", param.Type))
							continue
						}
					}
				}
			}

			if convertedTypeInfo.TypeID == "" && convertedTypeInfo.MapKey == nil {
				continue
			}

			paramDocs := extractComments(param.Doc, param.Comment)
			paramAnnotations := tags.ParseTags(paramDocs)

			if len(param.Names) > 0 {
				for _, name := range param.Names {
					method.Args = append(method.Args, &model.Variable{
						TypeRef: model.TypeRef{
							TypeID:           convertedTypeInfo.TypeID,
							NumberOfPointers: convertedTypeInfo.NumberOfPointers,
							IsSlice:          convertedTypeInfo.IsSlice,
							ArrayLen:         convertedTypeInfo.ArrayLen,
							IsEllipsis:       convertedTypeInfo.IsEllipsis,
							ElementPointers:  convertedTypeInfo.ElementPointers,
							MapKey:           convertedTypeInfo.MapKey,
							MapValue:         convertedTypeInfo.MapValue,
						},
						Name:        name.Name,
						Docs:        removeAnnotationsFromDocs(paramDocs),
						Annotations: paramAnnotations,
					})
				}
			} else {
				argName := fmt.Sprintf("arg%d", len(method.Args)+1)
				method.Args = append(method.Args, &model.Variable{
					TypeRef: model.TypeRef{
						TypeID:           convertedTypeInfo.TypeID,
						NumberOfPointers: convertedTypeInfo.NumberOfPointers,
						IsSlice:          convertedTypeInfo.IsSlice,
						ArrayLen:         convertedTypeInfo.ArrayLen,
						IsEllipsis:       convertedTypeInfo.IsEllipsis,
						ElementPointers:  convertedTypeInfo.ElementPointers,
						MapKey:           convertedTypeInfo.MapKey,
						MapValue:         convertedTypeInfo.MapValue,
					},
					Name:        argName,
					Docs:        removeAnnotationsFromDocs(paramDocs),
					Annotations: paramAnnotations,
				})
			}
		}
	}

	if funcType.Results != nil {
		for _, result := range funcType.Results.List {
			resultTypeInfo := convertTypeFromAST(result.Type, pkgPath, imports, project, loader, typeInfo)
			if resultTypeInfo.TypeID == "" && resultTypeInfo.MapKey == nil {
				if typeInfo != nil {
					typ := typeInfo.TypeOf(result.Type)
					if typ != nil {
						baseTyp := typ
						pointers := 0
						for {
							if ptr, ok := baseTyp.(*types.Pointer); ok {
								pointers++
								baseTyp = ptr.Elem()
								continue
							}
							break
						}
						typeID := generateTypeIDFromGoTypes(baseTyp)
						if typeID != "" {
							resultTypeInfo.TypeID = typeID
							resultTypeInfo.NumberOfPointers = pointers
							_, _ = ensureTypeInProject(typeID, baseTyp, "", nil, project, loader)
						} else {
							typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
							if typeInfoResult.TypeID != "" {
								resultTypeInfo = typeInfoResult
							} else {
								if named, ok := baseTyp.(*types.Named); ok && named.Obj() != nil {
									typeName := named.Obj().Name()
									if named.Obj().Pkg() != nil {
										importPkgPath := named.Obj().Pkg().Path()
										typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
										resultTypeInfo.TypeID = typeID
										resultTypeInfo.NumberOfPointers = pointers
										_, _ = ensureTypeInProject(typeID, typ, importPkgPath, nil, project, loader)
									} else {
										resultTypeInfo.TypeID = typeName
										resultTypeInfo.NumberOfPointers = pointers
									}
								} else if alias, ok := baseTyp.(*types.Alias); ok && alias.Obj() != nil {
									typeName := alias.Obj().Name()
									if alias.Obj().Pkg() != nil {
										importPkgPath := alias.Obj().Pkg().Path()
										typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
										resultTypeInfo.TypeID = typeID
										resultTypeInfo.NumberOfPointers = pointers
										_, _ = ensureTypeInProject(typeID, typ, importPkgPath, nil, project, loader)
									} else {
										resultTypeInfo.TypeID = typeName
										resultTypeInfo.NumberOfPointers = pointers
									}
								}
							}
						}
					}
				}
				if resultTypeInfo.TypeID == "" && resultTypeInfo.MapKey == nil {
					if typeInfo != nil {
						typ := typeInfo.TypeOf(result.Type)
						if typ != nil {
							typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
							if typeInfoResult.TypeID != "" {
								resultTypeInfo = typeInfoResult
							} else {
								if named, ok := typ.(*types.Named); ok && named.Obj() != nil {
									typeName := named.Obj().Name()
									if named.Obj().Pkg() != nil {
										importPkgPath := named.Obj().Pkg().Path()
										typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
										resultTypeInfo.TypeID = typeID
										_, _ = ensureTypeInProject(typeID, typ, importPkgPath, imports, project, loader)
									} else {
										resultTypeInfo.TypeID = typeName
									}
								} else if alias, ok := typ.(*types.Alias); ok && alias.Obj() != nil {
									typeName := alias.Obj().Name()
									if alias.Obj().Pkg() != nil {
										importPkgPath := alias.Obj().Pkg().Path()
										typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
										resultTypeInfo.TypeID = typeID
										_, _ = ensureTypeInProject(typeID, typ, importPkgPath, imports, project, loader)
									} else {
										resultTypeInfo.TypeID = typeName
									}
								}
							}
						}
					}
					if resultTypeInfo.TypeID == "" && resultTypeInfo.MapKey == nil {
						if ident, ok := result.Type.(*ast.Ident); ok {
							if isBuiltinTypeName(ident.Name) {
								resultTypeInfo.TypeID = ident.Name
							} else {
								slog.Debug(i18n.Msg("Failed to convert type for result in method"), slog.String("method", methodName), slog.String("type", ident.Name))
								continue
							}
						} else if selExpr, ok := result.Type.(*ast.SelectorExpr); ok {
							if x, ok := selExpr.X.(*ast.Ident); ok {
								importAlias := x.Name
								typeName := selExpr.Sel.Name
								importPkgPath, ok := imports[importAlias]
								if ok {
									typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
									resultTypeInfo.TypeID = typeID
									_, _ = ensureTypeInProject(typeID, nil, "", nil, project, loader)
								} else {
									slog.Debug(i18n.Msg("Failed to convert type for result in method"), slog.String("method", methodName), slog.String("importAlias", importAlias))
									continue
								}
							} else {
								slog.Debug(i18n.Msg("Failed to convert type for result in method"), slog.String("method", methodName), slog.Any("type", result.Type))
								continue
							}
						} else {
							slog.Debug(i18n.Msg("Failed to convert type for result in method"), slog.String("method", methodName), slog.Any("type", result.Type))
							continue
						}
					}
				}
			}

			if resultTypeInfo.TypeID == "" && resultTypeInfo.MapKey == nil {
				continue
			}

			resultDocs := extractComments(result.Doc, result.Comment)
			resultAnnotations := tags.ParseTags(resultDocs)

			if len(result.Names) > 0 {
				for _, name := range result.Names {
					method.Results = append(method.Results, &model.Variable{
						TypeRef: model.TypeRef{
							TypeID:           resultTypeInfo.TypeID,
							NumberOfPointers: resultTypeInfo.NumberOfPointers,
							IsSlice:          resultTypeInfo.IsSlice,
							ArrayLen:         resultTypeInfo.ArrayLen,
							IsEllipsis:       resultTypeInfo.IsEllipsis,
							ElementPointers:  resultTypeInfo.ElementPointers,
							MapKey:           resultTypeInfo.MapKey,
							MapValue:         resultTypeInfo.MapValue,
						},
						Name:        name.Name,
						Docs:        removeAnnotationsFromDocs(resultDocs),
						Annotations: resultAnnotations,
					})
				}
			} else {
				resultName := fmt.Sprintf("result%d", len(method.Results)+1)
				method.Results = append(method.Results, &model.Variable{
					TypeRef: model.TypeRef{
						TypeID:           resultTypeInfo.TypeID,
						NumberOfPointers: resultTypeInfo.NumberOfPointers,
						IsSlice:          resultTypeInfo.IsSlice,
						ArrayLen:         resultTypeInfo.ArrayLen,
						IsEllipsis:       resultTypeInfo.IsEllipsis,
						ElementPointers:  resultTypeInfo.ElementPointers,
						MapKey:           resultTypeInfo.MapKey,
						MapValue:         resultTypeInfo.MapValue,
					},
					Name:        resultName,
					Docs:        removeAnnotationsFromDocs(resultDocs),
					Annotations: resultAnnotations,
				})
			}
		}
	}

	method.Handler = extractHandlerInfo(method.Annotations)

	return
}

func extractHandlerInfo(methodTags tags.DocTags) (handlerInfo *model.HandlerInfo) {

	var handlerValue any
	var exists bool
	if handlerValue, exists = methodTags["handler"]; !exists {
		if handlerValue, exists = methodTags["http-response"]; !exists {
			return
		}
	}

	tokens := fmt.Sprintf("%v", handlerValue)
	parts := strings.Split(tokens, ":")
	if len(parts) != 2 {
		return
	}

	handlerInfo = &model.HandlerInfo{
		PkgPath: parts[0],
		Name:    parts[1],
	}
	return
}

type typeConversionInfo struct {
	TypeID           string
	NumberOfPointers int
	IsSlice          bool
	ArrayLen         int
	IsEllipsis       bool
	ElementPointers  int // Для элементов массивов/слайсов и значений map
	MapKey           *model.TypeRef
	MapValue         *model.TypeRef
}

func convertTypeFromAST(astType ast.Expr, pkgPath string, imports map[string]string, project *model.Project, loader *AutonomousPackageLoader, typeInfo *types.Info) (info typeConversionInfo) {

	if astType == nil {
		return
	}

	pkgInfo, ok := loader.GetPackage(pkgPath)
	if !ok || pkgInfo == nil {
		slog.Debug(i18n.Msg("Package not found or has no TypeInfo"), slog.String("package", pkgPath))
		return
	}

	if typeInfo == nil {
		typeInfo = pkgInfo.TypeInfo
	}
	if typeInfo == nil {
		slog.Debug(i18n.Msg("Package not found or has no TypeInfo"), slog.String("package", pkgPath))
		return
	}

	if ellipsis, ok := astType.(*ast.Ellipsis); ok {
		info.IsEllipsis = true
		info.IsSlice = true
		if ellipsis.Elt != nil {
			eltTyp := typeInfo.TypeOf(ellipsis.Elt)
			if eltTyp != nil {
				if basic, ok := eltTyp.(*types.Basic); ok && basic.Name() == "invalid type" {
					eltTyp = nil
				} else {
					eltInfo := convertTypeFromGoTypesToInfo(eltTyp, pkgPath, imports, project, loader)
					if eltInfo.TypeID != "" && eltInfo.TypeID != "invalid type" {
						info.TypeID = eltInfo.TypeID
						info.ElementPointers = eltInfo.NumberOfPointers
					} else {
						eltTyp = nil
					}
				}
			}
			if eltTyp == nil || info.TypeID == "" {
				if ident, ok := ellipsis.Elt.(*ast.Ident); ok {
					if isBuiltinTypeName(ident.Name) {
						info.TypeID = ident.Name
					} else if pkgInfo.Types != nil {
						obj := pkgInfo.Types.Scope().Lookup(ident.Name)
						if obj != nil {
							if typeName, ok := obj.(*types.TypeName); ok {
								typeID := generateTypeIDFromGoTypes(typeName.Type())
								if typeID != "" && typeID != "invalid type" {
									info.TypeID = typeID
								} else {
									typeNameStr := typeName.Name()
									if typeName.Pkg() != nil {
										typeID = fmt.Sprintf("%s:%s", typeName.Pkg().Path(), typeNameStr)
									} else {
										typeID = typeNameStr
									}
									info.TypeID = typeID
								}
							}
						}
					}
				} else if selExpr, ok := ellipsis.Elt.(*ast.SelectorExpr); ok {
					if x, ok := selExpr.X.(*ast.Ident); ok {
						importAlias := x.Name
						typeName := selExpr.Sel.Name
						importPkgPath, ok := imports[importAlias]
						if ok {
							typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
							info.TypeID = typeID
							_, _ = ensureTypeInProject(typeID, nil, "", nil, project, loader)
						}
					}
				}
			}
			// Если тип элемента variadic всё ещё не определён (например, ...map[string]any),
			// разбираем элемент через полную логику convertTypeFromAST (в т.ч. *ast.MapType).
			if info.TypeID == "" && info.MapKey == nil && ellipsis.Elt != nil {
				eltInfo := convertTypeFromAST(ellipsis.Elt, pkgPath, imports, project, loader, typeInfo)
				info.TypeID = eltInfo.TypeID
				info.NumberOfPointers = eltInfo.NumberOfPointers
				info.ArrayLen = eltInfo.ArrayLen
				info.ElementPointers = eltInfo.ElementPointers
				info.MapKey = eltInfo.MapKey
				info.MapValue = eltInfo.MapValue
			}
		}
		return
	}

	if ident, ok := astType.(*ast.Ident); ok {
		if isBuiltinTypeName(ident.Name) {
			info.TypeID = ident.Name
			return
		}
	}

	// Массивы и слайсы обрабатываем до указателей — иначе базовые типы в []T обрабатываются неверно.
	if arrayType, ok := astType.(*ast.ArrayType); ok {
		info.IsSlice = arrayType.Len == nil
		if arrayType.Len != nil {
			if basicLit, ok := arrayType.Len.(*ast.BasicLit); ok {
				if basicLit.Kind == token.INT {
					if arrayLen, err := strconv.Atoi(basicLit.Value); err == nil {
						info.ArrayLen = arrayLen
					}
				}
			}
		}
		if arrayType.Elt != nil {
			// Указатели на элементе ([]*T) обрабатываем отдельно от типа слайса.
			eltASTType := arrayType.Elt
			eltPointers := 0
			for {
				if starExpr, ok := eltASTType.(*ast.StarExpr); ok {
					eltPointers++
					eltASTType = starExpr.X
					continue
				}
				break
			}
			eltTyp := typeInfo.TypeOf(arrayType.Elt)
			if eltTyp != nil {
				if basic, ok := eltTyp.(*types.Basic); ok && basic.Name() == "invalid type" {
					eltTyp = nil
				} else {
					baseEltTyp := eltTyp
					for i := 0; i < eltPointers; i++ {
						if ptr, ok := baseEltTyp.(*types.Pointer); ok {
							baseEltTyp = ptr.Elem()
						} else {
							break
						}
					}
					eltInfo := convertTypeFromGoTypesToInfo(baseEltTyp, pkgPath, imports, project, loader)
					if eltInfo.TypeID != "" && eltInfo.TypeID != "invalid type" {
						info.TypeID = eltInfo.TypeID
						info.ElementPointers = eltPointers
					} else {
						eltTyp = nil
					}
				}
			}
			if eltTyp == nil || info.TypeID == "" {
				if ident, ok := eltASTType.(*ast.Ident); ok {
					if isBuiltinTypeName(ident.Name) {
						info.TypeID = ident.Name
						info.ElementPointers = eltPointers
					} else if pkgInfo.Types != nil {
						obj := pkgInfo.Types.Scope().Lookup(ident.Name)
						if obj != nil {
							if typeName, ok := obj.(*types.TypeName); ok {
								typeID := generateTypeIDFromGoTypes(typeName.Type())
								if typeID != "" {
									info.TypeID = typeID
									info.ElementPointers = eltPointers
								}
							}
						}
					}
				} else if selExpr, ok := eltASTType.(*ast.SelectorExpr); ok {
					if x, ok := selExpr.X.(*ast.Ident); ok {
						importAlias := x.Name
						typeName := selExpr.Sel.Name
						importPkgPath, ok := imports[importAlias]
						if ok {
							typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
							info.TypeID = typeID
							info.ElementPointers = eltPointers
							_, _ = ensureTypeInProject(typeID, nil, "", nil, project, loader)
						}
					}
				}
			}
		}
		return
	}

	if mapType, ok := astType.(*ast.MapType); ok {
		if mapType.Key != nil {
			keyTyp := typeInfo.TypeOf(mapType.Key)
			if keyTyp != nil {
				keyInfo := convertFieldType(keyTyp, pkgPath, imports, project, loader, make(map[string]bool))
				if keyInfo.TypeID != "" && keyInfo.TypeID != "invalid type" {
					info.MapKey = fieldTypeInfoToTypeRef(keyInfo)
				}
			}
		}
		if mapType.Value != nil {
			valueTyp := typeInfo.TypeOf(mapType.Value)
			if valueTyp != nil {
				valueInfo := convertFieldType(valueTyp, pkgPath, imports, project, loader, make(map[string]bool))
				if valueInfo.TypeID != "" && valueInfo.TypeID != "invalid type" {
					info.MapValue = fieldTypeInfoToTypeRef(valueInfo)
				}
			}
		}
		return
	}

	baseASTType := astType
	for {
		if starExpr, ok := baseASTType.(*ast.StarExpr); ok {
			info.NumberOfPointers++
			baseASTType = starExpr.X
			continue
		}
		break
	}

	var typ types.Type
	if selExpr, ok := baseASTType.(*ast.SelectorExpr); ok {
		if x, ok := selExpr.X.(*ast.Ident); ok {
			importAlias := x.Name
			typeName := selExpr.Sel.Name
			importPkgPath, ok := imports[importAlias]
			if !ok {
				slog.Debug(i18n.Msg("Import alias not found in imports map"),
					slog.String("alias", importAlias),
					slog.String("typeName", typeName),
					slog.Any("availableImports", imports))
				if typeInfo != nil {
					typ = typeInfo.TypeOf(astType)
					if typ != nil {
						typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
						if typeInfoResult.TypeID != "" {
							info = typeInfoResult
							if info.NumberOfPointers == 0 {
								info.NumberOfPointers = typeInfoResult.NumberOfPointers
							}
							return
						}
					}
				}
				return
			}
			importPkgInfo, ok := loader.GetPackage(importPkgPath)
			if !ok || importPkgInfo == nil || importPkgInfo.Types == nil {
				slog.Debug(i18n.Msg("Failed to get package info"), slog.String("package", importPkgPath), slog.String("typeName", typeName))
				if typeInfo != nil {
					typ = typeInfo.TypeOf(astType)
					if typ != nil {
						if basic, ok := typ.(*types.Basic); ok && basic.Name() == "invalid type" {
							typ = nil
						} else {
							baseTyp := typ
							for {
								if ptr, ok := baseTyp.(*types.Pointer); ok {
									info.NumberOfPointers++
									baseTyp = ptr.Elem()
									continue
								}
								break
							}
							typeID := generateTypeIDFromGoTypes(baseTyp)
							if typeID != "" && typeID != "invalid type" {
								info.TypeID = typeID
								_, _ = ensureTypeInProject(typeID, baseTyp, "", nil, project, loader)
								return
							} else {
								typ = nil
							}
						}
					}
				}
				if typ == nil {
					typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
					info.TypeID = typeID
					_, _ = ensureTypeInProject(typeID, nil, "", nil, project, loader)
					return
				}
				return
			}
			obj := importPkgInfo.Types.Scope().Lookup(typeName)
			if obj == nil {
				allNames := importPkgInfo.Types.Scope().Names()
				slog.Debug(i18n.Msg("Type not found in package"),
					slog.String("type", typeName),
					slog.String("package", importPkgPath),
					slog.Any("availableTypes", allNames))
				if typeInfo != nil {
					typ = typeInfo.TypeOf(astType)
					if typ != nil {
						typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
						if typeInfoResult.TypeID != "" {
							info = typeInfoResult
							if info.NumberOfPointers == 0 {
								info.NumberOfPointers = typeInfoResult.NumberOfPointers
							}
							return
						}
					}
				}
				return
			}
			typeNameObj, ok := obj.(*types.TypeName)
			if !ok {
				slog.Debug(i18n.Msg("Object in package is not a TypeName"), slog.String("object", typeName), slog.String("package", importPkgPath))
				if typeInfo != nil {
					typ = typeInfo.TypeOf(astType)
					if typ != nil {
						typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
						if typeInfoResult.TypeID != "" {
							info = typeInfoResult
							if info.NumberOfPointers == 0 {
								info.NumberOfPointers = typeInfoResult.NumberOfPointers
							}
							return
						}
					}
				}
				return
			}
			typ = typeNameObj.Type()
			typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
			if typeID != "" {
				info.TypeID = typeID
				if _, exists := project.Types[typeID]; !exists {
					processingSet := make(map[string]bool)
					coreType := convertTypeFromGoTypes(typ, importPkgPath, importPkgInfo.Imports, project, loader, processingSet)
					if coreType != nil {
						detectInterfaces(typ, coreType, project, loader)
						project.Types[typeID] = coreType
						if alias, ok := typ.(*types.Alias); ok {
							underlying := types.Unalias(alias)
							if named, ok := underlying.(*types.Named); ok {
								baseTypeID := generateTypeIDFromGoTypes(named)
								if baseTypeID != "" && baseTypeID != typeID {
									if _, exists := project.Types[baseTypeID]; !exists {
										if named.Obj() != nil && named.Obj().Pkg() != nil {
											basePkgPath := named.Obj().Pkg().Path()
											basePkgInfo, ok := loader.GetPackage(basePkgPath)
											if ok && basePkgInfo != nil {
												baseCoreType := convertTypeFromGoTypes(named, basePkgPath, basePkgInfo.Imports, project, loader, processingSet)
												if baseCoreType != nil {
													project.Types[baseTypeID] = baseCoreType
												}
											}
										}
									}
								}
							}
						}
					}
				}
				return
			}
		}
	}

	if typ == nil {
		typ = typeInfo.TypeOf(astType)
		if typ == nil {
			slog.Debug(i18n.Msg("TypeOf returned nil"),
				slog.String("astType", fmt.Sprintf("%T", astType)),
				slog.String("pkgPath", pkgPath))
			if selExpr, ok := astType.(*ast.SelectorExpr); ok {
				if x, ok := selExpr.X.(*ast.Ident); ok {
					importAlias := x.Name
					typeName := selExpr.Sel.Name
					importPkgPath, ok := imports[importAlias]
					if ok {
						typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
						info.TypeID = typeID
						_, _ = ensureTypeInProject(typeID, nil, "", nil, project, loader)
						return
					}
				}
			}
		} else {
			baseTyp := typ
			pointers := 0
			for {
				if ptr, ok := baseTyp.(*types.Pointer); ok {
					pointers++
					baseTyp = ptr.Elem()
					continue
				}
				break
			}
			typeID := generateTypeIDFromGoTypes(baseTyp)
			if typeID != "" {
				info.TypeID = typeID
				info.NumberOfPointers = pointers
				_, _ = ensureTypeInProject(typeID, baseTyp, "", nil, project, loader)
				if slice, ok := baseTyp.(*types.Slice); ok {
					info.IsSlice = true
					if slice.Elem() != nil {
						eltInfo := convertTypeFromGoTypesToInfo(slice.Elem(), pkgPath, imports, project, loader)
						info.TypeID = eltInfo.TypeID
						info.ElementPointers = eltInfo.NumberOfPointers
					}
					return
				} else if array, ok := baseTyp.(*types.Array); ok {
					info.IsSlice = false
					info.ArrayLen = int(array.Len())
					if array.Elem() != nil {
						eltInfo := convertTypeFromGoTypesToInfo(array.Elem(), pkgPath, imports, project, loader)
						info.TypeID = eltInfo.TypeID
						info.ElementPointers = eltInfo.NumberOfPointers
					}
					return
				} else if mapType, ok := baseTyp.(*types.Map); ok {
					if mapType.Key() != nil {
						keyInfo := convertFieldType(mapType.Key(), pkgPath, imports, project, loader, make(map[string]bool))
						if keyInfo.TypeID != "" && keyInfo.TypeID != "invalid type" {
							info.MapKey = fieldTypeInfoToTypeRef(keyInfo)
						}
					}
					if mapType.Elem() != nil {
						valueInfo := convertFieldType(mapType.Elem(), pkgPath, imports, project, loader, make(map[string]bool))
						if valueInfo.TypeID != "" && valueInfo.TypeID != "invalid type" {
							info.MapValue = fieldTypeInfoToTypeRef(valueInfo)
						}
					}
					return
				}
				return
			}
		}
	}
	if typ == nil {
		if ident, ok := baseASTType.(*ast.Ident); ok {
			if isBuiltinTypeName(ident.Name) {
				info.TypeID = ident.Name
				return
			}
			if pkgInfo.Types != nil {
				obj := pkgInfo.Types.Scope().Lookup(ident.Name)
				if obj != nil {
					if typeName, ok := obj.(*types.TypeName); ok {
						typ = typeName.Type()
						if alias, ok := typ.(*types.Alias); ok {
							typ = types.Unalias(alias)
						}
						typeID := generateTypeIDFromGoTypes(typ)
						if typeID != "" {
							info.TypeID = typeID
							_, _ = ensureTypeInProject(typeID, typ, pkgPath, imports, project, loader)
							return
						}
					}
				}
			}
		}
		if typ == nil {
			if selExpr, ok := baseASTType.(*ast.SelectorExpr); ok {
				if x, ok := selExpr.X.(*ast.Ident); ok {
					importAlias := x.Name
					typeName := selExpr.Sel.Name
					importPkgPath, ok := imports[importAlias]
					if ok {
						typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
						info.TypeID = typeID
						_, _ = ensureTypeInProject(typeID, nil, "", nil, project, loader)
						return
					}
				}
			}
			slog.Debug(i18n.Msg("Failed to get type from AST"), slog.Any("astType", astType))
			return
		}
	}

	if alias, ok := typ.(*types.Alias); ok {
		typ = types.Unalias(alias)
	}

	for {
		if ptr, ok := typ.(*types.Pointer); ok {
			typ = ptr.Elem()
			continue
		}
		break
	}

	switch t := typ.(type) {
	case *types.Slice:
		info.IsSlice = true
		if t.Elem() != nil {
			if arrayType, ok := baseASTType.(*ast.ArrayType); ok && arrayType.Elt != nil {
				eltTyp := typeInfo.TypeOf(arrayType.Elt)
				if eltTyp != nil {
					eltInfo := convertTypeFromGoTypesToInfo(eltTyp, pkgPath, imports, project, loader)
					info.TypeID = eltInfo.TypeID
					info.ElementPointers = eltInfo.NumberOfPointers
				}
			} else {
				eltInfo := convertTypeFromGoTypesToInfo(t.Elem(), pkgPath, imports, project, loader)
				info.TypeID = eltInfo.TypeID
				info.ElementPointers = eltInfo.NumberOfPointers
			}
		}
		return

	case *types.Array:
		info.IsSlice = false
		info.ArrayLen = int(t.Len())
		if t.Elem() != nil {
			if arrayType, ok := baseASTType.(*ast.ArrayType); ok && arrayType.Elt != nil {
				eltTyp := typeInfo.TypeOf(arrayType.Elt)
				if eltTyp != nil {
					eltInfo := convertTypeFromGoTypesToInfo(eltTyp, pkgPath, imports, project, loader)
					info.TypeID = eltInfo.TypeID
					info.ElementPointers = eltInfo.NumberOfPointers
				}
			} else {
				eltInfo := convertTypeFromGoTypesToInfo(t.Elem(), pkgPath, imports, project, loader)
				info.TypeID = eltInfo.TypeID
				info.ElementPointers = eltInfo.NumberOfPointers
			}
		}
		return

	case *types.Map:
		if t.Key() != nil {
			keyInfo := convertFieldType(t.Key(), pkgPath, imports, project, loader, make(map[string]bool))
			if keyInfo.TypeID != "" && keyInfo.TypeID != "invalid type" {
				info.MapKey = fieldTypeInfoToTypeRef(keyInfo)
			}
		}
		if t.Elem() != nil {
			valueInfo := convertFieldType(t.Elem(), pkgPath, imports, project, loader, make(map[string]bool))
			if valueInfo.TypeID != "" && valueInfo.TypeID != "invalid type" {
				info.MapValue = fieldTypeInfoToTypeRef(valueInfo)
			}
		}
		return
	}

	typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
	info.TypeID = typeInfoResult.TypeID
	if info.NumberOfPointers == 0 {
		info.NumberOfPointers = typeInfoResult.NumberOfPointers
	}

	return
}

func ensureTypeInProject(typeID string, typ types.Type, pkgPath string, imports map[string]string, project *model.Project, loader *AutonomousPackageLoader) (coreType *model.Type, err error) {

	if existing, exists := project.Types[typeID]; exists {
		return existing, nil
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
		pkgInfo, err = loader.LoadPackageForType(pkgPath, typeName)
		if err != nil {
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
				if n, ok := typ.(*types.Named); ok && n.Obj() != nil {
					typeNameForLoad = n.Obj().Name()
				} else if a, ok := typ.(*types.Alias); ok && a.Obj() != nil {
					typeNameForLoad = a.Obj().Name()
				}
				var loadErr error
				pkgInfo, loadErr = loader.LoadPackageForType(pkgPath, typeNameForLoad)
				if loadErr != nil {
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
	coreType = convertTypeFromGoTypes(typ, pkgPath, imports, project, loader, processingSet)
	if coreType == nil {
		return nil, nil
	}
	detectInterfaces(typ, coreType, project, loader)
	project.Types[typeID] = coreType
	return coreType, nil
}

func convertTypeFromGoTypesToInfo(typ types.Type, pkgPath string, imports map[string]string, project *model.Project, loader *AutonomousPackageLoader) (info typeConversionInfo) {

	if typ == nil {
		return
	}

	for {
		if ptr, ok := typ.(*types.Pointer); ok {
			info.NumberOfPointers++
			typ = ptr.Elem()
			continue
		}
		break
	}

	typeID := generateTypeIDFromGoTypes(typ)
	if typeID == "" {
		if basic, ok := typ.(*types.Basic); ok {
			typeID = basic.Name()
		} else if named, ok := typ.(*types.Named); ok {
			if named.Obj() != nil {
				typeName := named.Obj().Name()
				if named.Obj().Pkg() != nil {
					importPkgPath := named.Obj().Pkg().Path()
					typeID = fmt.Sprintf("%s:%s", importPkgPath, typeName)
				} else {
					typeID = typeName
				}
			} else {
				underlying := named.Underlying()
				if underlying != nil {
					if underlyingID := generateTypeIDFromGoTypes(underlying); underlyingID != "" {
						typeID = underlyingID
					} else if underlyingNamed, ok := underlying.(*types.Named); ok && underlyingNamed.Obj() != nil {
						typeName := underlyingNamed.Obj().Name()
						if underlyingNamed.Obj().Pkg() != nil {
							importPkgPath := underlyingNamed.Obj().Pkg().Path()
							typeID = fmt.Sprintf("%s:%s", importPkgPath, typeName)
						} else {
							typeID = typeName
						}
					}
				}
			}
		} else if alias, ok := typ.(*types.Alias); ok {
			if alias.Obj() != nil {
				typeName := alias.Obj().Name()
				if alias.Obj().Pkg() != nil {
					importPkgPath := alias.Obj().Pkg().Path()
					typeID = fmt.Sprintf("%s:%s", importPkgPath, typeName)
				} else {
					typeID = typeName
				}
			} else {
				underlying := types.Unalias(alias)
				if underlying != nil {
					if underlyingID := generateTypeIDFromGoTypes(underlying); underlyingID != "" {
						typeID = underlyingID
					} else if underlyingNamed, ok := underlying.(*types.Named); ok && underlyingNamed.Obj() != nil {
						typeName := underlyingNamed.Obj().Name()
						if underlyingNamed.Obj().Pkg() != nil {
							importPkgPath := underlyingNamed.Obj().Pkg().Path()
							typeID = fmt.Sprintf("%s:%s", importPkgPath, typeName)
						} else {
							typeID = typeName
						}
					}
				}
			}
		} else {
			// Fallback: тип не удалось обработать через go/types — пробуем строковое представление.
			typeStr := typ.String()
			if typeStr != "" && typeStr != "<nil>" {
				if strings.Contains(typeStr, ".") {
					parts := strings.Split(typeStr, ".")
					if len(parts) == 2 {
						pkgName := parts[0]
						typeName := parts[1]
						for alias, pkgPath := range imports {
							if alias == pkgName {
								typeID = fmt.Sprintf("%s:%s", pkgPath, typeName)
								break
							}
						}
						if typeID == "" {
							typeID = typeStr
						}
					}
				}
			}
		}
	}

	info.TypeID = typeID

	if typeID != "" && !isBuiltinTypeName(typeID) {
		_, _ = ensureTypeInProject(typeID, typ, "", nil, project, loader)
	}

	return
}
