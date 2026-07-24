// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal"
	"tgp/internal/model"
	"tgp/internal/tags"
	"tgp/plugins/kafka-pub-go/renderer"
)

func TestRenderPublisher(t *testing.T) {

	root := t.TempDir()
	outDir := filepath.Join(root, "kafka")
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goMod := "module example.com/app\n\ngo 1.26\n\nrequire (\n\tgithub.com/prometheus/client_golang v1.23.2\n\tgithub.com/twmb/franz-go v1.21.5\n\tgo.opentelemetry.io/otel v1.39.0\n)\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	contractSource := `package contracts
import "context"
type Order struct { ID string }
type OrderEvents interface {
	OrderCreated(ctx context.Context, orderID string, traceID string, event Order) (err error)
	OrderBulk(ctx context.Context, tenantID string, events ...Order) (err error)
}
`
	if err := os.WriteFile(filepath.Join(contractsDir, "contracts.go"), []byte(contractSource), 0o644); err != nil {
		t.Fatal(err)
	}
	project := &model.Project{
		Types: map[string]*model.Type{
			"example.com/app/contracts:Order": {TypeName: "Order", ImportPkgPath: "example.com/app/contracts"},
		},
		Contracts: []*model.Contract{{
			Name: "OrderEvents", PkgPath: "example.com/app/contracts", Annotations: tags.DocTags{"kafka": "", "metrics": "", "trace": ""},
			Methods: []*model.Method{
				{Name: "OrderCreated", Annotations: tags.DocTags{"kafka-topic": "orders.created", "kafka-key": "orderID", "kafka-headers": "traceID|x-trace-id", "kafka-message": "event"}, Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}}, {Name: "orderID", TypeRef: model.TypeRef{TypeID: "string"}}, {Name: "traceID", TypeRef: model.TypeRef{TypeID: "string"}}, {Name: "event", TypeRef: model.TypeRef{TypeID: "example.com/app/contracts:Order"}},
				}, Results: []*model.Variable{{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}}}},
				{Name: "OrderBulk", Annotations: tags.DocTags{"kafka-topic": "orders.bulk", "kafka-acks": "noAck", "kafka-key": "tenantID", "kafka-message": "events"}, Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}}, {Name: "tenantID", TypeRef: model.TypeRef{TypeID: "string"}}, {Name: "events", TypeRef: model.TypeRef{TypeID: "example.com/app/contracts:Order", IsEllipsis: true}},
				}, Results: []*model.Variable{{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}}}},
			},
		}},
	}
	renderer := renderer.New(project, outDir, "example.com/app", "kafka")
	if err := renderer.Render(); err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, name := range []string{"client.go", "options.go", "adapters.go", "codec.go", "produce.go", "security.go", "version.go", "metrics.go", "tracing.go"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	body, err := os.ReadFile(filepath.Join(outDir, "adapters.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"func New", "func Brokers", "OrderEvents", "produceAndWait", "type codec"} {
		if expected == "func New" || expected == "func Brokers" || expected == "type codec" {
			continue
		}
		if !strings.Contains(string(body), expected) {
			t.Fatalf("adapters.go does not contain %q", expected)
		}
	}
	for _, name := range []string{"client.go", "options.go", "codec.go"} {
		body, err = os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), map[string]string{"client.go": "func New", "options.go": "func Brokers", "codec.go": "type codec"}[name]) {
			t.Fatalf("%s misses public API", name)
		}
	}
	body, err = os.ReadFile(filepath.Join(outDir, "adapters.go"))
	if err != nil {
		t.Fatal(err)
	}
	adapters := string(body)
	if strings.Contains(adapters, "make([]kgo.RecordHeader, 0, 0)") {
		t.Fatal("adapters.go must not allocate empty headers with cap 0")
	}
	encodeIdx := strings.Index(adapters, `observeProduceCall("OrderEvents", "OrderBulk", "orders.bulk", len(events), time.Since(started), err, "encode")`)
	if encodeIdx >= 0 {
		snippet := adapters[encodeIdx:min(len(adapters), encodeIdx+350)]
		if strings.Contains(snippet, "kafka produce completed") {
			t.Fatal("encode failure path must not log produce completed")
		}
	}
	if !strings.Contains(adapters, "_ = index") {
		t.Fatal("OrderBulk without indexed key/headers should blank unused index")
	}
	body, err = os.ReadFile(filepath.Join(outDir, "version.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `VersionASTg = "`+internal.Version+`"`) {
		t.Fatalf("version.go does not contain VersionASTg %q", internal.Version)
	}
	if strings.Contains(string(body), "PluginVersion") {
		t.Fatal("version.go must not contain PluginVersion")
	}
	body, err = os.ReadFile(filepath.Join(outDir, "client.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "PluginVersion") || strings.Contains(string(body), "VersionASTg") {
		t.Fatal("client.go must not declare version constant")
	}
	for _, expected := range []string{"kgo.NoAck()", "kgo.AllISRAcks()"} {
		if !strings.Contains(string(body), expected) {
			t.Fatalf("client.go misses %q", expected)
		}
	}
	if strings.Contains(string(body), "kgo.LeaderAck()") {
		t.Fatal("client.go must not create unused leaderAck client")
	}
	if !strings.Contains(string(body), "_ = kafkaClient.Flush(context.Background())") {
		t.Fatal("client.go must check Flush error via blank assign")
	}
	adaptersBody, err := os.ReadFile(filepath.Join(outDir, "adapters.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(adaptersBody), "var outcome produceOutcome") {
		t.Fatal("adapters.go must merge outcome declaration with assignment")
	}
	if !strings.Contains(string(adaptersBody), "outcome := produceAndWait(") {
		t.Fatal("adapters.go must assign produceAndWait with :=")
	}
	tracing, err := os.ReadFile(filepath.Join(outDir, "tracing.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(tracing), "func traceError") {
		t.Fatal("tracing.go must not contain unused traceError")
	}
	metrics, err := os.ReadFile(filepath.Join(outDir, "metrics.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"produce_inflight", "prometheus.DefBuckets", "VersionASTg"} {
		if !strings.Contains(string(metrics), expected) {
			t.Fatalf("metrics.go does not contain %q", expected)
		}
	}
	command := exec.Command("go", "build", "-mod=mod", "./kafka")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated package does not build: %v\n%s", err, output)
	}
}
