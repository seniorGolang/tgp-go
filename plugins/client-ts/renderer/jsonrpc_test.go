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
					model.TagServerJSONRPC: "",
					model.TagHTTPPrefix:    "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Echo",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "msg", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "out", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
					},
					{
						Name: "ReturnsError",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
						},
						Results: []*model.Variable{
							{Name: "err", TypeID: "error"},
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
	if err := renderer.RenderJSONRPCClientClass(project.Contracts[0]); err != nil {
		t.Fatalf("RenderJSONRPCClientClass: %v", err)
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
	if err = renderer.RenderClient(); err != nil {
		t.Fatalf("RenderClient: %v", err)
	}
	clientBody, err := os.ReadFile(filepath.Join(dir, "client.ts"))
	if err != nil {
		t.Fatalf("read client.ts: %v", err)
	}
	clientSrc := string(clientBody)
	for _, fragment := range []string{
		"requestIDs.has(request.rpcRequest.id)",
		"duplicate batch request id:",
		"callbacks.delete(id)",
		"for (const callback of callbacks.values())",
		"batch response missing",
	} {
		if !strings.Contains(clientSrc, fragment) {
			t.Fatalf("batch client missing %q:\n%s", fragment, clientSrc)
		}
	}
}
