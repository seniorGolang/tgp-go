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

func TestRenderJsonRPCClient_AsyncThrowsDecodedRPCError(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Version:    "1.0.0",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "RPC",
				PkgPath: "example/contracts",
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
					{
						Name: "ReturnsError",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
						},
						Results: []*model.Variable{
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
				},
			},
		},
	}

	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderExchangeTypes(project.Contracts[0]); err != nil {
		t.Fatalf("RenderExchangeTypes: %v", err)
	}
	if err := renderer.RenderJsonRPCClientClass(project.Contracts[0]); err != nil {
		t.Fatalf("RenderJsonRPCClientClass: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "rpc.ts"))
	if err != nil {
		t.Fatalf("read rpc.ts: %v", err)
	}
	src := string(body)
	if !strings.Contains(src, "throw this.baseClient.decodeRPCError(execResult.error)") {
		t.Fatalf("async methods must throw decodeRPCError(execResult.error):\n%s", src)
	}
	if strings.Contains(src, "throw execResult.error") {
		t.Fatalf("async methods must not throw raw execResult.error:\n%s", src)
	}
	if strings.Contains(src, "as ReturnsErrorError") {
		t.Fatalf("async path must not cast/throw typed error unions without decode:\n%s", src)
	}
}
