package main

import (
	_ "embed"
	"fmt"
	"log/slog"

	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/core/plugin"
	"tgp/internal/helper"
	"tgp/internal/model"
	"tgp/internal/stats"
	"tgp/plugins/swagger/generator"
	"tgp/plugins/swagger/server"
)

//go:embed plugin.md
var pluginDoc string

// SwaggerPlugin реализует интерфейс Plugin.
type SwaggerPlugin struct{}

// Execute выполняет основную логику плагина.
func (p *SwaggerPlugin) Execute(rootDir string, request data.Storage, path ...string) (response data.Storage, err error) {

	slog.Info(i18n.Msg("swagger plugin started"))

	response = data.NewStorage()

	// Получаем project из request
	var project *model.Project
	if project, err = helper.GetProject(request); err != nil {
		return
	}

	// Получаем список контрактов для фильтрации
	var contracts []string
	if contracts, err = helper.ParseStringList(request, "contracts"); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.Msg("failed to parse contracts"), err)
	}

	// Собираем статистику для логирования
	swaggerStats := stats.CollectSwaggerStats(project, contracts)

	swaggerDoc := generator.GenerateDoc(project, contracts...)

	// Получаем output из request (опциональный, так как может быть только serve)
	var output string
	if output, err = helper.GetOutput(request); err != nil {
		return
	}

	if output != "" {
		// Логируем начало генерации с деталями
		attrs := stats.StartSwaggerGenerationAttrs(swaggerStats, output)
		slog.Info(i18n.Msg("generating Swagger documentation"), attrs...)

		if err = generator.SaveFile(swaggerDoc, output); err != nil {
			slog.Error(i18n.Msg("failed to generate Swagger documentation"), "error", err)
			return nil, fmt.Errorf("%s: %w", i18n.Msg("generate Swagger"), err)
		}

		// Подсчитываем количество типов (приблизительно, из project.Types)
		swaggerStats.SetTotalTypes(len(project.Types))

		// Логируем завершение генерации с деталями
		attrs = stats.CompleteSwaggerGenerationAttrs(swaggerStats, output)
		slog.Info(i18n.Msg("Swagger documentation generated successfully"), attrs...)

		// Сохраняем результат в response
		if err = response.Set("out", output); err != nil {
			return nil, fmt.Errorf("%s: %w", i18n.Msg("failed to set response"), err)
		}
	}

	var addr string
	if addr, _ = data.Get[string](request, "serve"); addr != "" {

		if err = server.Serve(addr, swaggerDoc); err != nil {
			slog.Error(i18n.Msg("failed to start swagger server"), "error", err)
			return nil, fmt.Errorf("%s: %w", i18n.Msg("failed to start swagger server"), err)
		}
	} else {
		slog.Info(i18n.Msg("swagger plugin completed"))
	}

	return
}

// Info возвращает информацию о плагине.
func (p *SwaggerPlugin) Info() (info plugin.Info, err error) {

	info = plugin.Info{
		Name:         "swagger",
		Doc:          pluginDoc,
		Description:  i18n.Msg("Swagger/OpenAPI documentation generator for contracts"),
		Author:       "AlexK <seniorGolang@gmail.com>",
		License:      "MIT",
		Category:     "docs",
		Dependencies: []string{"astg"}, // Зависимость от astg для получения project
		Commands: []plugin.Command{
			{
				Path:        []string{"swagger"},
				Description: i18n.Msg("Generate Swagger/OpenAPI documentation"),
				Options: []plugin.Option{
					{
						Name:        "out",
						Short:       "o",
						Type:        "string",
						Description: i18n.Msg("Path to output file (.json and .yaml/.yml supported)"),
						Required:    false,
					},
					{
						Name:        "serve",
						Type:        "string",
						Description: i18n.Msg("Start HTTP server with Swagger UI on specified address (e.g., :8080 or localhost:3000)"),
						Required:    false,
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
		AllowedPaths: map[string]string{
			"@go": "w",
		},
		AllowedShellCMDs: []string{
			"open",
			"xdg-open",
			"cmd",
			"uname",
		},
		AllowedEnvVars: []string{
			"OSTYPE",
		},
	}
	return
}
