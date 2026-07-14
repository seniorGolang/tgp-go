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

func TestRenderJsonRPC_ServiceBatchUsesScopedMethodMap(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Rpc",
				PkgPath: "example/contracts",
				ID:      "Rpc",
				Annotations: tags.DocTags{
					model.TagServerJsonRPC: "",
					model.TagHttpPrefix:    "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Echo",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "msg", TypeRef: model.TypeRef{TypeID: "string"}},
						},
						Results: []*model.Variable{
							{Name: "out", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderJsonRPC(); err != nil {
		t.Fatalf("RenderJsonRPC: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "rpc-jsonrpc.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, `methods := http.srv.jsonRPCMethodMaps["/api/v1/rpc"]`) {
		t.Fatalf("expected scoped methods lookup for /api/v1/rpc, got:\n%s", source)
	}
	if strings.Contains(source, `jsonRPCMethodMaps["/"]`) {
		t.Fatalf("service serveBatch must not use root method map, got:\n%s", source)
	}
	if !strings.Contains(source, "http.srv.doSingleBatch(") {
		t.Fatalf("expected srv.doSingleBatch for single request, got:\n%s", source)
	}
	if !strings.Contains(source, "http.srv.doBatch(") {
		t.Fatalf("expected srv.doBatch for batch request, got:\n%s", source)
	}
	if strings.Contains(source, "func (http *httpRpc) doSingleBatch(") {
		t.Fatalf("local doSingleBatch must be removed, got:\n%s", source)
	}
}
