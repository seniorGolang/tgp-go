package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureEmptyOutRejectsGoSources(t *testing.T) {

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("write go file: %v", err)
	}
	if err := ensureEmptyOut(dir); err == nil {
		t.Fatal("expected error for directory with .go files")
	}
}

func TestEnsureEmptyOutAllowsEmpty(t *testing.T) {

	dir := t.TempDir()
	if err := ensureEmptyOut(dir); err != nil {
		t.Fatalf("empty dir: %v", err)
	}
	missing := filepath.Join(dir, "missing")
	if err := ensureEmptyOut(missing); err != nil {
		t.Fatalf("missing dir: %v", err)
	}
}

func TestEnsureEmptyOutAllowsNonGoFiles(t *testing.T) {

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# demo\n"), 0o600); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	if err := ensureEmptyOut(dir); err != nil {
		t.Fatalf("dir with non-go files: %v", err)
	}
}
