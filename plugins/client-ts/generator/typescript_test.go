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

func TestGenerateClient_httpGetOmitsContentTypeWithoutBody(t *testing.T) {

	dir := t.TempDir()
	if err := GenerateClient(tscTestProject(), dir, Options{ClientIdentity: false}); err != nil {
		t.Fatalf("GenerateClient: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "annotations-http.ts"))
	if err != nil {
		t.Fatalf("read annotations-http.ts: %v", err)
	}
	source := string(content)

	headerRequiredIdx := strings.Index(source, "public async headerRequired")
	resultHeaderIdx := strings.Index(source, "public async resultHeader")
	if headerRequiredIdx < 0 || resultHeaderIdx < 0 {
		t.Fatalf("expected headerRequired and resultHeader methods")
	}
	headerRequiredBlock := source[headerRequiredIdx:resultHeaderIdx]
	if strings.Contains(headerRequiredBlock, `"Content-Type"`) {
		t.Fatalf("GET without body must not set Content-Type, got:\n%s", headerRequiredBlock)
	}
	if !strings.Contains(headerRequiredBlock, `"Accept"`) {
		t.Fatalf("GET must still set Accept")
	}
}

func TestGenerateClient_resultHeaderBodyModeReadsJSONAndHeader(t *testing.T) {

	dir := t.TempDir()
	if err := GenerateClient(tscTestProject(), dir, Options{ClientIdentity: false}); err != nil {
		t.Fatalf("GenerateClient: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "annotations-http.ts"))
	if err != nil {
		t.Fatalf("read annotations-http.ts: %v", err)
	}
	source := string(content)
	resultHeaderIdx := strings.Index(source, "public async resultHeader")
	if resultHeaderIdx < 0 {
		t.Fatal("expected resultHeader method")
	}
	block := source[resultHeaderIdx:]
	if !strings.Contains(block, ".json()") {
		t.Fatalf("body-mode resultHeader must parse JSON body:\n%s", block)
	}
	if !strings.Contains(block, "X-Correlation-Id") {
		t.Fatalf("body-mode resultHeader must read X-Correlation-Id header:\n%s", block)
	}
	if strings.Contains(block, `!== ""`) || strings.Contains(block, `!= null &&`) {
		t.Fatalf("header must always override body (including empty), got guard:\n%s", block)
	}
	bodyAssign := strings.Index(block, "mergedResult.correlationId = _responseData_")
	headerAssign := strings.Index(block, `parseFormValue(_response_.headers.get("X-Correlation-Id")`)
	if bodyAssign < 0 || headerAssign < 0 || bodyAssign > headerAssign {
		t.Fatalf("expected body assign then unconditional header assign:\n%s", block)
	}
}
