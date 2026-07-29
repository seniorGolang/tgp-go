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

	"tgp/internal/common"
	"tgp/internal/generated"
	"tgp/internal/model"
)

//go:embed pkg-tmpl
var pkgTmplFS embed.FS

//go:embed stream/parts.go
var streamPartsGo string

//go:embed stream/rpc.go
var streamRPCGo string

//go:embed stream/sse.go
var streamSSEGo string

//go:embed stream/context.go
var streamContextGo string

type baseRenderer struct {
	outDir   string
	project  *model.Project
	contract *model.Contract
}

func newBaseRenderer(project *model.Project, contract *model.Contract, outDir string) (r *baseRenderer) {
	return &baseRenderer{
		outDir:   outDir,
		project:  project,
		contract: contract,
	}
}

func (r *baseRenderer) pkgPath(dir string) (s string) {

	pkgDir := filepath.ToSlash(dir)

	pkgDir = strings.TrimPrefix(pkgDir, "./")

	if pkgDir != "" && !strings.HasPrefix(pkgDir, "/") {
		pkgDir = "/" + pkgDir
	}

	return r.project.ModulePath + pkgDir
}

func (r *baseRenderer) contractsSorted() (out []*model.Contract) {

	m := make(map[string]*model.Contract, len(r.project.Contracts))
	for _, c := range r.project.Contracts {
		m[c.Name] = c
	}
	names := common.SortedKeys(m)
	out = make([]*model.Contract, 0, len(names))
	for _, n := range names {
		out = append(out, m[n])
	}
	return
}

func methodsSorted(methods []*model.Method) (out []*model.Method) {

	if len(methods) == 0 {
		return
	}
	m := make(map[string]*model.Method, len(methods))
	for _, method := range methods {
		m[method.Name] = method
	}
	names := common.SortedKeys(m)
	out = make([]*model.Method, 0, len(names))
	for _, n := range names {
		out = append(out, m[n])
	}
	return
}

func (r *baseRenderer) pkgRenderTo(pkg string, dst string, data *pkgTemplateData) (err error) {

	pattern := "pkg-tmpl/" + pkg + "/*.go.tmpl"
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

func (r *baseRenderer) renderStreamPackage() (err error) {

	dir := path.Join(r.outDir, "stream")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return
	}
	files := map[string]string{
		"parts.go":   streamPartsGo,
		"rpc.go":     streamRPCGo,
		"sse.go":     streamSSEGo,
		"context.go": streamContextGo,
	}
	for name, body := range files {
		if idx := strings.Index(body, "\npackage "); idx >= 0 {
			body = body[idx+1:]
		}
		content := generated.ByToolGatewayComment + "\n\n" + body
		if err = os.WriteFile(path.Join(dir, name), []byte(content), 0600); err != nil {
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

func (r *baseRenderer) hasJsonRPC() (ok bool) {

	for _, contract := range r.contractsSorted() {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerJsonRPC) {
			return true
		}
	}
	return false
}

func (r *baseRenderer) hasSSE() (ok bool) {

	for _, contract := range r.contractsSorted() {
		if model.ContractHasSSE(r.project, contract) {
			return true
		}
	}
	return false
}

func (r *baseRenderer) hasHTTPServerContracts() (ok bool) {

	for _, contract := range r.contractsSorted() {
		if model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
			return true
		}
	}
	return false
}

func (r *baseRenderer) needsCookieType() (ok bool) {

	for _, contract := range r.contractsSorted() {
		if !model.IsAnnotationSet(r.project, contract, nil, nil, model.TagServerHTTP) {
			continue
		}
		for _, method := range contract.Methods {
			if !methodIsHTTP(r.project, contract, method) {
				continue
			}
			if len(model.HTTPResultCookieMapForResponse(r.project, contract, method)) > 0 {
				return true
			}
		}
	}
	return false
}

func (r *baseRenderer) hasMetrics() (ok bool) {

	for _, contract := range r.contractsSorted() {
		if !model.ContractIsHTTPFamily(r.project, contract) {
			continue
		}
		if model.IsAnnotationSet(r.project, contract, nil, nil, TagMetrics) {
			return true
		}
	}
	return false
}

func (r *baseRenderer) hasTrace() (ok bool) {

	for _, contract := range r.contractsSorted() {
		if !model.ContractIsHTTPFamily(r.project, contract) {
			continue
		}
		if model.IsAnnotationSet(r.project, contract, nil, nil, TagTrace) {
			return true
		}
	}
	return false
}

func (r *baseRenderer) getPackageJSON() (s string) {

	s = model.GetAnnotationValue(r.project, r.contract, nil, nil, TagPackageJSON, PackageStdJSON)
	return
}

type contractRenderer struct {
	*baseRenderer
}

func NewContractRenderer(project *model.Project, contract *model.Contract, outDir string) (r ContractRenderer) {
	return &contractRenderer{
		baseRenderer: newBaseRenderer(project, contract, outDir),
	}
}

type transportRenderer struct {
	*baseRenderer
}

func NewTransportRenderer(project *model.Project, outDir string) (r TransportRenderer) {
	return &transportRenderer{
		baseRenderer: newBaseRenderer(project, nil, outDir),
	}
}
