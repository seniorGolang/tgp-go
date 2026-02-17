package main

import (
	_ "embed"
	"log/slog"

	"tgp/core/data"
	"tgp/core/i18n"
	"tgp/core/plugin"
	"tgp/internal/cdb"
	"tgp/internal/model"
)

//go:embed plugin.md
var docContent string

const optionFromDB = "from-db"

// AstgHookPlugin реализует post-плагин: сохраняет project из response в локальную базу контрактов.
type AstgHookPlugin struct{}

func (p *AstgHookPlugin) Execute(request data.Storage) (response data.Storage, err error) {

	response = request
	if request == nil || !request.Has("project") {
		return
	}
	if request.Has(optionFromDB) {
		return
	}

	var project *model.Project
	if project, err = data.Get[*model.Project](request, "project"); err != nil || project == nil {
		return
	}

	var origin, modulePath, version, projectKey string
	var kind cdb.VersionKind
	if project.Git != nil {
		origin, modulePath, version, kind = cdb.OriginFromProject(project)
		projectKey = cdb.RemoteURLToProjectKey(project.Git.RemoteURL)
		if projectKey == "" {
			projectKey = cdb.ModulePathToProjectKey(modulePath)
		}
	} else {
		modulePath = project.ModulePath
		origin = project.ModulePath
		version = "default"
		kind = cdb.VersionKindBranch
		projectKey = cdb.ModulePathToProjectKey(modulePath)
	}
	projectKey = cdb.ProjectKeyForStorage(projectKey)
	if projectKey == "" {
		return
	}

	root, rootErr := cdb.Root()
	if rootErr != nil {
		slog.Debug(i18n.Msg("contracts db root unavailable"), slog.String("error", rootErr.Error()))
		return
	}

	idx, loadErr := cdb.LoadIndex(root)
	if loadErr != nil {
		slog.Debug(i18n.Msg("contracts db index load failed"), slog.String("error", loadErr.Error()))
		return
	}

	relPath, upsertErr := cdb.UpsertProject(root, idx, projectKey, origin, modulePath, version, kind)
	if upsertErr != nil {
		slog.Debug(i18n.Msg("contracts db upsert failed"), slog.String("error", upsertErr.Error()))
		return
	}

	if writeErr := cdb.WriteProject(root, relPath, project); writeErr != nil {
		slog.Debug(i18n.Msg("contracts db write project failed"), slog.String("error", writeErr.Error()))
		return
	}

	if saveErr := cdb.SaveIndex(root, idx); saveErr != nil {
		slog.Debug(i18n.Msg("contracts db save index failed"), slog.String("error", saveErr.Error()))
		return
	}

	slog.Debug(i18n.Msg("contract saved to db"), slog.String("ref", projectKey+"@"+version))
	return
}

func (p *AstgHookPlugin) Info() (info plugin.Info, err error) {

	info = plugin.Info{
		Name:         "astg-hook",
		Description:  i18n.Msg("Плагин astg-hook"),
		Author:       "AlexK <seniorGolang@gmail.com>",
		License:      "MIT",
		Category:     "utility",
		Doc:          docContent,
		Kind:         "post",
		Dependencies: []string{"astg"},
		AllowedPaths: map[string]string{
			"@tg/astg/db": "w",
		},
	}
	return
}
