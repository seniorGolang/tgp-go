// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/ast"
	"go/token"
)

func extractImportsFromExportedAndAliases(files []*ast.File) (requiredImports map[string]bool) {

	requiredImports = make(map[string]bool)
	importAliases := collectImports(files)

	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			genDecl, ok := n.(*ast.GenDecl)
			if ok && genDecl.Tok == token.TYPE {
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || typeSpec.Name == nil {
						continue
					}
					name := typeSpec.Name.Name
					if token.IsExported(name) || typeSpec.Assign != token.NoPos {
						extractImportsFromType(typeSpec.Type, importAliases, requiredImports)
					}
				}
				return true
			}

			funcDecl, ok := n.(*ast.FuncDecl)
			if ok && funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				recvType := funcDecl.Recv.List[0].Type
				if starExpr, ok := recvType.(*ast.StarExpr); ok {
					recvType = starExpr.X
				}
				if ident, ok := recvType.(*ast.Ident); ok && token.IsExported(ident.Name) {
					if funcDecl.Type != nil {
						extractImportsFromFieldList(funcDecl.Type.Params, importAliases, requiredImports)
						extractImportsFromFieldList(funcDecl.Type.Results, importAliases, requiredImports)
					}
				}
				return true
			}

			return true
		})
	}

	return requiredImports
}

func extractImportsFromTypeDefinition(files []*ast.File, typeName string) (requiredImports map[string]bool) {

	requiredImports = make(map[string]bool)
	importAliases := collectImports(files)

	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			genDecl, ok := n.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				return true
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				if typeSpec.Name == nil || typeSpec.Name.Name != typeName {
					continue
				}

				extractImportsFromType(typeSpec.Type, importAliases, requiredImports)
			}

			return true
		})
	}

	return
}

func extractImportsFromErrorType(files []*ast.File, typeName string) (requiredImports map[string]bool) {

	requiredImports = make(map[string]bool)
	importAliases := collectImports(files)

	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			genDecl, ok := n.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				return true
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				if typeSpec.Name == nil || typeSpec.Name.Name != typeName {
					continue
				}

				extractImportsFromType(typeSpec.Type, importAliases, requiredImports)

				for _, file2 := range files {
					for _, decl := range file2.Decls {
						funcDecl, ok := decl.(*ast.FuncDecl)
						if !ok {
							continue
						}

						if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
							continue
						}

						recvType := funcDecl.Recv.List[0].Type
						if starExpr, ok := recvType.(*ast.StarExpr); ok {
							recvType = starExpr.X
						}

						if ident, ok := recvType.(*ast.Ident); ok && ident.Name == typeName {
							if funcDecl.Name != nil {
								methodName := funcDecl.Name.Name
								if methodName == "Error" || methodName == "Code" {
									if funcDecl.Type != nil {
										extractImportsFromFieldList(funcDecl.Type.Params, importAliases, requiredImports)
										extractImportsFromFieldList(funcDecl.Type.Results, importAliases, requiredImports)
									}
								}
							}
						}
					}
				}
			}

			return true
		})
	}

	return
}

func extractImportsFromFieldList(fieldList *ast.FieldList, importAliases map[string]string, requiredImports map[string]bool) {

	if fieldList == nil {
		return
	}

	for _, field := range fieldList.List {
		extractImportsFromType(field.Type, importAliases, requiredImports)
	}
}

func extractImportsFromType(expr ast.Expr, importAliases map[string]string, requiredImports map[string]bool) {
	if expr == nil {
		return
	}

	switch t := expr.(type) {
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			if impPath, ok := importAliases[ident.Name]; ok {
				requiredImports[impPath] = true
			}
		}
	case *ast.StarExpr:
		extractImportsFromType(t.X, importAliases, requiredImports)
	case *ast.ArrayType:
		extractImportsFromType(t.Elt, importAliases, requiredImports)
	case *ast.MapType:
		extractImportsFromType(t.Key, importAliases, requiredImports)
		extractImportsFromType(t.Value, importAliases, requiredImports)
	case *ast.ChanType:
		extractImportsFromType(t.Value, importAliases, requiredImports)
	case *ast.FuncType:
		extractImportsFromFieldList(t.Params, importAliases, requiredImports)
		extractImportsFromFieldList(t.Results, importAliases, requiredImports)
	case *ast.Ellipsis:
		extractImportsFromType(t.Elt, importAliases, requiredImports)
	case *ast.StructType:
		if t.Fields != nil {
			for _, field := range t.Fields.List {
				extractImportsFromType(field.Type, importAliases, requiredImports)
			}
		}
	case *ast.InterfaceType:
		if t.Methods != nil {
			for _, method := range t.Methods.List {
				if len(method.Names) > 0 {
					if funcType, ok := method.Type.(*ast.FuncType); ok {
						extractImportsFromFieldList(funcType.Params, importAliases, requiredImports)
						extractImportsFromFieldList(funcType.Results, importAliases, requiredImports)
					}
				} else {
					extractImportsFromType(method.Type, importAliases, requiredImports)
				}
			}
		}
	}
}
