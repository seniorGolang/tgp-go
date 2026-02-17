package main

import (
	_ "embed"
	"fmt"
	"log/slog"

	"tgp/core"
	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/core/plugin"
	"tgp/internal/cdb"
)

//go:embed plugin.md
var docContent string

const optionFromDB = "from-db"

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

	refStr, _ := data.Get[string](request, optionFromDB)

	root, rootErr := cdb.Root()
	if rootErr != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("contracts db root"), rootErr)
		return
	}

	idx, loadErr := cdb.LoadIndex(root)
	if loadErr != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("contracts db index"), loadErr)
		return
	}

	if refStr == "" {
		refs := cdb.ListRefs(idx)
		if len(refs) == 0 {
			slog.Debug(i18n.Msg("contracts db empty, skipping interactive"))
			return
		}
		selected, selErr := core.InteractiveSelect(i18n.Msg("Select contract ref (project@version)"), refs, false, nil)
		if selErr != nil || len(selected) == 0 {
			err = selErr
			return
		}
		refStr = selected[0]
	}

	parsed, parseErr := cdb.ParseRef(refStr)
	if parseErr != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("invalid contract ref"), parseErr)
		return
	}

	_, projectFile, resolveErr := cdb.ResolveRef(idx, parsed)
	if resolveErr != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("resolve contract ref"), resolveErr)
		return
	}

	project, readErr := cdb.ReadProject(root, projectFile)
	if readErr != nil {
		err = fmt.Errorf("%s: %w", i18n.Msg("load project from db"), readErr)
		return
	}

	contractNamesFromDB := make([]string, 0, len(project.Contracts))
	for _, c := range project.Contracts {
		contractNamesFromDB = append(contractNamesFromDB, c.Name+" ("+c.ID+")")
	}
	slog.Debug(i18n.Msg("contracts in project from db"), slog.Int("count", len(project.Contracts)), slog.Any("contracts", contractNamesFromDB))

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
				Description: i18n.Msg("Load project from local contracts DB: ref (e.g. project@v1.0.1 or project:Contract1@main) or empty for interactive selection"),
			},
		},
	}
	return
}
