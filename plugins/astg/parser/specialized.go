// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/types"
	"os"
	"strings"
)

// LoadPackageForErrorType загружает пакет с оптимизацией для проверки типов ошибок.
func (l *AutonomousPackageLoader) LoadPackageForErrorType(pkgPath string, typeName string) (info *PackageInfo, err error) {

	l.mu.RLock()
	var ok bool
	if info, ok = l.cache[pkgPath]; ok && info != nil {
		l.mu.RUnlock()
		return
	}
	l.mu.RUnlock()

	var pkgDir string
	if pkgDir, err = l.resolver.Resolve(pkgPath); err != nil {
		err = fmt.Errorf("failed to resolve package path %s: %w", pkgPath, err)
		return
	}

	buildCtx := buildContext()
	var files []*ast.File
	if files, err = l.parsePackageFiles(pkgDir, &buildCtx); err != nil {
		err = fmt.Errorf("failed to parse package files in %s: %w", pkgDir, err)
		return
	}

	if len(files) == 0 {
		err = fmt.Errorf("no Go files found in package %s", pkgPath)
		return
	}

	requiredImports := extractImportsFromErrorType(files, typeName)
	importer := &FileSystemImporter{
		loader:          l,
		cache:           make(map[string]*types.Package),
		buildCtx:        &buildCtx,
		lazyMode:        true,
		importedSet:     make(map[string]bool),
		requiredImports: requiredImports,
	}

	cfg := &types.Config{
		Importer: importer,
		Error: func(err error) {
			// Обрабатываем ошибки type checking
			// Ошибки в телах методов (из-за stub-пакетов) не критичны и не логируются
		},
	}

	typeInfo := createTypeInfo()
	var pkg *types.Package
	if pkg, err = cfg.Check(pkgPath, l.fset, files, typeInfo); err != nil {
		// Ошибки type checking не критичны, если pkg != nil
		// Ошибки в телах методов (из-за stub-пакетов) не влияют на method sets
		if pkg == nil {
			// Критичная ошибка - пакет не создан
			err = fmt.Errorf("type checking failed for %s: %w", pkgPath, err)
			return
		}
		// Некритичные ошибки (pkg != nil) игнорируются - не логируются
		err = nil
	}

	imports := collectImports(files)

	info = &PackageInfo{
		PkgPath:     pkgPath,
		PackageName: pkg.Name(),
		Dir:         pkgDir,
		Files:       files,
		Types:       pkg,
		TypeInfo:    typeInfo,
		Fset:        l.fset,
		Imports:     imports,
	}

	l.mu.Lock()
	l.cache[pkgPath] = info
	l.mu.Unlock()

	return
}

// LoadPackageForType загружает пакет для конкретного типа, извлекая импорты только из определения этого типа и его полей.
func (l *AutonomousPackageLoader) LoadPackageForType(pkgPath string, typeName string) (info *PackageInfo, err error) {

	l.mu.RLock()
	var ok bool
	if info, ok = l.cache[pkgPath]; ok && info != nil {
		if info.Types != nil && info.Types.Scope().Lookup(typeName) != nil {
			l.mu.RUnlock()
			return
		}
	}
	l.mu.RUnlock()

	var pkgDir string
	if pkgDir, err = l.resolver.Resolve(pkgPath); err != nil {
		err = fmt.Errorf("failed to resolve package path %s: %w", pkgPath, err)
		return
	}

	buildCtx := buildContext()
	var files []*ast.File
	if files, err = l.parsePackageFiles(pkgDir, &buildCtx); err != nil {
		err = fmt.Errorf("failed to parse package files in %s: %w", pkgDir, err)
		return
	}

	if len(files) == 0 {
		err = fmt.Errorf("no Go files found in package %s", pkgPath)
		return
	}

	requiredImports := extractImportsFromTypeDefinition(files, typeName)
	importer := &FileSystemImporter{
		loader:          l,
		cache:           make(map[string]*types.Package),
		buildCtx:        &buildCtx,
		lazyMode:        true,
		importedSet:     make(map[string]bool),
		requiredImports: requiredImports,
	}

	cfg := &types.Config{
		Importer: importer,
		Error: func(err error) {
			// Обрабатываем ошибки type checking
			// Ошибки в телах методов (из-за stub-пакетов) не критичны и не логируются
		},
	}

	typeInfo := createTypeInfo()
	var pkg *types.Package
	if pkg, err = cfg.Check(pkgPath, l.fset, files, typeInfo); err != nil {
		// Ошибки type checking не критичны, если pkg != nil
		// Ошибки в телах методов (из-за stub-пакетов) не влияют на method sets
		if pkg == nil {
			// Критичная ошибка - пакет не создан
			err = fmt.Errorf("type checking failed for %s: %w", pkgPath, err)
			return
		}
		// Некритичные ошибки (pkg != nil) игнорируются - не логируются
		err = nil
	}

	imports := collectImports(files)

	info = &PackageInfo{
		PkgPath:     pkgPath,
		PackageName: pkg.Name(),
		Dir:         pkgDir,
		Files:       files,
		Types:       pkg,
		TypeInfo:    typeInfo,
		Fset:        l.fset,
		Imports:     imports,
	}

	l.mu.Lock()
	l.cache[pkgPath] = info
	l.mu.Unlock()

	return
}

// LoadPackageMinimal загружает пакет с минимальными зависимостями - только необходимые типы.
func (l *AutonomousPackageLoader) LoadPackageMinimal(pkgPath string, requiredImports map[string]bool) (info *PackageInfo, err error) {

	l.mu.RLock()
	var ok bool
	if info, ok = l.cache[pkgPath]; ok && info != nil {
		l.mu.RUnlock()
		return
	}
	l.mu.RUnlock()

	l.mu.Lock()
	if info, ok = l.cache[pkgPath]; ok && info != nil {
		l.mu.Unlock()
		return
	}
	l.cache[pkgPath] = nil
	l.mu.Unlock()

	var pkgDir string
	if pkgDir, err = l.resolver.Resolve(pkgPath); err != nil {
		l.mu.Lock()
		delete(l.cache, pkgPath)
		l.mu.Unlock()
		err = fmt.Errorf("failed to resolve package path %s: %w", pkgPath, err)
		return
	}

	buildCtx := buildContext()
	var files []*ast.File
	if files, err = l.parsePackageFiles(pkgDir, &buildCtx); err != nil {
		l.mu.Lock()
		delete(l.cache, pkgPath)
		l.mu.Unlock()
		err = fmt.Errorf("failed to parse package files in %s: %w", pkgDir, err)
		return
	}

	if len(files) == 0 {
		l.mu.Lock()
		delete(l.cache, pkgPath)
		l.mu.Unlock()
		err = fmt.Errorf("no Go files found in package %s", pkgPath)
		return
	}

	var importsToLoad map[string]bool
	if len(requiredImports) > 0 {
		importsToLoad = requiredImports
	} else {
		importsToLoad = extractImportsFromMethodSignatures(files)
	}

	importer := &FileSystemImporter{
		loader:          l,
		cache:           make(map[string]*types.Package),
		buildCtx:        &buildCtx,
		lazyMode:        true,
		importedSet:     make(map[string]bool),
		requiredImports: importsToLoad,
	}

	cfg := &types.Config{
		Importer: importer,
		Error: func(err error) {
			// Обрабатываем ошибки type checking
			// Ошибки в телах методов (из-за stub-пакетов) не критичны и не логируются
		},
	}

	typeInfo := createTypeInfo()
	var pkg *types.Package
	if pkg, err = cfg.Check(pkgPath, l.fset, files, typeInfo); err != nil {
		// Ошибки type checking не критичны, если pkg != nil
		// Ошибки в телах методов (из-за stub-пакетов) не влияют на method sets
		if pkg == nil {
			// Критичная ошибка - пакет не создан
			l.mu.Lock()
			delete(l.cache, pkgPath)
			l.mu.Unlock()
			err = fmt.Errorf("type checking failed for %s: %w", pkgPath, err)
			return
		}
		// Некритичные ошибки (pkg != nil) игнорируются - не логируются
		err = nil
	}

	imports := collectImports(files)

	info = &PackageInfo{
		PkgPath:     pkgPath,
		PackageName: pkg.Name(),
		Dir:         pkgDir,
		Files:       files,
		Types:       pkg,
		TypeInfo:    typeInfo,
		Fset:        l.fset,
		Imports:     imports,
	}

	l.cache[pkgPath] = info

	return
}

// buildContext создает build.Context с настройками из переменных окружения.
func buildContext() (buildCtx build.Context) {

	buildCtx = build.Default
	var goos string
	if goos = os.Getenv("GOOS"); goos != "" {
		buildCtx.GOOS = goos
	}
	var goarch string
	if goarch = os.Getenv("GOARCH"); goarch != "" {
		buildCtx.GOARCH = goarch
	}
	return
}

// createTypeInfo создает types.Info для type checking.
func createTypeInfo() (typeInfo *types.Info) {

	typeInfo = &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}
	return
}

// collectImports собирает импорты из файлов пакета.
func collectImports(files []*ast.File) (imports map[string]string) {

	imports = make(map[string]string)
	for _, file := range files {
		for _, imp := range file.Imports {
			impPath := strings.Trim(imp.Path.Value, `"`)
			var alias string
			if imp.Name != nil {
				alias = imp.Name.Name
			} else {
				parts := strings.Split(impPath, "/")
				if len(parts) > 0 {
					alias = parts[len(parts)-1]
				}
			}
			imports[alias] = impPath
		}
	}
	return
}
