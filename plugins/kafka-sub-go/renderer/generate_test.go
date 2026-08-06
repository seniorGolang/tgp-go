// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal"
	"tgp/internal/generated"
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
					{Name: "ctx", TypeID: "context:Context"},
					{Name: "event", TypeID: "example.com/contracts:Order"},
				},
				Results: []*model.Variable{{Name: "err", TypeID: "error"}},
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
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "runDone")
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "shutdownTimeout")
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "AllowRebalance")
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "CloseAllowingRebalance")
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "kafka handler panic")
	mustContain(t, filepath.Join(outDir, "options.go"), "func ShutdownTimeout")
	mustContain(t, filepath.Join(outDir, "options.go"), "at-most-once")
	mustAllGenerated(t, outDir)
	body, err := os.ReadFile(filepath.Join(outDir, "subscriber.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	if strings.Contains(source, "PluginVersion") || strings.Contains(source, "VersionASTg") {
		t.Fatal("subscriber.go must not declare version constant")
	}
	if strings.Contains(source, "lagDone") || strings.Contains(source, "startLagLoop") {
		t.Fatal("subscriber without metrics must not generate lag loop")
	}
	if strings.Contains(source, "2 * time.Second") {
		t.Fatal("Close must not use a hardcoded grace timeout")
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
					{Name: "ctx", TypeID: "context:Context"},
					{Name: "event", TypeID: "example.com/contracts:Order"},
				},
				Results: []*model.Variable{{Name: "err", TypeID: "error"}},
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
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "lagDone")
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "<-lagDone")
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "started = time.Now()")
	mustContain(t, filepath.Join(outDir, "version.go"), `VersionASTg = "`+internal.Version+`"`)
	body, err := os.ReadFile(filepath.Join(outDir, "subscriber.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	stopIdx := strings.Index(source, "if stop != nil")
	lagStopIdx := strings.Index(source, "if lagStop != nil")
	runDoneIdx := strings.Index(source, "if runDone != nil")
	lagDoneIdx := strings.Index(source, "if lagDone != nil")
	finalCloseIdx := strings.LastIndex(source, "client.client.Close()")
	if stopIdx < 0 || lagStopIdx < 0 || runDoneIdx < 0 || lagDoneIdx < 0 || finalCloseIdx < 0 {
		t.Fatal("Close must cancel and join run/lag before client.Close")
	}
	if stopIdx >= lagStopIdx || lagStopIdx >= runDoneIdx || runDoneIdx >= lagDoneIdx || lagDoneIdx >= finalCloseIdx {
		t.Fatal("Close must cancel run then lag, wait run then lag, then Close")
	}
	if !strings.Contains(source, "shutdown timeout") {
		t.Fatal("Close must use shutdown timeout when waiting for Run")
	}
	mustContain(t, filepath.Join(outDir, "metrics.go"), "registerOrGet")
	if !strings.Contains(source, "if err = ctx.Err(); err != nil") {
		t.Fatal("scrapeLag must exit early when ctx is canceled")
	}
	mustAllGenerated(t, outDir)
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
						{Name: "ctx", TypeID: "context:Context"},
						{Name: "event", TypeID: "example.com/contracts:Order"},
					},
					Results: []*model.Variable{{Name: "err", TypeID: "error"}},
				},
				{
					Name:        "Updated",
					Annotations: tags.DocTags{"kafka-topic": "orders.updated"},
					Args: []*model.Variable{
						{Name: "ctx", TypeID: "context:Context"},
						{Name: "event", TypeID: "example.com/contracts:Order"},
					},
					Results: []*model.Variable{{Name: "err", TypeID: "error"}},
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

func mustAllGenerated(t *testing.T, outDir string) {

	t.Helper()
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		count++
		mustContain(t, filepath.Join(outDir, entry.Name()), generated.ByToolGateway)
	}
	if count == 0 {
		t.Fatal("expected generated .go files")
	}
}
