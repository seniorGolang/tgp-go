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
