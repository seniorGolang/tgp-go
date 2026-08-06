package generator

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
)

func TestCmdTemplate_ServeHealthUsesOptionsAPI(t *testing.T) {

	source := renderCmdTemplate(t)

	required := []string{
		`transport.ServeHealth(serverLogger, "/liveness", cfg.LivenessBind)`,
		`transport.ServeHealth(serverLogger, "/readiness", cfg.ReadinessBind)`,
	}
	for _, snippet := range required {
		if !strings.Contains(source, snippet) {
			t.Fatalf("expected snippet %q in cmd.tmpl output:\n%s", snippet, source)
		}
	}

	forbidden := []string{
		`ServeHealth(serverLogger, "/liveness", config.Service().LivenessBind, "ok")`,
		`ServeHealth(serverLogger, "/readiness", config.Service().ReadinessBind, "ok")`,
		`WithResponse(`,
	}
	for _, snippet := range forbidden {
		if strings.Contains(source, snippet) {
			t.Fatalf("unexpected snippet %q in cmd.tmpl output:\n%s", snippet, source)
		}
	}
}

func TestCmdTemplate_LifecycleUsesNotifyContext(t *testing.T) {

	source := renderCmdTemplate(t)

	required := []string{
		`cfg := config.Service()`,
		`"example.com/demo/pkg/errs"`,
		`ctx, stop := signal.NotifyContext(log.Logger.WithContext(context.Background()), syscall.SIGINT, syscall.SIGTERM)`,
		`defer stop()`,
		`if shutdownErr := srv.ShutdownWithTimeout(cfg.ShutdownTimeout); shutdownErr != nil`,
		`if listenErr := srv.Fiber().Listen(cfg.Bind); listenErr != nil`,
		`errs.PanicOnError(ctx, listenErr, "server listen error")`,
		`<-ctx.Done()`,
		"\tstop()",
	}
	for _, snippet := range required {
		if !strings.Contains(source, snippet) {
			t.Fatalf("expected snippet %q in cmd.tmpl output:\n%s", snippet, source)
		}
	}

	forbidden := []string{
		`signal.Notify(`,
		`make(chan os.Signal`,
		`_ = srv.Shutdown()`,
		`log.Panic()`,
		`config.Service().Bind`,
		`config.Service().LivenessBind`,
		`config.Service().ReadinessBind`,
		`config.Service().MetricsBind`,
	}
	for _, snippet := range forbidden {
		if strings.Contains(source, snippet) {
			t.Fatalf("unexpected snippet %q in cmd.tmpl output:\n%s", snippet, source)
		}
	}
	if strings.Contains(source, "srv.Shutdown()") {
		t.Fatalf("unexpected srv.Shutdown() without timeout in cmd.tmpl output:\n%s", source)
	}
}

func TestConfigTemplate_ShutdownTimeout(t *testing.T) {

	tmpl, err := template.ParseFS(templates, "templates/*.tmpl")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	var buf bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buf, "config.tmpl", nil); err != nil {
		t.Fatalf("execute config.tmpl: %v", err)
	}
	source := buf.String()

	snippet := `ShutdownTimeout time.Duration ` + "`" + `envconfig:"SHUTDOWN_TIMEOUT" default:"10s"` + "`"
	if !strings.Contains(source, snippet) {
		t.Fatalf("expected snippet %q in config.tmpl output:\n%s", snippet, source)
	}
}

func renderCmdTemplate(t *testing.T) (source string) {

	t.Helper()
	tmpl, err := template.ParseFS(templates, "templates/*.tmpl")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	data := struct {
		Module          string
		ServiceName     string
		ServicePackages []InterfaceSpec
	}{
		Module:      "example.com/demo",
		ServiceName: "demo",
		ServicePackages: []InterfaceSpec{
			{PublicName: "Some", FileBase: "some"},
		},
	}

	var buf bytes.Buffer
	if err = tmpl.ExecuteTemplate(&buf, "cmd.tmpl", data); err != nil {
		t.Fatalf("execute cmd.tmpl: %v", err)
	}
	return buf.String()
}
