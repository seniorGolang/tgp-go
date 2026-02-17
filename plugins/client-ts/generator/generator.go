// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"log/slog"

	"tgp/core/i18n"
	"tgp/internal/model"
	"tgp/internal/validate"
	"tgp/plugins/client-ts/renderer"
)

type DocOptions struct {
	Enabled  bool   // Включена ли генерация документации (по умолчанию true)
	FilePath string // Полный путь к файлу документации (пусто = outDir/readme.md)
}

func (d DocOptions) IsEnabled() (ok bool) {

	return d.Enabled
}

func (d DocOptions) GetFilePath() (s string) {

	return d.FilePath
}

func GenerateClient(project *model.Project, outDir string, docOpts DocOptions) (err error) {

	if err = validate.Project(project); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}

	slog.Debug(i18n.Msg("generating TypeScript client"), slog.String("outDir", outDir))

	gen := &generator{
		project:  project,
		outDir:   outDir,
		renderer: renderer.NewClientRenderer(project, outDir),
	}

	if err = gen.generate(docOpts); err != nil {
		slog.Error(i18n.Msg("failed to generate TypeScript client"), slog.String("error", err.Error()))
		return
	}

	slog.Debug(i18n.Msg("TypeScript client generated successfully"))
	return
}

type generator struct {
	project  *model.Project
	outDir   string
	renderer *renderer.ClientRenderer
}

func (g *generator) generate(docOpts DocOptions) (err error) {

	for _, contract := range g.project.Contracts {
		if err = validate.Contract(contract, g.project); err != nil {
			return fmt.Errorf("validate contract %q: %w", contract.Name, err)
		}
	}

	if g.renderer.HasJsonRPC() || g.renderer.HasHTTP() {
		if err = g.renderer.RenderClientOptions(); err != nil {
			return
		}
		if err = g.renderer.RenderVersion(); err != nil {
			return
		}
		if g.renderer.HasJsonRPC() {
			if err = g.renderer.RenderJsonRPCLibrary(); err != nil {
				return
			}
		}
		if err = g.renderer.RenderClient(); err != nil {
			return
		}
		if err = g.renderer.RenderClientError(); err != nil {
			return
		}
		if g.renderer.HasJsonRPC() {
			if err = g.renderer.RenderClientBatch(); err != nil {
				return
			}
		}
	}

	contractsForClient := make([]*model.Contract, 0, len(g.project.Contracts))
	for _, contractName := range g.renderer.ContractKeys() {
		contract := g.renderer.FindContract(contractName)
		if contract == nil {
			continue
		}
		if model.IsAnnotationSet(g.project, contract, nil, nil, model.TagServerJsonRPC) || model.IsAnnotationSet(g.project, contract, nil, nil, model.TagServerHTTP) {
			contractsForClient = append(contractsForClient, contract)
		}
	}
	allCollectedTypeIDs := g.renderer.CollectTypeIDsForExchangeFromContracts(contractsForClient)

	slog.Debug(i18n.Msg("generator.generate: collected typeIDs"), slog.Int("totalTypeIDs", len(allCollectedTypeIDs)), slog.Int("projectTypes", len(g.project.Types)))

	for _, contract := range contractsForClient {
		if err = g.renderer.RenderExchangeTypes(contract); err != nil {
			return
		}
		if model.IsAnnotationSet(g.project, contract, nil, nil, model.TagServerJsonRPC) {
			if err = g.renderer.RenderJsonRPCClientClass(contract); err != nil {
				return
			}
		}
		if model.IsAnnotationSet(g.project, contract, nil, nil, model.TagServerHTTP) {
			if err = g.renderer.RenderHTTPClientClass(contract); err != nil {
				return
			}
		}
	}

	if g.renderer.HasJsonRPC() || g.renderer.HasHTTP() {
		if err = g.renderer.RenderTsConfig(); err != nil {
			return
		}
	}

	if docOpts.Enabled && (g.renderer.HasJsonRPC() || g.renderer.HasHTTP()) {
		rendererDocOpts := renderer.DocOptions{
			Enabled:  docOpts.Enabled,
			FilePath: docOpts.FilePath,
		}
		if err = g.renderer.RenderReadmeTS(rendererDocOpts); err != nil {
			return
		}
	}

	return
}
