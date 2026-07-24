// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/core/plugin"
	"tgp/internal/cleanup"
	"tgp/internal/helper"
	"tgp/internal/model"
	"tgp/internal/validate"
	"tgp/plugins/kafka-sub-go/generator"
	"tgp/plugins/kafka-sub-go/goimports"
)

//go:embed plugin.md
var pluginDoc string

// KafkaSubGoPlugin генерирует Go-подписчик Kafka.
type KafkaSubGoPlugin struct{}

// Execute запускает генерацию плагина.
func (p *KafkaSubGoPlugin) Execute(request data.Storage) (response data.Storage, err error) {

	response = request
	var project *model.Project
	if project, err = helper.GetProject(request); err != nil {
		return
	}
	var output string
	if output, err = helper.GetOutput(request); err != nil || output == "" {
		return
	}
	if err = os.MkdirAll(output, 0o700); err != nil {
		return
	}
	targetModulePath, moduleRoot := goimports.GetModuleInfo(filepath.Join(output, "_.go"))
	if targetModulePath == "" {
		return nil, fmt.Errorf("go.mod not found for output directory %s", output)
	}
	var outputRelPath string
	if outputRelPath, err = filepath.Rel(moduleRoot, output); err != nil {
		return nil, fmt.Errorf("output path outside module: %w", err)
	}
	if err = validate.Project(project); err != nil {
		return nil, fmt.Errorf("invalid project: %w", err)
	}
	if err = validate.KafkaProject(project); err != nil {
		return nil, fmt.Errorf("invalid kafka project: %w", err)
	}
	for _, contract := range project.Contracts {
		if err = validate.Contract(contract, project); err != nil {
			return nil, fmt.Errorf("validate contract %q: %w", contract.Name, err)
		}
	}
	var contracts []string
	if contracts, err = helper.ParseStringList(request, "contracts"); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.Msg("failed to parse contracts"), err)
	}
	filtered := *project
	filtered.Contracts = helper.FilterContracts(project, contracts)
	if err = cleanup.GeneratedFiles(output); err != nil {
		return nil, fmt.Errorf("cleanup generated files: %w", err)
	}
	if err = generator.Generate(&filtered, output, targetModulePath, outputRelPath); err != nil {
		return nil, fmt.Errorf("generate kafka-sub-go: %w", err)
	}
	return response, nil
}

// Info возвращает описание плагина.
func (p *KafkaSubGoPlugin) Info() (info plugin.Info, err error) {

	info = plugin.Info{
		Name:        "kafka-sub-go",
		Doc:         pluginDoc,
		Description: i18n.Msg("Kafka subscriber generator (franz-go)"),
		Author:      "AlexK (seniorGolang@gmail.com)",
		License:     "MIT",
		Category:    "broker",
		Dependencies: []string{
			"astg",
		},
		Commands: []plugin.Command{{
			Path:        []string{"kafka", "sub", "go"},
			Description: i18n.Msg("Generate Kafka Go subscriber"),
			Options: []plugin.Option{
				{Name: "contracts-dir", Type: "string", Description: i18n.Msg("Path to contracts folder (relative to rootDir)"), Default: "contracts"},
				{Name: "out", Type: "string", Description: i18n.Msg("Path to output directory (package name = basename)"), Required: true},
				{Name: "contracts", Type: "string", Description: i18n.Msg("Comma-separated list of contracts for filtering")},
			},
		}},
		AllowedEnvVars: []string{"GOPATH", "GOROOT", "GOMODCACHE"},
		AllowedPaths:   map[string]string{"@go": "w", "$GOPATH/src": "r", "$GOROOT": "r", "$GOMODCACHE": "r"},
	}
	return info, nil
}
