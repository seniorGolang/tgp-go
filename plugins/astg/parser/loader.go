// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sync"

	"golang.org/x/mod/modfile"
)

// AutonomousPackageLoader загружает пакеты автономно без go list.
type AutonomousPackageLoader struct {
	modulePath       string
	modFile          *modfile.File
	resolver         *PackageResolver
	cache            map[string]*PackageInfo
	versionTgCache   map[string]bool
	versionTgCacheMu sync.RWMutex
	fset             *token.FileSet
	mu               sync.RWMutex

	loadPackageStats   map[string]*loadPackageStat
	loadPackageStatsMu sync.RWMutex
}

// NewAutonomousPackageLoader создает новый загрузчик пакетов.
func NewAutonomousPackageLoader(modFile *modfile.File) (loader *AutonomousPackageLoader, err error) {

	var resolver *PackageResolver
	if resolver, err = NewPackageResolver(modFile); err != nil {
		return
	}

	var modulePath string
	if modFile != nil && modFile.Module != nil {
		modulePath = modFile.Module.Mod.Path
	}

	loader = &AutonomousPackageLoader{
		modulePath:       modulePath,
		modFile:          modFile,
		resolver:         resolver,
		cache:            make(map[string]*PackageInfo),
		versionTgCache:   make(map[string]bool),
		fset:             token.NewFileSet(),
		loadPackageStats: make(map[string]*loadPackageStat),
	}
	return
}

// LoadPackageLazy загружает пакет лениво (только если еще не загружен).
func (l *AutonomousPackageLoader) LoadPackageLazy(pkgPath string) (info *PackageInfo, err error) {

	l.mu.RLock()
	var ok bool
	if info, ok = l.cache[pkgPath]; ok {
		l.mu.RUnlock()
		return
	}
	l.mu.RUnlock()

	l.mu.Lock()
	if info, ok = l.cache[pkgPath]; ok {
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

	requiredImports := extractImportsFromMethodSignatures(files)

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
			// Обрабатываем ошибки type checking, чтобы предотвратить панику
			// Ошибки в телах методов (из-за stub-пакетов) не критичны и не логируются
			// Особенно важно для стандартных библиотек с внутренними пакетами
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

	l.cache[pkgPath] = info

	return
}

// LoadPackageFromFiles загружает пакет из уже распарсенных файлов без повторного парсинга.
func (l *AutonomousPackageLoader) LoadPackageFromFiles(pkgPath string, pkgDir string, fset *token.FileSet, files []*ast.File) (info *PackageInfo, err error) {

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

	if len(files) == 0 {
		l.mu.Lock()
		delete(l.cache, pkgPath)
		l.mu.Unlock()
		err = fmt.Errorf("no files provided for package %s", pkgPath)
		return
	}

	buildCtx := buildContext()
	requiredImports := extractImportsFromMethodSignatures(files)

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
	if pkg, err = cfg.Check(pkgPath, fset, files, typeInfo); err != nil {
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
		Fset:        fset,
		Imports:     imports,
	}

	l.mu.Lock()
	l.cache[pkgPath] = info
	l.mu.Unlock()

	return
}
