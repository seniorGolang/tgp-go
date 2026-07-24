// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
)

func TestRenderViewer_RespectsLogValuer(t *testing.T) {

	project := &model.Project{
		ModulePath: "example.com/app",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Demo",
				PkgPath: "example.com/app/contracts",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					TagLogger:           "",
				},
				Methods: []*model.Method{
					{
						Name: "Ping",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
						},
						Results: []*model.Variable{
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod: "GET",
							model.TagHttpPath:   "/ping",
						},
					},
				},
			},
		},
	}
	root := t.TempDir()
	dir := filepath.Join(root, "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderLogger(); err != nil {
		t.Fatalf("RenderLogger: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "viewer", "json.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, fragment := range []string{
		"slog.LogValuer",
		"slog.AnyValue(lv).Resolve()",
		"func slogValueToNode(",
		"typeJSONRawMessage",
		"json.RawMessage(v.Bytes())",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("viewer json.go missing %q:\n%s", fragment, source)
		}
	}

	testSource := "package viewer_test\n\n" +
		"import (\n" +
		"\t\"bytes\"\n" +
		"\t\"encoding/hex\"\n" +
		"\t\"encoding/json\"\n" +
		"\t\"io\"\n" +
		"\t\"log/slog\"\n" +
		"\t\"testing\"\n\n" +
		"\t\"example.com/app/transport/viewer\"\n" +
		")\n\n" +
		"type safeDTO struct {\n" +
		"\tName string\n" +
		"\tBody io.Reader\n" +
		"}\n\n" +
		"func (s safeDTO) LogValue() slog.Value {\n" +
		"\treturn slog.StringValue(\"<safeDTO>\")\n" +
		"}\n\n" +
		"func TestAnyUsesLogValuer(t *testing.T) {\n" +
		"\tattr := viewer.Any(\"request\", safeDTO{Name: \"x\", Body: bytes.NewReader([]byte(\"secret\"))})\n" +
		"\tval := attr.Value.Resolve()\n" +
		"\tif val.Kind() != slog.KindString || val.String() != \"<safeDTO>\" {\n" +
		"\t\tt.Fatalf(\"resolved = kind=%v value=%q, want KindString <safeDTO>\", val.Kind(), val.String())\n" +
		"\t}\n" +
		"}\n\n" +
		"type withParams struct {\n" +
		"\tParams json.RawMessage\n" +
		"\tBlob   []byte\n" +
		"}\n\n" +
		"func TestAnyKeepsRawMessageNotHex(t *testing.T) {\n" +
		"\traw := json.RawMessage(`{\"hooks\":true}`)\n" +
		"\tattr := viewer.Any(\"response\", withParams{Params: raw, Blob: []byte{0x01, 0x02}})\n" +
		"\tval := attr.Value.Resolve()\n" +
		"\ttree, ok := val.Any().(map[string]any)\n" +
		"\tif !ok {\n" +
		"\t\tt.Fatalf(\"tree type = %T, want map\", val.Any())\n" +
		"\t}\n" +
		"\tparams, ok := tree[\"Params\"].(json.RawMessage)\n" +
		"\tif !ok || string(params) != string(raw) {\n" +
		"\t\tt.Fatalf(\"Params = %#v, want raw JSON\", tree[\"Params\"])\n" +
		"\t}\n" +
		"\tblob, ok := tree[\"Blob\"].(string)\n" +
		"\tif !ok || blob != hex.EncodeToString([]byte{0x01, 0x02}) {\n" +
		"\t\tt.Fatalf(\"Blob = %#v, want hex of binary\", tree[\"Blob\"])\n" +
		"\t}\n" +
		"}\n"
	if err = os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "viewer", "logvaluer_test.go"), []byte(testSource), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "-mod=mod", "./transport/viewer/")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go test viewer: %v\n%s", err, output)
	}
}
