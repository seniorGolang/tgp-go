// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates
var templates embed.FS

func Generate(rootDir string, moduleName string) (err error) {

	astgDir := filepath.Join(rootDir, "astg")
	if err = os.MkdirAll(astgDir, 0755); err != nil {
		err = fmt.Errorf("failed to create astg directory: %w", err)
		return
	}

	// Подготавливаем метаданные для шаблонов
	meta := map[string]string{
		"moduleName": moduleName,
	}

	var tmpl *template.Template
	if tmpl, err = template.ParseFS(templates, "templates/*.tmpl"); err != nil {
		err = fmt.Errorf("failed to parse templates: %w", err)
		return
	}

	if err = renderFile(tmpl, "tags.go.tmpl", filepath.Join(astgDir, "tags.go"), meta); err != nil {
		err = fmt.Errorf("failed to render tags.go: %w", err)
		return
	}

	if err = renderFile(tmpl, "tag_scanner.go.tmpl", filepath.Join(astgDir, "tag_scanner.go"), meta); err != nil {
		err = fmt.Errorf("failed to render tag_scanner.go: %w", err)
		return
	}

	if err = renderFile(tmpl, "unquote.go.tmpl", filepath.Join(astgDir, "unquote.go"), meta); err != nil {
		err = fmt.Errorf("failed to render unquote.go: %w", err)
		return
	}

	if err = renderFile(tmpl, "project.go.tmpl", filepath.Join(astgDir, "project.go"), meta); err != nil {
		err = fmt.Errorf("failed to render project.go: %w", err)
		return
	}

	if err = renderFile(tmpl, "typeref.go.tmpl", filepath.Join(astgDir, "typeref.go"), meta); err != nil {
		err = fmt.Errorf("failed to render typeref.go: %w", err)
		return
	}

	if err = renderFile(tmpl, "types.go.tmpl", filepath.Join(astgDir, "types.go"), meta); err != nil {
		err = fmt.Errorf("failed to render types.go: %w", err)
		return
	}

	if err = renderFile(tmpl, "service.go.tmpl", filepath.Join(astgDir, "service.go"), meta); err != nil {
		err = fmt.Errorf("failed to render service.go: %w", err)
		return
	}

	if err = renderFile(tmpl, "contract.go.tmpl", filepath.Join(astgDir, "contract.go"), meta); err != nil {
		err = fmt.Errorf("failed to render contract.go: %w", err)
		return
	}

	if err = renderFile(tmpl, "method.go.tmpl", filepath.Join(astgDir, "method.go"), meta); err != nil {
		err = fmt.Errorf("failed to render method.go: %w", err)
		return
	}

	if err = renderFile(tmpl, "variable.go.tmpl", filepath.Join(astgDir, "variable.go"), meta); err != nil {
		err = fmt.Errorf("failed to render variable.go: %w", err)
		return
	}

	if err = renderFile(tmpl, "readme.md.tmpl", filepath.Join(astgDir, "readme.md"), meta); err != nil {
		err = fmt.Errorf("failed to render readme.md: %w", err)
		return
	}

	return
}

func Cleanup(rootDir string) (err error) {

	astgDir := filepath.Join(rootDir, "astg")
	if err = os.RemoveAll(astgDir); err != nil && !os.IsNotExist(err) {
		err = fmt.Errorf("failed to remove astg directory: %w", err)
		return
	}

	return
}

func renderFile(tmpl *template.Template, templateName, filePath string, data any) (err error) {

	_ = os.Remove(filePath)
	dir := filepath.Dir(filePath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return
	}
	var buf bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		return
	}
	err = os.WriteFile(filePath, buf.Bytes(), 0600)
	return
}
