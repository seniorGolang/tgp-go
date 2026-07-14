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

func TestRenderTransportServer_WithLogSingleAssignmentPerHybridContract(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	assertSingleWithLogAssignment(t, source, "httpHybrid")
	assertSingleWithLogAssignment(t, source, "httpShapes")
	assertSingleWithMetricsAssignment(t, source, "httpHybrid")
	assertSingleWithMetricsAssignment(t, source, "httpShapes")
}

func TestRenderTransportServer_ClientIDMiddlewareAlwaysRegistered(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "func (srv *Server) clientIDMiddleware(ftx *fiber.Ctx) error") {
		t.Fatalf("clientIDMiddleware must always be generated, got:\n%s", source)
	}
	if !strings.Contains(source, `srv.srvHTTP.Use(srv.clientIDMiddleware)`) {
		t.Fatalf("clientIDMiddleware must always be registered, got:\n%s", source)
	}
}

func TestRenderTransportServer_ClientIDMiddlewareWithoutMetrics(t *testing.T) {

	project := transportServerLogOnlyProject()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if strings.Contains(source, "inFlightMiddleware") {
		t.Fatalf("metrics middleware must not be generated without @tg metrics, got:\n%s", source)
	}
	if !strings.Contains(source, `srv.srvHTTP.Use(srv.clientIDMiddleware)`) {
		t.Fatalf("clientIDMiddleware must be registered without @tg metrics, got:\n%s", source)
	}
}

func TestRenderTransportServer_WithLogKeepsHTTPOnlyAndJsonRPCOnlyContracts(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "srv.httpModes = srv.httpModes.WithLog()") {
		t.Fatalf("expected single WithLog assignment for HTTP-only contract Modes, got:\n%s", source)
	}
	if !strings.Contains(source, "srv.httpRpc = srv.httpRpc.WithLog()") {
		t.Fatalf("expected single WithLog assignment for JSON-RPC-only contract Rpc, got:\n%s", source)
	}
}

func TestRenderTransportServer_JSONRPCBatchMountsByPrefix(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "ABACV2",
				PkgPath: "example/contracts",
				ID:      "ABACV2",
				Annotations: tags.DocTags{
					model.TagServerJsonRPC: "",
					model.TagHttpPrefix:    "api/v2",
					model.TagHttpPath:      "abac",
				},
				Methods: []*model.Method{
					{
						Name: "CheckAccess",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
						},
						Results: []*model.Variable{
							{Name: "ok", TypeRef: model.TypeRef{TypeID: "bool"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
				},
			},
			{
				Name:    "Users",
				PkgPath: "example/contracts",
				ID:      "Users",
				Annotations: tags.DocTags{
					model.TagServerJsonRPC: "",
					model.TagHttpPrefix:    "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Get",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
						},
						Results: []*model.Variable{
							{Name: "id", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	for _, mount := range []string{"/", "/api", "/api/v1", "/api/v2"} {
		want := `srv.srvHTTP.Post("` + mount + `", srv.batchHandler("` + mount + `"))`
		if !strings.Contains(source, want) {
			t.Fatalf("expected batch registration %q, got:\n%s", want, source)
		}
	}
}

func TestRenderTransportJsonRPC_MethodMapsScopedByPrefix(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Users",
				PkgPath: "example/contracts",
				ID:      "Users",
				Annotations: tags.DocTags{
					model.TagServerJsonRPC: "",
					model.TagHttpPrefix:    "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Get",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
						},
						Results: []*model.Variable{
							{Name: "id", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
				},
			},
			{
				Name:    "Catalog",
				PkgPath: "example/contracts",
				ID:      "Catalog",
				Annotations: tags.DocTags{
					model.TagServerJsonRPC: "",
					model.TagHttpPrefix:    "api/v2",
				},
				Methods: []*model.Method{
					{
						Name: "List",
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
						},
						Results: []*model.Variable{
							{Name: "items", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportJsonRPC(); err != nil {
		t.Fatalf("RenderTransportJsonRPC: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "jsonrpc.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "func initJsonRPCMethodMaps(srv *Server)") {
		t.Fatalf("expected initJsonRPCMethodMaps, got:\n%s", source)
	}
	if !strings.Contains(source, `srv.jsonRPCMethodMaps["/api/v1"]["users.get"] = handler`) {
		t.Fatalf("expected users.get on /api/v1 scope, got:\n%s", source)
	}
	if !strings.Contains(source, `srv.jsonRPCMethodMaps["/api"]["users.get"] = handler`) {
		t.Fatalf("expected users.get on /api ancestor, got:\n%s", source)
	}
	if !strings.Contains(source, `srv.jsonRPCMethodMaps["/"]["users.get"] = handler`) {
		t.Fatalf("expected users.get on /, got:\n%s", source)
	}
	if strings.Contains(source, `srv.jsonRPCMethodMaps["/api/v2"]["users.get"]`) {
		t.Fatalf("users.get must not appear on /api/v2, got:\n%s", source)
	}
	if !strings.Contains(source, `srv.jsonRPCMethodMaps["/api/v2"]["catalog.list"] = handler`) {
		t.Fatalf("expected catalog.list on /api/v2, got:\n%s", source)
	}
	if strings.Contains(source, `srv.jsonRPCMethodMaps["/api/v1"]["catalog.list"]`) {
		t.Fatalf("catalog.list must not appear on /api/v1, got:\n%s", source)
	}
}

func assertSingleWithLogAssignment(t *testing.T, source string, field string) {

	t.Helper()
	pattern := "srv." + field + " = srv." + field + ".WithLog()"
	if strings.Count(source, pattern) != 1 {
		t.Fatalf("expected exactly one %q, got %d in:\n%s", pattern, strings.Count(source, pattern), source)
	}
	if strings.Contains(source, "srv."+field+" = srv."+strings.TrimPrefix(field, "http")+"().WithLog()") {
		t.Fatalf("must not use accessor WithLog for %s:\n%s", field, source)
	}
}

func assertSingleWithMetricsAssignment(t *testing.T, source string, field string) {

	t.Helper()
	pattern := "srv." + field + " = srv." + field + ".WithMetrics(srv.metrics)"
	if strings.Count(source, pattern) != 1 {
		t.Fatalf("expected exactly one %q, got %d in:\n%s", pattern, strings.Count(source, pattern), source)
	}
}

func transportServerTestProject() (project *model.Project) {

	httpMethod := func(name string) *model.Method {
		return &model.Method{
			Name: name,
			Args: []*model.Variable{
				{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
			},
			Results: []*model.Variable{
				{Name: "out", TypeRef: model.TypeRef{TypeID: "string"}},
				{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
			},
			Annotations: tags.DocTags{
				model.TagHTTPMethod: "GET",
				model.TagHttpPath:   "/test/" + strings.ToLower(name),
			},
		}
	}

	rpcMethod := func(name string) *model.Method {
		return &model.Method{
			Name: name,
			Args: []*model.Variable{
				{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
			},
			Results: []*model.Variable{
				{Name: "out", TypeRef: model.TypeRef{TypeID: "string"}},
				{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
			},
		}
	}

	project = &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Hybrid",
				PkgPath: "example/contracts",
				ID:      "Hybrid",
				Annotations: tags.DocTags{
					model.TagServerHTTP:    "",
					model.TagServerJsonRPC: "",
					TagLogger:              "",
					TagMetrics:             "",
				},
				Methods: []*model.Method{
					rpcMethod("RpcPing"),
					httpMethod("HttpEcho"),
				},
			},
			{
				Name:    "Shapes",
				PkgPath: "example/contracts",
				ID:      "Shapes",
				Annotations: tags.DocTags{
					model.TagServerHTTP:    "",
					model.TagServerJsonRPC: "",
					TagLogger:              "",
					TagMetrics:             "",
				},
				Methods: []*model.Method{
					rpcMethod("EchoEntity"),
					httpMethod("PostEntity"),
				},
			},
			{
				Name:    "Modes",
				PkgPath: "example/contracts",
				ID:      "Modes",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					TagLogger:           "",
				},
				Methods: []*model.Method{
					httpMethod("BodyHeader"),
				},
			},
			{
				Name:    "Rpc",
				PkgPath: "example/contracts",
				ID:      "Rpc",
				Annotations: tags.DocTags{
					model.TagServerJsonRPC: "",
					TagLogger:              "",
				},
				Methods: []*model.Method{
					rpcMethod("Ping"),
				},
			},
		},
	}
	return project
}

func transportServerLogOnlyProject() (project *model.Project) {

	project = &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Modes",
				PkgPath: "example/contracts",
				ID:      "Modes",
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
							{Name: "out", TypeRef: model.TypeRef{TypeID: "string"}},
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
	return project
}
