// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"tgp/internal/model"
)

//go:embed pkg
var pkgFiles embed.FS

type baseRenderer struct {
	project  *model.Project
	contract *model.Contract
	outDir   string
}

func newBaseRenderer(project *model.Project, contract *model.Contract, outDir string) *baseRenderer {
	return &baseRenderer{
		project:  project,
		contract: contract,
		outDir:   outDir,
	}
}

func (r *baseRenderer) pkgPath(dir string) string {

	pkgDir := filepath.ToSlash(dir)

	pkgDir = strings.TrimPrefix(pkgDir, "./")

	if pkgDir != "" && !strings.HasPrefix(pkgDir, "/") {
		pkgDir = "/" + pkgDir
	}

	return r.project.ModulePath + pkgDir
}

func (r *baseRenderer) pkgCopyTo(pkg, dst string) (err error) {

	pkgPath := path.Join("pkg", pkg)
	var entries []fs.DirEntry
	if entries, err = pkgFiles.ReadDir(pkgPath); err != nil {
		return
	}
	for _, entry := range entries {
		var fileContent []byte
		if fileContent, err = pkgFiles.ReadFile(fmt.Sprintf("%s/%s", pkgPath, entry.Name())); err != nil {
			return
		}
		if err = os.MkdirAll(path.Join(dst, pkg), 0700); err != nil {
			return
		}
		filename := path.Join(dst, pkg, entry.Name())
		if err = os.WriteFile(filename, fileContent, 0600); err != nil {
			return
		}
	}
	return
}

func (r *baseRenderer) hasJsonRPC() bool {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
			return true
		}
	}
	return false
}

func (r *baseRenderer) hasMetrics() bool {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
			return true
		}
	}
	return false
}

func (r *baseRenderer) hasTrace() bool {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, TagTrace) {
			return true
		}
	}
	return false
}

func (r *baseRenderer) getPackageJSON() string {

	return model.GetAnnotationValue(r.project, r.contract, nil, nil, TagPackageJSON, PackageStdJSON)
}

type contractRenderer struct {
	*baseRenderer
}

func NewContractRenderer(project *model.Project, contract *model.Contract, outDir string) Renderer {
	return &contractRenderer{
		baseRenderer: newBaseRenderer(project, contract, outDir),
	}
}

type transportRenderer struct {
	*baseRenderer
}

func NewTransportRenderer(project *model.Project, outDir string) Renderer {
	return &transportRenderer{
		baseRenderer: newBaseRenderer(project, nil, outDir),
	}
}
