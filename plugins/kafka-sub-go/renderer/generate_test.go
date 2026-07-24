// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal"
	"tgp/internal/model"
	"tgp/internal/tags"
	"tgp/plugins/kafka-sub-go/renderer"
)

func TestRenderAllSubscriber(t *testing.T) {

	outDir := filepath.Join(t.TempDir(), "kafka")
	project := &model.Project{
		Types: map[string]*model.Type{
			"example.com/contracts:Order": {TypeName: "Order", ImportPkgPath: "example.com/contracts"},
		},
		Contracts: []*model.Contract{{
			Name:        "OrderEvents",
			PkgPath:     "example.com/contracts",
			Annotations: tags.DocTags{"kafka": ""},
			Methods: []*model.Method{{
				Name:        "OrderCreated",
				Annotations: tags.DocTags{"kafka-topic": "orders"},
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
					{Name: "event", TypeRef: model.TypeRef{TypeID: "example.com/contracts:Order"}},
				},
				Results: []*model.Variable{{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}}},
			}},
		}},
	}
	render := renderer.NewRenderer(project, outDir, "example.com/app", "kafka")
	if err := render.RenderAll(); err != nil {
		t.Fatalf("RenderAll: %v", err)
	}
	mustContain(t, filepath.Join(outDir, "handlers.go"), "type OrderEventsMetaHandler interface")
	mustContain(t, filepath.Join(outDir, "options.go"), "func Codec(name string, c codec) Option")
	mustContain(t, filepath.Join(outDir, "options.go"), "func CommitAfterBatch() Option")
	mustContain(t, filepath.Join(outDir, "options.go"), "func TLS(config *tls.Config) Option")
	mustContain(t, filepath.Join(outDir, "security.go"), "func saslMechanism")
	mustContain(t, filepath.Join(outDir, "version.go"), `VersionASTg = "`+internal.Version+`"`)
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "func New(log *slog.Logger, options ...Option)")
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "func (client *Client) Run(ctx context.Context) (err error)")
	body, err := os.ReadFile(filepath.Join(outDir, "subscriber.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "PluginVersion") || strings.Contains(string(body), "VersionASTg") {
		t.Fatal("subscriber.go must not declare version constant")
	}
}

func TestRenderAllSubscriberWithMetrics(t *testing.T) {

	outDir := filepath.Join(t.TempDir(), "kafka")
	project := &model.Project{
		Types: map[string]*model.Type{
			"example.com/contracts:Order": {TypeName: "Order", ImportPkgPath: "example.com/contracts"},
		},
		Contracts: []*model.Contract{{
			Name:        "OrderEvents",
			PkgPath:     "example.com/contracts",
			Annotations: tags.DocTags{"kafka": "", "metrics": "", "trace": ""},
			Methods: []*model.Method{{
				Name:        "OrderCreated",
				Annotations: tags.DocTags{"kafka-topic": "orders"},
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
					{Name: "event", TypeRef: model.TypeRef{TypeID: "example.com/contracts:Order"}},
				},
				Results: []*model.Variable{{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}}},
			}},
		}},
	}
	render := renderer.NewRenderer(project, outDir, "example.com/app", "kafka")
	if err := render.RenderAll(); err != nil {
		t.Fatalf("RenderAll: %v", err)
	}
	mustContain(t, filepath.Join(outDir, "metrics.go"), "consume_lag_records")
	mustContain(t, filepath.Join(outDir, "tracing.go"), "kafka.consume/")
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "startLagLoop")
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "started = time.Now()")
	mustContain(t, filepath.Join(outDir, "version.go"), `VersionASTg = "`+internal.Version+`"`)
}

func TestRenderSubscriberDedupsCodecChecks(t *testing.T) {

	outDir := filepath.Join(t.TempDir(), "kafka")
	project := &model.Project{
		Types: map[string]*model.Type{
			"example.com/contracts:Order": {TypeName: "Order", ImportPkgPath: "example.com/contracts"},
		},
		Contracts: []*model.Contract{{
			Name:        "OrderEvents",
			PkgPath:     "example.com/contracts",
			Annotations: tags.DocTags{"kafka": ""},
			Methods: []*model.Method{
				{
					Name:        "Created",
					Annotations: tags.DocTags{"kafka-topic": "orders.created"},
					Args: []*model.Variable{
						{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
						{Name: "event", TypeRef: model.TypeRef{TypeID: "example.com/contracts:Order"}},
					},
					Results: []*model.Variable{{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}}},
				},
				{
					Name:        "Updated",
					Annotations: tags.DocTags{"kafka-topic": "orders.updated"},
					Args: []*model.Variable{
						{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
						{Name: "event", TypeRef: model.TypeRef{TypeID: "example.com/contracts:Order"}},
					},
					Results: []*model.Variable{{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}}},
				},
			},
		}},
	}
	render := renderer.NewRenderer(project, outDir, "example.com/app", "kafka")
	if err := render.RenderAll(); err != nil {
		t.Fatalf("RenderAll: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(outDir, "subscriber.go"))
	if err != nil {
		t.Fatal(err)
	}
	needle := `kafka subscriber: codec json is required`
	if strings.Count(string(body), needle) != 1 {
		t.Fatalf("expected one json codec check, got %d", strings.Count(string(body), needle))
	}
}

func mustContain(t *testing.T, filePath string, expected string) {

	t.Helper()
	body, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), expected) {
		t.Fatalf("%s does not contain %q", filePath, expected)
	}
}
