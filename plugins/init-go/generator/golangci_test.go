package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestRenderGolangci(t *testing.T) {

	dir := t.TempDir()
	tmpl, err := template.ParseFS(templates, "templates/*.tmpl")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	outPath := filepath.Join(dir, fileGolangci)
	if err = renderFile(tmpl, "golangci.tmpl", outPath, newParams("demo", "some", "")); err != nil {
		t.Fatalf("render golangci: %v", err)
	}
	var got []byte
	if got, err = os.ReadFile(outPath); err != nil {
		t.Fatalf("read %s: %v", outPath, err)
	}
	content := string(got)
	for _, want := range []string{
		`version: "2"`,
		"modules-download-mode: readonly",
		"- revive",
		"- inamedparam",
		"max-issues-per-linter: 0",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("golangci content missing %q", want)
		}
	}
	if strings.Contains(content, "{{") || strings.Contains(content, "}}") {
		t.Fatal("golangci template left unresolved actions")
	}
}

func TestRenderGolangciMatchesRepo(t *testing.T) {

	var want []byte
	var err error
	if want, err = os.ReadFile(filepath.Join("..", "..", "..", fileGolangci)); err != nil {
		t.Fatalf("read repo %s: %v", fileGolangci, err)
	}
	dir := t.TempDir()
	tmpl, err := template.ParseFS(templates, "templates/*.tmpl")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	outPath := filepath.Join(dir, fileGolangci)
	if err = renderFile(tmpl, "golangci.tmpl", outPath, newParams("demo", "", "example")); err != nil {
		t.Fatalf("render golangci: %v", err)
	}
	var got []byte
	if got, err = os.ReadFile(outPath); err != nil {
		t.Fatalf("read rendered: %v", err)
	}
	if strings.TrimSpace(string(got)) != strings.TrimSpace(string(want)) {
		t.Fatalf("rendered %s diverges from repo root %s", fileGolangci, fileGolangci)
	}
}

func TestRenderFileUnknownTemplate(t *testing.T) {

	tmpl, err := template.ParseFS(templates, "templates/*.tmpl")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	if err = renderFile(tmpl, "missing.tmpl", filepath.Join(t.TempDir(), "x.yml"), newParams("demo", "", "")); err == nil {
		t.Fatal("expected error for unknown template")
	}
}
