// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"path/filepath"
	"strings"

	"tgp/internal/model"
)

type ClientRenderer struct {
	project        *model.Project
	outDir         string
	typeAnchorsSet map[string]bool // множество якорей типов из секции «Общие типы» (заполняется при генерации readme)
}

func NewClientRenderer(project *model.Project, outDir string) *ClientRenderer {
	return &ClientRenderer{
		project: project,
		outDir:  outDir,
	}
}

func (r *ClientRenderer) pkgPath(dir string) string {

	pkgDir := filepath.ToSlash(dir)

	pkgDir = strings.TrimPrefix(pkgDir, "./")

	if pkgDir != "" && !strings.HasPrefix(pkgDir, "/") {
		pkgDir = "/" + pkgDir
	}

	return r.project.ModulePath + pkgDir
}

func (r *ClientRenderer) HasJsonRPC() bool {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerJsonRPC) {
			return true
		}
	}
	return false
}

func (r *ClientRenderer) HasHTTP() bool {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, TagServerHTTP) {
			return true
		}
	}
	return false
}

func (r *ClientRenderer) HasMetrics() bool {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
			return true
		}
	}
	return false
}

// Все методы Render* реализованы в соответствующих файлах:
// - RenderClientOptions - options.go
// - RenderVersion - version.go
// - RenderClient - client.go
// - RenderClientError - error.go
// - RenderClientBatch - batch.go
// - CollectTypeIDsForExchange - collector.go
// - RenderClientTypes - types.go
// - RenderExchange - exchange.go
// - RenderServiceClient - service.go
// - RenderClientMetrics - metrics.go
// - RenderReadmeGo - readme.go
