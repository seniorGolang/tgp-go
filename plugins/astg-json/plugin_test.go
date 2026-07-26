package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tgp/core/data"
	"tgp/internal/model"
)

func TestExecuteWritesFile(t *testing.T) {

	t.Parallel()

	out := filepath.Join(t.TempDir(), "project.json")
	request := data.NewStorage()
	if err := request.Set("project", &model.Project{ModulePath: "example.com/app"}); err != nil {
		t.Fatalf("Set project: %v", err)
	}
	if err := request.Set(optionOut, out); err != nil {
		t.Fatalf("Set out: %v", err)
	}

	plugin := &AstgJsonPlugin{}
	if _, err := plugin.Execute(request); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var decoded model.Project
	if err = json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.ModulePath != "example.com/app" {
		t.Fatalf("modulePath: got %q", decoded.ModulePath)
	}
}

func TestExecuteMissingProject(t *testing.T) {

	t.Parallel()

	plugin := &AstgJsonPlugin{}
	if _, err := plugin.Execute(data.NewStorage()); err == nil {
		t.Fatal("expected error when project is missing")
	}
}

func TestInfo(t *testing.T) {

	t.Parallel()

	plugin := &AstgJsonPlugin{}
	info, err := plugin.Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Name != "astg-json" {
		t.Fatalf("name: got %q want astg-json", info.Name)
	}
	if len(info.Commands) != 1 {
		t.Fatalf("commands: got %d want 1", len(info.Commands))
	}
	wantPath := []string{"astg", "json"}
	if got := info.Commands[0].Path; len(got) != len(wantPath) || got[0] != wantPath[0] || got[1] != wantPath[1] {
		t.Fatalf("path: got %v want %v", got, wantPath)
	}
	if len(info.Dependencies) != 1 || info.Dependencies[0] != "astg" {
		t.Fatalf("dependencies: got %v want [astg]", info.Dependencies)
	}
}
