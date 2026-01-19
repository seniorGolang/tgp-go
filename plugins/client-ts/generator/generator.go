// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"log/slog"
	"sort"

	"tgp/core/i18n"
	"tgp/internal/common"
	"tgp/internal/model"
	"tgp/plugins/client-ts/renderer"
)

// DocOptions содержит опции для генерации документации
type DocOptions struct {
	Enabled  bool   // Включена ли генерация документации (по умолчанию true)
	FilePath string // Полный путь к файлу документации (пусто = outDir/readme.md)
}

// IsEnabled возвращает, включена ли генерация документации.
func (d DocOptions) IsEnabled() bool {

	return d.Enabled
}

// GetFilePath возвращает путь к файлу документации.
func (d DocOptions) GetFilePath() string {

	return d.FilePath
}

// GenerateClient генерирует клиент для всех контрактов.
func GenerateClient(project *model.Project, outDir string, docOpts DocOptions) error {

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

	// Генерируем базовые файлы клиента один раз для всех контрактов
	if g.renderer.HasJsonRPC() || g.renderer.HasHTTP() {
		if err := g.renderer.RenderClientOptions(); err != nil {
			return err
		}
		if err := g.renderer.RenderVersion(); err != nil {
			return err
		}
		// Генерируем JSON-RPC библиотеку перед генерацией клиента
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

	// Собираем typeID типов из всех контрактов перед генерацией
	allCollectedTypeIDs := make(map[string]bool)
	// Используем отсортированный список контрактов для гарантии детерминированного порядка
	contracts := make([]*model.Contract, len(g.project.Contracts))
	copy(contracts, g.project.Contracts)
	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].Name < contracts[j].Name
	})
	for _, contract := range contracts {
		if contract.Annotations.IsSet(renderer.TagServerJsonRPC) || contract.Annotations.IsSet(renderer.TagServerHTTP) {
			contractTypeIDs := g.renderer.CollectTypeIDsForExchange(contract)
			// Объединяем typeID из всех контрактов
			// Используем отсортированные ключи для детерминированного порядка
			for typeID := range common.SortedPairs(contractTypeIDs) {
				allCollectedTypeIDs[typeID] = true
			}
		}
	}

	slog.Debug(i18n.Msg("generator.generate: collected typeIDs"), slog.Int("totalTypeIDs", len(allCollectedTypeIDs)), slog.Int("projectTypes", len(g.project.Types)))

	// Генерируем локальные версии типов один раз для всех контрактов
	// ВАЖНО: для TS нужно генерировать ВСЕ типы, включая внешние либы
	if len(allCollectedTypeIDs) > 0 {
		if err := g.renderer.RenderClientTypes(allCollectedTypeIDs); err != nil {
			return err
		}
	} else {
		slog.Debug(i18n.Msg("generator.generate: no typeIDs collected, skipping RenderClientTypes"))
	}

	// Генерируем клиент для каждого контракта
	// Переиспользуем уже отсортированный список контрактов
	for _, contract := range contracts {
		if contract.Annotations.IsSet(renderer.TagServerJsonRPC) || contract.Annotations.IsSet(renderer.TagServerHTTP) {
			// Генерируем exchange для клиента
			if err := g.renderer.RenderExchangeTypes(contract); err != nil {
				return err
			}
			// Генерируем JSON-RPC клиент
			if contract.Annotations.IsSet(renderer.TagServerJsonRPC) {
				if err := g.renderer.RenderJsonRPCClientClass(contract); err != nil {
					return err
				}
			}
			// Генерируем HTTP клиент
			if contract.Annotations.IsSet(renderer.TagServerHTTP) {
				if err := g.renderer.RenderHTTPClientClass(contract); err != nil {
					return err
				}
			}
		}
	}

	// Генерируем tsconfig.json для IDE поддержки
	if g.renderer.HasJsonRPC() || g.renderer.HasHTTP() {
		if err := g.renderer.RenderTsConfig(); err != nil {
			return err
		}
	}

	// Генерируем документацию
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
