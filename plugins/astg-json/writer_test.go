package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"tgp/internal/model"
)

func TestWriteModelToFile(t *testing.T) {

	t.Parallel()

	project := &model.Project{
		ModulePath: "example.com/app",
		Contracts: []*model.Contract{
			{Name: "Alpha", ID: "alpha"},
		},
	}
	out := filepath.Join(t.TempDir(), "project.json")

	if err := writeModel(project, out); err != nil {
		t.Fatalf("writeModel: %v", err)
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
		t.Fatalf("modulePath: got %q want %q", decoded.ModulePath, "example.com/app")
	}
	if len(decoded.Contracts) != 1 || decoded.Contracts[0].Name != "Alpha" {
		t.Fatalf("contracts mismatch: %+v", decoded.Contracts)
	}
}

func TestWriteModelToStdout(t *testing.T) {

	project := &model.Project{ModulePath: "example.com/app"}

	captured := captureStdout(t, func() {
		if err := writeModel(project, ""); err != nil {
			t.Fatalf("writeModel: %v", err)
		}
	})

	var decoded model.Project
	if err := json.Unmarshal([]byte(captured), &decoded); err != nil {
		t.Fatalf("Unmarshal stdout: %v (raw=%q)", err, captured)
	}
	if decoded.ModulePath != "example.com/app" {
		t.Fatalf("modulePath: got %q want %q", decoded.ModulePath, "example.com/app")
	}
}

func TestWriteModelInvalidPath(t *testing.T) {

	t.Parallel()

	project := &model.Project{ModulePath: "example.com/app"}
	out := filepath.Join(t.TempDir(), "missing", "nested", "project.json")

	if err := writeModel(project, out); err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func captureStdout(t *testing.T, fn func()) (output string) {

	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	os.Stdout = writer

	fn()

	if err = writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	os.Stdout = original

	var data []byte
	if data, err = io.ReadAll(reader); err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return string(data)
}
