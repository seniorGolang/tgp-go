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

func TestRenderTransportServer_IgnoresKafkaTraceAndMetrics(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Http",
				PkgPath: "example/contracts",
				ID:      "Http",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					TagLogger:           "",
				},
				Methods: []*model.Method{{
					Name: "Ping",
					Annotations: tags.DocTags{
						model.TagHTTPMethod: "GET",
						model.TagHttpPath:   "/ping",
					},
					Args:    []*model.Variable{{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}}},
					Results: []*model.Variable{{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}}},
				}},
			},
			{
				Name:    "OrderEvents",
				PkgPath: "example/contracts",
				ID:      "OrderEvents",
				Annotations: tags.DocTags{
					model.TagKafka: "",
					TagLogger:      "",
					TagMetrics:     "",
					TagTrace:       "",
				},
				Methods: []*model.Method{{
					Name:        "Publish",
					Annotations: tags.DocTags{model.TagKafkaTopic: "demo.orders"},
					Args: []*model.Variable{
						{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
						{Name: "event", TypeRef: model.TypeRef{TypeID: "string"}},
					},
					Results: []*model.Variable{{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}}},
				}},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}
	if err := renderer.RenderTransportMetrics(); err != nil {
		t.Fatalf("RenderTransportMetrics: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if strings.Contains(source, "tracer.Middleware") {
		t.Fatalf("tracer middleware must not come from kafka @tg trace, got:\n%s", source)
	}
	if strings.Contains(source, "func (srv *Server) WithTrace") {
		t.Fatalf("WithTrace must not come from kafka @tg trace, got:\n%s", source)
	}
	if strings.Contains(source, "inFlightMiddleware") {
		t.Fatalf("metrics middleware must not come from kafka @tg metrics, got:\n%s", source)
	}
	if _, err = os.Stat(filepath.Join(dir, "tracer")); !os.IsNotExist(err) {
		t.Fatalf("tracer package must not be generated from kafka @tg trace, err=%v", err)
	}
	if _, err = os.Stat(filepath.Join(dir, "metrics.go")); !os.IsNotExist(err) {
		t.Fatalf("metrics.go must not be generated from kafka @tg metrics, err=%v", err)
	}
}

func TestRenderTransportServer_HTTPTraceAndMetricsStillWork(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Http",
				PkgPath: "example/contracts",
				ID:      "Http",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					TagLogger:           "",
					TagMetrics:          "",
					TagTrace:            "",
				},
				Methods: []*model.Method{{
					Name: "Ping",
					Annotations: tags.DocTags{
						model.TagHTTPMethod: "GET",
						model.TagHttpPath:   "/ping",
					},
					Args:    []*model.Variable{{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}}},
					Results: []*model.Variable{{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}}},
				}},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}
	if err := renderer.RenderTransportMetrics(); err != nil {
		t.Fatalf("RenderTransportMetrics: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "tracer.Middleware") {
		t.Fatalf("expected tracer middleware for HTTP @tg trace, got:\n%s", source)
	}
	if !strings.Contains(source, "func (srv *Server) WithTrace") {
		t.Fatalf("expected WithTrace for HTTP @tg trace, got:\n%s", source)
	}
	if !strings.Contains(source, "inFlightMiddleware") {
		t.Fatalf("expected metrics middleware for HTTP @tg metrics, got:\n%s", source)
	}
	if _, err = os.Stat(filepath.Join(dir, "tracer")); err != nil {
		t.Fatalf("expected tracer package for HTTP @tg trace: %v", err)
	}
	if _, err = os.Stat(filepath.Join(dir, "metrics.go")); err != nil {
		t.Fatalf("expected metrics.go for HTTP @tg metrics: %v", err)
	}
}
