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

func CollectWithExcludeDirs(version string, svcDir string, excludeDirs []string, ifaces ...string) (project *model.Project, err error) {

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
		err = fmt.Errorf("failed to get go.mod path: %w", err)
		project = nil
		return
	}

	modBytes, err := os.ReadFile(modPath)
	var modFile *modfile.File
	if err == nil {
		modFile, err = modfile.Parse(modPath, modBytes, nil)
		if err != nil {
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
		err = fmt.Errorf("failed to create package loader: %w", err)
		project = nil
		return
	}

	svcDirAbs := filepath.Join(internal.ProjectRoot, svcDir)

	var files []os.DirEntry
	if files, err = os.ReadDir(svcDirAbs); err != nil {
		err = fmt.Errorf("failed to read service directory: %w", err)
		project = nil
		return
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
				err = fmt.Errorf("failed to parse file %s: %w", filePathAbs, parseErr)
				project = nil
				return
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

		pkgInfo, err := loader.LoadPackageLazy(pkgPath)
		if err != nil {
			slog.Debug(i18n.Msg("Package not found, skipping file"),
				slog.String("package", pkgPath),
				slog.String("file", filePathAbs),
				slog.String("error", err.Error()))
			continue
		}

		var astFile *ast.File
		fileName := filepath.Base(filePathAbs)
		found := false
		for _, file := range pkgInfo.Files {
			if file != nil {
				if file.Package.IsValid() {
					pos := pkgInfo.Fset.Position(file.Package)
					if pos.Filename == filePathAbs || filepath.Base(pos.Filename) == fileName {
						astFile = file
						found = true
						break
					}
				}
			}
		}

		if !found && len(pkgInfo.Files) > 0 {
			for _, file := range pkgInfo.Files {
				if file != nil {
					astFile = file
					found = true
					break
				}
			}
		}

		if !found {
			fset := token.NewFileSet()
			astFile, err = parser.ParseFile(fset, filePathAbs, nil, parser.ParseComments)
			if err != nil {
				slog.Debug(i18n.Msg("Failed to parse file"), slog.String("file", filePathAbs), slog.String("error", err.Error()))
				continue
			}
		}

		imports := collectImports([]*ast.File{astFile})

		if astFile.Doc != nil && len(project.Annotations) == 0 {
			packageDocs := extractComments(astFile.Doc)
			project.Annotations = project.Annotations.Merge(tags.ParseTags(packageDocs))
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
				contract := &model.Contract{
					ID:          contractID,
					Name:        interfaceName,
					PkgPath:     pkgPath,
					FilePath:    filePathRel,
					Docs:        removeAnnotationsFromDocs(interfaceDocs),
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
			err = fmt.Errorf("failed to compute relative path from %s to %s: %w", internal.ProjectRoot, dir, err)
			return
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

func extractComments(commentGroups ...*ast.CommentGroup) (comments []string) {

	for _, group := range commentGroups {
		if group == nil {
			continue
		}
		for _, comment := range group.List {
			comments = append(comments, comment.Text)
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

func removeAnnotationsFromDocs(docs []string) (filtered []string) {

	if len(docs) == 0 {
		filtered = docs
		return
	}

	filtered = make([]string, 0, len(docs))
	for _, doc := range docs {
		trimmed := strings.TrimSpace(strings.TrimPrefix(doc, "//"))
		if strings.HasPrefix(trimmed, "@tg") {
			continue
		}
		filtered = append(filtered, doc)
	}
	return
}
