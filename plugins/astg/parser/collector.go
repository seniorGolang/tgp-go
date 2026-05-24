// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"

	"tgp/core/i18n"
	"tgp/internal"
	"tgp/internal/model"
	"tgp/internal/tags"
)

func CollectWithExcludeDirs(version string, svcDir string, excludeDirs []string) (project *model.Project, err error) {

	project = &model.Project{
		Version:      version,
		ContractsDir: svcDir,
		Types:        make(map[string]*model.Type),
		Contracts:    make([]*model.Contract, 0),
		Services:     make([]*model.Service, 0),
		ExcludeDirs:  excludeDirs,
	}

	var modPath string
	if modPath, err = findGoModPath(); err != nil {
		return nil, fmt.Errorf("failed to get go.mod path: %w", err)
	}

	modBytes, err := os.ReadFile(modPath)
	var modFile *modfile.File
	if err == nil {
		if modFile, err = modfile.Parse(modPath, modBytes, nil); err != nil {
			modFile = nil
		}
	}

	if modFile != nil && modFile.Module != nil {
		project.ModulePath = modFile.Module.Mod.Path
	} else {
		project.ModulePath = svcDir
	}

	if err = collectGitInfo(project); err != nil {
		slog.Debug(i18n.Msg("Failed to collect git info"), slog.String("error", err.Error()))
	}

	var loader *AutonomousPackageLoader
	if loader, err = NewAutonomousPackageLoader(modFile); err != nil {
		return nil, fmt.Errorf("failed to create package loader: %w", err)
	}

	svcDirAbs := filepath.Join(internal.ProjectRoot, svcDir)

	var files []os.DirEntry
	if files, err = os.ReadDir(svcDirAbs); err != nil {
		return nil, fmt.Errorf("failed to read service directory: %w", err)
	}

	contractsMap := make(map[string]*model.Contract)

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") {
			continue
		}

		filePathAbs := filepath.Join(svcDirAbs, file.Name())

		dir := filepath.Dir(filePathAbs)
		var pkgPath string
		var pkgPathErr error
		if pkgPath, pkgPathErr = getPkgPathFromDir(dir, project.ModulePath); pkgPathErr != nil {
			slog.Debug(i18n.Msg("Failed to get package path"),
				slog.String("path", filePathAbs),
				slog.String("dir", dir),
				slog.String("modulePath", project.ModulePath),
				slog.String("error", pkgPathErr.Error()))
			fset := token.NewFileSet()
			var astFile *ast.File
			var parseErr error
			if astFile, parseErr = parser.ParseFile(fset, filePathAbs, nil, parser.ParseComments); parseErr != nil {
				return nil, fmt.Errorf("failed to parse file %s: %w", filePathAbs, parseErr)
			}
			relPath, relErr := filepath.Rel(internal.ProjectRoot, filepath.Dir(filePathAbs))
			if relErr == nil {
				pkgRelPath := filepath.ToSlash(relPath)
				pkgRelPath = strings.TrimPrefix(pkgRelPath, "./")
				if pkgRelPath == "" || pkgRelPath == "." {
					pkgPath = project.ModulePath + "/" + astFile.Name.Name
				} else {
					pkgPath = project.ModulePath + "/" + pkgRelPath
				}
			} else {
				pkgPath = project.ModulePath + "/" + astFile.Name.Name
			}
		}

		var pkgInfo *PackageInfo
		if pkgInfo, err = loader.LoadPackageLazy(pkgPath); err != nil {
			slog.Debug(i18n.Msg("Package not found, skipping file"),
				slog.String("package", pkgPath),
				slog.String("file", filePathAbs),
				slog.String("error", err.Error()))
			continue
		}

		var astFile *ast.File
		fileName := filepath.Base(filePathAbs)
		found := false
		for _, pkgFile := range pkgInfo.Files {
			if pkgFile != nil {
				if pkgFile.Package.IsValid() {
					pos := pkgInfo.Fset.Position(pkgFile.Package)
					if pos.Filename == filePathAbs || filepath.Base(pos.Filename) == fileName {
						astFile = pkgFile
						found = true
						break
					}
				}
			}
		}

		if !found && len(pkgInfo.Files) > 0 {
			for _, pkgFile := range pkgInfo.Files {
				if pkgFile != nil {
					astFile = pkgFile
					found = true
					break
				}
			}
		}

		if !found {
			fset := token.NewFileSet()
			if astFile, err = parser.ParseFile(fset, filePathAbs, nil, parser.ParseComments); err != nil {
				slog.Debug(i18n.Msg("Failed to parse file"), slog.String("file", filePathAbs), slog.String("error", err.Error()))
				continue
			}
		}

		imports := collectImports([]*ast.File{astFile}, loader.resolver)

		if astFile.Doc != nil {
			packageLines := extractComments(astFile.Doc)
			if len(project.Docs) == 0 {
				project.Docs, project.Directives = splitDocsAndDirectives(packageLines)
			}
			if len(project.Annotations) == 0 {
				project.Annotations = project.Annotations.Merge(tags.ParseTags(packageLines))
			}
		}

		filePathRel := makeRelativePath(filePathAbs)
		for _, decl := range astFile.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}

				interfaceName := typeSpec.Name.Name

				interfaceDocs := extractComments(genDecl.Doc, typeSpec.Doc, typeSpec.Comment)
				ifaceAnnotations := tags.ParseTags(interfaceDocs)
				if len(ifaceAnnotations) == 0 {
					continue
				}

				contractID := fmt.Sprintf("%s:%s", pkgPath, interfaceName)
				interfaceDocsOut, interfaceDirectives := splitDocsAndDirectives(interfaceDocs)
				contract := &model.Contract{
					ID:          contractID,
					Name:        interfaceName,
					PkgPath:     pkgPath,
					FilePath:    filePathRel,
					Docs:        interfaceDocsOut,
					Directives:  interfaceDirectives,
					Annotations: ifaceAnnotations,
					Methods:     make([]*model.Method, 0),
				}

				if interfaceType.Methods != nil {
					typeInfo := pkgInfo.TypeInfo
					if typeInfo == nil {
						cfg := &types.Config{Importer: &FileSystemImporter{loader: loader, cache: make(map[string]*types.Package)}}
						typeInfo = &types.Info{
							Types:      make(map[ast.Expr]types.TypeAndValue),
							Defs:       make(map[*ast.Ident]types.Object),
							Uses:       make(map[*ast.Ident]types.Object),
							Implicits:  make(map[ast.Node]types.Object),
							Selections: make(map[*ast.SelectorExpr]*types.Selection),
							Scopes:     make(map[ast.Node]*types.Scope),
						}
						_, err := cfg.Check(pkgPath, pkgInfo.Fset, pkgInfo.Files, typeInfo)
						if err != nil {
							slog.Debug(i18n.Msg("Failed to create TypeInfo for contract package"), slog.String("package", pkgPath), slog.String("error", err.Error()))
							continue
						}
						pkgInfo.TypeInfo = typeInfo
					}
					for _, methodField := range interfaceType.Methods.List {
						if _, ok := methodField.Type.(*ast.Ident); ok {
							continue
						}
						if _, ok := methodField.Type.(*ast.SelectorExpr); ok {
							continue
						}

						funcType, ok := methodField.Type.(*ast.FuncType)
						if !ok {
							continue
						}

						methodName := ""
						if len(methodField.Names) > 0 && methodField.Names[0] != nil {
							methodName = methodField.Names[0].Name
						}
						if methodName == "" {
							continue
						}

						method := convertMethod(methodName, funcType, extractComments(methodField.Doc, methodField.Comment), contractID, pkgPath, imports, typeInfo, project, loader)
						if method != nil {
							contract.Methods = append(contract.Methods, method)
						}
					}
				}

				contractsMap[contractID] = contract
			}
		}
	}

	for _, contract := range contractsMap {
		project.Contracts = append(project.Contracts, contract)
	}

	if err = analyzeProject(project, loader); err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	return
}

func findGoModPath() (modPath string, err error) {

	if _, err = os.Stat("/go.mod"); err != nil {
		err = errors.New(i18n.Msg("go.mod not found: @go resolution not provided or go.mod is missing in /go.mod"))
		return
	}
	modPath = "/go.mod"
	return
}

func getPkgPathFromDir(dir string, modulePath string) (pkgPath string, err error) {

	var relPath string
	if relPath, err = filepath.Rel(internal.ProjectRoot, dir); err != nil {
		if strings.HasPrefix(dir, "/") {
			relPath = strings.TrimPrefix(dir, "/")
		} else {
			return "", fmt.Errorf("failed to compute relative path from %s to %s: %w", internal.ProjectRoot, dir, err)
		}
	}
	pkgRelPath := filepath.ToSlash(relPath)
	pkgRelPath = strings.TrimPrefix(pkgRelPath, "./")
	if pkgRelPath == "" || pkgRelPath == "." {
		pkgPath = modulePath
		return
	}
	pkgPath = modulePath + "/" + pkgRelPath
	return
}

// normalizeCommentLine убирает префикс комментария (//, /*, */, * ) и лишние пробелы.
// Пустая строка после нормализации возвращается как "".
func normalizeCommentLine(line string) (out string) {

	s := strings.TrimSpace(line)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "//") {
		return strings.TrimSpace(s[2:])
	}
	if strings.HasPrefix(s, "/*") {
		s = strings.TrimSpace(strings.TrimSuffix(s[2:], "*/"))
		return strings.TrimSpace(s)
	}
	if strings.HasPrefix(s, "*") {
		return strings.TrimSpace(s[1:])
	}
	return s
}

func extractComments(commentGroups ...*ast.CommentGroup) (comments []string) {

	for _, group := range commentGroups {
		if group == nil {
			continue
		}
		for _, comment := range group.List {
			normalized := normalizeCommentLine(comment.Text)
			if normalized != "" {
				comments = append(comments, normalized)
			}
		}
	}
	return
}

func makeRelativePath(absPath string) (relPath string) {

	var err error
	if relPath, err = filepath.Rel(internal.ProjectRoot, absPath); err != nil {
		relPath = absPath
		return
	}
	relPath = filepath.ToSlash(relPath)
	return
}

func isDirective(line string) (ok bool) {
	return strings.HasPrefix(line, "go:") || strings.HasPrefix(line, "+")
}

func splitDocsAndDirectives(lines []string) (docs, directives []string) {

	for _, line := range lines {
		if isDirective(line) {
			directives = append(directives, line)
			continue
		}
		if strings.Contains(line, "@tg") {
			continue
		}
		docs = append(docs, line)
	}
	return
}
