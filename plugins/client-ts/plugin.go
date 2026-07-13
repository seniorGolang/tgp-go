// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.

package main

import (
	_ "embed"
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
	"tgp/plugins/client-ts/generator"
	"tgp/plugins/client-ts/path"
)

//go:embed plugin.md
var pluginDoc string

type ClientTsPlugin struct{}

func (p *ClientTsPlugin) Execute(request data.Storage) (response data.Storage, err error) {

	response = request
	var project *model.Project
	if project, err = helper.GetProject(request); err != nil {
		return
	}

	var output string
	if output, err = path.ResolveOutput(request); err != nil {
		return nil, err
	}

	opts := generator.Options{
		Doc:            generator.DocOptions{Enabled: true},
		ClientIdentity: true,
	}

	var noDoc bool
	if noDoc, err = data.Get[bool](request, "no-doc"); err == nil {
		opts.Doc.Enabled = !noDoc
	}

	var noClientID bool
	if noClientID, err = data.Get[bool](request, "no-client-id"); err == nil && noClientID {
		opts.ClientIdentity = false
	}

	var docFile string
	if docFile, err = data.Get[string](request, "doc-file"); err != nil {
		docFile = ""
	}
	if docFile != "" {
		if opts.Doc.FilePath, err = path.ResolveRaw(docFile); err != nil {
			return nil, err
		}
	} else if opts.Doc.Enabled {
		opts.Doc.FilePath = filepath.Join(output, "readme.md")
	}

	var packageJSONPath string
	var hasPackageJSON bool
	if packageJSONPath, hasPackageJSON, err = path.ResolveOptional(request, "package-json"); err != nil {
		return nil, err
	}
	if hasPackageJSON {
		opts.PackageJSONPath = packageJSONPath
	}

	var contracts []string
	if contracts, err = helper.ParseStringList(request, "contracts"); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.Msg("failed to parse contracts"), err)
	}

	project.Contracts = helper.FilterContracts(project, contracts)

	clientStats := stats.CollectClientStats(project)

	attrs := stats.StartGenerationAttrs(clientStats, output, opts.Doc)
	slog.Info(i18n.Msg("generation started"), attrs...)

	if err = cleanup.GeneratedFiles(output); err != nil {
		slog.Debug(i18n.Msg("failed to cleanup generated files"), slog.String("error", err.Error()))
	}

	if err = generator.GenerateClient(project, output, opts); err != nil {
		slog.Error(i18n.Msg("failed to generate TypeScript client"), slog.String("error", err.Error()))
		err = fmt.Errorf("%s: %w", i18n.Msg("generate TypeScript client"), err)
		return
	}

	clientStats.SetTotalTypes(len(project.Types))

	attrs = stats.CompleteGenerationAttrs(clientStats, output, opts.Doc)
	slog.Info(i18n.Msg("TypeScript client generation completed"), attrs...)

	return
}

func (p *ClientTsPlugin) Info() (info plugin.Info, err error) {

	info = plugin.Info{
		Name:         "client-ts",
		Doc:          pluginDoc,
		Description:  i18n.Msg("HTTP/JSON-RPC TypeScript client generator"),
		Author:       "AlexK (seniorGolang@gmail.com)",
		License:      "MIT",
		Category:     "client",
		Dependencies: []string{"astg"},
		Commands: []plugin.Command{
			{
				Path:        []string{"client", "ts"},
				Description: i18n.Msg("Generate TypeScript client"),
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
						Name:        "package-json",
						Type:        "string",
						Description: i18n.Msg("Path to generated package.json (requires @tg npmName)"),
						Required:    false,
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
					{
						Name:        "no-client-id",
						Type:        "bool",
						Description: i18n.Msg("Disable X-Client-Id header generation and sending"),
						Required:    false,
						Default:     false,
					},
				},
			},
		},
		AllowedPaths: map[string]string{
			"@root": "w",
		},
	}
	return
}
