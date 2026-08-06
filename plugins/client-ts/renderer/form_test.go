// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

var nodeOnlyFormPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\bprocess\b`),
	regexp.MustCompile(`\brequire\s*\(`),
	regexp.MustCompile(`["']node:`),
	regexp.MustCompile(`\bBuffer\b`),
	regexp.MustCompile(`\b__dirname\b`),
	regexp.MustCompile(`\b__filename\b`),
	regexp.MustCompile(`\bfs\b`),
	regexp.MustCompile(`\blocalStorage\b`),
	regexp.MustCompile(`\bwindow\b`),
	regexp.MustCompile(`\bdocument\b`),
}

func TestRenderForm_isomorphicModule(t *testing.T) {

	dir := t.TempDir()
	renderer := NewClientRenderer(&model.Project{ModulePath: "example"}, dir, false, "", true)
	if err := renderer.RenderForm(); err != nil {
		t.Fatalf("RenderForm: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "form.ts"))
	if err != nil {
		t.Fatalf("read form.ts: %v", err)
	}
	source := string(content)
	for _, want := range []string{
		"export function parseFormValue",
		"export function parseCSVFormValue",
		`val.split(',')`,
		"elemStr",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("form.ts must contain %q, got:\n%s", want, source)
		}
	}
	assertNoNodeOnlyTokens(t, "form.ts", source)
}

func TestRenderForm_runtimeCSVVectors(t *testing.T) {

	dir := t.TempDir()
	renderer := NewClientRenderer(&model.Project{ModulePath: "example"}, dir, false, "", true)
	if err := renderer.RenderForm(); err != nil {
		t.Fatalf("RenderForm: %v", err)
	}

	harness := filepath.Join(dir, "form_check.mjs")
	script := `import { parseCSVFormValue, parseFormValue } from './form.ts';

const cases = [
  { name: 'numbers', got: parseCSVFormValue('4,5,18', 'number'), want: [4, 5, 18] },
  { name: 'trim', got: parseCSVFormValue(' 4, 5 ,18 ', 'number'), want: [4, 5, 18] },
  { name: 'empty-parts', got: parseCSVFormValue('4,,18', 'number'), want: [4, 18] },
  { name: 'strings', got: parseCSVFormValue('a,b,c', 'string'), want: ['a', 'b', 'c'] },
  { name: 'null', got: parseCSVFormValue(null, 'number'), want: undefined },
  { name: 'bad-number-zero', got: parseCSVFormValue('4,x,18', 'number'), want: [4, 0, 18] },
  { name: 'scalar-number', got: parseFormValue('42', 'number'), want: 42 },
];

for (const c of cases) {
  const gotJSON = JSON.stringify(c.got);
  const wantJSON = JSON.stringify(c.want);
  if (gotJSON !== wantJSON) {
    console.error(c.name + ': got ' + gotJSON + ' want ' + wantJSON);
    process.exit(1);
  }
}
console.log('ok');
`
	if err := os.WriteFile(harness, []byte(script), 0o644); err != nil {
		t.Fatalf("write harness: %v", err)
	}

	cmd := exec.Command("node", "--experimental-strip-types", harness)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("node form runtime check failed: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok") {
		t.Fatalf("unexpected harness output: %s", stdout.String())
	}
}

func TestRenderHTTPClient_importsIsomorphicFormHelpers(t *testing.T) {

	project := &model.Project{ModulePath: "example"}
	contract := &model.Contract{
		Name:    "Http",
		PkgPath: "example/contracts",
		Annotations: tags.DocTags{
			model.TagServerHTTP: "",
			model.TagHTTPPrefix: "api/v1",
		},
		Methods: []*model.Method{{
			Name: "EchoIDHeader",
			Annotations: tags.DocTags{
				model.TagHTTPMethod: "GET",
				model.TagHTTPPath:   "/items/echo-ids-header",
				model.TagHTTPHeader: "ids|X-Item-Ids",
			},
			Results: []*model.Variable{
				{Name: "ids", TypeID: "int64", IsSlice: true},
			},
		}},
	}

	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderForm(); err != nil {
		t.Fatalf("RenderForm: %v", err)
	}
	if err := renderer.RenderHTTPClientClass(contract); err != nil {
		t.Fatalf("RenderHTTPClientClass: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "http-http.ts"))
	if err != nil {
		t.Fatalf("read http-http.ts: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, `import {parseCSVFormValue} from './form'`) &&
		!strings.Contains(source, `import {parseFormValue, parseCSVFormValue} from './form'`) &&
		!strings.Contains(source, `import {parseCSVFormValue, parseFormValue} from './form'`) {
		t.Fatalf("HTTP client must import form helpers from ./form, got:\n%s", source)
	}
	if strings.Contains(source, "function parseCSVFormValue") || strings.Contains(source, "function parseFormValue") {
		t.Fatalf("HTTP client must not inline form helpers:\n%s", source)
	}
	if !strings.Contains(source, `parseCSVFormValue(_response_.headers.get("X-Item-Ids"), "number")`) {
		t.Fatalf("EchoIDHeader must CSV-parse X-Item-Ids as number[], got:\n%s", source)
	}
	assertNoNodeOnlyTokens(t, "http-http.ts", source)
}

func assertNoNodeOnlyTokens(t *testing.T, name string, source string) {

	t.Helper()
	patterns := nodeOnlyFormPatterns
	if name != "form.ts" {
		patterns = []*regexp.Regexp{
			regexp.MustCompile(`\bprocess\b`),
			regexp.MustCompile(`\brequire\s*\(`),
			regexp.MustCompile(`["']node:`),
			regexp.MustCompile(`\b__dirname\b`),
			regexp.MustCompile(`\b__filename\b`),
			regexp.MustCompile(`\blocalStorage\b`),
		}
	}
	for _, pattern := range patterns {
		if pattern.MatchString(source) {
			t.Fatalf("%s must stay isomorphic (browser+Node), matched %q", name, pattern.String())
		}
	}
}
