// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
	"tgp/plugins/kafka-pub-go/renderer"
)

func TestRenderPublisherSecurityOptions(t *testing.T) {

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
	OrderCreated(ctx context.Context, event Order) (err error)
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
			Methods: []*model.Method{{
				Name: "OrderCreated", Annotations: tags.DocTags{"kafka-topic": "orders.created", "kafka-message": "event"},
				Args: []*model.Variable{
					{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
					{Name: "event", TypeRef: model.TypeRef{TypeID: "example.com/app/contracts:Order"}},
				},
				Results: []*model.Variable{{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}}},
			}},
		}},
	}
	if err := renderer.New(project, outDir, "example.com/app", "kafka").Render(); err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, name := range []string{"security.go", "options.go", "client.go"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	optionsBody, err := os.ReadFile(filepath.Join(outDir, "options.go"))
	if err != nil {
		t.Fatal(err)
	}
	options := string(optionsBody)
	for _, expected := range []string{
		"func TLS(config *tls.Config)",
		"func Auth(user string, password string)",
		"func SASL(mechanism string)",
		"func validateSecurity",
		"kafka Auth and SASL must be set together",
		"SCRAM-SHA-256",
	} {
		if !strings.Contains(options, expected) {
			t.Fatalf("options.go misses %q", expected)
		}
	}
	clientBody, err := os.ReadFile(filepath.Join(outDir, "client.go"))
	if err != nil {
		t.Fatal(err)
	}
	client := string(clientBody)
	for _, expected := range []string{"DialTLSConfig", "kgo.SASL", "saslMechanism", "validateSecurity"} {
		if !strings.Contains(client, expected) {
			t.Fatalf("client.go misses %q", expected)
		}
	}
	securityBody, err := os.ReadFile(filepath.Join(outDir, "security.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(securityBody), "func saslMechanism") {
		t.Fatal("security.go misses saslMechanism")
	}
	testSource := `package kafka

import (
	"crypto/tls"
	"strings"
	"testing"
)

func TestSecurityOptions(t *testing.T) {
	if err := TLS(nil)(&setup{}); err == nil || !strings.Contains(err.Error(), "TLS config is nil") {
		t.Fatalf("TLS(nil): %v", err)
	}
	if err := Auth("", "pass")(&setup{}); err == nil || !strings.Contains(err.Error(), "user is required") {
		t.Fatalf("Auth empty user: %v", err)
	}
	if err := Auth("user", "")(&setup{}); err == nil || !strings.Contains(err.Error(), "password is required") {
		t.Fatalf("Auth empty password: %v", err)
	}
	if err := SASL("GSSAPI")(&setup{}); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("SASL unknown: %v", err)
	}
	cfg := defaultSetup()
	if err := Auth("user", "pass")(&cfg); err != nil {
		t.Fatal(err)
	}
	if err := validateSecurity(cfg); err == nil || !strings.Contains(err.Error(), "together") {
		t.Fatalf("Auth without SASL: %v", err)
	}
	cfg = defaultSetup()
	if err := SASL("plain")(&cfg); err != nil {
		t.Fatal(err)
	}
	if err := validateSecurity(cfg); err == nil || !strings.Contains(err.Error(), "together") {
		t.Fatalf("SASL without Auth: %v", err)
	}
	cfg = defaultSetup()
	if err := Auth("user", "pass")(&cfg); err != nil {
		t.Fatal(err)
	}
	if err := SASL("scram-sha-256")(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.saslName != "SCRAM-SHA-256" {
		t.Fatalf("saslName=%q", cfg.saslName)
	}
	if err := validateSecurity(cfg); err != nil {
		t.Fatal(err)
	}
	if err := TLS(&tls.Config{})(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.tlsConfig == nil {
		t.Fatal("tlsConfig unset")
	}
	if _, err := saslMechanism("PLAIN", "u", "p"); err != nil {
		t.Fatal(err)
	}
	if _, err := saslMechanism("nope", "u", "p"); err == nil {
		t.Fatal("expected unsupported mechanism")
	}
}
`
	if err = os.WriteFile(filepath.Join(outDir, "security_test.go"), []byte(testSource), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "-mod=mod", "./kafka", "-count=1", "-run", "TestSecurityOptions")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("security options test failed: %v\n%s", err, output)
	}
}
