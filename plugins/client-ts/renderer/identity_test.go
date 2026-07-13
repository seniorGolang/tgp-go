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

func testHTTPProject(version string) (project *model.Project) {

	return &model.Project{
		ModulePath: "example",
		Version:    version,
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
		}},
	}
}

func TestRenderIdentity_writesIdentityFile(t *testing.T) {

	dir := t.TempDir()
	renderer := NewClientRenderer(testHTTPProject("1.2.3"), dir, false, "", true)
	if err := renderer.RenderIdentity(); err != nil {
		t.Fatalf("RenderIdentity: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "identity.ts"))
	if err != nil {
		t.Fatalf("read identity.ts: %v", err)
	}
	source := string(content)

	for _, want := range []string{
		"export function resolveDefaultClientName",
		"resolveBrowserToken",
		"resolveInstanceId",
		"readNodeHostname",
		"globalThis",
		`import {VersionASTg} from "./version"`,
		"tgp-client-instance-id",
		"_astg_ts_${VersionASTg}",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("identity.ts must contain %q, got:\n%s", want, source)
		}
	}
}

func TestRenderHeaders_writesHeadersFile(t *testing.T) {

	dir := t.TempDir()
	renderer := NewClientRenderer(testHTTPProject("1.0.0"), dir, false, "", true)
	if err := renderer.RenderHeaders(); err != nil {
		t.Fatalf("RenderHeaders: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "headers.ts"))
	if err != nil {
		t.Fatalf("read headers.ts: %v", err)
	}
	source := string(content)

	for _, want := range []string{
		"export const headerClientId = \"X-Client-Id\"",
		"export async function buildClientHeaders",
		"options.clientName",
		"headerClientId in result",
		"userHeaders = await options.headers()",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("headers.ts must contain %q, got:\n%s", want, source)
		}
	}
}

func TestRenderHeaders_preservesExplicitClientIdHeader(t *testing.T) {

	dir := t.TempDir()
	renderer := NewClientRenderer(testHTTPProject("1.0.0"), dir, false, "", true)
	if err := renderer.RenderHeaders(); err != nil {
		t.Fatalf("RenderHeaders: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "headers.ts"))
	if err != nil {
		t.Fatalf("read headers.ts: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "if (!(headerClientId in result))") {
		t.Fatalf("headers.ts must not override explicit X-Client-Id, got:\n%s", source)
	}
}

func TestRenderClientOptions_includesClientName(t *testing.T) {

	dir := t.TempDir()
	renderer := NewClientRenderer(testHTTPProject("1.0.0"), dir, false, "", true)
	if err := renderer.RenderClientOptions(); err != nil {
		t.Fatalf("RenderClientOptions: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "options.ts"))
	if err != nil {
		t.Fatalf("read options.ts: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "clientName") || !strings.Contains(source, "string") {
		t.Fatalf("options.ts must declare clientName, got:\n%s", source)
	}
}

func TestRenderClient_usesBuildClientHeadersAndDefaultClientName(t *testing.T) {

	dir := t.TempDir()
	renderer := NewClientRenderer(testHTTPProject("1.0.0"), dir, false, "", true)
	if err := renderer.RenderIdentity(); err != nil {
		t.Fatalf("RenderIdentity: %v", err)
	}
	if err := renderer.RenderHeaders(); err != nil {
		t.Fatalf("RenderHeaders: %v", err)
	}
	if err := renderer.RenderVersion(); err != nil {
		t.Fatalf("RenderVersion: %v", err)
	}
	if err := renderer.RenderClientError(); err != nil {
		t.Fatalf("RenderClientError: %v", err)
	}
	if err := renderer.RenderJsonRPCLibrary(); err != nil {
		t.Fatalf("RenderJsonRPCLibrary: %v", err)
	}
	if err := renderer.RenderClient(); err != nil {
		t.Fatalf("RenderClient: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "client.ts"))
	if err != nil {
		t.Fatalf("read client.ts: %v", err)
	}
	source := string(content)

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
}

func TestRenderJsonRPCClient_usesBuildClientHeaders(t *testing.T) {

	dir := t.TempDir()
	renderer := NewClientRenderer(testHTTPProject("1.0.0"), dir, false, "", true)
	if err := renderer.RenderHeaders(); err != nil {
		t.Fatalf("RenderHeaders: %v", err)
	}
	if err := renderer.RenderJsonRPCLibrary(); err != nil {
		t.Fatalf("RenderJsonRPCLibrary: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "jsonrpc", "client.ts"))
	if err != nil {
		t.Fatalf("read jsonrpc/client.ts: %v", err)
	}
	source := string(content)

	for _, want := range []string{
		`import {buildClientHeaders} from '../headers'`,
		"await buildClientHeaders(this.options)",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("jsonrpc/client.ts must contain %q, got:\n%s", want, source)
		}
	}
	if strings.Contains(source, "getHeaders") {
		t.Fatalf("jsonrpc/client.ts must not contain private getHeaders, got:\n%s", source)
	}
}

func TestRenderHTTPClient_includesGetHeadersFromBaseClient(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Version:    "1.0.0",
		Contracts: []*model.Contract{{
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
					{Name: "token", TypeRef: model.TypeRef{TypeID: "string"}},
				},
				Results: []*model.Variable{
					{Name: "ok", TypeRef: model.TypeRef{TypeID: "bool"}},
				},
			}},
		}},
	}

	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderHTTPClientClass(project.Contracts[0]); err != nil {
		t.Fatalf("RenderHTTPClientClass: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "annotations-http.ts"))
	if err != nil {
		t.Fatalf("read annotations-http.ts: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "this.baseClient.getHeaders()") {
		t.Fatalf("HTTP client must use baseClient.getHeaders(), got:\n%s", source)
	}
}

func TestRenderIdentity_skipsFileWhenClientIdentityDisabled(t *testing.T) {

	dir := t.TempDir()
	renderer := NewClientRenderer(testHTTPProject("1.0.0"), dir, false, "", false)
	if err := renderer.RenderIdentity(); err != nil {
		t.Fatalf("RenderIdentity: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "identity.ts")); !os.IsNotExist(err) {
		t.Fatalf("identity.ts must not exist when client identity disabled, err=%v", err)
	}
}

func TestRenderHeaders_plainTemplateWhenClientIdentityDisabled(t *testing.T) {

	dir := t.TempDir()
	renderer := NewClientRenderer(testHTTPProject("1.0.0"), dir, false, "", false)
	if err := renderer.RenderHeaders(); err != nil {
		t.Fatalf("RenderHeaders: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "headers.ts"))
	if err != nil {
		t.Fatalf("read headers.ts: %v", err)
	}
	source := string(content)

	for _, mustNot := range []string{
		"X-Client-Id",
		"headerClientId",
		"clientName",
	} {
		if strings.Contains(source, mustNot) {
			t.Fatalf("headers.ts must not contain %q when client identity disabled, got:\n%s", mustNot, source)
		}
	}
	if !strings.Contains(source, "export async function buildClientHeaders") {
		t.Fatalf("headers.ts must export buildClientHeaders, got:\n%s", source)
	}
}

func TestRenderClientOptions_omitsClientNameWhenClientIdentityDisabled(t *testing.T) {

	dir := t.TempDir()
	renderer := NewClientRenderer(testHTTPProject("1.0.0"), dir, false, "", false)
	if err := renderer.RenderClientOptions(); err != nil {
		t.Fatalf("RenderClientOptions: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "options.ts"))
	if err != nil {
		t.Fatalf("read options.ts: %v", err)
	}
	source := string(content)

	if strings.Contains(source, "clientName") {
		t.Fatalf("options.ts must not declare clientName when client identity disabled, got:\n%s", source)
	}
}

func TestRenderClient_omitsIdentityWhenClientIdentityDisabled(t *testing.T) {

	dir := t.TempDir()
	renderer := NewClientRenderer(testHTTPProject("1.0.0"), dir, false, "", false)
	if err := renderer.RenderHeaders(); err != nil {
		t.Fatalf("RenderHeaders: %v", err)
	}
	if err := renderer.RenderVersion(); err != nil {
		t.Fatalf("RenderVersion: %v", err)
	}
	if err := renderer.RenderClientError(); err != nil {
		t.Fatalf("RenderClientError: %v", err)
	}
	if err := renderer.RenderJsonRPCLibrary(); err != nil {
		t.Fatalf("RenderJsonRPCLibrary: %v", err)
	}
	if err := renderer.RenderClient(); err != nil {
		t.Fatalf("RenderClient: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "client.ts"))
	if err != nil {
		t.Fatalf("read client.ts: %v", err)
	}
	source := string(content)

	for _, mustNot := range []string{
		`import {resolveDefaultClientName} from './identity'`,
		"clientName:resolveDefaultClientName()",
	} {
		if strings.Contains(source, mustNot) {
			t.Fatalf("client.ts must not contain %q when client identity disabled, got:\n%s", mustNot, source)
		}
	}
	if !strings.Contains(source, "return buildClientHeaders(this.options)") {
		t.Fatalf("client.ts must still use buildClientHeaders, got:\n%s", source)
	}
}
