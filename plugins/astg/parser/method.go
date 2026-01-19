// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
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

// convertMethod преобразует ast.FuncType в Method.
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

	// Преобразуем аргументы
	if funcType.Params != nil {
		for _, param := range funcType.Params.List {
			convertedTypeInfo := convertTypeFromAST(param.Type, pkgPath, imports, project, loader, typeInfo)
			// Если TypeID пустой, пытаемся определить тип через go/types
			if convertedTypeInfo.TypeID == "" && convertedTypeInfo.MapKeyID == "" {
				if typeInfo != nil {
					typ := typeInfo.TypeOf(param.Type)
					if typ != nil {
						// Сначала пытаемся получить TypeID напрямую из types.Type
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
							// Успешно получили typeID
							convertedTypeInfo.TypeID = typeID
							convertedTypeInfo.NumberOfPointers = pointers
							// Пытаемся сохранить тип в project.Types
							if _, exists := project.Types[typeID]; !exists {
								if named, ok := baseTyp.(*types.Named); ok && named.Obj() != nil && named.Obj().Pkg() != nil {
									actualPkgPath := named.Obj().Pkg().Path()
									if pkgInfo, err := loader.LoadPackageForType(actualPkgPath, named.Obj().Name()); err == nil {
										processingSet := make(map[string]bool)
										coreType := convertTypeFromGoTypes(baseTyp, actualPkgPath, pkgInfo.Imports, project, loader, processingSet)
										if coreType != nil {
											detectInterfaces(baseTyp, coreType, project, loader)
											project.Types[typeID] = coreType
										}
									}
								} else if alias, ok := baseTyp.(*types.Alias); ok && alias.Obj() != nil && alias.Obj().Pkg() != nil {
									actualPkgPath := alias.Obj().Pkg().Path()
									if pkgInfo, err := loader.LoadPackageForType(actualPkgPath, alias.Obj().Name()); err == nil {
										processingSet := make(map[string]bool)
										coreType := convertTypeFromGoTypes(baseTyp, actualPkgPath, pkgInfo.Imports, project, loader, processingSet)
										if coreType != nil {
											detectInterfaces(baseTyp, coreType, project, loader)
											project.Types[typeID] = coreType
										}
									}
								}
							}
						} else {
							// Если generateTypeIDFromGoTypes вернул пустую строку, используем convertTypeFromGoTypesToInfo
							typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
							if typeInfoResult.TypeID != "" {
								convertedTypeInfo = typeInfoResult
							} else {
								// Пытаемся обработать тип напрямую
								if named, ok := baseTyp.(*types.Named); ok && named.Obj() != nil {
									typeName := named.Obj().Name()
									if named.Obj().Pkg() != nil {
										importPkgPath := named.Obj().Pkg().Path()
										typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
										convertedTypeInfo.TypeID = typeID
										convertedTypeInfo.NumberOfPointers = pointers
										// Сохраняем тип в project.Types
										if _, exists := project.Types[typeID]; !exists {
											if pkgInfo, err := loader.LoadPackageForType(importPkgPath, typeName); err == nil {
												processingSet := make(map[string]bool)
												coreType := convertTypeFromGoTypes(typ, importPkgPath, pkgInfo.Imports, project, loader, processingSet)
												if coreType != nil {
													detectInterfaces(typ, coreType, project, loader)
													project.Types[typeID] = coreType
												}
											}
										}
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
										// Сохраняем тип в project.Types
										if _, exists := project.Types[typeID]; !exists {
											if pkgInfo, err := loader.LoadPackageForType(importPkgPath, typeName); err == nil {
												processingSet := make(map[string]bool)
												coreType := convertTypeFromGoTypes(typ, importPkgPath, pkgInfo.Imports, project, loader, processingSet)
												if coreType != nil {
													detectInterfaces(typ, coreType, project, loader)
													project.Types[typeID] = coreType
												}
											}
										}
									} else {
										convertedTypeInfo.TypeID = typeName
										convertedTypeInfo.NumberOfPointers = pointers
									}
								}
							}
						}
					}
				}
				// Если все еще пусто, пытаемся определить из AST
				if convertedTypeInfo.TypeID == "" && convertedTypeInfo.MapKeyID == "" {
					// Пытаемся обработать через go/types, если typeInfo доступен
					if typeInfo != nil {
						typ := typeInfo.TypeOf(param.Type)
						if typ != nil {
							typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
							if typeInfoResult.TypeID != "" {
								convertedTypeInfo = typeInfoResult
							} else {
								// Пытаемся обработать тип напрямую
								if named, ok := typ.(*types.Named); ok && named.Obj() != nil {
									typeName := named.Obj().Name()
									if named.Obj().Pkg() != nil {
										importPkgPath := named.Obj().Pkg().Path()
										typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
										convertedTypeInfo.TypeID = typeID
										// Сохраняем тип в project.Types
										if _, exists := project.Types[typeID]; !exists {
											processingSet := make(map[string]bool)
											coreType := convertTypeFromGoTypes(typ, importPkgPath, imports, project, loader, processingSet)
											if coreType != nil {
												detectInterfaces(typ, coreType, project, loader)
												project.Types[typeID] = coreType
											}
										}
									} else {
										convertedTypeInfo.TypeID = typeName
									}
								} else if alias, ok := typ.(*types.Alias); ok && alias.Obj() != nil {
									typeName := alias.Obj().Name()
									if alias.Obj().Pkg() != nil {
										importPkgPath := alias.Obj().Pkg().Path()
										typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
										convertedTypeInfo.TypeID = typeID
										// Сохраняем тип в project.Types
										if _, exists := project.Types[typeID]; !exists {
											processingSet := make(map[string]bool)
											coreType := convertTypeFromGoTypes(typ, importPkgPath, imports, project, loader, processingSet)
											if coreType != nil {
												detectInterfaces(typ, coreType, project, loader)
												project.Types[typeID] = coreType
											}
										}
									} else {
										convertedTypeInfo.TypeID = typeName
									}
								}
							}
						}
					}
					// Если все еще пусто, пытаемся определить из AST напрямую
					if convertedTypeInfo.TypeID == "" && convertedTypeInfo.MapKeyID == "" {
						if ident, ok := param.Type.(*ast.Ident); ok {
							// Проверяем базовые типы
							if isBuiltinTypeName(ident.Name) {
								convertedTypeInfo.TypeID = ident.Name
							} else {
								slog.Debug(i18n.Msg("Failed to convert type for parameter in method"), slog.String("method", methodName), slog.String("type", ident.Name))
								continue
							}
						} else if selExpr, ok := param.Type.(*ast.SelectorExpr); ok {
							// Обрабатываем SelectorExpr напрямую
							if x, ok := selExpr.X.(*ast.Ident); ok {
								importAlias := x.Name
								typeName := selExpr.Sel.Name
								importPkgPath, ok := imports[importAlias]
								if ok {
									typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
									convertedTypeInfo.TypeID = typeID
									// Сохраняем тип в project.Types
									if _, exists := project.Types[typeID]; !exists {
										importPkgInfo, ok := loader.GetPackage(importPkgPath)
										if ok && importPkgInfo != nil && importPkgInfo.Types != nil {
											obj := importPkgInfo.Types.Scope().Lookup(typeName)
											if obj != nil {
												if typeNameObj, ok := obj.(*types.TypeName); ok {
													processingSet := make(map[string]bool)
													coreType := convertTypeFromGoTypes(typeNameObj.Type(), importPkgPath, importPkgInfo.Imports, project, loader, processingSet)
													if coreType != nil {
														detectInterfaces(typeNameObj.Type(), coreType, project, loader)
														project.Types[typeID] = coreType
													}
												}
											}
										}
									}
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

			// Если TypeID всё ещё пустой после всех попыток, пропускаем параметр
			if convertedTypeInfo.TypeID == "" && convertedTypeInfo.MapKeyID == "" {
				continue
			}

			paramDocs := extractComments(param.Doc, param.Comment)
			paramAnnotations := tags.ParseTags(paramDocs)

			// Обрабатываем имена параметров
			if len(param.Names) > 0 {
				for _, name := range param.Names {
					method.Args = append(method.Args, &model.Variable{
						Name:             name.Name,
						TypeID:           convertedTypeInfo.TypeID,
						NumberOfPointers: convertedTypeInfo.NumberOfPointers,
						IsSlice:          convertedTypeInfo.IsSlice,
						ArrayLen:         convertedTypeInfo.ArrayLen,
						IsEllipsis:       convertedTypeInfo.IsEllipsis,
						ElementPointers:  convertedTypeInfo.ElementPointers,
						MapKeyID:         convertedTypeInfo.MapKeyID,
						MapValueID:       convertedTypeInfo.MapValueID,
						MapKeyPointers:   convertedTypeInfo.MapKeyPointers,
						Docs:             removeAnnotationsFromDocs(paramDocs),
						Annotations:      paramAnnotations,
					})
				}
			} else {
				// Анонимный параметр
				method.Args = append(method.Args, &model.Variable{
					Name:             "",
					TypeID:           convertedTypeInfo.TypeID,
					NumberOfPointers: convertedTypeInfo.NumberOfPointers,
					IsSlice:          convertedTypeInfo.IsSlice,
					ArrayLen:         convertedTypeInfo.ArrayLen,
					IsEllipsis:       convertedTypeInfo.IsEllipsis,
					ElementPointers:  convertedTypeInfo.ElementPointers,
					MapKeyID:         convertedTypeInfo.MapKeyID,
					MapValueID:       convertedTypeInfo.MapValueID,
					MapKeyPointers:   convertedTypeInfo.MapKeyPointers,
					Docs:             removeAnnotationsFromDocs(paramDocs),
					Annotations:      paramAnnotations,
				})
			}
		}
	}

	// Преобразуем результаты
	if funcType.Results != nil {
		for _, result := range funcType.Results.List {
			resultTypeInfo := convertTypeFromAST(result.Type, pkgPath, imports, project, loader, typeInfo)
			// Если TypeID пустой, пытаемся определить тип через go/types
			if resultTypeInfo.TypeID == "" && resultTypeInfo.MapKeyID == "" {
				if typeInfo != nil {
					typ := typeInfo.TypeOf(result.Type)
					if typ != nil {
						// Сначала пытаемся получить TypeID напрямую из types.Type
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
							// Успешно получили typeID
							resultTypeInfo.TypeID = typeID
							resultTypeInfo.NumberOfPointers = pointers
							// Пытаемся сохранить тип в project.Types
							if _, exists := project.Types[typeID]; !exists {
								if named, ok := baseTyp.(*types.Named); ok && named.Obj() != nil && named.Obj().Pkg() != nil {
									actualPkgPath := named.Obj().Pkg().Path()
									if pkgInfo, err := loader.LoadPackageForType(actualPkgPath, named.Obj().Name()); err == nil {
										processingSet := make(map[string]bool)
										coreType := convertTypeFromGoTypes(baseTyp, actualPkgPath, pkgInfo.Imports, project, loader, processingSet)
										if coreType != nil {
											detectInterfaces(baseTyp, coreType, project, loader)
											project.Types[typeID] = coreType
										}
									}
								} else if alias, ok := baseTyp.(*types.Alias); ok && alias.Obj() != nil && alias.Obj().Pkg() != nil {
									actualPkgPath := alias.Obj().Pkg().Path()
									if pkgInfo, err := loader.LoadPackageForType(actualPkgPath, alias.Obj().Name()); err == nil {
										processingSet := make(map[string]bool)
										coreType := convertTypeFromGoTypes(baseTyp, actualPkgPath, pkgInfo.Imports, project, loader, processingSet)
										if coreType != nil {
											detectInterfaces(baseTyp, coreType, project, loader)
											project.Types[typeID] = coreType
										}
									}
								}
							}
						} else {
							// Если generateTypeIDFromGoTypes вернул пустую строку, используем convertTypeFromGoTypesToInfo
							typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
							if typeInfoResult.TypeID != "" {
								resultTypeInfo = typeInfoResult
							} else {
								// Пытаемся обработать тип напрямую
								if named, ok := baseTyp.(*types.Named); ok && named.Obj() != nil {
									typeName := named.Obj().Name()
									if named.Obj().Pkg() != nil {
										importPkgPath := named.Obj().Pkg().Path()
										typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
										resultTypeInfo.TypeID = typeID
										resultTypeInfo.NumberOfPointers = pointers
										// Сохраняем тип в project.Types
										if _, exists := project.Types[typeID]; !exists {
											if pkgInfo, err := loader.LoadPackageForType(importPkgPath, typeName); err == nil {
												processingSet := make(map[string]bool)
												coreType := convertTypeFromGoTypes(typ, importPkgPath, pkgInfo.Imports, project, loader, processingSet)
												if coreType != nil {
													detectInterfaces(typ, coreType, project, loader)
													project.Types[typeID] = coreType
												}
											}
										}
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
										// Сохраняем тип в project.Types
										if _, exists := project.Types[typeID]; !exists {
											if pkgInfo, err := loader.LoadPackageForType(importPkgPath, typeName); err == nil {
												processingSet := make(map[string]bool)
												coreType := convertTypeFromGoTypes(typ, importPkgPath, pkgInfo.Imports, project, loader, processingSet)
												if coreType != nil {
													detectInterfaces(typ, coreType, project, loader)
													project.Types[typeID] = coreType
												}
											}
										}
									} else {
										resultTypeInfo.TypeID = typeName
										resultTypeInfo.NumberOfPointers = pointers
									}
								}
							}
						}
					}
				}
				// Если все еще пусто, пытаемся определить из AST
				if resultTypeInfo.TypeID == "" && resultTypeInfo.MapKeyID == "" {
					// Пытаемся обработать через go/types, если typeInfo доступен
					if typeInfo != nil {
						typ := typeInfo.TypeOf(result.Type)
						if typ != nil {
							typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
							if typeInfoResult.TypeID != "" {
								resultTypeInfo = typeInfoResult
							} else {
								// Пытаемся обработать тип напрямую
								if named, ok := typ.(*types.Named); ok && named.Obj() != nil {
									typeName := named.Obj().Name()
									if named.Obj().Pkg() != nil {
										importPkgPath := named.Obj().Pkg().Path()
										typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
										resultTypeInfo.TypeID = typeID
										// Сохраняем тип в project.Types
										if _, exists := project.Types[typeID]; !exists {
											processingSet := make(map[string]bool)
											coreType := convertTypeFromGoTypes(typ, importPkgPath, imports, project, loader, processingSet)
											if coreType != nil {
												detectInterfaces(typ, coreType, project, loader)
												project.Types[typeID] = coreType
											}
										}
									} else {
										resultTypeInfo.TypeID = typeName
									}
								} else if alias, ok := typ.(*types.Alias); ok && alias.Obj() != nil {
									typeName := alias.Obj().Name()
									if alias.Obj().Pkg() != nil {
										importPkgPath := alias.Obj().Pkg().Path()
										typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
										resultTypeInfo.TypeID = typeID
										// Сохраняем тип в project.Types
										if _, exists := project.Types[typeID]; !exists {
											processingSet := make(map[string]bool)
											coreType := convertTypeFromGoTypes(typ, importPkgPath, imports, project, loader, processingSet)
											if coreType != nil {
												detectInterfaces(typ, coreType, project, loader)
												project.Types[typeID] = coreType
											}
										}
									} else {
										resultTypeInfo.TypeID = typeName
									}
								}
							}
						}
					}
					// Если все еще пусто, пытаемся определить из AST напрямую
					if resultTypeInfo.TypeID == "" && resultTypeInfo.MapKeyID == "" {
						if ident, ok := result.Type.(*ast.Ident); ok {
							// Проверяем базовые типы
							if isBuiltinTypeName(ident.Name) {
								resultTypeInfo.TypeID = ident.Name
							} else {
								slog.Debug(i18n.Msg("Failed to convert type for result in method"), slog.String("method", methodName), slog.String("type", ident.Name))
								continue
							}
						} else if selExpr, ok := result.Type.(*ast.SelectorExpr); ok {
							// Обрабатываем SelectorExpr напрямую
							if x, ok := selExpr.X.(*ast.Ident); ok {
								importAlias := x.Name
								typeName := selExpr.Sel.Name
								importPkgPath, ok := imports[importAlias]
								if ok {
									typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
									resultTypeInfo.TypeID = typeID
									// Сохраняем тип в project.Types
									if _, exists := project.Types[typeID]; !exists {
										importPkgInfo, ok := loader.GetPackage(importPkgPath)
										if ok && importPkgInfo != nil && importPkgInfo.Types != nil {
											obj := importPkgInfo.Types.Scope().Lookup(typeName)
											if obj != nil {
												if typeNameObj, ok := obj.(*types.TypeName); ok {
													processingSet := make(map[string]bool)
													coreType := convertTypeFromGoTypes(typeNameObj.Type(), importPkgPath, importPkgInfo.Imports, project, loader, processingSet)
													if coreType != nil {
														detectInterfaces(typeNameObj.Type(), coreType, project, loader)
														project.Types[typeID] = coreType
													}
												}
											}
										}
									}
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

			// Если TypeID всё ещё пустой после всех попыток, пропускаем результат
			if resultTypeInfo.TypeID == "" && resultTypeInfo.MapKeyID == "" {
				continue
			}

			resultDocs := extractComments(result.Doc, result.Comment)
			resultAnnotations := tags.ParseTags(resultDocs)

			// Обрабатываем имена результатов
			if len(result.Names) > 0 {
				for _, name := range result.Names {
					method.Results = append(method.Results, &model.Variable{
						Name:             name.Name,
						TypeID:           resultTypeInfo.TypeID,
						NumberOfPointers: resultTypeInfo.NumberOfPointers,
						IsSlice:          resultTypeInfo.IsSlice,
						ArrayLen:         resultTypeInfo.ArrayLen,
						IsEllipsis:       resultTypeInfo.IsEllipsis,
						ElementPointers:  resultTypeInfo.ElementPointers,
						MapKeyID:         resultTypeInfo.MapKeyID,
						MapValueID:       resultTypeInfo.MapValueID,
						MapKeyPointers:   resultTypeInfo.MapKeyPointers,
						Docs:             removeAnnotationsFromDocs(resultDocs),
						Annotations:      resultAnnotations,
					})
				}
			} else {
				// Анонимный результат
				method.Results = append(method.Results, &model.Variable{
					Name:             "",
					TypeID:           resultTypeInfo.TypeID,
					NumberOfPointers: resultTypeInfo.NumberOfPointers,
					IsSlice:          resultTypeInfo.IsSlice,
					ArrayLen:         resultTypeInfo.ArrayLen,
					IsEllipsis:       resultTypeInfo.IsEllipsis,
					ElementPointers:  resultTypeInfo.ElementPointers,
					MapKeyID:         resultTypeInfo.MapKeyID,
					MapValueID:       resultTypeInfo.MapValueID,
					MapKeyPointers:   resultTypeInfo.MapKeyPointers,
					Docs:             removeAnnotationsFromDocs(resultDocs),
					Annotations:      resultAnnotations,
				})
			}
		}
	}

	// Извлечение информации о handler
	method.Handler = extractHandlerInfo(method.Annotations)

	return
}

// extractHandlerInfo извлекает информацию о handler из аннотаций.
// Поддерживает аннотации: handler и http-response
func extractHandlerInfo(methodTags tags.DocTags) (handlerInfo *model.HandlerInfo) {

	// Проверяем handler
	var handlerValue interface{}
	var exists bool
	if handlerValue, exists = methodTags["handler"]; !exists {
		// Проверяем http-response
		if handlerValue, exists = methodTags["http-response"]; !exists {
			return
		}
	}

	// Формат: package:HandlerName
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

// typeConversionInfo содержит информацию о преобразованном типе.
type typeConversionInfo struct {
	TypeID           string
	NumberOfPointers int
	IsSlice          bool
	ArrayLen         int
	IsEllipsis       bool
	ElementPointers  int // Для элементов массивов/слайсов и значений map
	MapKeyID         string
	MapValueID       string
	MapKeyPointers   int
}

// convertTypeFromAST преобразует AST тип в typeConversionInfo.
// Основано на подходе из gopls: используем go/types напрямую.
func convertTypeFromAST(astType ast.Expr, pkgPath string, imports map[string]string, project *model.Project, loader *AutonomousPackageLoader, typeInfo *types.Info) (info typeConversionInfo) {

	if astType == nil {
		return
	}

	// Получаем информацию о пакете
	pkgInfo, ok := loader.GetPackage(pkgPath)
	if !ok || pkgInfo == nil {
		slog.Debug(i18n.Msg("Package not found or has no TypeInfo"), slog.String("package", pkgPath))
		return
	}

	// Используем переданный typeInfo, если он есть, иначе используем из pkgInfo
	if typeInfo == nil {
		typeInfo = pkgInfo.TypeInfo
	}
	if typeInfo == nil {
		slog.Debug(i18n.Msg("Package not found or has no TypeInfo"), slog.String("package", pkgPath))
		return
	}

	// Проверяем ellipsis ДО обработки через go/types
	if ellipsis, ok := astType.(*ast.Ellipsis); ok {
		info.IsEllipsis = true
		info.IsSlice = true
		if ellipsis.Elt != nil {
			eltTyp := typeInfo.TypeOf(ellipsis.Elt)
			if eltTyp != nil {
				// Проверяем, не является ли это "invalid type"
				if basic, ok := eltTyp.(*types.Basic); ok && basic.Name() == "invalid type" {
					eltTyp = nil
				} else {
					eltInfo := convertTypeFromGoTypesToInfo(eltTyp, pkgPath, imports, project, loader)
					if eltInfo.TypeID != "" && eltInfo.TypeID != "invalid type" {
						info.TypeID = eltInfo.TypeID
						info.ElementPointers = eltInfo.NumberOfPointers
					} else {
						// TypeID пустой или "invalid type" - используем прямое извлечение из AST
						eltTyp = nil
					}
				}
			}
			if eltTyp == nil || info.TypeID == "" {
				// Если TypeOf вернул nil или TypeID пустой, пытаемся обработать напрямую из AST
				if ident, ok := ellipsis.Elt.(*ast.Ident); ok {
					if isBuiltinTypeName(ident.Name) {
						info.TypeID = ident.Name
					} else if pkgInfo.Types != nil {
						// Это может быть тип из текущего пакета
						obj := pkgInfo.Types.Scope().Lookup(ident.Name)
						if obj != nil {
							if typeName, ok := obj.(*types.TypeName); ok {
								typeID := generateTypeIDFromGoTypes(typeName.Type())
								if typeID != "" && typeID != "invalid type" {
									info.TypeID = typeID
								} else {
									// Fallback: используем имя типа и путь пакета
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
					// Обрабатываем SelectorExpr (например, dto.SomeStruct)
					if x, ok := selExpr.X.(*ast.Ident); ok {
						importAlias := x.Name
						typeName := selExpr.Sel.Name
						importPkgPath, ok := imports[importAlias]
						if ok {
							typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
							info.TypeID = typeID
							// Сохраняем тип в project.Types
							if _, exists := project.Types[typeID]; !exists {
								importPkgInfo, ok := loader.GetPackage(importPkgPath)
								if ok && importPkgInfo != nil && importPkgInfo.Types != nil {
									obj := importPkgInfo.Types.Scope().Lookup(typeName)
									if obj != nil {
										if typeNameObj, ok := obj.(*types.TypeName); ok {
											processingSet := make(map[string]bool)
											coreType := convertTypeFromGoTypes(typeNameObj.Type(), importPkgPath, importPkgInfo.Imports, project, loader, processingSet)
											if coreType != nil {
												// Определяем интерфейсы для типа
												detectInterfaces(typeNameObj.Type(), coreType, project, loader)
												project.Types[typeID] = coreType
											}
										}
									}
								} else {
									// Пакет не загружен - пытаемся загрузить
									if pkgInfo, err := loader.LoadPackageForType(importPkgPath, typeName); err == nil {
										obj := pkgInfo.Types.Scope().Lookup(typeName)
										if obj != nil {
											if typeNameObj, ok := obj.(*types.TypeName); ok {
												processingSet := make(map[string]bool)
												coreType := convertTypeFromGoTypes(typeNameObj.Type(), importPkgPath, pkgInfo.Imports, project, loader, processingSet)
												if coreType != nil {
													detectInterfaces(typeNameObj.Type(), coreType, project, loader)
													project.Types[typeID] = coreType
												}
											}
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

	// Сначала проверяем базовые типы напрямую из AST
	if ident, ok := astType.(*ast.Ident); ok {
		// Проверяем, является ли это базовым типом
		if isBuiltinTypeName(ident.Name) {
			info.TypeID = ident.Name
			return
		}
	}

	// Обрабатываем массивы и слайсы ДО обработки указателей
	// Это важно для правильной обработки базовых типов в массивах
	if arrayType, ok := astType.(*ast.ArrayType); ok {
		info.IsSlice = arrayType.Len == nil // Если Len == nil, это слайс, иначе массив
		if arrayType.Len != nil {
			// Это массив, пытаемся получить длину
			if basicLit, ok := arrayType.Len.(*ast.BasicLit); ok {
				// Парсим длину массива
				if basicLit.Kind == token.INT {
					if arrayLen, err := strconv.Atoi(basicLit.Value); err == nil {
						info.ArrayLen = arrayLen
					}
				}
			}
		}
		if arrayType.Elt != nil {
			// Обрабатываем элемент массива/слайса
			// ВАЖНО: обрабатываем указатели на элементе (например, []*dto.SomeStruct)
			eltASTType := arrayType.Elt
			eltPointers := 0
			// Подсчитываем указатели на элементе из AST
			for {
				if starExpr, ok := eltASTType.(*ast.StarExpr); ok {
					eltPointers++
					eltASTType = starExpr.X
					continue
				}
				break
			}
			// Получаем тип элемента через go/types
			eltTyp := typeInfo.TypeOf(arrayType.Elt)
			if eltTyp != nil {
				// Проверяем, не является ли это "invalid type"
				if basic, ok := eltTyp.(*types.Basic); ok && basic.Name() == "invalid type" {
					// Используем прямое извлечение из AST
					eltTyp = nil
				} else {
					// Убираем указатели из типа, так как мы уже учли их в eltPointers
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
						// TypeID пустой или "invalid type" - используем прямое извлечение из AST
						eltTyp = nil
					}
				}
			}
			if eltTyp == nil || info.TypeID == "" {
				// Если TypeOf вернул nil или TypeID пустой, пытаемся обработать напрямую из AST
				// Если TypeOf вернул nil, пытаемся обработать напрямую из AST
				if ident, ok := eltASTType.(*ast.Ident); ok {
					if isBuiltinTypeName(ident.Name) {
						info.TypeID = ident.Name
						info.ElementPointers = eltPointers
					} else if pkgInfo.Types != nil {
						// Это может быть тип из текущего пакета
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
					// Обрабатываем SelectorExpr (например, dto.SomeStruct)
					if x, ok := selExpr.X.(*ast.Ident); ok {
						importAlias := x.Name
						typeName := selExpr.Sel.Name
						importPkgPath, ok := imports[importAlias]
						if ok {
							typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
							info.TypeID = typeID
							info.ElementPointers = eltPointers
							// Сохраняем тип в project.Types
							if _, exists := project.Types[typeID]; !exists {
								importPkgInfo, ok := loader.GetPackage(importPkgPath)
								if ok && importPkgInfo != nil && importPkgInfo.Types != nil {
									obj := importPkgInfo.Types.Scope().Lookup(typeName)
									if obj != nil {
										if typeNameObj, ok := obj.(*types.TypeName); ok {
											processingSet := make(map[string]bool)
											coreType := convertTypeFromGoTypes(typeNameObj.Type(), importPkgPath, importPkgInfo.Imports, project, loader, processingSet)
											if coreType != nil {
												// Определяем интерфейсы для типа
												detectInterfaces(typeNameObj.Type(), coreType, project, loader)
												project.Types[typeID] = coreType
											}
										}
									}
								} else {
									// Пакет не загружен - пытаемся загрузить
									if pkgInfo, err := loader.LoadPackageForType(importPkgPath, typeName); err == nil {
										obj := pkgInfo.Types.Scope().Lookup(typeName)
										if obj != nil {
											if typeNameObj, ok := obj.(*types.TypeName); ok {
												processingSet := make(map[string]bool)
												coreType := convertTypeFromGoTypes(typeNameObj.Type(), importPkgPath, pkgInfo.Imports, project, loader, processingSet)
												if coreType != nil {
													detectInterfaces(typeNameObj.Type(), coreType, project, loader)
													project.Types[typeID] = coreType
												}
											}
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

	// Обрабатываем мапы ДО обработки указателей
	if mapType, ok := astType.(*ast.MapType); ok {
		if mapType.Key != nil {
			// Обрабатываем указатели на ключе (например, map[*string]int)
			keyASTType := mapType.Key
			keyPointers := 0
			for {
				if starExpr, ok := keyASTType.(*ast.StarExpr); ok {
					keyPointers++
					keyASTType = starExpr.X
					continue
				}
				break
			}
			keyTyp := typeInfo.TypeOf(mapType.Key)
			if keyTyp != nil {
				// Проверяем, не является ли это "invalid type"
				if basic, ok := keyTyp.(*types.Basic); ok && basic.Name() == "invalid type" {
					keyTyp = nil
				} else {
					// Убираем указатели из типа
					baseKeyTyp := keyTyp
					for i := 0; i < keyPointers; i++ {
						if ptr, ok := baseKeyTyp.(*types.Pointer); ok {
							baseKeyTyp = ptr.Elem()
						} else {
							break
						}
					}
					keyInfo := convertTypeFromGoTypesToInfo(baseKeyTyp, pkgPath, imports, project, loader)
					if keyInfo.TypeID != "" && keyInfo.TypeID != "invalid type" {
						info.MapKeyID = keyInfo.TypeID
						info.MapKeyPointers = keyPointers
					} else {
						// TypeID пустой или "invalid type" - используем прямое извлечение из AST
						keyTyp = nil
					}
				}
			}
			if keyTyp == nil || info.MapKeyID == "" {
				// Если TypeOf вернул nil или TypeID пустой, пытаемся обработать напрямую из AST
				// Если TypeOf вернул nil, пытаемся обработать напрямую из AST
				if ident, ok := mapType.Key.(*ast.Ident); ok {
					if isBuiltinTypeName(ident.Name) {
						info.MapKeyID = ident.Name
					} else if pkgInfo.Types != nil {
						// Это может быть тип из текущего пакета
						obj := pkgInfo.Types.Scope().Lookup(ident.Name)
						if obj != nil {
							if typeName, ok := obj.(*types.TypeName); ok {
								typeID := generateTypeIDFromGoTypes(typeName.Type())
								if typeID != "" {
									info.MapKeyID = typeID
								}
							}
						}
					}
				} else if selExpr, ok := mapType.Key.(*ast.SelectorExpr); ok {
					// Обрабатываем SelectorExpr (например, dto.UserID)
					if x, ok := selExpr.X.(*ast.Ident); ok {
						importAlias := x.Name
						typeName := selExpr.Sel.Name
						importPkgPath, ok := imports[importAlias]
						if ok {
							typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
							info.MapKeyID = typeID
							// Сохраняем тип в project.Types
							if _, exists := project.Types[typeID]; !exists {
								importPkgInfo, ok := loader.GetPackage(importPkgPath)
								if ok && importPkgInfo != nil && importPkgInfo.Types != nil {
									obj := importPkgInfo.Types.Scope().Lookup(typeName)
									if obj != nil {
										if typeNameObj, ok := obj.(*types.TypeName); ok {
											processingSet := make(map[string]bool)
											coreType := convertTypeFromGoTypes(typeNameObj.Type(), importPkgPath, importPkgInfo.Imports, project, loader, processingSet)
											if coreType != nil {
												// Определяем интерфейсы для типа
												detectInterfaces(typeNameObj.Type(), coreType, project, loader)
												project.Types[typeID] = coreType
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
		if mapType.Value != nil {
			// ВАЖНО: обрабатываем указатели на значении (например, map[string]*dto.SomeStruct)
			valueASTType := mapType.Value
			valuePointers := 0
			// Подсчитываем указатели на значении из AST
			for {
				if starExpr, ok := valueASTType.(*ast.StarExpr); ok {
					valuePointers++
					valueASTType = starExpr.X
					continue
				}
				break
			}
			valueTyp := typeInfo.TypeOf(mapType.Value)
			if valueTyp != nil {
				// Проверяем, не является ли это "invalid type"
				if basic, ok := valueTyp.(*types.Basic); ok && basic.Name() == "invalid type" {
					valueTyp = nil
				} else {
					// Убираем указатели из типа, так как мы уже учли их в valuePointers
					baseValueTyp := valueTyp
					for i := 0; i < valuePointers; i++ {
						if ptr, ok := baseValueTyp.(*types.Pointer); ok {
							baseValueTyp = ptr.Elem()
						} else {
							break
						}
					}
					valueInfo := convertTypeFromGoTypesToInfo(baseValueTyp, pkgPath, imports, project, loader)
					if valueInfo.TypeID != "" && valueInfo.TypeID != "invalid type" {
						info.MapValueID = valueInfo.TypeID
						info.ElementPointers = valuePointers
					} else {
						// TypeID пустой или "invalid type" - используем прямое извлечение из AST
						valueTyp = nil
					}
				}
			}
			if valueTyp == nil || info.MapValueID == "" {
				// Если TypeOf вернул nil или TypeID пустой, пытаемся обработать напрямую из AST
				// Если TypeOf вернул nil, пытаемся обработать напрямую из AST
				info.ElementPointers = valuePointers
				if ident, ok := valueASTType.(*ast.Ident); ok {
					if isBuiltinTypeName(ident.Name) {
						info.MapValueID = ident.Name
					}
				} else if selExpr, ok := valueASTType.(*ast.SelectorExpr); ok {
					// Обрабатываем SelectorExpr (например, dto.SomeStruct)
					if x, ok := selExpr.X.(*ast.Ident); ok {
						importAlias := x.Name
						typeName := selExpr.Sel.Name
						importPkgPath, ok := imports[importAlias]
						if ok {
							typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
							info.MapValueID = typeID
							info.ElementPointers = valuePointers
							// Сохраняем тип в project.Types
							if _, exists := project.Types[typeID]; !exists {
								importPkgInfo, ok := loader.GetPackage(importPkgPath)
								if ok && importPkgInfo != nil && importPkgInfo.Types != nil {
									obj := importPkgInfo.Types.Scope().Lookup(typeName)
									if obj != nil {
										if typeNameObj, ok := obj.(*types.TypeName); ok {
											processingSet := make(map[string]bool)
											coreType := convertTypeFromGoTypes(typeNameObj.Type(), importPkgPath, importPkgInfo.Imports, project, loader, processingSet)
											if coreType != nil {
												// Определяем интерфейсы для типа
												detectInterfaces(typeNameObj.Type(), coreType, project, loader)
												project.Types[typeID] = coreType
											}
										}
									}
								} else {
									// Пакет не загружен - пытаемся загрузить
									if pkgInfo, err := loader.LoadPackageForType(importPkgPath, typeName); err == nil {
										obj := pkgInfo.Types.Scope().Lookup(typeName)
										if obj != nil {
											if typeNameObj, ok := obj.(*types.TypeName); ok {
												processingSet := make(map[string]bool)
												coreType := convertTypeFromGoTypes(typeNameObj.Type(), importPkgPath, pkgInfo.Imports, project, loader, processingSet)
												if coreType != nil {
													detectInterfaces(typeNameObj.Type(), coreType, project, loader)
													project.Types[typeID] = coreType
												}
											}
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

	// Обрабатываем указатели из AST
	baseASTType := astType
	for {
		if starExpr, ok := baseASTType.(*ast.StarExpr); ok {
			info.NumberOfPointers++
			baseASTType = starExpr.X
			continue
		}
		break
	}

	// Обрабатываем *ast.SelectorExpr (например, dto.JSONWebKeySet или *dto.JSONWebKeySet после удаления указателя)
	var typ types.Type
	if selExpr, ok := baseASTType.(*ast.SelectorExpr); ok {
		if x, ok := selExpr.X.(*ast.Ident); ok {
			importAlias := x.Name
			typeName := selExpr.Sel.Name
			// Находим путь пакета по алиасу импорта
			importPkgPath, ok := imports[importAlias]
			if !ok {
				slog.Debug(i18n.Msg("Import alias not found in imports map"),
					slog.String("alias", importAlias),
					slog.String("typeName", typeName),
					slog.Any("availableImports", imports))
				// Пытаемся использовать typeInfo для получения типа
				if typeInfo != nil {
					typ = typeInfo.TypeOf(astType)
					if typ != nil {
						typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
						if typeInfoResult.TypeID != "" {
							info = typeInfoResult
							// Учитываем указатели, которые уже обработаны
							if info.NumberOfPointers == 0 {
								info.NumberOfPointers = typeInfoResult.NumberOfPointers
							}
							return
						}
					}
				}
				return
			}
			// Загружаем тип из импортированного пакета
			importPkgInfo, ok := loader.GetPackage(importPkgPath)
			if !ok || importPkgInfo == nil || importPkgInfo.Types == nil {
				slog.Debug(i18n.Msg("Failed to get package info"), slog.String("package", importPkgPath), slog.String("typeName", typeName))
				// Пытаемся использовать typeInfo для получения типа
				if typeInfo != nil {
					typ = typeInfo.TypeOf(astType)
					if typ != nil {
						// Проверяем, не является ли это "invalid type" - это ошибка typeInfo
						if basic, ok := typ.(*types.Basic); ok && basic.Name() == "invalid type" {
							// typeInfo содержит ошибку - используем прямое извлечение из AST
							typ = nil
						} else {
							// Убираем указатели для правильной обработки
							baseTyp := typ
							for {
								if ptr, ok := baseTyp.(*types.Pointer); ok {
									info.NumberOfPointers++
									baseTyp = ptr.Elem()
									continue
								}
								break
							}
							// Генерируем typeID напрямую из types.Type
							typeID := generateTypeIDFromGoTypes(baseTyp)
							if typeID != "" && typeID != "invalid type" {
								// Успешно получили typeID из types.Type
								info.TypeID = typeID
								// Пытаемся сохранить тип в project.Types, если его еще нет
								if _, exists := project.Types[typeID]; !exists {
									// Для сохранения типа нужен загруженный пакет, но мы можем создать минимальную запись
									if named, ok := baseTyp.(*types.Named); ok && named.Obj() != nil {
										if named.Obj().Pkg() != nil {
											actualPkgPath := named.Obj().Pkg().Path()
											// Пытаемся загрузить пакет для сохранения типа
											if pkgInfo, err := loader.LoadPackageForType(actualPkgPath, named.Obj().Name()); err == nil {
												processingSet := make(map[string]bool)
												coreType := convertTypeFromGoTypes(baseTyp, actualPkgPath, pkgInfo.Imports, project, loader, processingSet)
												if coreType != nil {
													detectInterfaces(baseTyp, coreType, project, loader)
													project.Types[typeID] = coreType
												}
											}
										}
									} else if alias, ok := baseTyp.(*types.Alias); ok && alias.Obj() != nil {
										if alias.Obj().Pkg() != nil {
											actualPkgPath := alias.Obj().Pkg().Path()
											// Пытаемся загрузить пакет для сохранения типа
											if pkgInfo, err := loader.LoadPackageForType(actualPkgPath, alias.Obj().Name()); err == nil {
												processingSet := make(map[string]bool)
												coreType := convertTypeFromGoTypes(baseTyp, actualPkgPath, pkgInfo.Imports, project, loader, processingSet)
												if coreType != nil {
													detectInterfaces(baseTyp, coreType, project, loader)
													project.Types[typeID] = coreType
												}
											}
										}
									}
								}
								return
							} else {
								// typeID пустой или "invalid type" - используем прямое извлечение из AST
								typ = nil
							}
						}
					}
				}
				// Если typeInfo.TypeOf вернул "invalid type" или не вернул тип, используем прямое извлечение из AST
				if typ == nil {
					// Прямое извлечение typeID из AST для SelectorExpr
					typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
					info.TypeID = typeID
					// Пытаемся загрузить пакет для сохранения типа
					if _, exists := project.Types[typeID]; !exists {
						if pkgInfo, err := loader.LoadPackageForType(importPkgPath, typeName); err == nil {
							obj := pkgInfo.Types.Scope().Lookup(typeName)
							if obj != nil {
								if typeNameObj, ok := obj.(*types.TypeName); ok {
									processingSet := make(map[string]bool)
									coreType := convertTypeFromGoTypes(typeNameObj.Type(), importPkgPath, pkgInfo.Imports, project, loader, processingSet)
									if coreType != nil {
										detectInterfaces(typeNameObj.Type(), coreType, project, loader)
										project.Types[typeID] = coreType
									}
								}
							}
						}
					}
					return
				}
				return
			}
			// Пакет загружен - продолжаем обработку
			obj := importPkgInfo.Types.Scope().Lookup(typeName)
			if obj == nil {
				// Отладочная информация: выводим все доступные типы в пакете
				allNames := importPkgInfo.Types.Scope().Names()
				slog.Debug(i18n.Msg("Type not found in package"),
					slog.String("type", typeName),
					slog.String("package", importPkgPath),
					slog.Any("availableTypes", allNames))
				// Пытаемся использовать typeInfo для получения типа
				if typeInfo != nil {
					typ = typeInfo.TypeOf(astType)
					if typ != nil {
						typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
						if typeInfoResult.TypeID != "" {
							info = typeInfoResult
							// Учитываем указатели, которые уже обработаны
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
				// Пытаемся использовать typeInfo для получения типа
				if typeInfo != nil {
					typ = typeInfo.TypeOf(astType)
					if typ != nil {
						typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
						if typeInfoResult.TypeID != "" {
							info = typeInfoResult
							// Учитываем указатели, которые уже обработаны
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
			// Генерируем typeID для типа (может быть алиасом)
			// Используем typeNameObj напрямую для правильной обработки алиасов
			typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
			if typeID != "" {
				info.TypeID = typeID
				// Сохраняем тип в project.Types
				if _, exists := project.Types[typeID]; !exists {
					// Создаем processingSet для защиты от рекурсии
					processingSet := make(map[string]bool)
					coreType := convertTypeFromGoTypes(typ, importPkgPath, importPkgInfo.Imports, project, loader, processingSet)
					if coreType != nil {
						// Определяем интерфейсы для типа
						detectInterfaces(typ, coreType, project, loader)
						project.Types[typeID] = coreType
						// Если это алиас, базовый тип уже должен быть обработан в convertTypeFromGoTypes
						// Но убеждаемся, что он есть в project.Types
						if alias, ok := typ.(*types.Alias); ok {
							underlying := types.Unalias(alias)
							if named, ok := underlying.(*types.Named); ok {
								baseTypeID := generateTypeIDFromGoTypes(named)
								if baseTypeID != "" && baseTypeID != typeID {
									if _, exists := project.Types[baseTypeID]; !exists {
										// Базовый тип должен был быть обработан в convertTypeFromGoTypes
										// Но если его нет, обрабатываем его
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

	// Используем go/types для получения типа
	if typ == nil {
		typ = typeInfo.TypeOf(astType)
		if typ == nil {
			// Отладочная информация
			slog.Debug(i18n.Msg("TypeOf returned nil"),
				slog.String("astType", fmt.Sprintf("%T", astType)),
				slog.String("pkgPath", pkgPath))
			// Пытаемся обработать SelectorExpr напрямую, если это SelectorExpr
			if selExpr, ok := astType.(*ast.SelectorExpr); ok {
				if x, ok := selExpr.X.(*ast.Ident); ok {
					importAlias := x.Name
					typeName := selExpr.Sel.Name
					importPkgPath, ok := imports[importAlias]
					if ok {
						typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
						info.TypeID = typeID
						// Сохраняем тип в project.Types
						if _, exists := project.Types[typeID]; !exists {
							importPkgInfo, ok := loader.GetPackage(importPkgPath)
							if ok && importPkgInfo != nil && importPkgInfo.Types != nil {
								obj := importPkgInfo.Types.Scope().Lookup(typeName)
								if obj != nil {
									if typeNameObj, ok := obj.(*types.TypeName); ok {
										processingSet := make(map[string]bool)
										coreType := convertTypeFromGoTypes(typeNameObj.Type(), importPkgPath, importPkgInfo.Imports, project, loader, processingSet)
										if coreType != nil {
											detectInterfaces(typeNameObj.Type(), coreType, project, loader)
											project.Types[typeID] = coreType
										}
									}
								}
							} else {
								// Пакет не загружен, но мы можем создать минимальную запись типа
								// Пытаемся загрузить пакет для сохранения типа
								if pkgInfo, err := loader.LoadPackageForType(importPkgPath, typeName); err == nil {
									obj := pkgInfo.Types.Scope().Lookup(typeName)
									if obj != nil {
										if typeNameObj, ok := obj.(*types.TypeName); ok {
											processingSet := make(map[string]bool)
											coreType := convertTypeFromGoTypes(typeNameObj.Type(), importPkgPath, pkgInfo.Imports, project, loader, processingSet)
											if coreType != nil {
												detectInterfaces(typeNameObj.Type(), coreType, project, loader)
												project.Types[typeID] = coreType
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
		} else {
			// typeInfo.TypeOf вернул тип - пытаемся получить TypeID напрямую
			// Убираем указатели для правильной обработки
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
			// Генерируем typeID напрямую из types.Type
			typeID := generateTypeIDFromGoTypes(baseTyp)
			if typeID != "" {
				info.TypeID = typeID
				info.NumberOfPointers = pointers
				// Пытаемся сохранить тип в project.Types, если его еще нет
				if _, exists := project.Types[typeID]; !exists {
					if named, ok := baseTyp.(*types.Named); ok && named.Obj() != nil && named.Obj().Pkg() != nil {
						actualPkgPath := named.Obj().Pkg().Path()
						// Пытаемся загрузить пакет для сохранения типа
						if pkgInfo, err := loader.LoadPackageForType(actualPkgPath, named.Obj().Name()); err == nil {
							processingSet := make(map[string]bool)
							coreType := convertTypeFromGoTypes(baseTyp, actualPkgPath, pkgInfo.Imports, project, loader, processingSet)
							if coreType != nil {
								detectInterfaces(baseTyp, coreType, project, loader)
								project.Types[typeID] = coreType
							}
						}
					} else if alias, ok := baseTyp.(*types.Alias); ok && alias.Obj() != nil && alias.Obj().Pkg() != nil {
						actualPkgPath := alias.Obj().Pkg().Path()
						// Пытаемся загрузить пакет для сохранения типа
						if pkgInfo, err := loader.LoadPackageForType(actualPkgPath, alias.Obj().Name()); err == nil {
							processingSet := make(map[string]bool)
							coreType := convertTypeFromGoTypes(baseTyp, actualPkgPath, pkgInfo.Imports, project, loader, processingSet)
							if coreType != nil {
								detectInterfaces(baseTyp, coreType, project, loader)
								project.Types[typeID] = coreType
							}
						}
					}
				}
				// Если TypeID успешно получен, возвращаем info
				// Но нужно обработать слайсы, массивы и мапы отдельно
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
						keyInfo := convertTypeFromGoTypesToInfo(mapType.Key(), pkgPath, imports, project, loader)
						info.MapKeyID = keyInfo.TypeID
						info.MapKeyPointers = keyInfo.NumberOfPointers
					}
					if mapType.Elem() != nil {
						valueInfo := convertTypeFromGoTypesToInfo(mapType.Elem(), pkgPath, imports, project, loader)
						info.MapValueID = valueInfo.TypeID
						info.ElementPointers = valueInfo.NumberOfPointers
					}
					return
				}
				return
			}
		}
	}
	if typ == nil {
		// Если TypeOf вернул nil, пытаемся обработать напрямую из AST
		if ident, ok := baseASTType.(*ast.Ident); ok {
			// Проверяем базовые типы
			if isBuiltinTypeName(ident.Name) {
				info.TypeID = ident.Name
				return
			}
			// Это может быть тип из текущего пакета
			if pkgInfo.Types != nil {
				obj := pkgInfo.Types.Scope().Lookup(ident.Name)
				if obj != nil {
					if typeName, ok := obj.(*types.TypeName); ok {
						typ = typeName.Type()
						// Обрабатываем алиасы
						if alias, ok := typ.(*types.Alias); ok {
							typ = types.Unalias(alias)
						}
						// Генерируем typeID
						typeID := generateTypeIDFromGoTypes(typ)
						if typeID != "" {
							info.TypeID = typeID
							// Сохраняем тип в project.Types
							if _, exists := project.Types[typeID]; !exists {
								processingSet := make(map[string]bool)
								coreType := convertTypeFromGoTypes(typ, pkgPath, imports, project, loader, processingSet)
								if coreType != nil {
									// Определяем интерфейсы для типа
									detectInterfaces(typ, coreType, project, loader)
									project.Types[typeID] = coreType
								}
							}
							return
						}
					}
				}
			}
		}
		if typ == nil {
			// Пытаемся обработать SelectorExpr напрямую, если это SelectorExpr
			if selExpr, ok := baseASTType.(*ast.SelectorExpr); ok {
				if x, ok := selExpr.X.(*ast.Ident); ok {
					importAlias := x.Name
					typeName := selExpr.Sel.Name
					importPkgPath, ok := imports[importAlias]
					if ok {
						typeID := fmt.Sprintf("%s:%s", importPkgPath, typeName)
						info.TypeID = typeID
						// Сохраняем тип в project.Types
						if _, exists := project.Types[typeID]; !exists {
							importPkgInfo, ok := loader.GetPackage(importPkgPath)
							if ok && importPkgInfo != nil && importPkgInfo.Types != nil {
								obj := importPkgInfo.Types.Scope().Lookup(typeName)
								if obj != nil {
									if typeNameObj, ok := obj.(*types.TypeName); ok {
										processingSet := make(map[string]bool)
										coreType := convertTypeFromGoTypes(typeNameObj.Type(), importPkgPath, importPkgInfo.Imports, project, loader, processingSet)
										if coreType != nil {
											detectInterfaces(typeNameObj.Type(), coreType, project, loader)
											project.Types[typeID] = coreType
										}
									}
								}
							}
						}
						return
					}
				}
			}
			slog.Debug(i18n.Msg("Failed to get type from AST"), slog.Any("astType", astType))
			return
		}
	}

	// Обрабатываем алиасы
	if alias, ok := typ.(*types.Alias); ok {
		typ = types.Unalias(alias)
	}

	// Убираем указатели из typ, так как мы уже учли их в info.NumberOfPointers
	for {
		if ptr, ok := typ.(*types.Pointer); ok {
			typ = ptr.Elem()
			continue
		}
		break
	}

	// Обрабатываем слайсы и массивы
	switch t := typ.(type) {
	case *types.Slice:
		info.IsSlice = true
		if t.Elem() != nil {
			// Находим элемент в AST для правильной обработки указателей на элементе
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
			keyInfo := convertTypeFromGoTypesToInfo(t.Key(), pkgPath, imports, project, loader)
			info.MapKeyID = keyInfo.TypeID
			info.MapKeyPointers = keyInfo.NumberOfPointers
		}
		if t.Elem() != nil {
			valueInfo := convertTypeFromGoTypesToInfo(t.Elem(), pkgPath, imports, project, loader)
			info.MapValueID = valueInfo.TypeID
			info.ElementPointers = valueInfo.NumberOfPointers
		}
		return
	}

	// Для остальных типов используем convertTypeFromGoTypesToInfo
	typeInfoResult := convertTypeFromGoTypesToInfo(typ, pkgPath, imports, project, loader)
	info.TypeID = typeInfoResult.TypeID
	// NumberOfPointers уже установлен при обработке указателей из AST
	// Но если тип был получен через go/types, нужно учесть указатели из typeInfoResult
	if info.NumberOfPointers == 0 {
		info.NumberOfPointers = typeInfoResult.NumberOfPointers
	}

	return
}

// convertTypeFromGoTypesToInfo конвертирует types.Type в typeConversionInfo.
func convertTypeFromGoTypesToInfo(typ types.Type, pkgPath string, imports map[string]string, project *model.Project, loader *AutonomousPackageLoader) (info typeConversionInfo) {

	if typ == nil {
		return
	}

	// Убираем указатели
	for {
		if ptr, ok := typ.(*types.Pointer); ok {
			info.NumberOfPointers++
			typ = ptr.Elem()
			continue
		}
		break
	}

	// Генерируем typeID
	typeID := generateTypeIDFromGoTypes(typ)
	if typeID == "" {
		if basic, ok := typ.(*types.Basic); ok {
			typeID = basic.Name()
		} else if named, ok := typ.(*types.Named); ok {
			// Обрабатываем именованные типы, которые не были обработаны generateTypeIDFromGoTypes
			if named.Obj() != nil {
				typeName := named.Obj().Name()
				if named.Obj().Pkg() != nil {
					importPkgPath := named.Obj().Pkg().Path()
					typeID = fmt.Sprintf("%s:%s", importPkgPath, typeName)
				} else {
					typeID = typeName
				}
			} else {
				// Если Obj() == nil, пытаемся получить информацию из underlying типа
				underlying := named.Underlying()
				if underlying != nil {
					// Пытаемся получить typeID из underlying типа
					if underlyingID := generateTypeIDFromGoTypes(underlying); underlyingID != "" {
						typeID = underlyingID
					} else if underlyingNamed, ok := underlying.(*types.Named); ok && underlyingNamed.Obj() != nil {
						// Если underlying - именованный тип, используем его
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
			// Обрабатываем алиасы, которые не были обработаны generateTypeIDFromGoTypes
			if alias.Obj() != nil {
				typeName := alias.Obj().Name()
				if alias.Obj().Pkg() != nil {
					importPkgPath := alias.Obj().Pkg().Path()
					typeID = fmt.Sprintf("%s:%s", importPkgPath, typeName)
				} else {
					typeID = typeName
				}
			} else {
				// Если Obj() == nil, пытаемся получить информацию из underlying типа
				underlying := types.Unalias(alias)
				if underlying != nil {
					// Пытаемся получить typeID из underlying типа
					if underlyingID := generateTypeIDFromGoTypes(underlying); underlyingID != "" {
						typeID = underlyingID
					} else if underlyingNamed, ok := underlying.(*types.Named); ok && underlyingNamed.Obj() != nil {
						// Если underlying - именованный тип, используем его
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
			// Для остальных типов пытаемся использовать строковое представление
			// Это fallback для случаев, когда тип не может быть обработан стандартным способом
			typeStr := typ.String()
			if typeStr != "" && typeStr != "<nil>" {
				// Пытаемся извлечь информацию из строкового представления
				// Например, "context.Context" -> "context:Context"
				if strings.Contains(typeStr, ".") {
					parts := strings.Split(typeStr, ".")
					if len(parts) == 2 {
						// Пытаемся найти пакет в imports
						pkgName := parts[0]
						typeName := parts[1]
						for alias, pkgPath := range imports {
							if alias == pkgName {
								typeID = fmt.Sprintf("%s:%s", pkgPath, typeName)
								break
							}
						}
						// Если не нашли в imports, используем строковое представление как есть
						if typeID == "" {
							typeID = typeStr
						}
					}
				}
			}
		}
	}

	info.TypeID = typeID

	// Сохраняем тип в project.Types, если это именованный тип
	if typeID != "" && !isBuiltinTypeName(typeID) {
		if _, exists := project.Types[typeID]; !exists {
			if named, ok := typ.(*types.Named); ok {
				if named.Obj() != nil && named.Obj().Pkg() != nil {
					importPkgPath := named.Obj().Pkg().Path()
					pkgInfo, ok := loader.GetPackage(importPkgPath)
					if ok && pkgInfo != nil {
						processingSet := make(map[string]bool)
						coreType := convertTypeFromGoTypes(typ, importPkgPath, pkgInfo.Imports, project, loader, processingSet)
						if coreType != nil {
							// Определяем интерфейсы для типа
							detectInterfaces(typ, coreType, project, loader)
							project.Types[typeID] = coreType
						}
					}
				}
			} else if alias, ok := typ.(*types.Alias); ok {
				if alias.Obj() != nil && alias.Obj().Pkg() != nil {
					importPkgPath := alias.Obj().Pkg().Path()
					pkgInfo, ok := loader.GetPackage(importPkgPath)
					if ok && pkgInfo != nil {
						processingSet := make(map[string]bool)
						coreType := convertTypeFromGoTypes(typ, importPkgPath, pkgInfo.Imports, project, loader, processingSet)
						if coreType != nil {
							// Определяем интерфейсы для типа
							detectInterfaces(typ, coreType, project, loader)
							project.Types[typeID] = coreType
						}
					}
				}
			}
		}
	}

	return
}
