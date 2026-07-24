// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tgp/internal/model"
	"tgp/internal/tags"
	"tgp/plugins/kafka-sub-go/renderer"
)

func TestRenderSubscriberSecurityOptions(t *testing.T) {

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
	if err := renderer.NewRenderer(project, outDir, "example.com/app", "kafka").RenderAll(); err != nil {
		t.Fatalf("RenderAll: %v", err)
	}
	mustContain(t, filepath.Join(outDir, "security.go"), "func saslMechanism")
	mustContain(t, filepath.Join(outDir, "options.go"), "func TLS(config *tls.Config) Option")
	mustContain(t, filepath.Join(outDir, "options.go"), "func Auth(user string, password string) Option")
	mustContain(t, filepath.Join(outDir, "options.go"), "func SASL(mechanism string) Option")
	mustContain(t, filepath.Join(outDir, "options.go"), "kafka Auth and SASL must be set together")
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "DialTLSConfig")
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "kgo.SASL")
	mustContain(t, filepath.Join(outDir, "subscriber.go"), "saslMechanism")

	optionsBody, err := os.ReadFile(filepath.Join(outDir, "options.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(optionsBody), "SCRAM-SHA-512") {
		t.Fatal("options.go misses SCRAM-SHA-512")
	}
}
