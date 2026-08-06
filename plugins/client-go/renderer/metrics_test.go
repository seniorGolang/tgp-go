// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderClientMetrics_DefaultSecondsBuckets(t *testing.T) {

	project := httpClientTestProject()
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, "example", "client")
	if err := renderer.RenderClientMetrics(); err != nil {
		t.Fatalf("RenderClientMetrics: %v", err)
	}
	if err := renderer.RenderClientOptions(); err != nil {
		t.Fatalf("RenderClientOptions: %v", err)
	}
	if err := renderer.RenderClient(); err != nil {
		t.Fatalf("RenderClient: %v", err)
	}

	metricsContent, err := os.ReadFile(filepath.Join(dir, "metrics.go"))
	if err != nil {
		t.Fatalf("read metrics.go: %v", err)
	}
	metricsSource := string(metricsContent)
	for _, want := range []string{
		"var defaultSecondsBuckets = []float64{",
		"0.001", "0.0025", "0.015", "0.02", "0.035", "0.075",
		"func validBuckets(buckets []float64) (ok bool)",
		"buckets := cli.latencyBuckets",
		"buckets = defaultSecondsBuckets",
		"Buckets:   buckets",
	} {
		if !strings.Contains(metricsSource, want) {
			t.Fatalf("metrics.go must contain %q, got:\n%s", want, metricsSource)
		}
	}

	optionsContent, err := os.ReadFile(filepath.Join(dir, "options.go"))
	if err != nil {
		t.Fatalf("read options.go: %v", err)
	}
	optionsSource := string(optionsContent)
	for _, want := range []string{
		"func LatencyBuckets(buckets ...float64) (opt Option)",
		"if !validBuckets(buckets)",
		"cli.latencyBuckets = append([]float64{}, buckets...)",
		"func WithMetrics() (opt Option)",
	} {
		if !strings.Contains(optionsSource, want) {
			t.Fatalf("options.go must contain %q, got:\n%s", want, optionsSource)
		}
	}

	clientContent, err := os.ReadFile(filepath.Join(dir, "client.go"))
	if err != nil {
		t.Fatalf("read client.go: %v", err)
	}
	if !strings.Contains(string(clientContent), "latencyBuckets []float64") {
		t.Fatalf("Client must have latencyBuckets field, got:\n%s", clientContent)
	}
}
