// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func (l *AutonomousPackageLoader) HasVersionASTgConstant(pkgPath string) (hasVersionASTg bool) {

	l.versionASTgCacheMu.RLock()
	var cached bool
	var ok bool
	if cached, ok = l.versionASTgCache[pkgPath]; ok {
		l.versionASTgCacheMu.RUnlock()
		hasVersionASTg = cached
		return
	}
	l.versionASTgCacheMu.RUnlock()

	var pkgDir string
	var err error
	if pkgDir, err = l.resolver.Resolve(pkgPath); err != nil {
		l.versionASTgCacheMu.Lock()
		l.versionASTgCache[pkgPath] = false
		l.versionASTgCacheMu.Unlock()
		return
	}

	hasVersionASTg = l.checkVersionASTgInDir(pkgDir)

	l.versionASTgCacheMu.Lock()
	l.versionASTgCache[pkgPath] = hasVersionASTg
	l.versionASTgCacheMu.Unlock()

	return
}

func (l *AutonomousPackageLoader) checkVersionASTgInDir(pkgDir string) (hasVersionASTg bool) {

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
							if name.Name == "VersionASTg" {
								hasVersionASTg = true
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
