// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"tgp/core/i18n"
)

func (l *AutonomousPackageLoader) parsePackageFiles(pkgDir string, buildCtx *build.Context) (files []*ast.File, err error) {

	var entries []os.DirEntry
	if entries, err = os.ReadDir(pkgDir); err != nil {
		return
	}

	files = make([]*ast.File, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(pkgDir, entry.Name())
		var match bool
		if match, err = buildCtx.MatchFile(pkgDir, entry.Name()); err != nil {
			slog.Debug(i18n.Msg("Failed to check build tags"), slog.String("file", filePath), slog.Any("error", err))
			continue
		}
		if !match {
			continue
		}

		var file *ast.File
		if file, err = parser.ParseFile(l.fset, filePath, nil, parser.ParseComments); err != nil {
			slog.Debug(i18n.Msg("Failed to parse file"), slog.String("file", filePath), slog.Any("error", err))
			continue
		}

		files = append(files, file)
	}

	return
}

func (l *AutonomousPackageLoader) ParsePackageFilesOnly(pkgPath string) (files []*ast.File, fset *token.FileSet, err error) {

	var pkgDir string
	if pkgDir, err = l.resolver.Resolve(pkgPath); err != nil {
		err = fmt.Errorf("failed to resolve package path %s: %w", pkgPath, err)
		return
	}

	buildCtx := buildContext()
	if files, err = l.parsePackageFiles(pkgDir, &buildCtx); err != nil {
		err = fmt.Errorf("failed to parse package files in %s: %w", pkgDir, err)
		return
	}

	if len(files) == 0 {
		err = fmt.Errorf("no Go files found in package %s", pkgPath)
		return
	}

	fset = l.fset
	return
}
