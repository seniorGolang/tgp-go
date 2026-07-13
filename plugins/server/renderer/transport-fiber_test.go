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

func TestRenderTransportFiber_RequestLoggerAlwaysIncludesClientID(t *testing.T) {

	project := transportFiberTestProject()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportFiber(); err != nil {
		t.Fatalf("RenderTransportFiber: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "fiber.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "func (srv *Server) requestLogger(ctx context.Context) (log *slog.Logger)") {
		t.Fatalf("expected requestLogger method, got:\n%s", source)
	}
	if !strings.Contains(source, `log = srv.log.With("clientID", srvctx.GetClientID(ctx))`) {
		t.Fatalf("requestLogger must always include clientID, got:\n%s", source)
	}
	if strings.Count(source, "srv.requestLogger(ctx)") < 3 {
		t.Fatalf("setLogger must use requestLogger in all branches, got:\n%s", source)
	}
	if strings.Contains(source, `SetUserContext(srvctx.WithCtx[*slog.Logger](ctx, srv.log))`) {
		t.Fatalf("setLogger must not put bare srv.log into context, got:\n%s", source)
	}
}

func TestRenderTransportFiber_RequestLoggerPreservesClientIDWithLogLevel(t *testing.T) {

	project := transportFiberTestProject()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportFiber(); err != nil {
		t.Fatalf("RenderTransportFiber: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "fiber.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "log := srv.requestLogger(ctx)") {
		t.Fatalf("X-Log-Level branch must build level logger from requestLogger, got:\n%s", source)
	}
	if !strings.Contains(source, "baseHandler := log.Handler()") {
		t.Fatalf("level handler must preserve request logger attrs, got:\n%s", source)
	}
}

func transportFiberTestProject() (project *model.Project) {

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
