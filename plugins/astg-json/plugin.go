package main

import (
	_ "embed"

	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/core/plugin"
	"tgp/internal/helper"
	"tgp/internal/model"
)

//go:embed plugin.md
var docContent string

const (
	optionOut          = "out"
	optionAllContracts = "all-contracts"
)

// AstgJsonPlugin реализует command-плагин: выводит модель проекта astg в формате JSON.
type AstgJsonPlugin struct{}

func (p *AstgJsonPlugin) Execute(request data.Storage) (response data.Storage, err error) {

	response = request

	var project *model.Project
	if project, err = helper.GetProject(request); err != nil {
		return
	}

	var out string
	out, _ = data.Get[string](request, optionOut)

	if err = writeModel(project, out); err != nil {
		return
	}
	return
}

func (p *AstgJsonPlugin) Info() (info plugin.Info, err error) {

	info = plugin.Info{
		Name:          "astg-json",
		Description:   i18n.Msg("Export astg project model as JSON"),
		Author:        "AlexK <seniorGolang@gmail.com>",
		License:       "MIT",
		Category:      "utility",
		Doc:           docContent,
		Dependencies:  []string{"astg"},
		AllowedStdOut: true,
		AllowedStdErr: true,
		AllowedPaths:  map[string]string{"@go": "w"},
		Commands: []plugin.Command{
			{
				Path:        []string{"astg", "json"},
				Description: i18n.Msg("Export astg project model as JSON to stdout or file"),
				Options: []plugin.Option{
					{Name: optionOut, Short: "o", Type: "string", Description: i18n.Msg("Path to output file (default: stdout)")},
					{Name: "contracts-dir", Type: "string", Description: i18n.Msg("Path to contracts folder (relative to rootDir)"), Default: "contracts"},
					{Name: optionAllContracts, Type: "bool", Description: i18n.Msg("Load full model from DB without interactive contract selection"), Default: true},
				},
			},
		},
	}
	return
}
