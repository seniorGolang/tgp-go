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

func GenerateServer(project *model.Project, contractID string, outDir string) (err error) {

	if err = validate.Project(project); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}
	if err = validate.ContractID(contractID); err != nil {
		return fmt.Errorf("invalid contractID: %w", err)
	}
	if err = validate.OutDir(outDir); err != nil {
		return fmt.Errorf("invalid outDir: %w", err)
	}

	var contract *model.Contract
	if contract, err = validate.FindContract(project, contractID); err != nil {
		return fmt.Errorf("find contract: %w", err)
	}

	if err = validate.Contract(contract, project); err != nil {
		return fmt.Errorf("validate contract: %w", err)
	}

	resetStats()
	setupCacheLogger()
	renderer.SetOnFileSaved(onFileSaved)
	slog.Debug(i18n.Msg("generating server"), slog.String("contract", contractID), slog.String("outDir", outDir))

	gen := &generator{
		project:          project,
		contract:         contract,
		outDir:           outDir,
		contractRenderer: renderer.NewContractRenderer(project, contract, outDir),
	}

	if err = gen.generate(); err != nil {
		slog.Error(i18n.Msg("failed to generate server"), slog.String("contract", contractID), slog.String("error", err.Error()))
		return
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
	return
}

func getServerType(project *model.Project, contract *model.Contract) (serverType string) {

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
	return
}

func GenerateTransportFiles(project *model.Project, outDir string, contracts ...string) (err error) {

	if err = validate.Project(project); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}
	if err = validate.OutDir(outDir); err != nil {
		return fmt.Errorf("invalid outDir: %w", err)
	}

	resetStats()
	setupCacheLogger()
	renderer.SetOnFileSaved(onFileSaved)

	gen := &generator{
		project:           project,
		outDir:            outDir,
		transportRenderer: renderer.NewTransportRenderer(project, outDir),
	}

	if len(contracts) > 0 {
		var filteredProject *model.Project
		if filteredProject, err = filterContracts(project, contracts); err != nil {
			return fmt.Errorf("filter contracts: %w", err)
		}
		gen.project = filteredProject
		gen.transportRenderer = renderer.NewTransportRenderer(filteredProject, outDir)
	}

	if err = gen.generateTransport(); err != nil {
		slog.Error(i18n.Msg("failed to generate transport files"), slog.String("outDir", outDir), slog.String("error", err.Error()))
		return
	}
	return
}

type generator struct {
	project           *model.Project
	contract          *model.Contract
	outDir            string
	contractRenderer  renderer.ContractRenderer
	transportRenderer renderer.TransportRenderer
}

func (g *generator) generate() (err error) {

	if err = g.contractRenderer.RenderHTTP(); err != nil {
		return fmt.Errorf("render HTTP: %w", err)
	}

	if err = g.contractRenderer.RenderServer(); err != nil {
		return fmt.Errorf("render server: %w", err)
	}

	if err = g.contractRenderer.RenderExchange(); err != nil {
		return fmt.Errorf("render exchange: %w", err)
	}

	if err = g.contractRenderer.RenderMiddleware(); err != nil {
		return fmt.Errorf("render middleware: %w", err)
	}

	if model.IsAnnotationSet(g.project, g.contract, nil, nil, "trace") {
		if err = g.contractRenderer.RenderTrace(); err != nil {
			return fmt.Errorf("render trace: %w", err)
		}
	}
	if model.IsAnnotationSet(g.project, g.contract, nil, nil, "metrics") {
		if err = g.contractRenderer.RenderMetrics(); err != nil {
			return fmt.Errorf("render metrics: %w", err)
		}
	}
	if model.IsAnnotationSet(g.project, g.contract, nil, nil, "log") {
		if err = g.contractRenderer.RenderLogger(); err != nil {
			return fmt.Errorf("render logger: %w", err)
		}
	}
	if model.IsAnnotationSet(g.project, g.contract, nil, nil, "jsonRPC-server") {
		if err = g.contractRenderer.RenderJsonRPC(); err != nil {
			return fmt.Errorf("render JSON-RPC: %w", err)
		}
	}
	if model.IsAnnotationSet(g.project, g.contract, nil, nil, "http-server") {
		if err = g.contractRenderer.RenderREST(); err != nil {
			return fmt.Errorf("render REST: %w", err)
		}
	}

	return
}

func (g *generator) generateTransport() (err error) {

	if err = g.transportRenderer.RenderTransportContext(); err != nil {
		return fmt.Errorf("render transport context: %w", err)
	}

	if err = g.transportRenderer.RenderTransportFiber(); err != nil {
		return fmt.Errorf("render transport fiber: %w", err)
	}

	if err = g.transportRenderer.RenderTransportHeader(); err != nil {
		return fmt.Errorf("render transport header: %w", err)
	}

	if err = g.transportRenderer.RenderTransportErrors(); err != nil {
		return fmt.Errorf("render transport errors: %w", err)
	}

	if err = g.transportRenderer.RenderTransportServer(); err != nil {
		return fmt.Errorf("render transport server: %w", err)
	}

	if err = g.transportRenderer.RenderTransportOptions(); err != nil {
		return fmt.Errorf("render transport options: %w", err)
	}

	if err = g.transportRenderer.RenderTransportMetrics(); err != nil {
		return fmt.Errorf("render transport metrics: %w", err)
	}

	if err = g.transportRenderer.RenderTransportVersion(); err != nil {
		return fmt.Errorf("render transport version: %w", err)
	}

	if g.hasJsonRPC() {
		if err = g.transportRenderer.RenderTransportJsonRPC(); err != nil {
			return fmt.Errorf("render transport JSON-RPC: %w", err)
		}
	}

	return
}

func filterContracts(project *model.Project, contractNames []string) (filteredProject *model.Project, err error) {

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

	projCopy := *project
	projCopy.Contracts = filteredContracts
	filteredProject = &projCopy
	return
}

func (g *generator) hasJsonRPC() (found bool) {

	for _, contract := range g.project.Contracts {
		if model.IsAnnotationSet(g.project, contract, nil, nil, "jsonRPC-server") {
			return true
		}
	}
	return
}
