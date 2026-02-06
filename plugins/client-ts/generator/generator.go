// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"log/slog"
	"sort"

	"tgp/core/i18n"
	"tgp/internal/common"
	"tgp/internal/model"
	"tgp/internal/validate"
	"tgp/plugins/client-ts/renderer"
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

	slog.Debug(i18n.Msg("generating TypeScript client"), slog.String("outDir", outDir))

	gen := &generator{
		project:  project,
		outDir:   outDir,
		renderer: renderer.NewClientRenderer(project, outDir),
	}

	if err := gen.generate(docOpts); err != nil {
		slog.Error(i18n.Msg("failed to generate TypeScript client"), slog.String("error", err.Error()))
		return err
	}

	slog.Debug(i18n.Msg("TypeScript client generated successfully"))
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
		if g.renderer.HasJsonRPC() {
			if err := g.renderer.RenderJsonRPCLibrary(); err != nil {
				return err
			}
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

	allCollectedTypeIDs := make(map[string]bool)
	contracts := make([]*model.Contract, len(g.project.Contracts))
	copy(contracts, g.project.Contracts)
	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].Name < contracts[j].Name
	})
	for _, contract := range contracts {
		if model.IsAnnotationSet(g.project, contract, nil, nil, renderer.TagServerJsonRPC) || model.IsAnnotationSet(g.project, contract, nil, nil, renderer.TagServerHTTP) {
			contractTypeIDs := g.renderer.CollectTypeIDsForExchange(contract)
			// Объединяем typeID из всех контрактов
			for typeID := range common.SortedPairs(contractTypeIDs) {
				allCollectedTypeIDs[typeID] = true
			}
		}
	}

	slog.Debug(i18n.Msg("generator.generate: collected typeIDs"), slog.Int("totalTypeIDs", len(allCollectedTypeIDs)), slog.Int("projectTypes", len(g.project.Types)))

	// ВАЖНО: для TS нужно генерировать ВСЕ типы, включая внешние либы
	if len(allCollectedTypeIDs) > 0 {
		if err := g.renderer.RenderClientTypes(allCollectedTypeIDs); err != nil {
			return err
		}
	} else {
		slog.Debug(i18n.Msg("generator.generate: no typeIDs collected, skipping RenderClientTypes"))
	}

	for _, contract := range contracts {
		if model.IsAnnotationSet(g.project, contract, nil, nil, renderer.TagServerJsonRPC) || model.IsAnnotationSet(g.project, contract, nil, nil, renderer.TagServerHTTP) {
			if err := g.renderer.RenderExchangeTypes(contract); err != nil {
				return err
			}
			if model.IsAnnotationSet(g.project, contract, nil, nil, renderer.TagServerJsonRPC) {
				if err := g.renderer.RenderJsonRPCClientClass(contract); err != nil {
					return err
				}
			}
			if model.IsAnnotationSet(g.project, contract, nil, nil, renderer.TagServerHTTP) {
				if err := g.renderer.RenderHTTPClientClass(contract); err != nil {
					return err
				}
			}
		}
	}

	if g.renderer.HasJsonRPC() || g.renderer.HasHTTP() {
		if err := g.renderer.RenderTsConfig(); err != nil {
			return err
		}
	}

	if docOpts.Enabled && (g.renderer.HasJsonRPC() || g.renderer.HasHTTP()) {
		rendererDocOpts := renderer.DocOptions{
			Enabled:  docOpts.Enabled,
			FilePath: docOpts.FilePath,
		}
		if err := g.renderer.RenderReadmeTS(rendererDocOpts); err != nil {
			return err
		}
	}

	return nil
}
