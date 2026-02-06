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
	"tgp/plugins/astg/generator"
	"tgp/plugins/astg/marker"
	"tgp/plugins/astg/parser"
)

//go:embed plugin.md
var pluginDoc string

type AstgPlugin struct{}

func (p *AstgPlugin) Execute(rootDir string, request data.Storage, path ...string) (response data.Storage, err error) {

	// Настраиваем дефолтный slog с контекстом плагина
	slog.SetDefault(slog.Default().With(
		slog.String("plugin", "astg"),
	))

	response = request

	// Если project уже есть в request, не пересоздаем его
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
	var currentMarker string
	var project *model.Project

	if noCache {
		// Если no-cache установлен, игнорируем кэш и всегда выполняем парсинг
		fromCache = false
		// Вычисляем projectID и marker заранее для последующего сохранения в кэш
		if projectID, err = cache.GetProjectID(rootDir); err != nil {
			slog.Debug(i18n.Msg("failed to compute project ID"), slog.String("error", err.Error()))
		}
		if currentMarker, err = marker.ComputeMarker(rootDir); err != nil {
			slog.Debug(i18n.Msg("failed to compute marker"), slog.String("error", err.Error()))
		}
	} else {
		project, fromCache, projectID, currentMarker = cache.GetProject(internal.ProjectRoot)
	}

	// Если проект не из кэша, выполняем парсинг и сохраняем в кэш
	if !fromCache {
		// Выполняем парсинг проекта (всегда собираем все контракты, игнорируя ifaces)
		if project, err = parser.CollectWithExcludeDirs(internal.Version, contractsDir, nil); err != nil {
			err = fmt.Errorf("%s: %w", i18n.Msg("failed to collect project"), err)
			return
		}

		// Если projectID и marker уже вычислены в GetProject, используем их
		// Если нет, вычисляем заново
		if projectID == "" {
			if projectID, err = cache.GetProjectID(rootDir); err == nil {
				project.ProjectID = projectID
			}
		} else {
			project.ProjectID = projectID
		}

		if currentMarker == "" {
			if currentMarker, err = marker.ComputeMarker(rootDir); err == nil {
				project.Marker = currentMarker
			}
		} else {
			project.Marker = currentMarker
		}

		// Сохраняем проект в кэш (используем уже вычисленные projectID и marker)
		// В кэш сохраняются полные данные без фильтрации
		if projectID != "" {
			cache.SaveProject(projectID, currentMarker, project)
		}
	}

	// Применяем фильтрацию по contracts после загрузки проекта (из кэша или после парсинга)
	// Фильтрация применяется к копии проекта, чтобы не изменять данные в кэше
	if len(contractsFilter) > 0 {
		var filteredContracts []*model.Contract
		if filteredContracts, err = helper.FilterContractsByInterfaces(project, contractsFilter); err != nil {
			err = fmt.Errorf("%s: %w", i18n.Msg("failed to filter contracts"), err)
			return
		}
		// Создаем копию проекта с отфильтрованными контрактами
		filteredProject := *project
		filteredProject.Contracts = filteredContracts
		project = &filteredProject
	}

	// Выводим сводку по каждому контракту отдельно
	for _, contract := range project.Contracts {
		// Подсчитываем уникальные ошибки во всех реализациях
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
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var jsonData []byte
	if jsonData, err = json.MarshalIndent(project, "", "  "); err != nil {
		return err
	}

	path := filepath.Join(dir, "project.json")
	return os.WriteFile(path, jsonData, 0600)
}

func (p *AstgPlugin) Info() (info plugin.Info, err error) {

	info = plugin.Info{
		Name:        "astg",
		Doc:         pluginDoc,
		Description: i18n.Msg("Project AST parser plugin"),
		Author:      "AlexK <seniorGolang@gmail.com>",
		License:     "MIT",
		Category:    "parser",
		AllowedEnvVars: []string{
			"GOPATH",     // Для поиска пакетов в GOPATH/src и модулей в GOPATH/pkg/mod
			"GOROOT",     // Для поиска стандартной библиотеки
			"GOMODCACHE", // Для поиска модулей в кэше модулей
			"GOOS",       // Для build.Context при парсинге файлов с build tags
			"GOARCH",     // Для build.Context при парсинге файлов с build tags
		},
		InitPkgs: []string{"astg"},
		AllowedPaths: map[string]string{
			"@go":            "w", // Доступ к директории с go.mod (монтируется хостом в корень "/")
			"$GOPATH":        "r", // Для чтения пакетов из GOPATH/src (для go/types и go/build)
			"$GOROOT":        "r", // Для чтения стандартной библиотеки Go (для go/types и go/build)
			"$GOMODCACHE":    "r", // Для чтения модулей из кэша (для go/types и go/build)
			"@tg/cache/astg": "w", // Доступ на запись к кэшу проектов
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
