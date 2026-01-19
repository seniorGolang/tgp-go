// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"tgp/core/i18n"
	"tgp/internal"
	"tgp/internal/model"
)

// findServices находит все main файлы в проекте и создает сервисы.
func findServices(project *model.Project, loader *AutonomousPackageLoader) (err error) {

	servicesMap := make(map[string]*model.Service)

	// Сначала быстро сканируем файлы на наличие "func main" без полного парсинга
	candidateFiles := make([]string, 0)
	totalFiles := 0

	err = filepath.Walk(internal.ProjectRoot, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		if info.IsDir() {
			// Пропускаем служебные директории
			dirName := info.Name()
			if dirName == "vendor" || dirName == "node_modules" || dirName == ".git" ||
				dirName == ".tg" || dirName == "dist" || dirName == "build" {
				return filepath.SkipDir
			}
			if shouldExcludeDir(filePath, project.ExcludeDirs) {
				return filepath.SkipDir
			}
			return nil
		}

		if shouldExcludeDir(filepath.Dir(filePath), project.ExcludeDirs) {
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		// Пропускаем тестовые файлы
		if strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		if isGeneratedFile(filePath) {
			return nil
		}

		totalFiles++

		// Быстрое сканирование на наличие "func main" без полного парсинга
		if hasMainFunction(filePath) {
			candidateFiles = append(candidateFiles, filePath)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Теперь парсим только файлы с func main
	for _, filePath := range candidateFiles {
		fset := token.NewFileSet()
		var file *ast.File
		if file, err = parser.ParseFile(fset, filePath, nil, parser.ParseComments); err != nil {
			slog.Debug(i18n.Msg("Failed to parse candidate file"), slog.String("file", filePath), slog.String("error", err.Error()))
			continue
		}

		var mainFunc *ast.FuncDecl
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name != nil && fn.Name.Name == "main" {
				mainFunc = fn
				break
			}
		}

		if mainFunc == nil {
			continue
		}

		if !isServiceMain(file, mainFunc, project, loader) {
			continue
		}

		serviceName := extractServiceName(filePath)
		mainPathRel := makeRelativePath(filePath)
		service := &model.Service{
			Name:        serviceName,
			MainPath:    mainPathRel,
			ContractIDs: make([]string, 0),
		}

		contractIDs := findContractsInMainFile(file, filePath, project, loader)
		service.ContractIDs = contractIDs

		servicesMap[filePath] = service

		// DEBUG: логируем найденный сервис
		slog.Debug(i18n.Msg("Service found"),
			slog.String("name", serviceName),
			slog.String("mainPath", mainPathRel),
			slog.Int("contractsCount", len(contractIDs)))
	}

	for _, service := range servicesMap {
		project.Services = append(project.Services, service)
	}

	return
}

// extractServiceName извлекает имя сервиса из пути к main файлу.
func extractServiceName(mainPath string) (name string) {

	var relPath string
	var err error
	if relPath, err = filepath.Rel(internal.ProjectRoot, mainPath); err != nil {
		name = filepath.Base(filepath.Dir(mainPath))
		return
	}

	name = strings.TrimSuffix(filepath.Base(relPath), ".go")
	if name == "main" {
		name = filepath.Base(filepath.Dir(relPath))
	}

	return
}

// findContractsInMainFile находит контракты, используемые в main файле.
func findContractsInMainFile(file *ast.File, filePath string, project *model.Project, loader *AutonomousPackageLoader) (contractIDs []string) {

	contractIDs = make([]string, 0)
	contractNames := make(map[string]*model.Contract)
	for _, contract := range project.Contracts {
		contractNames[contract.Name] = contract
	}

	importAliases := make(map[string]string)
	for _, imp := range file.Imports {
		impPath := strings.Trim(imp.Path.Value, "\"")
		var alias string
		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			parts := strings.Split(impPath, "/")
			alias = parts[len(parts)-1]
		}
		importAliases[alias] = impPath
	}

	transportAliases := make(map[string]bool)
	for alias, impPath := range importAliases {
		if hasVersionTgConstant(impPath, project, loader) {
			transportAliases[alias] = true
		}
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if node, ok := n.(*ast.CallExpr); ok {
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				if transportAlias, ok := sel.X.(*ast.Ident); ok {
					if transportAliases[transportAlias.Name] {
						if contract, exists := contractNames[sel.Sel.Name]; exists {
							contractID := contract.ID
							found := false
							for _, id := range contractIDs {
								if id == contractID {
									found = true
									break
								}
							}
							if !found {
								contractIDs = append(contractIDs, contractID)
							}
						}
					}
				}
			}
		}
		return true
	})

	return
}

// isGeneratedFile проверяет, является ли файл сгенерированным.
func isGeneratedFile(filePath string) (isGenerated bool) {

	var file *os.File
	var err error
	if file, err = os.Open(filePath); err != nil {
		return
	}
	defer file.Close()

	buf := make([]byte, 200)
	var n int
	n, _ = file.Read(buf)
	content := string(buf[:n])

	isGenerated = strings.Contains(content, "GENERATED BY 'T'ransport 'G'enerator. DO NOT EDIT.") ||
		strings.Contains(content, "This file is auto-generated. Do not edit manually.")
	return
}

// hasMainFunction быстро проверяет наличие "func main" в файле без полного парсинга.
func hasMainFunction(filePath string) (hasMain bool) {

	var file *os.File
	var err error
	if file, err = os.Open(filePath); err != nil {
		return
	}
	defer file.Close()

	// Читаем первые 8KB файла - этого достаточно для поиска func main
	buf := make([]byte, 8192)
	var n int
	if n, err = file.Read(buf); err != nil && err != io.EOF {
		return
	}

	content := string(buf[:n])

	// Ищем паттерн "func main" с учетом возможных пробелов и переносов строк
	// Проверяем несколько вариантов для надежности
	patterns := []string{
		"func main(",
		"func main (",
		"func\tmain(",
		"func\nmain(",
		"func\r\nmain(",
	}

	for _, pattern := range patterns {
		if strings.Contains(content, pattern) {
			hasMain = true
			return
		}
	}

	return
}

// isServiceMain проверяет, является ли main файл сервисом.
func isServiceMain(file *ast.File, mainFunc *ast.FuncDecl, project *model.Project, loader *AutonomousPackageLoader) (isService bool) {

	contractNames := make(map[string]bool)
	for _, contract := range project.Contracts {
		contractNames[contract.Name] = true
	}

	transportAliases := make(map[string]string)
	for _, imp := range file.Imports {
		impPath := strings.Trim(imp.Path.Value, "\"")
		var alias string
		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			parts := strings.Split(impPath, "/")
			alias = parts[len(parts)-1]
		}

		if hasVersionTgConstant(impPath, project, loader) {
			transportAliases[alias] = impPath
		}
	}

	if len(transportAliases) == 0 {
		return false
	}

	transportAlias := ""
	for alias := range transportAliases {
		if isTransportUsedInMain(mainFunc, alias, contractNames) {
			transportAlias = alias
			break
		}
	}

	if transportAlias == "" {
		return false
	}

	hasServerCreation := false
	hasContractRegistration := false
	hasListenCall := false

	ast.Inspect(mainFunc.Body, func(n ast.Node) bool {
		if node, ok := n.(*ast.CallExpr); ok {
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				// Проверяем transportAlias.New() - создание сервера
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == transportAlias {
					if sel.Sel.Name == "New" {
						hasServerCreation = true
						for _, arg := range node.Args {
							if checkContractRegistration(arg, contractNames, transportAlias) {
								hasContractRegistration = true
								break
							}
						}
					}
				}

				// Проверяем transportAlias.ContractName() - регистрация контрактов
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == transportAlias {
					if contractNames[sel.Sel.Name] {
						hasContractRegistration = true
					}
				}

				// Проверяем методы запуска сервера
				methodName := sel.Sel.Name
				if methodName == "Listen" || methodName == "Serve" || methodName == "ServeMetrics" {
					hasListenCall = true
				}

				// Проверяем цепочку вызовов типа srv.Fiber().Listen()
				// Если это вызов метода на результате другого вызова
				if _, ok := sel.X.(*ast.CallExpr); ok {
					// Проверяем, что это вызов метода на результате другого вызова
					// (может быть srv.Fiber().Listen() или подобное)
					if methodName == "Listen" || methodName == "Serve" {
						hasListenCall = true
					}
				}
			}
		}
		return true
	})

	isService = hasServerCreation && hasContractRegistration && hasListenCall
	return
}

// checkContractRegistration проверяет, является ли вызов регистрацией контракта.
func checkContractRegistration(node ast.Node, contractNames map[string]bool, transportAlias string) (isRegistration bool) {

	var callExpr *ast.CallExpr
	var ok bool
	if callExpr, ok = node.(*ast.CallExpr); !ok {
		var compLit *ast.CompositeLit
		if compLit, ok = node.(*ast.CompositeLit); ok {
			for _, elt := range compLit.Elts {
				if checkContractRegistration(elt, contractNames, transportAlias) {
					isRegistration = true
					return
				}
			}
		}
		return
	}

	var sel *ast.SelectorExpr
	if sel, ok = callExpr.Fun.(*ast.SelectorExpr); ok {
		var x *ast.Ident
		if x, ok = sel.X.(*ast.Ident); ok && x.Name == transportAlias {
			contractName := sel.Sel.Name
			if contractNames[contractName] {
				isRegistration = true
				return
			}
		}
	}

	return
}

// hasVersionTgConstant проверяет наличие константы VersionTg в пакете.
func hasVersionTgConstant(pkgPath string, project *model.Project, loader *AutonomousPackageLoader) (hasVersion bool) {

	hasVersion = loader.HasVersionTgConstant(pkgPath)
	return
}

// isTransportUsedInMain проверяет, используется ли transport пакет в main.
func isTransportUsedInMain(mainFunc *ast.FuncDecl, alias string, contractNames map[string]bool) (isUsed bool) {

	hasNewCall := false
	hasContractCall := false

	ast.Inspect(mainFunc.Body, func(n ast.Node) bool {
		var callExpr *ast.CallExpr
		var ok bool
		if callExpr, ok = n.(*ast.CallExpr); ok {
			var sel *ast.SelectorExpr
			if sel, ok = callExpr.Fun.(*ast.SelectorExpr); ok {
				var x *ast.Ident
				if x, ok = sel.X.(*ast.Ident); ok && x.Name == alias {
					if sel.Sel.Name == "New" {
						hasNewCall = true
					}
					if contractNames[sel.Sel.Name] {
						hasContractCall = true
					}
				}
			}
		}
		return true
	})

	isUsed = hasNewCall && hasContractCall
	return
}

// shouldExcludeDir проверяет, нужно ли исключить директорию из анализа.
func shouldExcludeDir(dirPath string, excludeDirs []string) (shouldExclude bool) {

	if len(excludeDirs) == 0 {
		return
	}

	var relPath string
	var err error
	if relPath, err = filepath.Rel(internal.ProjectRoot, dirPath); err != nil {
		return
	}

	relPath = strings.TrimPrefix(relPath, "./")
	relPath = strings.TrimPrefix(relPath, ".\\")
	relPath = filepath.ToSlash(relPath)

	separator := "/"

	for _, excludeDir := range excludeDirs {
		excludeDir = strings.TrimPrefix(excludeDir, "./")
		excludeDir = strings.TrimPrefix(excludeDir, ".\\")
		excludeDir = filepath.ToSlash(excludeDir)

		if relPath == excludeDir {
			shouldExclude = true
			return
		}

		if strings.HasPrefix(relPath, excludeDir+separator) {
			shouldExclude = true
			return
		}
	}

	return
}
