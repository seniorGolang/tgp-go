// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"embed"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"tgp/internal/model"
)

//go:embed pkg
var schemaPkgFiles embed.FS

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

func (r *ClientRenderer) copySchemaTo(dst string) (err error) {

	embedPath := "pkg/schema"
	var entries []fs.DirEntry
	if entries, err = schemaPkgFiles.ReadDir(embedPath); err != nil {
		return err
	}
	schemaDir := path.Join(dst, "schema")
	for _, entry := range entries {
		var fileContent []byte
		if fileContent, err = schemaPkgFiles.ReadFile(path.Join(embedPath, entry.Name())); err != nil {
			return err
		}
		if err = os.MkdirAll(schemaDir, 0700); err != nil {
			return err
		}
		if err = os.WriteFile(path.Join(schemaDir, entry.Name()), fileContent, 0600); err != nil {
			return err
		}
	}
	return nil
}

func (r *ClientRenderer) HasJsonRPC() bool {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
			return true
		}
	}
	return false
}

func (r *ClientRenderer) HasHTTP() bool {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
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
