package main

import (
	_ "embed"
	"fmt"
	"log/slog"

	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/core/plugin"
	"tgp/internal/cdb"
	"tgp/internal/helper"
	"tgp/internal/model"
)

//go:embed plugin.md
var docContent string

const (
	optionFromDB       = "from-db"
	optionAllContracts = "all-contracts"
)

// AstgDbPlugin реализует pre-плагин: подставляет project из локальной базы контрактов в request.
type AstgDbPlugin struct{}

func (p *AstgDbPlugin) Execute(request data.Storage) (response data.Storage, err error) {

	response = request
	if request == nil {
		return
	}
	if request.Has("project") {
		return
	}
	if !request.Has(optionFromDB) {
		return
	}

	var root string
	if root, err = cdb.Root(); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("contracts db root"), err)
		return
	}

	response, err = loadFromDB(request, root)
	return
}

func loadFromDB(request data.Storage, root string) (response data.Storage, err error) {

	response = request

	refStr, _ := data.Get[string](request, optionFromDB)

	var idx *cdb.Index
	if idx, err = cdb.LoadIndex(root); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("contracts db index"), err)
		return
	}

	if refStr == "" {
		if refStr, err = selectRef(idx); err != nil {
			return
		}
	}

	var parsed cdb.Ref
	if parsed, err = cdb.ParseRef(refStr); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("invalid contract ref"), err)
		return
	}

	var projectFile string
	if _, projectFile, err = cdb.ResolveRef(idx, parsed); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("resolve contract ref"), err)
		return
	}

	var project *model.Project
	if project, err = cdb.ReadProject(root, projectFile); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("load project from db"), err)
		return
	}

	var contractsOpt []string
	if contractsOpt, err = helper.ParseStringList(request, "contracts"); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("failed to parse contracts"), err)
		return
	}

	allContracts, _ := data.Get[bool](request, optionAllContracts)
	if !allContracts && len(parsed.Contracts) == 0 && len(contractsOpt) == 0 {
		if parsed.Contracts, err = selectContracts(project); err != nil {
			return
		}
	}

	project = cdb.FilterProject(project, parsed.Contracts)
	slog.Debug(i18n.Msg("contracts after filter"), slog.Any("filter", parsed.Contracts), slog.Int("count", len(project.Contracts)))

	if err = response.Set("project", project); err != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("set project in request"), err)
		return
	}

	slog.Debug(i18n.Msg("project loaded from contracts db"), slog.String("ref", refStr))
	return
}

func (p *AstgDbPlugin) Info() (info plugin.Info, err error) {

	info = plugin.Info{
		Name:         "astg-db",
		Description:  i18n.Msg("Плагин astg-db"),
		Author:       "AlexK <seniorGolang@gmail.com>",
		License:      "MIT",
		Category:     "utility",
		Doc:          docContent,
		Kind:         "pre",
		Dependencies: []string{"astg"},
		AllowedPaths: map[string]string{
			"@tg/astg/db": "w",
		},
		Options: []plugin.Option{
			{
				Name:        optionFromDB,
				Type:        "string",
				Default:     "",
				Description: i18n.Msg("Load project from local contracts DB: ref (e.g. project@v1.0.1 or project:Contract1@main) or empty for interactive project selection; contracts are selected interactively unless listed in ref or --contracts"),
			},
		},
	}
	return
}
