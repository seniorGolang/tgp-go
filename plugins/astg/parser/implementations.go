// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"tgp/core/i18n"
	"tgp/internal"
	"tgp/internal/model"
	"tgp/plugins/astg/parser/utils"
)

type structInfo struct {
	Name   string
	Fields []*ast.Field
	Doc    *ast.CommentGroup
}

func findImplementations(project *model.Project, loader *AutonomousPackageLoader) (err error) {

	for _, contract := range project.Contracts {
		contract.Implementations = make([]*model.ImplementationInfo, 0)
	}

	implementsCache := make(map[string]bool)
	implementsCacheMu := sync.RWMutex{}
	methodSetCache := make(map[string]*types.MethodSet)
	pointerMethodSetCache := make(map[string]*types.MethodSet)
	methodSetCacheMu := sync.RWMutex{}
	contractMethodNames := make(map[string]map[string]bool)
	for _, contract := range project.Contracts {
		methodNames := make(map[string]bool)
		for _, method := range contract.Methods {
			methodNames[method.Name] = true
		}
		contractMethodNames[contract.ID] = methodNames
	}

	packages := make(map[string][]string)

	err = filepath.Walk(internal.ProjectRoot, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		if info.IsDir() {
			if info.Name() == "vendor" {
				return filepath.SkipDir
			}
			if shouldExcludeDir(filePath, project.ExcludeDirs) {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		if isGeneratedFile(filePath) {
			return nil
		}

		if shouldExcludeDir(filepath.Dir(filePath), project.ExcludeDirs) {
			return nil
		}

		pkgDir := filepath.Dir(filePath)
		pkgPath, err := utils.GetPkgPath(pkgDir, true)
		if err != nil {
			// В WASM используем альтернативный способ получения пути пакета
			relPath, relErr := filepath.Rel(internal.ProjectRoot, pkgDir)
			if relErr == nil {
				relPath = filepath.ToSlash(relPath)
				relPath = strings.TrimPrefix(relPath, "./")
				if relPath != "" && relPath != "." {
					pkgPath = project.ModulePath + "/" + relPath
				} else {
					pkgPath = project.ModulePath
				}
			} else {
				slog.Debug(i18n.Msg("Failed to get package path"), slog.String("file", filePath), slog.String("error", err.Error()))
				return nil
			}
		}

		pkgPath = filepath.ToSlash(pkgPath)

		if _, exists := packages[pkgPath]; !exists {
			packages[pkgPath] = make([]string, 0)
		}
		packages[pkgPath] = append(packages[pkgPath], filePath)

		return nil
	})

	if err != nil {
		slog.Debug(i18n.Msg("Failed to walk project directory"), slog.String("error", err.Error()))
		return
	}

	for pkgPath, goFiles := range packages {
		if len(goFiles) == 0 {
			continue
		}

		fset := token.NewFileSet()
		parsedFiles := make([]*ast.File, 0)
		for _, filePath := range goFiles {
			parsedFile, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
			if err != nil {
				continue
			}
			parsedFiles = append(parsedFiles, parsedFile)
		}

		if len(parsedFiles) == 0 {
			continue
		}

		//nolint:staticcheck // ast.Package deprecated, but ast.MergePackageFiles still requires it
		astPkg := &ast.Package{
			Name:  parsedFiles[0].Name.Name,
			Files: make(map[string]*ast.File),
		}
		for i, file := range parsedFiles {
			astPkg.Files[fmt.Sprintf("file%d.go", i)] = file
		}
		mergedFile := ast.MergePackageFiles(astPkg, ast.FilterUnassociatedComments|ast.FilterImportDuplicates)

		structs := findStructsInFile(mergedFile)
		structMethods := buildStructMethodsMap(structs, parsedFiles)

		var implPkgInfo *PackageInfo
		pkgLoaded := false

		for _, structType := range structs {
			structMethodSet := structMethods[structType.Name]
			for _, contract := range project.Contracts {
				methodNames := contractMethodNames[contract.ID]
				hasMethods := len(methodNames) <= len(structMethodSet)
				if hasMethods {
					for methodName := range methodNames {
						if !structMethodSet[methodName] {
							hasMethods = false
							break
						}
					}
				}
				if !hasMethods {
					continue
				}
				if !pkgLoaded {
					implPkgInfo, pkgLoaded = loader.GetPackage(pkgPath)
					if !pkgLoaded {
						pkgDir := filepath.Dir(goFiles[0])
						var err error
						implPkgInfo, err = loader.LoadPackageFromFiles(pkgPath, pkgDir, fset, parsedFiles)
						if err != nil {
							var err2 error
							implPkgInfo, err2 = loader.LoadPackageLazy(pkgPath)
							if err2 != nil {
								pkgLoaded = true
								implPkgInfo = nil
							} else {
								pkgLoaded = true
							}
						} else {
							pkgLoaded = true
						}
					} else {
						pkgLoaded = true
					}
				}

				if implPkgInfo == nil {
					continue
				}

				cacheKey := fmt.Sprintf("%s:%s:%s:%s", pkgPath, structType.Name, contract.PkgPath, contract.Name)
				implementsCacheMu.RLock()
				cached, inCache := implementsCache[cacheKey]
				implementsCacheMu.RUnlock()

				var implementsResult bool
				if inCache {
					if !cached {
						continue
					}
					implementsResult = true
				} else {
					implementsResult = implementsContract(structType, contract, mergedFile, goFiles[0], pkgPath, project, loader, implPkgInfo, methodSetCache, pointerMethodSetCache, &methodSetCacheMu)
					implementsCacheMu.Lock()
					implementsCache[cacheKey] = implementsResult
					implementsCacheMu.Unlock()
				}

				if implementsResult {
					impl := &model.ImplementationInfo{
						PkgPath:    pkgPath,
						StructName: structType.Name,
						MethodsMap: make(map[string]*model.ImplementationMethod),
					}

					structObj := implPkgInfo.Types.Scope().Lookup(structType.Name)
					if structObj == nil {
						continue
					}

					structTypeName, ok := structObj.(*types.TypeName)
					if !ok {
						continue
					}

					structType_ := structTypeName.Type()
					pointerType := types.NewPointer(structType_)
					mset := types.NewMethodSet(structType_)
					pointerMset := types.NewMethodSet(pointerType)

					for _, method := range contract.Methods {
						var methodObj *types.Selection
						if sel := mset.Lookup(implPkgInfo.Types, method.Name); sel != nil {
							methodObj = sel
						} else if sel := pointerMset.Lookup(implPkgInfo.Types, method.Name); sel != nil {
							methodObj = sel
						}

						if methodObj == nil {
							continue
						}

						methodAST := findMethodInFile(mergedFile, structType.Name, method.Name)

						implMethod := &model.ImplementationMethod{
							FilePath: makeRelativePath(goFiles[0]),
						}

						if methodAST != nil && methodAST.Body != nil {
							errorTypes := findErrorTypesInMethodBody(methodAST.Body, mergedFile, pkgPath, loader.resolver)
							implMethod.ErrorTypes = errorTypes
						}

						impl.MethodsMap[method.Name] = implMethod
					}

					if len(impl.MethodsMap) > 0 {
						contract.Implementations = append(contract.Implementations, impl)
					}
				}
			}
		}
	}

	return
}

func findAllReceiverMethodNames(file *ast.File, structName string) (methodNames map[string]bool) {

	methodNames = make(map[string]bool)
	if file == nil {
		return
	}
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			continue
		}
		recv := funcDecl.Recv.List[0]
		var recvTypeName string
		switch rt := recv.Type.(type) {
		case *ast.Ident:
			recvTypeName = rt.Name
		case *ast.StarExpr:
			if ident, ok := rt.X.(*ast.Ident); ok {
				recvTypeName = ident.Name
			}
		}
		if recvTypeName != structName {
			continue
		}
		if funcDecl.Name != nil {
			methodNames[funcDecl.Name.Name] = true
		}
	}
	return
}

func buildStructMethodsMap(structs []structInfo, parsedFiles []*ast.File) (structMethods map[string]map[string]bool) {

	structMethods = make(map[string]map[string]bool, len(structs))
	for _, st := range structs {
		allMethods := make(map[string]bool)
		for _, file := range parsedFiles {
			for name := range findAllReceiverMethodNames(file, st.Name) {
				allMethods[name] = true
			}
		}
		structMethods[st.Name] = allMethods
	}
	return
}

func findStructsInFile(file *ast.File) (structs []structInfo) {

	structs = make([]structInfo, 0)
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			structs = append(structs, structInfo{
				Name:   typeSpec.Name.Name,
				Fields: structType.Fields.List,
				Doc:    genDecl.Doc,
			})
		}
	}
	return
}

func implementsContract(structType structInfo, contract *model.Contract, file *ast.File, filePath string, pkgPath string, project *model.Project, loader *AutonomousPackageLoader, implPkgInfo *PackageInfo, methodSetCache map[string]*types.MethodSet, pointerMethodSetCache map[string]*types.MethodSet, methodSetCacheMu *sync.RWMutex) (implements bool) {

	var contractPkgInfo *PackageInfo
	var ok bool
	if contractPkgInfo, ok = loader.GetPackage(contract.PkgPath); !ok {
		var err error
		if contractPkgInfo, err = loader.LoadPackageLazy(contract.PkgPath); err != nil {
			slog.Debug(i18n.Msg("Failed to load contract package"),
				slog.String("package", contract.PkgPath),
				slog.String("error", err.Error()))
			return
		}
	}

	contractIfaceObj := contractPkgInfo.Types.Scope().Lookup(contract.Name)

	if contractIfaceObj == nil {
		slog.Debug(i18n.Msg("Contract interface not found"),
			slog.String("contract", contract.Name),
			slog.String("package", contract.PkgPath))
		return
	}

	var contractTypeName *types.TypeName
	if contractTypeName, ok = contractIfaceObj.(*types.TypeName); !ok {
		return
	}

	var contractIface *types.Interface
	if contractIface, ok = contractTypeName.Type().Underlying().(*types.Interface); !ok {
		return
	}

	structObj := implPkgInfo.Types.Scope().Lookup(structType.Name)
	if structObj == nil {
		return
	}

	var structTypeName *types.TypeName
	if structTypeName, ok = structObj.(*types.TypeName); !ok {
		return
	}

	structType_ := structTypeName.Type()
	methodSetKey := fmt.Sprintf("%s:%s", pkgPath, structType.Name)

	methodSetCacheMu.RLock()
	mset, hasMset := methodSetCache[methodSetKey]
	pointerMset, hasPointerMset := pointerMethodSetCache[methodSetKey]
	methodSetCacheMu.RUnlock()

	if !hasMset || !hasPointerMset {
		mset = types.NewMethodSet(structType_)
		pointerType := types.NewPointer(structType_)
		pointerMset = types.NewMethodSet(pointerType)

		methodSetCacheMu.Lock()
		methodSetCache[methodSetKey] = mset
		pointerMethodSetCache[methodSetKey] = pointerMset
		methodSetCacheMu.Unlock()
	}

	contractMethodCount := contractIface.NumMethods()
	if contractMethodCount == 0 {
		implements = true
		return
	}
	for i := 0; i < contractMethodCount; i++ {
		method := contractIface.Method(i)
		methodName := method.Name()
		var found *types.Selection
		if sel := mset.Lookup(implPkgInfo.Types, methodName); sel != nil {
			found = sel
		} else if sel := pointerMset.Lookup(implPkgInfo.Types, methodName); sel != nil {
			found = sel
		}
		if found == nil {
			if sel := mset.Lookup(nil, methodName); sel != nil {
				found = sel
			} else if sel := pointerMset.Lookup(nil, methodName); sel != nil {
				found = sel
			}
		}

		if found == nil {
			return
		}

		foundMethod := found.Obj().(*types.Func)
		foundSig := foundMethod.Type().(*types.Signature)
		contractSig := method.Type().(*types.Signature)

		if !types.Identical(foundSig.Params(), contractSig.Params()) {
			foundParams := foundSig.Params()
			contractParams := contractSig.Params()
			if foundParams.Len() != contractParams.Len() {
				return
			}
			for i := 0; i < foundParams.Len(); i++ {
				if foundParams.At(i).Type().String() != contractParams.At(i).Type().String() {
					return
				}
			}
		}

		if !types.Identical(foundSig.Results(), contractSig.Results()) {
			foundResults := foundSig.Results()
			contractResults := contractSig.Results()
			if foundResults.Len() != contractResults.Len() {
				return
			}
			for i := 0; i < foundResults.Len(); i++ {
				if foundResults.At(i).Type().String() != contractResults.At(i).Type().String() {
					return
				}
			}
		}
	}

	implements = true
	return
}

func findMethodInFile(file *ast.File, structName string, methodName string) (method *ast.FuncDecl) {

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if funcDecl.Name == nil {
			continue
		}
		if funcDecl.Name.Name != methodName {
			continue
		}
		if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			continue
		}
		recvType := funcDecl.Recv.List[0].Type
		if isReceiverForStruct(recvType, structName) {
			method = funcDecl
			return
		}
	}
	return
}

func findErrorTypesInMethodBody(body *ast.BlockStmt, file *ast.File, pkgPath string, resolver *PackageResolver) (errorTypes []*model.ErrorTypeReference) {

	errorTypes = make([]*model.ErrorTypeReference, 0)
	errorTypesMap := make(map[string]bool)

	ast.Inspect(body, func(n ast.Node) bool {
		if retStmt, ok := n.(*ast.ReturnStmt); ok {
			for _, result := range retStmt.Results {
				extractErrorTypeFromExpr(result, file, pkgPath, errorTypesMap, &errorTypes, resolver)
			}
			return true
		}

		if assignStmt, ok := n.(*ast.AssignStmt); ok {
			for i, lhs := range assignStmt.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok && ident.Name == "err" {
					if i < len(assignStmt.Rhs) {
						extractErrorTypeFromExpr(assignStmt.Rhs[i], file, pkgPath, errorTypesMap, &errorTypes, resolver)
					}
				}
			}
			return true
		}

		return true
	})

	return
}

func extractErrorTypeFromExpr(expr ast.Expr, file *ast.File, pkgPath string, errorTypesMap map[string]bool, errorTypes *[]*model.ErrorTypeReference, resolver *PackageResolver) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.UnaryExpr:
		if e.Op == token.AND || e.Op == token.MUL {
			extractErrorTypeFromExpr(e.X, file, pkgPath, errorTypesMap, errorTypes, resolver)
		}
	case *ast.CompositeLit:
		if e.Type == nil {
			return
		}
		extractErrorTypeFromTypeExpr(e.Type, file, pkgPath, errorTypesMap, errorTypes, resolver)
	case *ast.CallExpr:
		for _, arg := range e.Args {
			extractErrorTypeFromExpr(arg, file, pkgPath, errorTypesMap, errorTypes, resolver)
		}
		extractErrorTypeFromExpr(e.Fun, file, pkgPath, errorTypesMap, errorTypes, resolver)
	case *ast.SelectorExpr:
		extractErrorTypeFromTypeExpr(e, file, pkgPath, errorTypesMap, errorTypes, resolver)
	case *ast.Ident:
		if !isBuiltinTypeName(e.Name) {
			key := fmt.Sprintf("%s:%s", pkgPath, e.Name)
			if !errorTypesMap[key] {
				errorTypesMap[key] = true
				*errorTypes = append(*errorTypes, &model.ErrorTypeReference{
					PkgPath:  pkgPath,
					TypeName: e.Name,
					FullName: fmt.Sprintf("%s.%s", pkgPath, e.Name),
				})
			}
		}
	}
}

func extractErrorTypeFromTypeExpr(typeExpr ast.Expr, file *ast.File, pkgPath string, errorTypesMap map[string]bool, errorTypes *[]*model.ErrorTypeReference, resolver *PackageResolver) {
	if typeExpr == nil {
		return
	}

	switch t := typeExpr.(type) {
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			pkgName := x.Name
			typeName := t.Sel.Name
			importAliases := collectImports([]*ast.File{file}, resolver)
			if impPath, ok := importAliases[pkgName]; ok {
				key := fmt.Sprintf("%s:%s", impPath, typeName)
				if !errorTypesMap[key] {
					errorTypesMap[key] = true
					*errorTypes = append(*errorTypes, &model.ErrorTypeReference{
						PkgPath:  impPath,
						TypeName: typeName,
						FullName: fmt.Sprintf("%s.%s", impPath, typeName),
					})
				}
			}
		}
	case *ast.Ident:
		if !isBuiltinTypeName(t.Name) {
			key := fmt.Sprintf("%s:%s", pkgPath, t.Name)
			if !errorTypesMap[key] {
				errorTypesMap[key] = true
				*errorTypes = append(*errorTypes, &model.ErrorTypeReference{
					PkgPath:  pkgPath,
					TypeName: t.Name,
					FullName: fmt.Sprintf("%s.%s", pkgPath, t.Name),
				})
			}
		}
	}
}
