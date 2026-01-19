// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// HasVersionTgConstant проверяет наличие константы VersionTg в пакете с кэшированием.
func (l *AutonomousPackageLoader) HasVersionTgConstant(pkgPath string) (hasVersionTg bool) {

	l.versionTgCacheMu.RLock()
	var cached bool
	var ok bool
	if cached, ok = l.versionTgCache[pkgPath]; ok {
		l.versionTgCacheMu.RUnlock()
		hasVersionTg = cached
		return
	}
	l.versionTgCacheMu.RUnlock()

	var pkgDir string
	var err error
	if pkgDir, err = l.resolver.Resolve(pkgPath); err != nil {
		l.versionTgCacheMu.Lock()
		l.versionTgCache[pkgPath] = false
		l.versionTgCacheMu.Unlock()
		return
	}

	hasVersionTg = l.checkVersionTgInDir(pkgDir)

	l.versionTgCacheMu.Lock()
	l.versionTgCache[pkgPath] = hasVersionTg
	l.versionTgCacheMu.Unlock()

	return
}

// checkVersionTgInDir проверяет наличие константы VersionTg в директории пакета.
func (l *AutonomousPackageLoader) checkVersionTgInDir(pkgDir string) (hasVersionTg bool) {

	var entries []os.DirEntry
	var err error
	if entries, err = os.ReadDir(pkgDir); err != nil {
		return
	}

	fset := token.NewFileSet()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(pkgDir, entry.Name())
		var file *ast.File
		if file, err = parser.ParseFile(fset, filePath, nil, 0); err != nil {
			continue
		}

		for _, decl := range file.Decls {
			var genDecl *ast.GenDecl
			var ok bool
			if genDecl, ok = decl.(*ast.GenDecl); ok && genDecl.Tok == token.CONST {
				for _, spec := range genDecl.Specs {
					var valueSpec *ast.ValueSpec
					if valueSpec, ok = spec.(*ast.ValueSpec); ok {
						for _, name := range valueSpec.Names {
							if name.Name == "VersionTg" {
								hasVersionTg = true
								return
							}
						}
					}
				}
			}
		}
	}

	return
}
