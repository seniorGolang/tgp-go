// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file is subject to the terms and conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"log/slog"

	"tgp/core/i18n"
	"tgp/internal/common"
	"tgp/internal/model"
	"tgp/plugins/client-go/renderer"
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

	// Генерируем базовые файлы клиента один раз для всех контрактов
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

	// Собираем typeID типов из всех контрактов перед генерацией
	// Используем отсортированный список контрактов для гарантии детерминированного порядка
	allCollectedTypeIDs := make(map[string]bool)
	for _, contractName := range g.renderer.ContractKeys() {
		contract := g.renderer.FindContract(contractName)
		if contract == nil {
			continue
		}
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
	if len(allCollectedTypeIDs) > 0 {
		if err := g.renderer.RenderClientTypes(allCollectedTypeIDs); err != nil {
			return err
		}
	} else {
		slog.Debug(i18n.Msg("generator.generate: no typeIDs collected, skipping RenderClientTypes"))
	}

	// Генерируем клиент для каждого контракта
	// Используем отсортированный список контрактов для гарантии детерминированного порядка
	for _, contractName := range g.renderer.ContractKeys() {
		contract := g.renderer.FindContract(contractName)
		if contract == nil {
			continue
		}
		if contract.Annotations.IsSet(renderer.TagServerJsonRPC) || contract.Annotations.IsSet(renderer.TagServerHTTP) {
			// Генерируем exchange для клиента
			if err := g.renderer.RenderExchange(contract); err != nil {
				return err
			}
			// Генерируем service-client
			if err := g.renderer.RenderServiceClient(contract); err != nil {
				return err
			}
			// Генерируем метрики, если нужно
			if g.renderer.HasMetrics() && contract.Annotations.IsSet(renderer.TagMetrics) {
				if err := g.renderer.RenderClientMetrics(); err != nil {
					return err
				}
			}
		}
	}

	// Генерируем документацию
	if docOpts.Enabled && (g.renderer.HasJsonRPC() || g.renderer.HasHTTP()) {
		if err := g.renderer.RenderReadmeGo(docOpts); err != nil {
			return err
		}
	}

	return nil
}
