// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func testProject() (project *model.Project) {

	return &model.Project{
		ModulePath: "example",
		Version:    "2.0.0",
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
					{Name: "msg", TypeRef: model.TypeRef{TypeID: "string"}},
				},
				Results: []*model.Variable{
					{Name: "out", TypeRef: model.TypeRef{TypeID: "string"}},
				},
			}},
		}, {
			Name:    "Http",
			PkgPath: "example/contracts",
			Annotations: tags.DocTags{
				model.TagServerHTTP: "",
				model.TagHttpPrefix: "api/v1",
			},
			Methods: []*model.Method{{
				Name: "Get",
				Annotations: tags.DocTags{
					model.TagHTTPMethod: "GET",
					model.TagHttpPath:   "/items/{id}",
					model.TagHttpArg:    "id|id",
				},
				Args: []*model.Variable{
					{Name: "id", TypeRef: model.TypeRef{TypeID: "string"}},
				},
				Results: []*model.Variable{
					{Name: "name", TypeRef: model.TypeRef{TypeID: "string"}},
				},
			}},
		}},
	}
}

func TestGenerateClient_writesClientIdentityFiles(t *testing.T) {

	dir := t.TempDir()
	if err := GenerateClient(testProject(), dir, Options{ClientIdentity: true}); err != nil {
		t.Fatalf("GenerateClient: %v", err)
	}

	for _, fileName := range []string{"identity.ts", "headers.ts", "client.ts", "options.ts"} {
		if _, err := os.Stat(filepath.Join(dir, fileName)); err != nil {
			t.Fatalf("expected %s: %v", fileName, err)
		}
	}

	clientSource, err := os.ReadFile(filepath.Join(dir, "client.ts"))
	if err != nil {
		t.Fatalf("read client.ts: %v", err)
	}
	source := string(clientSource)
	for _, want := range []string{
		`import {resolveDefaultClientName} from './identity'`,
		`import {buildClientHeaders} from './headers'`,
		"clientName:resolveDefaultClientName()",
		"return buildClientHeaders(this.options)",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("client.ts must contain %q, got:\n%s", want, source)
		}
	}

	jsonrpcSource, err := os.ReadFile(filepath.Join(dir, "jsonrpc", "client.ts"))
	if err != nil {
		t.Fatalf("read jsonrpc/client.ts: %v", err)
	}
	jsonrpc := string(jsonrpcSource)
	if !strings.Contains(jsonrpc, "await buildClientHeaders(this.options)") {
		t.Fatalf("jsonrpc/client.ts must use buildClientHeaders, got:\n%s", jsonrpc)
	}
}

func TestGenerateClient_rejectsInvalidProject(t *testing.T) {

	dir := t.TempDir()
	project := &model.Project{}
	err := GenerateClient(project, dir, Options{})
	if err == nil {
		t.Fatal("expected error for empty project")
	}
}

func TestGenerateClient_omitsClientIdentityWhenDisabled(t *testing.T) {

	dir := t.TempDir()
	if err := GenerateClient(testProject(), dir, Options{ClientIdentity: false}); err != nil {
		t.Fatalf("GenerateClient: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "identity.ts")); !os.IsNotExist(err) {
		t.Fatalf("identity.ts must not exist when client identity disabled, err=%v", err)
	}

	optionsSource, err := os.ReadFile(filepath.Join(dir, "options.ts"))
	if err != nil {
		t.Fatalf("read options.ts: %v", err)
	}
	if strings.Contains(string(optionsSource), "clientName") {
		t.Fatalf("options.ts must not contain clientName, got:\n%s", optionsSource)
	}

	headersSource, err := os.ReadFile(filepath.Join(dir, "headers.ts"))
	if err != nil {
		t.Fatalf("read headers.ts: %v", err)
	}
	headers := string(headersSource)
	if strings.Contains(headers, "X-Client-Id") || strings.Contains(headers, "clientName") {
		t.Fatalf("headers.ts must not send X-Client-Id, got:\n%s", headers)
	}

	clientSource, err := os.ReadFile(filepath.Join(dir, "client.ts"))
	if err != nil {
		t.Fatalf("read client.ts: %v", err)
	}
	client := string(clientSource)
	if strings.Contains(client, "resolveDefaultClientName") {
		t.Fatalf("client.ts must not reference resolveDefaultClientName, got:\n%s", client)
	}
	if !strings.Contains(client, "buildClientHeaders") {
		t.Fatalf("client.ts must still use buildClientHeaders, got:\n%s", client)
	}
}
