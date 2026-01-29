// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package main

import (
	_ "embed"
	"errors"
	"log/slog"
	"strings"

	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/core/plugin"
	"tgp/internal/cleanup"
	"tgp/internal/helper"
	"tgp/internal/model"
	"tgp/plugins/server/generator"
)

//go:embed plugin.md
var pluginDoc string

type ServerPlugin struct{}

func (p *ServerPlugin) Execute(rootDir string, request data.Storage, path ...string) (response data.Storage, err error) {

	var project *model.Project
	if project, err = helper.GetProject(request); err != nil {
		return
	}

	var output string
	if output, err = helper.GetOutput(request); err != nil || output == "" {
		return nil, errors.New(i18n.Msg("out option is required and must be a string"))
	}

	// project уже отфильтрован по contracts в плагине astg (зависимость)

	// Очищаем старые сгенерированные файлы перед новой генерацией
	if err = cleanup.GeneratedFiles(output); err != nil {
		slog.Debug(i18n.Msg("failed to cleanup generated files"), slog.String("error", err.Error()))
		// Не возвращаем ошибку, так как очистка не критична
	}

	contractNames := make([]string, 0, len(project.Contracts))
	for _, c := range project.Contracts {
		contractNames = append(contractNames, c.Name)
	}
	slog.Info(i18n.Msg("generating transport files"),
		slog.String("output", output),
		slog.String("contracts", strings.Join(contractNames, ", ")),
	)
	if err = generator.GenerateTransportFiles(project, output); err != nil {
		slog.Error(i18n.Msg("failed to generate transport files"),
			slog.String("output", output),
			slog.String("error", err.Error()),
		)
		return
	}

	for _, contract := range project.Contracts {
		if err = generator.GenerateServer(project, contract.ID, output); err != nil {
			slog.Error(i18n.Msg("failed to generate server"),
				slog.String("contract", contract.ID),
				slog.String("error", err.Error()),
			)
			return
		}
	}

	slog.Info(i18n.Msg("generation completed"),
		slog.String("output", output),
	)

	// Создаем response
	if response, err = helper.CreateResponse(output); err != nil {
		return
	}

	return
}

func (p *ServerPlugin) Info() (info plugin.Info, err error) {

	info = plugin.Info{
		Name:         "server",
		Doc:          pluginDoc,
		Description:  i18n.Msg("HTTP/JSON-RPC server code generator based on Fiber"),
		Author:       "AlexK (seniorGolang@gmail.com)",
		License:      "MIT",
		Category:     "server",
		Dependencies: []string{"astg"},
		Commands: []plugin.Command{
			{
				Path:        []string{"server"},
				Description: i18n.Msg("Generate server code"),
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
						Short:       "o",
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
