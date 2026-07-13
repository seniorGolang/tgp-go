// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func TestRenderPackageJSON_sameDirAsOut(t *testing.T) {

	outDir := t.TempDir()
	packagePath := filepath.Join(outDir, "package.json")
	project := &model.Project{
		Annotations: tags.DocTags{
			tagNpmName:     "@test/api-client",
			tagVersion:     "v1.2.3",
			tagDesc:        "Test API",
			tagLicense:     "MIT",
			tagAuthor:      "Author",
			tagNpmPrivate:  "true",
			tagNpmRegistry: "https://registry.npmjs.org",
		},
	}
	renderer := NewClientRenderer(project, outDir, true, packagePath, true)
	if err := renderer.RenderPackageJSON(); err != nil {
		t.Fatalf("RenderPackageJSON: %v", err)
	}

	content, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}
	source := string(content)
	for _, want := range []string{
		`"name": "@test/api-client"`,
		`"version": "1.2.3"`,
		`"main": "./dist/client.js"`,
		`"types": "./dist/client.d.ts"`,
		`"private": true`,
		`"registry": "https://registry.npmjs.org"`,
		`"files": [`,
		`"dist"`,
		`"readme.md"`,
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("package.json missing %q:\n%s", want, source)
		}
	}
}

func TestRenderPackageJSON_rootPackageOutInSubdir(t *testing.T) {

	root := t.TempDir()
	outDir := filepath.Join(root, "web")
	if err := os.Mkdir(outDir, 0700); err != nil {
		t.Fatalf("mkdir web: %v", err)
	}
	packagePath := filepath.Join(root, "package.json")
	project := &model.Project{
		Annotations: tags.DocTags{
			tagNpmName: "@test/api-client",
			tagVersion: "2.0.0",
		},
	}
	renderer := NewClientRenderer(project, outDir, true, packagePath, true)
	if err := renderer.RenderPackageJSON(); err != nil {
		t.Fatalf("RenderPackageJSON: %v", err)
	}

	content, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatalf("read package.json: %v", err)
	}
	source := string(content)
	for _, want := range []string{
		`"main": "./web/dist/client.js"`,
		`"types": "./web/dist/client.d.ts"`,
		`"web/dist"`,
		`"web/readme.md"`,
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("package.json missing %q:\n%s", want, source)
		}
	}
}

func TestRenderPackageJSON_requiresNpmName(t *testing.T) {

	outDir := t.TempDir()
	renderer := NewClientRenderer(&model.Project{}, outDir, true, filepath.Join(outDir, "package.json"), true)
	if err := renderer.RenderPackageJSON(); err == nil {
		t.Fatal("expected error without npmName")
	}
}

func TestRenderTsConfig_emitDist(t *testing.T) {

	outDir := t.TempDir()
	renderer := NewClientRenderer(&model.Project{}, outDir, true, filepath.Join(outDir, "package.json"), true)
	if err := renderer.RenderTsConfig(); err != nil {
		t.Fatalf("RenderTsConfig: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(outDir, "tsconfig.json"))
	if err != nil {
		t.Fatalf("read tsconfig: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, `"outDir": "dist"`) {
		t.Fatalf("expected outDir dist, got:\n%s", source)
	}
}
