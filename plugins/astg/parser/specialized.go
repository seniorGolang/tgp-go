// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/ast"
	"go/types"
)

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
		return nil, fmt.Errorf("failed to resolve package path %s: %w", pkgPath, err)
	}

	buildCtx := buildContext()
	var files []*ast.File
	if files, err = l.parsePackageFiles(pkgDir, &buildCtx); err != nil {
		return nil, fmt.Errorf("failed to parse package files in %s: %w", pkgDir, err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no Go files found in package %s", pkgPath)
	}

	requiredImports := extractImportsFromErrorType(files, typeName, l.resolver)
	importer := &FileSystemImporter{
		loader:          l,
		cache:           make(map[string]*types.Package),
		lazyMode:        true,
		requiredImports: requiredImports,
	}

	cfg := &types.Config{
		Importer: importer,
		Error:    func(error) {},
	}

	typeInfo := createTypeInfo()
	var pkg *types.Package
	if pkg, err = cfg.Check(pkgPath, l.fset, files, typeInfo); err != nil {
		if pkg == nil {
			return nil, fmt.Errorf("type checking failed for %s: %w", pkgPath, err)
		}
		err = nil
	}

	imports := collectImports(files, l.resolver)

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
		return nil, fmt.Errorf("failed to resolve package path %s: %w", pkgPath, err)
	}

	buildCtx := buildContext()
	var files []*ast.File
	if files, err = l.parsePackageFiles(pkgDir, &buildCtx); err != nil {
		return nil, fmt.Errorf("failed to parse package files in %s: %w", pkgDir, err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no Go files found in package %s", pkgPath)
	}

	requiredImports := extractImportsFromTypeDefinition(files, typeName, l.resolver)
	importer := &FileSystemImporter{
		loader:          l,
		cache:           make(map[string]*types.Package),
		lazyMode:        true,
		requiredImports: requiredImports,
	}

	cfg := &types.Config{
		Importer: importer,
		Error:    func(error) {},
	}

	typeInfo := createTypeInfo()
	var pkg *types.Package
	if pkg, err = cfg.Check(pkgPath, l.fset, files, typeInfo); err != nil {
		if pkg == nil {
			return nil, fmt.Errorf("type checking failed for %s: %w", pkgPath, err)
		}
		err = nil
	}

	imports := collectImports(files, l.resolver)

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
		return nil, fmt.Errorf("failed to resolve package path %s: %w", pkgPath, err)
	}

	buildCtx := buildContext()
	var files []*ast.File
	if files, err = l.parsePackageFiles(pkgDir, &buildCtx); err != nil {
		l.mu.Lock()
		delete(l.cache, pkgPath)
		l.mu.Unlock()
		return nil, fmt.Errorf("failed to parse package files in %s: %w", pkgDir, err)
	}

	if len(files) == 0 {
		l.mu.Lock()
		delete(l.cache, pkgPath)
		l.mu.Unlock()
		return nil, fmt.Errorf("no Go files found in package %s", pkgPath)
	}

	importsToLoad := extractImportsFromExportedAndAliases(files, l.resolver)

	importer := &FileSystemImporter{
		loader:          l,
		cache:           make(map[string]*types.Package),
		lazyMode:        true,
		requiredImports: importsToLoad,
	}

	cfg := &types.Config{
		Importer: importer,
		Error:    func(error) {},
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
			return nil, fmt.Errorf("type checking failed for %s: %w", pkgPath, err)
		}
		// Некритичные ошибки (pkg != nil) игнорируются - не логируются
		err = nil
	}

	imports := collectImports(files, l.resolver)

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
