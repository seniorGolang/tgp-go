// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"os"
	"path/filepath"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func TestGenerateServerSkipsNonHTTPFamily(t *testing.T) {

	outDir := filepath.Join(t.TempDir(), "transport")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	project := &model.Project{
		ModulePath: "example.com/app",
		Contracts: []*model.Contract{{
			ID:          "OrderEvents",
			Name:        "OrderEvents",
			PkgPath:     "example.com/app/contracts",
			Annotations: tags.DocTags{model.TagKafka: ""},
			Methods: []*model.Method{{
				Name:        "Publish",
				Annotations: tags.DocTags{model.TagKafkaTopic: "demo.orders"},
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
					{Name: "event", TypeRef: model.TypeRef{TypeID: "string"}},
				},
				Results: []*model.Variable{{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}}},
			}},
		}},
	}
	if err := GenerateServer(project, "OrderEvents", outDir); err != nil {
		t.Fatalf("GenerateServer: %v", err)
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no files for kafka contract, got %d", len(entries))
	}
}

func TestGenerateServerAcceptsHTTPFamily(t *testing.T) {

	outDir := filepath.Join(t.TempDir(), "transport")
	project := &model.Project{
		ModulePath: "example.com/app",
		Contracts: []*model.Contract{{
			ID:          "Http",
			Name:        "Http",
			PkgPath:     "example.com/app/contracts",
			Annotations: tags.DocTags{model.TagServerHTTP: "", model.TagHttpPrefix: "api/v1"},
			Methods: []*model.Method{{
				Name: "Ping",
				Annotations: tags.DocTags{
					model.TagHTTPMethod: "GET",
					model.TagHttpPath:   "/ping",
				},
				Args:    []*model.Variable{{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}}},
				Results: []*model.Variable{{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}}},
			}},
		}},
	}
	if err := GenerateServer(project, "Http", outDir); err != nil {
		t.Fatalf("GenerateServer: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "http-http.go")); err != nil {
		t.Fatalf("expected http-http.go: %v", err)
	}
}
