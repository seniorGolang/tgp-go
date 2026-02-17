// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/core/plugin"
	"tgp/internal"
	"tgp/internal/helper"
	"tgp/internal/model"
	"tgp/plugins/astg/cache"
	"tgp/plugins/astg/descref"
	"tgp/plugins/astg/generator"
	"tgp/plugins/astg/parser"
)

//go:embed plugin.md
var pluginDoc string

type AstgPlugin struct{}

func (p *AstgPlugin) Execute(request data.Storage) (response data.Storage, err error) {

	slog.SetDefault(slog.Default().With(
		slog.String("plugin", "astg"),
	))

	response = request

	if request != nil && request.Has("project") {
		slog.Debug(i18n.Msg("project already exists in request, skipping analysis"))
		return
	}

	contractsDir := "./contracts"
	var contractsDirStr string
	if contractsDirStr, err = data.Get[string](request, "contracts-dir"); err == nil && contractsDirStr != "" {
		contractsDir = contractsDirStr
	}

	var contractsFilter []string
	if contractsFilter, err = helper.ParseStringList(request, "contracts"); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("failed to parse contracts"), err)
		return
	}

	contractsDisplay := "all"
	if len(contractsFilter) > 0 {
		contractsDisplay = strings.Join(contractsFilter, ", ")
	}

	noCache, _ := data.Get[bool](request, "no-cache")

	slog.Info(i18n.Msg("analyzing project"),
		slog.String("contractsDir", contractsDir),
		slog.String("contracts", contractsDisplay),
		slog.Bool("no-cache", noCache),
	)

	var fromCache bool
	var projectID string
	var project *model.Project

	if noCache {
		fromCache = false
		if projectID, err = cache.GetProjectID(internal.ProjectRoot); err != nil {
			slog.Debug(i18n.Msg("failed to compute project ID"), slog.String("error", err.Error()))
		}
	} else {
		project, fromCache, projectID = cache.GetProject(internal.ProjectRoot)
	}

	if !fromCache {
		if project, err = parser.CollectWithExcludeDirs(internal.Version, contractsDir, nil); err != nil || project == nil {
			err = fmt.Errorf("%s: %w", i18n.Msg("failed to collect project"), err)
			return
		}

		if projectID == "" {
			if projectID, err = cache.GetProjectID(internal.ProjectRoot); err == nil {
				project.ProjectID = projectID
			}
		} else {
			project.ProjectID = projectID
		}

		descref.ResolveFileRefsInProject(project, internal.ProjectRoot)
		if projectID != "" {
			cache.SaveProject(projectID, project, internal.ProjectRoot, contractsDir, nil)
		}
	}

	if len(contractsFilter) > 0 {
		var filteredContracts []*model.Contract
		if filteredContracts, err = helper.FilterContractsByInterfaces(project, contractsFilter); err != nil {
			err = fmt.Errorf("%s: %w", i18n.Msg("failed to filter contracts"), err)
			return
		}
		filteredProject := *project
		filteredProject.Contracts = filteredContracts
		project = &filteredProject
	}

	for _, contract := range project.Contracts {
		errorTypesMap := make(map[string]bool)
		for _, impl := range contract.Implementations {
			for _, method := range impl.MethodsMap {
				for _, errorType := range method.ErrorTypes {
					errorTypesMap[errorType.FullName] = true
				}
			}
		}

		slog.Info(contract.Name,
			slog.String("pkgPath", contract.PkgPath),
			slog.Int("methodsCount", len(contract.Methods)),
			slog.Int("implementationsCount", len(contract.Implementations)),
			slog.Int("errorsCount", len(errorTypesMap)),
		)
	}

	if logLevel, _ := data.Get[string](request, "log-level"); strings.EqualFold(logLevel, "debug") {
		if saveErr := saveProjectJSON(project); saveErr != nil {
			slog.Debug(i18n.Msg("failed to save project.json"), slog.String("error", saveErr.Error()))
		}
	}

	if err = response.Set("project", project); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("failed to set project in response"), err)
		return
	}

	slog.Info(i18n.Msg("analysis completed"))
	return
}

func saveProjectJSON(project *model.Project) (err error) {

	dir := filepath.Join(internal.ProjectRoot, ".tg")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return
	}

	var jsonData []byte
	if jsonData, err = json.MarshalIndent(project, "", "  "); err != nil {
		return
	}

	path := filepath.Join(dir, "project.json")
	return os.WriteFile(path, jsonData, 0600)
}

func (p *AstgPlugin) Info() (info plugin.Info, err error) {

	info = plugin.Info{
		Name:             "astg",
		Doc:              pluginDoc,
		Description:      i18n.Msg("Project AST parser plugin"),
		Author:           "AlexK <seniorGolang@gmail.com>",
		License:          "MIT",
		Category:         "parser",
		AllowedShellCMDs: []string{"go"},
		AllowedEnvVars: []string{
			"GOPATH",     // Для поиска пакетов в GOPATH/src и модулей в GOPATH/pkg/mod
			"GOROOT",     // Для поиска стандартной библиотеки
			"GOMODCACHE", // Для поиска модулей в кэше модулей
			"GOOS",       // Для build.Context при парсинге файлов с build tags
			"GOARCH",     // Для build.Context при парсинге файлов с build tags
			"GOCACHE",    // Для чтения export data скомпилированных пакетов
		},
		InitPkgs: []string{"astg"},
		AllowedPaths: map[string]string{
			"@go":            "w", // Доступ к директории с go.mod (монтируется хостом в корень "/")
			"$GOPATH":        "r", // Для чтения пакетов из GOPATH/src (для go/types и go/build)
			"$GOROOT":        "r", // Для чтения стандартной библиотеки Go (для go/types и go/build)
			"$GOMODCACHE":    "r", // Для чтения модулей из кэша (для go/types и go/build)
			"$GOCACHE":       "r", // Для чтения export data скомпилированных пакетов
			"@tg/astg/cache": "w", // Доступ на запись к кэшу проектов
		},
	}
	return
}

func (p *AstgPlugin) Generate(rootDir string, moduleName string) (err error) {

	slog.Info(i18n.Msg("generating astg package"), slog.String("rootDir", rootDir), slog.String("moduleName", moduleName))
	if err = generator.Generate(rootDir, moduleName); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("failed to generate astg package"), err)
		return
	}
	slog.Info(i18n.Msg("astg package generated successfully"))
	return
}

func (p *AstgPlugin) Cleanup(rootDir string) (err error) {

	slog.Info(i18n.Msg("cleaning up astg package"), slog.String("rootDir", rootDir))
	if err = generator.Cleanup(rootDir); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("failed to cleanup astg package"), err)
		return
	}
	slog.Info(i18n.Msg("astg package cleaned up successfully"))
	return
}
