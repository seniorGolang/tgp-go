// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"log/slog"
	"strings"

	"tgp/core/i18n"
	"tgp/internal/common"
	"tgp/internal/model"
	"tgp/internal/validate"
	"tgp/plugins/server/renderer"
)

func GenerateServer(project *model.Project, contractID string, outDir string) error {

	if err := validate.ValidateProject(project); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}
	if err := validate.ValidateContractID(contractID); err != nil {
		return fmt.Errorf("invalid contractID: %w", err)
	}
	if err := validate.ValidateOutDir(outDir); err != nil {
		return fmt.Errorf("invalid outDir: %w", err)
	}

	contract, err := validate.FindContract(project, contractID)
	if err != nil {
		return fmt.Errorf("find contract: %w", err)
	}

	if err := validate.ValidateContract(contract, project); err != nil {
		return fmt.Errorf("validate contract: %w", err)
	}

	resetStats()
	setupCacheLogger()
	renderer.SetOnFileSaved(onFileSaved)
	slog.Debug(i18n.Msg("generating server"), slog.String("contract", contractID), slog.String("outDir", outDir))

	gen := &generator{
		project:  project,
		contract: contract,
		outDir:   outDir,
		renderer: renderer.NewContractRenderer(project, contract, outDir),
	}

	if err := gen.generate(); err != nil {
		slog.Error(i18n.Msg("failed to generate server"), slog.String("contract", contractID), slog.String("error", err.Error()))
		return err
	}

	logStats(contractID)

	serverType := getServerType(project, contract)
	if serverType != "" {
		slog.Debug(i18n.Msg("server generated successfully"),
			slog.String("contract", contractID),
			slog.String("serverType", serverType))
	} else {
		slog.Debug(i18n.Msg("server generated successfully"),
			slog.String("contract", contractID))
	}
	return nil
}

func getServerType(project *model.Project, contract *model.Contract) string {

	for annotation := range common.SortedPairs(contract.Annotations) {
		if strings.HasSuffix(annotation, "-server") {
			return strings.TrimSuffix(annotation, "-server")
		}
	}
	if project != nil && project.Annotations != nil {
		for annotation := range common.SortedPairs(project.Annotations) {
			if strings.HasSuffix(annotation, "-server") {
				return strings.TrimSuffix(annotation, "-server")
			}
		}
	}
	return ""
}

func GenerateTransportFiles(project *model.Project, outDir string, contracts ...string) error {

	if err := validate.ValidateProject(project); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}
	if err := validate.ValidateOutDir(outDir); err != nil {
		return fmt.Errorf("invalid outDir: %w", err)
	}

	resetStats()
	setupCacheLogger()
	renderer.SetOnFileSaved(onFileSaved)

	gen := &generator{
		project:  project,
		outDir:   outDir,
		renderer: renderer.NewTransportRenderer(project, outDir),
	}

	if len(contracts) > 0 {
		filteredProject, err := filterContracts(project, contracts)
		if err != nil {
			return fmt.Errorf("filter contracts: %w", err)
		}
		gen.project = filteredProject
		gen.renderer = renderer.NewTransportRenderer(filteredProject, outDir)
	}

	if err := gen.generateTransport(); err != nil {
		slog.Error(i18n.Msg("failed to generate transport files"), slog.String("outDir", outDir), slog.String("error", err.Error()))
		return err
	}

	return nil
}

type generator struct {
	project  *model.Project
	contract *model.Contract
	outDir   string
	renderer renderer.Renderer
}

func (g *generator) generate() error {

	if err := g.renderer.RenderHTTP(); err != nil {
		return fmt.Errorf("render HTTP: %w", err)
	}

	if err := g.renderer.RenderServer(); err != nil {
		return fmt.Errorf("render server: %w", err)
	}

	if err := g.renderer.RenderExchange(); err != nil {
		return fmt.Errorf("render exchange: %w", err)
	}

	if err := g.renderer.RenderMiddleware(); err != nil {
		return fmt.Errorf("render middleware: %w", err)
	}

	if model.IsAnnotationSet(g.project, g.contract, nil, nil, "trace") {
		if err := g.renderer.RenderTrace(); err != nil {
			return fmt.Errorf("render trace: %w", err)
		}
	}
	if model.IsAnnotationSet(g.project, g.contract, nil, nil, "metrics") {
		if err := g.renderer.RenderMetrics(); err != nil {
			return fmt.Errorf("render metrics: %w", err)
		}
	}
	if model.IsAnnotationSet(g.project, g.contract, nil, nil, "log") {
		if err := g.renderer.RenderLogger(); err != nil {
			return fmt.Errorf("render logger: %w", err)
		}
	}
	if model.IsAnnotationSet(g.project, g.contract, nil, nil, "jsonRPC-server") {
		if err := g.renderer.RenderJsonRPC(); err != nil {
			return fmt.Errorf("render JSON-RPC: %w", err)
		}
	}
	if model.IsAnnotationSet(g.project, g.contract, nil, nil, "http-server") {
		if err := g.renderer.RenderREST(); err != nil {
			return fmt.Errorf("render REST: %w", err)
		}
	}

	return nil
}

func (g *generator) generateTransport() error {

	if err := g.renderer.RenderTransportHTTP(); err != nil {
		return fmt.Errorf("render transport HTTP: %w", err)
	}

	if err := g.renderer.RenderTransportContext(); err != nil {
		return fmt.Errorf("render transport context: %w", err)
	}

	if err := g.renderer.RenderTransportLogger(); err != nil {
		return fmt.Errorf("render transport logger: %w", err)
	}

	if err := g.renderer.RenderTransportFiber(); err != nil {
		return fmt.Errorf("render transport fiber: %w", err)
	}

	if err := g.renderer.RenderTransportHeader(); err != nil {
		return fmt.Errorf("render transport header: %w", err)
	}

	if err := g.renderer.RenderTransportErrors(); err != nil {
		return fmt.Errorf("render transport errors: %w", err)
	}

	if err := g.renderer.RenderTransportServer(); err != nil {
		return fmt.Errorf("render transport server: %w", err)
	}

	if err := g.renderer.RenderTransportOptions(); err != nil {
		return fmt.Errorf("render transport options: %w", err)
	}

	if err := g.renderer.RenderTransportMetrics(); err != nil {
		return fmt.Errorf("render transport metrics: %w", err)
	}

	if err := g.renderer.RenderTransportVersion(); err != nil {
		return fmt.Errorf("render transport version: %w", err)
	}

	if g.hasJsonRPC() {
		if err := g.renderer.RenderTransportJsonRPC(); err != nil {
			return fmt.Errorf("render transport JSON-RPC: %w", err)
		}
	}

	return nil
}

func filterContracts(project *model.Project, contractNames []string) (*model.Project, error) {

	contractMap := make(map[string]bool, len(contractNames))
	for _, name := range contractNames {
		contractMap[name] = true
	}

	filteredContracts := make([]*model.Contract, 0)
	for _, contract := range project.Contracts {
		if contractMap[contract.Name] || contractMap[contract.ID] {
			filteredContracts = append(filteredContracts, contract)
		}
	}

	filteredProject := *project
	filteredProject.Contracts = filteredContracts
	return &filteredProject, nil
}

func (g *generator) hasJsonRPC() bool {

	for _, contract := range g.project.Contracts {
		if model.IsAnnotationSet(g.project, contract, nil, nil, "jsonRPC-server") {
			return true
		}
	}
	return false
}
