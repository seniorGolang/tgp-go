// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func tscTestProject() (project *model.Project) {

	return &model.Project{
		ModulePath: "example",
		Version:    "1.0.0",
		Contracts: []*model.Contract{{
			Name:    "Rpc",
			PkgPath: "example/contracts",
			Annotations: tags.DocTags{
				model.TagServerJsonRPC: "",
				model.TagHttpPrefix:    "api/v1",
			},
			Methods: []*model.Method{{
				Name: "Echo",
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context.Context"}},
					{Name: "msg", TypeRef: model.TypeRef{TypeID: "string"}},
				},
				Results: []*model.Variable{
					{Name: "out", TypeRef: model.TypeRef{TypeID: "string"}},
				},
			}},
		}, {
			Name:    "Annotations",
			PkgPath: "example/contracts",
			Annotations: tags.DocTags{
				model.TagServerHTTP: "",
				model.TagHttpPrefix: "api/v1",
			},
			Methods: []*model.Method{{
				Name: "HeaderRequired",
				Annotations: tags.DocTags{
					model.TagHTTPMethod: "GET",
					model.TagHttpPath:   "/annotations/header-required",
					model.TagHttpHeader: "token|X-Auth-Token|explicit",
				},
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context.Context"}},
					{Name: "token", TypeRef: model.TypeRef{TypeID: "string"}},
				},
				Results: []*model.Variable{
					{Name: "ok", TypeRef: model.TypeRef{TypeID: "bool"}},
				},
			}, {
				Name: "QueryOptional",
				Annotations: tags.DocTags{
					model.TagHTTPMethod: "GET",
					model.TagHttpPath:   "/annotations/query-optional",
					model.TagHttpArg:    "filter|filter",
				},
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context.Context"}},
					{Name: "filter", TypeRef: model.TypeRef{TypeID: "string", NumberOfPointers: 1}},
				},
				Results: []*model.Variable{
					{Name: "value", TypeRef: model.TypeRef{TypeID: "string"}},
				},
			}, {
				Name: "ResultHeader",
				Annotations: tags.DocTags{
					model.TagHTTPMethod: "GET",
					model.TagHttpPath:   "/annotations/result-header",
					model.TagHttpHeader: "correlationId|X-Correlation-Id",
				},
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context.Context"}},
				},
				Results: []*model.Variable{
					{Name: "correlationId", TypeRef: model.TypeRef{TypeID: "string"}},
				},
			}},
		}},
	}
}

func runTscCheck(t *testing.T, dir string) {

	t.Helper()
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not available")
	}
	cmd := exec.Command("npx", "--yes", "-p", "typescript@5.4.5", "tsc", "--noEmit", "-p", "tsconfig.json")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tsc failed: %v\n%s", err, output)
	}
}

func TestGenerateClient_passesTypeScriptCheck_withClientIdentity(t *testing.T) {

	dir := t.TempDir()
	if err := GenerateClient(tscTestProject(), dir, Options{ClientIdentity: true}); err != nil {
		t.Fatalf("GenerateClient: %v", err)
	}
	runTscCheck(t, dir)
}

func TestGenerateClient_passesTypeScriptCheck_withoutClientIdentity(t *testing.T) {

	dir := t.TempDir()
	if err := GenerateClient(tscTestProject(), dir, Options{ClientIdentity: false}); err != nil {
		t.Fatalf("GenerateClient: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "identity.ts")); !os.IsNotExist(err) {
		t.Fatalf("identity.ts must not exist when client identity disabled")
	}
	runTscCheck(t, dir)
}

func TestGenerateClient_exchangeRequestMarksOptionalPointerFields(t *testing.T) {

	dir := t.TempDir()
	if err := GenerateClient(tscTestProject(), dir, Options{ClientIdentity: true}); err != nil {
		t.Fatalf("GenerateClient: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "annotations-exchange.ts"))
	if err != nil {
		t.Fatalf("read annotations-exchange.ts: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, "filter?:") {
		t.Fatalf("expected optional filter in exchange request, got:\n%s", source)
	}
}
