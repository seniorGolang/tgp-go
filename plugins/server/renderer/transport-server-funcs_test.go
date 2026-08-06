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

func TestRenderTransportServer_WithLogSingleAssignmentPerHybridContract(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	assertSingleWithLogAssignment(t, source, "httpHybrid")
	assertSingleWithLogAssignment(t, source, "httpShapes")
	assertSingleWithMetricsAssignment(t, source, "httpHybrid")
	assertSingleWithMetricsAssignment(t, source, "httpShapes")
	if !strings.Contains(source, "func New(log *slog.Logger, options ...Option) (srv *Server)") {
		t.Fatalf("New must return *Server only:\n%s", source)
	}
	if strings.Contains(source, "initErr") || strings.Contains(source, "func New(log *slog.Logger, options ...Option) (srv *Server, err error)") {
		t.Fatalf("New must not return error or use initErr:\n%s", source)
	}
}

func TestRenderTransportServer_ClientIDMiddlewareAlwaysRegistered(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "func (srv *Server) clientIDMiddleware(ftx *fiber.Ctx) error") {
		t.Fatalf("clientIDMiddleware must always be generated, got:\n%s", source)
	}
	if !strings.Contains(source, `srv.srvHTTP.Use(srv.clientIDMiddleware)`) {
		t.Fatalf("clientIDMiddleware must always be registered, got:\n%s", source)
	}
}

func TestRenderTransportServer_FiberConfigImmutable(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)
	idx := strings.Index(source, "Immutable:")
	if idx < 0 {
		t.Fatalf("New must set Immutable field, got:\n%s", source)
	}
	snippet := source[idx:]
	if end := strings.IndexAny(snippet, ",\n}"); end > 0 {
		snippet = snippet[:end]
	}
	if !strings.Contains(snippet, "true") {
		t.Fatalf("Immutable must be true, got %q in:\n%s", snippet, source)
	}
}

func TestRenderTransportServer_DoesNotRegisterOverlayMiddleware(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)
	if strings.Contains(source, "requestOverlayMiddleware") {
		t.Fatalf("overlay middleware must not be registered:\n%s", source)
	}
}

func TestRenderTransportServer_ClientIDMiddlewareWithoutMetrics(t *testing.T) {

	project := transportServerLogOnlyProject()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if strings.Contains(source, "inFlightMiddleware") {
		t.Fatalf("metrics middleware must not be generated without @tg metrics, got:\n%s", source)
	}
	if !strings.Contains(source, `srv.srvHTTP.Use(srv.clientIDMiddleware)`) {
		t.Fatalf("clientIDMiddleware must be registered without @tg metrics, got:\n%s", source)
	}
}

func TestRenderTransportServer_WithLogKeepsHTTPOnlyAndJsonRPCOnlyContracts(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "srv.httpModes = srv.httpModes.WithLog()") {
		t.Fatalf("expected single WithLog assignment for HTTP-only contract Modes, got:\n%s", source)
	}
	if !strings.Contains(source, "srv.httpRpc = srv.httpRpc.WithLog()") {
		t.Fatalf("expected single WithLog assignment for JSON-RPC-only contract Rpc, got:\n%s", source)
	}
}

func TestRenderTransportServer_WithMetricsWiresWSContract(t *testing.T) {

	project := streamTestProject("Live", model.TagServerWS)
	project.Contracts[0].Annotations[TagMetrics] = ""
	project.Contracts[0].Annotations[TagLogger] = ""
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, "srv.httpLive = srv.httpLive.WithMetrics(srv.metrics)") {
		t.Fatalf("WithMetrics must wire WS contract Live, got:\n%s", source)
	}
	if !strings.Contains(source, "srv.httpLive = srv.httpLive.WithLog()") {
		t.Fatalf("WithLog must wire WS contract Live, got:\n%s", source)
	}
}

func TestRenderTransportServer_JSONRPCBatchMountsByPrefix(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "ABACV2",
				PkgPath: "example/contracts",
				ID:      "ABACV2",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
					model.TagHTTPPrefix:    "api/v2",
					model.TagHTTPPath:      "abac",
				},
				Methods: []*model.Method{
					{
						Name: "CheckAccess",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
						},
						Results: []*model.Variable{
							{Name: "ok", TypeID: "bool"},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
			{
				Name:    "Users",
				PkgPath: "example/contracts",
				ID:      "Users",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
					model.TagHTTPPrefix:    "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Get",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
						},
						Results: []*model.Variable{
							{Name: "id", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	for _, mount := range []string{"/", "/api", "/api/v1", "/api/v2"} {
		want := `srv.srvHTTP.Post("` + mount + `", srv.batchHandler("` + mount + `"))`
		if !strings.Contains(source, want) {
			t.Fatalf("expected batch registration %q, got:\n%s", want, source)
		}
	}
}

func TestRenderTransportJsonRPC_MethodMapsScopedByPrefix(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Users",
				PkgPath: "example/contracts",
				ID:      "Users",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
					model.TagHTTPPrefix:    "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Get",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
						},
						Results: []*model.Variable{
							{Name: "id", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
			{
				Name:    "Catalog",
				PkgPath: "example/contracts",
				ID:      "Catalog",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
					model.TagHTTPPrefix:    "api/v2",
				},
				Methods: []*model.Method{
					{
						Name: "List",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
						},
						Results: []*model.Variable{
							{Name: "items", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportJSONRPC(); err != nil {
		t.Fatalf("RenderTransportJSONRPC: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "jsonrpc.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, "func initJSONRPCMethodMaps(srv *Server)") {
		t.Fatalf("expected initJSONRPCMethodMaps, got:\n%s", source)
	}
	if !strings.Contains(source, `srv.jsonRPCMethodMaps["/api/v1"]["users.get"] = handler`) {
		t.Fatalf("expected users.get on /api/v1 scope, got:\n%s", source)
	}
	if !strings.Contains(source, `srv.jsonRPCMethodMaps["/api"]["users.get"] = handler`) {
		t.Fatalf("expected users.get on /api ancestor, got:\n%s", source)
	}
	if !strings.Contains(source, `srv.jsonRPCMethodMaps["/"]["users.get"] = handler`) {
		t.Fatalf("expected users.get on /, got:\n%s", source)
	}
	if strings.Contains(source, `srv.jsonRPCMethodMaps["/api/v2"]["users.get"]`) {
		t.Fatalf("users.get must not appear on /api/v2, got:\n%s", source)
	}
	if !strings.Contains(source, `srv.jsonRPCMethodMaps["/api/v2"]["catalog.list"] = handler`) {
		t.Fatalf("expected catalog.list on /api/v2, got:\n%s", source)
	}
	if strings.Contains(source, `srv.jsonRPCMethodMaps["/api/v1"]["catalog.list"]`) {
		t.Fatalf("catalog.list must not appear on /api/v1, got:\n%s", source)
	}
	if !strings.Contains(source, `srv.jsonRPCMethodMaps["/api/v1/users"]["get"] = handler`) {
		t.Fatalf("expected short method get on service batch path /api/v1/users, got:\n%s", source)
	}
	if !strings.Contains(source, `srv.jsonRPCMethodMaps["/api/v2/catalog"]["list"] = handler`) {
		t.Fatalf("expected short method list on service batch path /api/v2/catalog, got:\n%s", source)
	}
	if strings.Contains(source, `srv.jsonRPCMethodMaps["/api/v1/users"]["users.get"]`) {
		t.Fatalf("service path must not register qualified users.get, got:\n%s", source)
	}
	if strings.Contains(source, `srv.jsonRPCMethodMaps["/api/v2/catalog"]["catalog.list"]`) {
		t.Fatalf("service path must not register qualified catalog.list, got:\n%s", source)
	}
	if strings.Contains(source, `srv.jsonRPCMethodMaps["/api/v1/users"]["list"]`) {
		t.Fatalf("catalog list must not appear on users service path, got:\n%s", source)
	}
	if strings.Contains(source, `srv.jsonRPCMethodMaps["/api/v2/catalog"]["get"]`) {
		t.Fatalf("users get must not appear on catalog service path, got:\n%s", source)
	}
}

func TestRenderTransportServer_ServeHealthOptions(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	required := []string{
		"const defaultHealthCheckTimeout = 5 * time.Second",
		"type HealthOption func(hs *HealthServer)",
		"func WithResponse(response any) (opt HealthOption)",
		"func WithCheck(check func(ctx context.Context) (err error)) (opt HealthOption)",
		"func (hs *HealthServer) handle(ftx *fiber.Ctx) (err error)",
		"func ServeHealth(log *slog.Logger, path string, address string, opts ...HealthOption) (hs *HealthServer)",
		"checkTimeout: defaultHealthCheckTimeout",
		"hs.srv.Get(path, hs.handle)",
		"checkCtx, cancel := context.WithTimeout(ftx.UserContext(), hs.checkTimeout)",
		"return ftx.SendStatus(fiber.StatusServiceUnavailable)",
		"_, err = ftx.Write(hs.responseBody)",
		"hs.responseBody, err = ",
		"hs.responseBody = []byte(\"\\\"ok\\\"\")",
		"if hs.response != nil",
	}
	for _, snippet := range required {
		if !strings.Contains(source, snippet) {
			t.Fatalf("expected snippet %q in generated server.go:\n%s", snippet, source)
		}
	}

	forbidden := []string{
		"func ServeHealth(log *slog.Logger, path string, address string, response any, opts ...HealthOption) (hs *HealthServer)",
		"func ServeHealth(log *slog.Logger, path string, address string, response any) *HealthServer",
		"ftx.JSON(\"ok\")",
	}
	for _, snippet := range forbidden {
		if strings.Contains(source, snippet) {
			t.Fatalf("unexpected snippet %q in generated server.go:\n%s", snippet, source)
		}
	}

	if !strings.Contains(source, "check        func(ctx context.Context) (err error)") &&
		!strings.Contains(source, "check func(ctx context.Context) (err error)") {
		t.Fatalf("HealthServer must store check func, got:\n%s", source)
	}
	if !strings.Contains(source, "response     any") &&
		!strings.Contains(source, "response any") {
		t.Fatalf("HealthServer must store response staging field, got:\n%s", source)
	}
}

func TestRenderTransportServer_Shutdown(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportServer(); err != nil {
		t.Fatalf("RenderTransportServer: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "server.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	required := []string{
		"func (srv *Server) Shutdown() (err error)",
		"return srv.ShutdownWithTimeout(defaultShutdownTimeout)",
		"func (srv *Server) ShutdownWithTimeout(timeout time.Duration) (err error)",
		"srv.shutdownOnce.Do(func() {",
		"srv.rootCancel()",
		"clear(srv.streams)",
		"stream.Close()",
		"httpErr = srv.srvHTTP.ShutdownWithTimeout(timeout)",
		"metricsErr = srv.srvMetrics.ShutdownWithTimeout(timeout)",
		"wg.Go(func() {",
		"wg.Wait()",
		"srv.streamsWG.Wait()",
		"case <-shutdownCtx.Done():",
		"srv.shutdownErr = errors.Join(httpErr, metricsErr, traceErr, streamErr)",
		"err = srv.shutdownErr",
	}
	for _, snippet := range required {
		if !strings.Contains(source, snippet) {
			t.Fatalf("expected snippet %q in generated server.go:\n%s", snippet, source)
		}
	}
	if !strings.Contains(source, "var (") || !strings.Contains(source, "sync.WaitGroup") {
		t.Fatalf("expected parallel WaitGroup shutdown locals in generated server.go:\n%s", source)
	}

	shutdownIdx := strings.Index(source, "func (srv *Server) Shutdown() (err error)")
	withTimeoutIdx := strings.Index(source, "func (srv *Server) ShutdownWithTimeout(timeout time.Duration) (err error)")
	if shutdownIdx < 0 || withTimeoutIdx < 0 || shutdownIdx > withTimeoutIdx {
		t.Fatalf("expected Shutdown before ShutdownWithTimeout in generated server.go:\n%s", source)
	}
}

func TestRenderTransportOptions_MaxBatchWorkersIgnoresNonPositive(t *testing.T) {

	project := transportServerTestProject()
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportOptions(); err != nil {
		t.Fatalf("RenderTransportOptions: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "options.go"))
	if err != nil {
		t.Fatalf("read generated options.go: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{
		"func MaxBatchWorkers(size int) Option",
		"if size <= 0",
		"srv.maxParallelBatch = size",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("expected snippet %q in generated options.go:\n%s", fragment, source)
		}
	}
	for _, forbidden := range []string{"initErr", "must be greater than zero"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("options.go must not contain %q:\n%s", forbidden, source)
		}
	}
}

func assertSingleWithLogAssignment(t *testing.T, source string, field string) {

	t.Helper()
	pattern := "srv." + field + " = srv." + field + ".WithLog()"
	if strings.Count(source, pattern) != 1 {
		t.Fatalf("expected exactly one %q, got %d in:\n%s", pattern, strings.Count(source, pattern), source)
	}
	if strings.Contains(source, "srv."+field+" = srv."+strings.TrimPrefix(field, "http")+"().WithLog()") {
		t.Fatalf("must not use accessor WithLog for %s:\n%s", field, source)
	}
}

func assertSingleWithMetricsAssignment(t *testing.T, source string, field string) {

	t.Helper()
	pattern := "srv." + field + " = srv." + field + ".WithMetrics(srv.metrics)"
	if strings.Count(source, pattern) != 1 {
		t.Fatalf("expected exactly one %q, got %d in:\n%s", pattern, strings.Count(source, pattern), source)
	}
}

func transportServerTestProject() (project *model.Project) {

	httpMethod := func(name string) *model.Method {
		return &model.Method{
			Name: name,
			Args: []*model.Variable{
				{Name: "ctx", TypeID: "context:Context"},
			},
			Results: []*model.Variable{
				{Name: "out", TypeID: "string"},
				{Name: "err", TypeID: "error"},
			},
			Annotations: tags.DocTags{
				model.TagHTTPMethod: "GET",
				model.TagHTTPPath:   "/test/" + strings.ToLower(name),
			},
		}
	}

	rpcMethod := func(name string) *model.Method {
		return &model.Method{
			Name: name,
			Args: []*model.Variable{
				{Name: "ctx", TypeID: "context:Context"},
			},
			Results: []*model.Variable{
				{Name: "out", TypeID: "string"},
				{Name: "err", TypeID: "error"},
			},
		}
	}

	project = &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Hybrid",
				PkgPath: "example/contracts",
				ID:      "Hybrid",
				Annotations: tags.DocTags{
					model.TagServerHTTP:    "",
					model.TagServerJSONRPC: "",
					TagLogger:              "",
					TagMetrics:             "",
				},
				Methods: []*model.Method{
					rpcMethod("RpcPing"),
					httpMethod("HttpEcho"),
				},
			},
			{
				Name:    "Shapes",
				PkgPath: "example/contracts",
				ID:      "Shapes",
				Annotations: tags.DocTags{
					model.TagServerHTTP:    "",
					model.TagServerJSONRPC: "",
					TagLogger:              "",
					TagMetrics:             "",
				},
				Methods: []*model.Method{
					rpcMethod("EchoEntity"),
					httpMethod("PostEntity"),
				},
			},
			{
				Name:    "Modes",
				PkgPath: "example/contracts",
				ID:      "Modes",
				Annotations: tags.DocTags{
					model.TagServerHTTP: "",
					TagLogger:           "",
				},
				Methods: []*model.Method{
					httpMethod("BodyHeader"),
				},
			},
			{
				Name:    "Rpc",
				PkgPath: "example/contracts",
				ID:      "Rpc",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
					TagLogger:              "",
				},
				Methods: []*model.Method{
					rpcMethod("Ping"),
				},
			},
		},
	}
	return project
}

func transportServerLogOnlyProject() (project *model.Project) {

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
							{Name: "ctx", TypeID: "context:Context"},
						},
						Results: []*model.Variable{
							{Name: "out", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPMethod: "GET",
							model.TagHTTPPath:   "/ping",
						},
					},
				},
			},
		},
	}
	return project
}
