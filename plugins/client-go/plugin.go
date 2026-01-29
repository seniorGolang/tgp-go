// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package main

import (
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/core/plugin"
	"tgp/internal/cleanup"
	"tgp/internal/helper"
	"tgp/internal/model"
	"tgp/internal/stats"
	"tgp/plugins/client-go/generator"
)

//go:embed plugin.md
var pluginDoc string

type ClientGoPlugin struct{}

func (p *ClientGoPlugin) Execute(rootDir string, request data.Storage, path ...string) (response data.Storage, err error) {

	var project *model.Project
	if project, err = helper.GetProject(request); err != nil {
		return
	}

	var output string
	if output, err = helper.GetOutput(request); err != nil || output == "" {
		return nil, errors.New(i18n.Msg("out option is required and must be a string"))
	}

	docOpts := generator.DocOptions{
		Enabled: true,
	}

	var noDoc bool
	if noDoc, err = data.Get[bool](request, "no-doc"); err == nil {
		docOpts.Enabled = !noDoc
	}

	if docOpts.FilePath, err = data.Get[string](request, "doc-file"); err != nil {
		docOpts.FilePath = ""
	}
	if docOpts.FilePath == "" && docOpts.Enabled {
		docOpts.FilePath = filepath.Join(output, "readme.md")
	}

	var contracts []string
	if contracts, err = helper.ParseStringList(request, "contracts"); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.Msg("failed to parse contracts"), err)
	}

	// Фильтруем контракты, если указаны
	project.Contracts = helper.FilterContracts(project, contracts)

	clientStats := stats.CollectClientStats(project)

	// Логируем начало генерации с деталями
	attrs := stats.StartGenerationAttrs(clientStats, output, docOpts)
	slog.Info(i18n.Msg("generation started"), attrs...)

	// Очищаем старые сгенерированные файлы перед новой генерацией
	if err = cleanup.GeneratedFiles(output); err != nil {
		slog.Debug(i18n.Msg("failed to cleanup generated files"), slog.String("error", err.Error()))
		// Не возвращаем ошибку, так как очистка не критична
	}

	if err = generator.GenerateClient(project, output, docOpts); err != nil {
		slog.Error(i18n.Msg("failed to generate Go client"), slog.String("error", err.Error()))
		err = fmt.Errorf("%s: %w", i18n.Msg("generate Go client"), err)
		return
	}

	// Подсчитываем количество типов (приблизительно, из project.Types)
	clientStats.SetTotalTypes(len(project.Types))

	// Логируем завершение генерации с деталями
	attrs = stats.CompleteGenerationAttrs(clientStats, output, docOpts)
	slog.Info(i18n.Msg("Go client generation completed"), attrs...)

	// Создаем response
	if response, err = helper.CreateResponse(output); err != nil {
		return nil, err
	}

	return
}

func (p *ClientGoPlugin) Info() (info plugin.Info, err error) {

	info = plugin.Info{
		Name:         "client-go",
		Doc:          pluginDoc,
		Description:  i18n.Msg("HTTP/JSON-RPC Go client generator"),
		Author:       "AlexK (seniorGolang@gmail.com)",
		License:      "MIT",
		Category:     "client",
		Dependencies: []string{"astg"},
		Commands: []plugin.Command{
			{
				Path:        []string{"client", "go"},
				Description: i18n.Msg("Generate Go client"),
				Options: []plugin.Option{
					{
						Name:        "contracts-dir",
						Type:        "string",
						Description: i18n.Msg("Path to contracts folder (relative to rootDir)"),
						Required:    false,
						Default:     "contracts",
					},
					{
						Name:        "out",
						Type:        "string",
						Description: i18n.Msg("Path to output directory"),
						Required:    true,
					},
					{
						Name:        "contracts",
						Type:        "string",
						Description: i18n.Msg("Comma-separated list of contracts for filtering (e.g., \"Contract1,Contract2\")"),
						Required:    false,
					},
					{
						Name:        "doc-file",
						Type:        "string",
						Description: i18n.Msg("Path to documentation file (default: <out>/readme.md)"),
						Required:    false,
					},
					{
						Name:        "no-doc",
						Type:        "bool",
						Description: i18n.Msg("Disable documentation generation"),
						Required:    false,
						Default:     false,
					},
				},
			},
		},
		AllowedEnvVars: []string{
			"GOPATH",     // Для поиска пакетов в GOPATH/src и модулей в GOPATH/pkg/mod
			"GOROOT",     // Для поиска стандартной библиотеки Go
			"GOMODCACHE", // Для поиска модулей в кэше модулей
		},
		AllowedPaths: map[string]string{
			"@go/":        "w", // Доступ к директории с go.mod (монтируется хостом в корень "/")
			"$GOPATH/src": "r", // Для чтения пакетов из GOPATH/src (для goimports)
			"$GOROOT":     "r", // Для чтения стандартной библиотеки Go (для goimports)
			"$GOMODCACHE": "r", // Для чтения модулей из кэша (для goimports)
		},
	}
	return
}
