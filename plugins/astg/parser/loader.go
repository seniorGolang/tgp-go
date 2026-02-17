// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/mod/modfile"

	"tgp/core/exec"
	"tgp/internal/helper"
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
	gcImporter         types.Importer
	exportIndex        map[string]string
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

	fset := token.NewFileSet()
	loader = &AutonomousPackageLoader{
		modulePath:       modulePath,
		modFile:          modFile,
		resolver:         resolver,
		cache:            make(map[string]*PackageInfo),
		versionASTgCache: make(map[string]bool),
		fset:             fset,
		exportIndex:      make(map[string]string),
	}
	_ = loader.buildExportIndex()
	loader.gcImporter = importer.ForCompiler(fset, "gc", loader.exportLookup)
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

	requiredImports := extractImportsFromExportedAndAliases(files, l.resolver)

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
		return nil, fmt.Errorf("no files provided for package %s", pkgPath)
	}

	requiredImports := extractImportsFromExportedAndAliases(files, l.resolver)

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
	return &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}
}

func getPackageNameFromPath(resolver *PackageResolver, pkgPath string) (pkgName string, err error) {

	var dir string
	if dir, err = resolver.Resolve(pkgPath); err != nil || dir == "" {
		return
	}
	var entries []os.DirEntry
	if entries, err = os.ReadDir(dir); err != nil {
		return
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !helper.IsRelevantGoFile(e.Name()) {
			continue
		}
		fpath := filepath.Join(dir, e.Name())
		var f *ast.File
		if f, err = parser.ParseFile(fset, fpath, nil, 0); err != nil {
			continue
		}
		if f.Name != nil {
			pkgName = f.Name.Name
			return
		}
	}
	return
}

func (l *AutonomousPackageLoader) buildExportIndex() (err error) {

	cmd := exec.Command("go", "list", "-e", "-json=ImportPath,Export", "-export", "-deps", "./...")
	cmd.Dir("/")
	if err = cmd.Start(); err != nil {
		return
	}
	var stdout io.ReadCloser
	if stdout, err = cmd.StdoutPipe(); err != nil {
		_ = cmd.Wait()
		return
	}
	type listEntry struct {
		ImportPath string `json:"ImportPath"`
		Export     string `json:"Export"`
	}
	dec := json.NewDecoder(stdout)
	for dec.More() {
		var entry listEntry
		if err = dec.Decode(&entry); err != nil {
			break
		}
		if entry.Export != "" {
			l.exportIndex[entry.ImportPath] = entry.Export
		}
	}
	_ = stdout.Close()
	_ = cmd.Wait()
	return
}

func (l *AutonomousPackageLoader) exportLookup(path string) (rc io.ReadCloser, err error) {

	exportPath, ok := l.exportIndex[path]
	if !ok || exportPath == "" {
		return nil, fmt.Errorf("no export data for %s", path)
	}
	return os.Open(exportPath)
}

func (l *AutonomousPackageLoader) isLocalPackage(pkgPath string) bool {

	return l.modulePath != "" && (pkgPath == l.modulePath || strings.HasPrefix(pkgPath, l.modulePath+"/"))
}

func collectImports(files []*ast.File, resolver *PackageResolver) (imports map[string]string) {

	imports = make(map[string]string)
	for _, file := range files {
		for _, imp := range file.Imports {
			impPath := strings.Trim(imp.Path.Value, `"`)
			var alias string
			if imp.Name != nil {
				alias = imp.Name.Name
			} else {
				alias, _ = getPackageNameFromPath(resolver, impPath)
			}
			if alias != "" {
				imports[alias] = impPath
			}
		}
	}
	return
}
