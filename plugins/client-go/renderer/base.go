// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"tgp/internal/generated"
	"tgp/internal/model"
)

//go:embed pkg_tmpl
var pkgTmplFS embed.FS

type ClientRenderer struct {
	outDir           string
	outputRelPath    string
	project          *model.Project
	targetModulePath string
	typeAnchorsSet   map[string]bool
}

func NewClientRenderer(project *model.Project, outDir string, targetModulePath string, outputRelPath string) (r *ClientRenderer) {
	return &ClientRenderer{
		outDir:           outDir,
		outputRelPath:    outputRelPath,
		project:          project,
		targetModulePath: targetModulePath,
	}
}

func (r *ClientRenderer) pkgPath(dir string) (s string) {

	rel, err := filepath.Rel(r.outDir, dir)
	if err != nil {
		rel = filepath.Base(dir)
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return r.targetModulePath + "/" + r.outputRelPath
	}
	return r.targetModulePath + "/" + r.outputRelPath + "/" + rel
}

func (r *ClientRenderer) pkgRenderTo(pkg string, dst string, data *pkgTemplateData) (err error) {

	pattern := "pkg_tmpl/" + pkg + "/*.go.tmpl"
	var names []string
	if names, err = fs.Glob(pkgTmplFS, pattern); err != nil {
		return
	}
	var tmpl *template.Template
	if tmpl, err = template.ParseFS(pkgTmplFS, pattern); err != nil {
		return
	}
	if err = os.MkdirAll(path.Join(dst, pkg), 0700); err != nil {
		return
	}
	for _, name := range names {
		var buf bytes.Buffer
		if err = tmpl.ExecuteTemplate(&buf, filepath.Base(name), data); err != nil {
			return
		}
		outName := strings.TrimSuffix(filepath.Base(name), ".tmpl")
		if err = os.WriteFile(path.Join(dst, pkg, outName), buf.Bytes(), 0600); err != nil {
			return
		}
	}
	return
}

type pkgTemplateData struct {
	DoNotEditComment string
}

func newPkgTemplateData() (data *pkgTemplateData) {
	return &pkgTemplateData{DoNotEditComment: generated.ByToolGatewayComment}
}

func (r *ClientRenderer) HasJsonRPC() (ok bool) {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
			return true
		}
	}
	return false
}

func (r *ClientRenderer) HasHTTP() (ok bool) {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
			return true
		}
	}
	return false
}

func (r *ClientRenderer) HasMetrics() (ok bool) {

	for _, contract := range r.project.Contracts {
		if model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
			return true
		}
	}
	return false
}
