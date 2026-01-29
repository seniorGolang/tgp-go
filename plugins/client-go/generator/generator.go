// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"log/slog"

	"tgp/core/i18n"
	"tgp/internal/model"
	"tgp/internal/validate"
	"tgp/plugins/client-go/renderer"
)

type DocOptions struct {
	Enabled  bool   // Включена ли генерация документации (по умолчанию true)
	FilePath string // Полный путь к файлу документации (пусто = outDir/readme.md)
}

func (d DocOptions) IsEnabled() bool {

	return d.Enabled
}

func (d DocOptions) GetFilePath() string {

	return d.FilePath
}

func GenerateClient(project *model.Project, outDir string, docOpts DocOptions) error {

	if err := validate.ValidateProject(project); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}

	slog.Debug(i18n.Msg("generating Go client"), slog.String("outDir", outDir))

	gen := &generator{
		project:  project,
		outDir:   outDir,
		renderer: renderer.NewClientRenderer(project, outDir),
	}

	if err := gen.generate(docOpts); err != nil {
		slog.Error(i18n.Msg("failed to generate Go client"), slog.String("error", err.Error()))
		return err
	}

	slog.Debug(i18n.Msg("Go client generated successfully"))
	return nil
}

type generator struct {
	project  *model.Project
	outDir   string
	renderer *renderer.ClientRenderer
}

func (g *generator) generate(docOpts DocOptions) error {

	for _, contract := range g.project.Contracts {
		if err := validate.ValidateContract(contract, g.project); err != nil {
			return fmt.Errorf("validate contract %q: %w", contract.Name, err)
		}
	}

	if g.renderer.HasJsonRPC() || g.renderer.HasHTTP() {
		if err := g.renderer.RenderClientOptions(); err != nil {
			return err
		}
		if err := g.renderer.RenderVersion(); err != nil {
			return err
		}
		if err := g.renderer.RenderClient(); err != nil {
			return err
		}
		if err := g.renderer.RenderClientError(); err != nil {
			return err
		}
		if g.renderer.HasJsonRPC() {
			if err := g.renderer.RenderClientBatch(); err != nil {
				return err
			}
		}
	}

	contractsForClient := make([]*model.Contract, 0, len(g.project.Contracts))
	for _, contractName := range g.renderer.ContractKeys() {
		contract := g.renderer.FindContract(contractName)
		if contract == nil {
			continue
		}
		if model.IsAnnotationSet(g.project, contract, nil, nil, renderer.TagServerJsonRPC) || model.IsAnnotationSet(g.project, contract, nil, nil, renderer.TagServerHTTP) {
			contractsForClient = append(contractsForClient, contract)
		}
	}
	allCollectedTypeIDs := g.renderer.CollectTypeIDsForExchangeFromContracts(contractsForClient)

	slog.Debug(i18n.Msg("generator.generate: collected typeIDs"), slog.Int("totalTypeIDs", len(allCollectedTypeIDs)), slog.Int("projectTypes", len(g.project.Types)))

	if len(allCollectedTypeIDs) > 0 {
		if err := g.renderer.RenderClientTypes(allCollectedTypeIDs); err != nil {
			return err
		}
	} else {
		slog.Debug(i18n.Msg("generator.generate: no typeIDs collected, skipping RenderClientTypes"))
	}

	for _, contractName := range g.renderer.ContractKeys() {
		contract := g.renderer.FindContract(contractName)
		if contract == nil {
			continue
		}
		if model.IsAnnotationSet(g.project, contract, nil, nil, renderer.TagServerJsonRPC) || model.IsAnnotationSet(g.project, contract, nil, nil, renderer.TagServerHTTP) {
			if err := g.renderer.RenderExchange(contract); err != nil {
				return err
			}
			if err := g.renderer.RenderServiceClient(contract); err != nil {
				return err
			}
			if g.renderer.HasMetrics() && model.IsAnnotationSet(g.project, contract, nil, nil, renderer.TagMetrics) {
				if err := g.renderer.RenderClientMetrics(); err != nil {
					return err
				}
			}
		}
	}

	if docOpts.Enabled && (g.renderer.HasJsonRPC() || g.renderer.HasHTTP()) {
		if err := g.renderer.RenderReadmeGo(docOpts); err != nil {
			return err
		}
	}

	return nil
}
