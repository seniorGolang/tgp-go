// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderTransportMetrics_DefaultSecondsBuckets(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportMetrics(); err != nil {
		t.Fatalf("RenderTransportMetrics: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "metrics.go"))
	if err != nil {
		t.Fatalf("read metrics.go: %v", err)
	}
	source := string(content)
	for _, want := range []string{
		"var defaultSecondsBuckets = []float64{",
		"0.001", "0.0025", "0.0075", "0.015", "0.02", "0.035", "0.075",
		"func NewMetrics() (m *Metrics)",
		"func newMetrics(durationBuckets []float64, latencyBuckets []float64) (m *Metrics)",
		"func validBuckets(buckets []float64) (ok bool)",
		"durationBuckets = defaultSecondsBuckets",
		"latencyBuckets = defaultSecondsBuckets",
		"Buckets:   durationBuckets",
		"Buckets:   latencyBuckets",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("metrics.go must contain %q, got:\n%s", want, source)
		}
	}
	if strings.Contains(source, "0.001, 0.005, 0.01, 0.025") {
		t.Fatalf("old coarse bucket ladder must not remain, got:\n%s", source)
	}
}

func TestRenderTransportOptions_DurationAndLatencyBuckets(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportOptions(); err != nil {
		t.Fatalf("RenderTransportOptions: %v", err)
	}
	if err := renderer.RenderTransportMetrics(); err != nil {
		t.Fatalf("RenderTransportMetrics: %v", err)
	}
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}

	optionsContent, err := os.ReadFile(filepath.Join(dir, "options.go"))
	if err != nil {
		t.Fatalf("read options.go: %v", err)
	}
	optionsSource := string(optionsContent)
	for _, want := range []string{
		"func DurationBuckets(buckets ...float64) (opt Option)",
		"func LatencyBuckets(buckets ...float64) (opt Option)",
		"if !validBuckets(buckets)",
		"srv.durationBuckets = append([]float64{}, buckets...)",
		"srv.latencyBuckets = append([]float64{}, buckets...)",
	} {
		if !strings.Contains(optionsSource, want) {
			t.Fatalf("options.go must contain %q, got:\n%s", want, optionsSource)
		}
	}

	serverContent, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read server.go: %v", err)
	}
	serverSource := string(serverContent)
	for _, want := range []string{
		"durationBuckets []float64",
		"latencyBuckets",
		"srv.metrics = newMetrics(srv.durationBuckets, srv.latencyBuckets)",
	} {
		if !strings.Contains(serverSource, want) {
			t.Fatalf("server.go must contain %q, got:\n%s", want, serverSource)
		}
	}
}

func TestRenderTransportOptions_BucketsOnlyWithMetrics(t *testing.T) {

	project := transportServerTestProject()
	for _, contract := range project.Contracts {
		delete(contract.Annotations, TagMetrics)
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportOptions(); err != nil {
		t.Fatalf("RenderTransportOptions: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "options.go"))
	if err != nil {
		t.Fatalf("read options.go: %v", err)
	}
	source := string(content)
	if strings.Contains(source, "DurationBuckets") || strings.Contains(source, "LatencyBuckets") {
		t.Fatalf("bucket options must not be generated without @tg metrics, got:\n%s", source)
	}
}
