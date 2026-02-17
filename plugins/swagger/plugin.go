// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package main

import (
	_ "embed"
	"fmt"
	"log/slog"

	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/core/plugin"
	"tgp/internal/common"
	"tgp/internal/helper"
	"tgp/internal/model"
	"tgp/internal/stats"
	"tgp/plugins/swagger/generator"
	"tgp/plugins/swagger/server"
)

const defaultServeAddr = ":8080"

//go:embed plugin.md
var pluginDoc string

type SwaggerPlugin struct{}

func (p *SwaggerPlugin) Execute(request data.Storage) (response data.Storage, err error) {

	slog.Info(i18n.Msg("swagger plugin started"))

	response = request

	var project *model.Project
	if project, err = helper.GetProject(request); err != nil {
		return
	}

	var contracts []string
	if contracts, err = helper.ParseStringList(request, "contracts"); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.Msg("failed to parse contracts"), err)
	}

	swaggerStats := stats.CollectSwaggerStats(project, contracts)

	swaggerDoc, err := generator.GenerateDoc(project, contracts...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.Msg("generate Swagger"), err)
	}

	var addr string
	var output string
	if rawOut, _ := data.Get[string](request, "out"); rawOut != "" {
		output = common.NormalizeWASMPath(rawOut)
	}
	if output != "" {
		attrs := stats.StartSwaggerGenerationAttrs(swaggerStats, output)
		slog.Info(i18n.Msg("generating Swagger documentation"), attrs...)

		if err = generator.SaveFile(swaggerDoc, output); err != nil {
			slog.Error(i18n.Msg("failed to generate Swagger documentation"), "error", err)
			return nil, fmt.Errorf("%s: %w", i18n.Msg("generate Swagger"), err)
		}

		swaggerStats.SetTotalTypes(len(project.Types))
		attrs = stats.CompleteSwaggerGenerationAttrs(swaggerStats, output)
		slog.Info(i18n.Msg("Swagger documentation generated successfully"), attrs...)
	}

	addr, _ = data.Get[string](request, "serve")
	if addr == "" && output == "" {
		addr = defaultServeAddr
	}
	if addr != "" {
		if err = server.Serve(addr, swaggerDoc); err != nil {
			slog.Error(i18n.Msg("failed to start swagger server"), "error", err)
			return nil, fmt.Errorf("%s: %w", i18n.Msg("failed to start swagger server"), err)
		}
	} else {
		slog.Info(i18n.Msg("swagger plugin completed"))
	}

	return
}

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
						Name:        "contracts-dir",
						Type:        "string",
						Description: i18n.Msg("Path to contracts folder (relative to rootDir)"),
						Required:    false,
						Default:     "contracts",
					},
					{
						Name:        "out",
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
