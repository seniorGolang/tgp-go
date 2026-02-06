// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"go/types"
	"os"
	"strings"
	"sync"

	"golang.org/x/mod/modfile"
)

type AutonomousPackageLoader struct {
	modulePath         string
	modFile            *modfile.File
	resolver           *PackageResolver
	cache              map[string]*PackageInfo
	versionASTgCache   map[string]bool
	versionASTgCacheMu sync.RWMutex
	fset               *token.FileSet
	mu                 sync.RWMutex
}

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
		versionASTgCache: make(map[string]bool),
		fset:             token.NewFileSet(),
	}
	return
}

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

	requiredImports := extractImportsFromExportedAndAliases(files)

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
			err = fmt.Errorf("type checking failed for %s: %w", pkgPath, err)
			return
		}
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

	requiredImports := extractImportsFromExportedAndAliases(files)

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
	if pkg, err = cfg.Check(pkgPath, fset, files, typeInfo); err != nil {
		if pkg == nil {
			l.mu.Lock()
			delete(l.cache, pkgPath)
			l.mu.Unlock()
			err = fmt.Errorf("type checking failed for %s: %w", pkgPath, err)
			return
		}
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
