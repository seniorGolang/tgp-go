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

func TestRenderWebSocket_ClientStreamJoinsFeedIn(t *testing.T) {

	project := streamTestProject("Live", model.TagServerWS)
	project.Contracts[0].Methods[0] = &model.Method{
		Name: "Ingest",
		Annotations: tags.DocTags{
			model.TagStream: model.StreamModeClient,
		},
		Args: []*model.Variable{
			{Name: "ctx", TypeID: "context:Context"},
			{Name: "samples", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 1},
		},
		Results: []*model.Variable{
			{Name: "accepted", TypeID: "int"},
			{Name: "err", TypeID: "error"},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderWebSocket(); err != nil {
		t.Fatalf("RenderWebSocket: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-websocket.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{
		"feedCtx, feedCancel := context.WithCancel(ctx)",
		"feedErr := make(chan error, 1)",
		"stream.FeedIn[string](feedCtx, rawIn, samples)",
		"feedCancel()",
		"e := <-feedErr",
		"http.svc.Ingest(feedCtx, samples)",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("missing %q in:\n%s", fragment, source)
		}
	}
	if strings.Contains(source, "_ = stream.FeedIn") {
		t.Fatal("FeedIn errors must not be discarded")
	}
}

func TestRenderWebSocket_UniqueConnAndMixedResults(t *testing.T) {

	project := streamTestProject("Live", model.TagServerWS)
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderWebSocket(); err != nil {
		t.Fatalf("RenderWebSocket: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-websocket.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{
		"type wsLiveConn struct",
		"ticks, response.Count, err = http.svc.Subscribe",
		"stream.MarshalResult(response)",
		"stream.NewSession(",
		"http.srv.maxOpenStreams",
		"http.srv.maxWSMessageSize",
		"conn.SetReadLimit(",
		`conn.Locals("wsCtx")`,
		"ctx := sessionCtx",
		"writeTimeout",
		"http.srv.registerStream(session)",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("missing %q in:\n%s", fragment, source)
		}
	}
	if strings.Contains(source, "stream.KeyOverlay") {
		t.Fatal("WS without HTTP keys must not put overlay into context")
	}
	if strings.Contains(source, "overlay :=") {
		t.Fatal("WS without HTTP keys must not allocate overlay map")
	}
	if strings.Contains(source, "acquireWS()") {
		t.Fatal("WS connection limit must be acquired in HTTP upgrade middleware, not serveWS")
	}
	if !strings.Contains(source, "defer http.srv.releaseWS()") {
		t.Fatal("serveWS must release WS slot after fasthttp Hijack owns the connection")
	}
	if strings.Contains(source, "context.WithValue(context.Background(), stream.KeyOverlay") {
		t.Fatal("WS session must inherit upgrade UserContext, not Background alone")
	}
	if strings.Contains(source, "type wsConn struct") {
		t.Fatal("generic wsConn must not be generated")
	}
}

func TestRenderWebSocket_OverlayKeepsClone(t *testing.T) {

	project := streamTestProject("Live", model.TagServerWS)
	method := project.Contracts[0].Methods[0]
	method.Args = append(method.Args, &model.Variable{Name: "token", TypeID: "string"})
	method.Annotations[model.TagHTTPHeader] = "token|X-Token"
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderWebSocket(); err != nil {
		t.Fatalf("RenderWebSocket: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-websocket.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, `strings.Clone(conn.Headers("X-Token"))`) {
		t.Fatalf("WS overlay must Clone conn values (Immutable does not cover websocket.Conn), got:\n%s", source)
	}
	if !strings.Contains(source, "context.WithValue(sessionCtx, stream.KeyOverlay, overlay)") {
		t.Fatalf("WS with HTTP keys must put overlay into context:\n%s", source)
	}
}

func TestRenderWebSocket_SkipsEmptyOverlay(t *testing.T) {

	project := streamTestProject("Live", model.TagServerWS)
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderWebSocket(); err != nil {
		t.Fatalf("RenderWebSocket: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-websocket.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	if strings.Contains(source, "KeyOverlay") {
		t.Fatalf("empty WS overlay must not use KeyOverlay:\n%s", source)
	}
	if !strings.Contains(source, "ctx := sessionCtx") {
		t.Fatalf("empty WS overlay must reuse sessionCtx:\n%s", source)
	}
}

func TestRenderSSE_MixedResultsFinalMarshal(t *testing.T) {

	project := streamTestProject("Live", model.TagServerSSE)
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderSSE(); err != nil {
		t.Fatalf("RenderSSE: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-sse.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{
		"var release func()",
		"http.srv.acquireSSE()",
		"var once sync.Once",
		"once.Do(http.srv.releaseSSE)",
		"defer release()",
		"parentCtx := ftx.UserContext()",
		"var writeErr error",
		"streamCtx, cancel := context.WithCancel(parentCtx)",
		"defer cancel()",
		"conn := ftx.Context().Conn()",
		"conn.SetWriteDeadline(time.Now().Add(http.srv.streamWriteTimeout))",
		"stream.OpenSSE(writer, beforeWrite)",
		"stream.PumpSSEServerStreamTyped[string]",
		"stream.MarshalResult(response)",
		"stream.WriteSSEError(writer, requestBase.ID, writeErr, beforeWrite)",
		"stream.DefaultSSEHeartbeat",
		"effectiveSSEHeartbeat()",
		"X-Accel-Buffering",
		"ticks, response.Count, writeErr = http.svc.Subscribe(streamCtx",
		"return nil",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("missing %q in:\n%s", fragment, source)
		}
	}
	if strings.Contains(source, "held = false") || strings.Contains(source, "held := false") {
		t.Fatal("SSE must use once-release instead of held handoff")
	}
	if strings.Contains(source, "ticks, response.Count, err = http.svc.Subscribe") {
		t.Fatal("BodyStreamWriter must not assign named handler err")
	}
	if !strings.Contains(source, "stream.MarshalResult(response)") || !strings.Contains(source, "stream.WriteSSEError(writer, requestBase.ID, writeErr, beforeWrite)") {
		t.Fatal("MarshalResult error must WriteSSEError")
	}
	idxAcquire := strings.Index(source, "http.srv.acquireSSE()")
	idxBody := strings.Index(source, "decodeBodyReader(ftx)")
	if idxAcquire < 0 || idxBody < 0 || idxAcquire > idxBody {
		t.Fatal("acquireSSE must run before body parse")
	}
	idxOnce := strings.Index(source, "once.Do(http.srv.releaseSSE)")
	idxWriter := strings.Index(source, "SetBodyStreamWriter")
	if idxOnce < 0 || idxWriter < 0 || idxOnce > idxWriter {
		t.Fatal("once release must be prepared before SetBodyStreamWriter")
	}
	if idxConn := strings.Index(source, "conn.SetWriteDeadline(time.Now().Add(http.srv.streamWriteTimeout))"); idxConn < 0 {
		t.Fatal("missing SetWriteDeadline")
	} else if idxOpen := strings.Index(source, "stream.OpenSSE(writer, beforeWrite)"); idxOpen < idxConn {
		t.Fatal("rolling deadline callback must be declared before OpenSSE")
	}
	idxRelease := strings.Index(source, "if release != nil")
	idxCancel := strings.Index(source, "defer cancel()")
	if idxRelease < 0 {
		t.Fatal("missing release handoff inside writer")
	} else if idxCancel < idxRelease {
		t.Fatal("defer cancel must follow release defer so cancel runs first on exit")
	}
	if strings.Contains(source, "streamCtx := ftx.UserContext()") {
		t.Fatal("SSE must not use UserContext as stream lifetime without WithCancel")
	}
	if strings.Contains(source, "ticks, err = http.svc.Subscribe") {
		t.Fatal("SSE must not drop non-chan results")
	}
	if strings.Contains(source, `fmt.Fprintf(writer, "data: %s`) {
		t.Fatal("SSE must use stream runtime instead of inline fmt.Fprintf")
	}
	if strings.Contains(source, "go.opentelemetry.io/otel/trace") || strings.Contains(source, "SpanFromContext") {
		t.Fatal("SSE without @tg trace must not import or call otel/trace")
	}
}

func TestRenderSSE_EndsSpanWhenTraceEnabled(t *testing.T) {

	project := streamTestProject("Live", model.TagServerSSE)
	project.Contracts[0].Annotations[TagTrace] = ""
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderSSE(); err != nil {
		t.Fatalf("RenderSSE: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-sse.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, `"go.opentelemetry.io/otel/trace"`) {
		t.Fatalf("SSE with @tg trace must import otel/trace, got:\n%s", source)
	}
	if !strings.Contains(source, "trace.SpanFromContext(streamCtx).End()") {
		t.Fatalf("SSE with @tg trace must end span at stream exit, got:\n%s", source)
	}
	idxCancel := strings.Index(source, "defer cancel()")
	idxEnd := strings.Index(source, "trace.SpanFromContext(streamCtx).End()")
	if idxCancel < 0 || idxEnd < 0 || idxEnd < idxCancel {
		t.Fatal("SpanFromContext.End must follow defer cancel so End runs before cancel on exit")
	}
}

func TestRenderSSE_CancelsStreamContextInsideWriter(t *testing.T) {

	project := streamTestProject("Live", model.TagServerSSE)
	project.Contracts[0].Methods[0].Results = []*model.Variable{
		{Name: "ticks", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2},
		{Name: "err", TypeID: "error"},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderSSE(); err != nil {
		t.Fatalf("RenderSSE: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-sse.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{
		"parentCtx := ftx.UserContext()",
		"SetBodyStreamWriter(func(writer *bufio.Writer)",
		"var writeErr error",
		"streamCtx, cancel := context.WithCancel(parentCtx)",
		"defer cancel()",
		"ticks, writeErr = http.svc.Subscribe(streamCtx",
		"stream.PumpSSEServerStreamTyped[string](streamCtx,",
		"stream.EmptyResult()",
		"return nil",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("missing %q in:\n%s", fragment, source)
		}
	}
	idxWriter := strings.Index(source, "SetBodyStreamWriter(func(writer *bufio.Writer)")
	idxCancel := strings.Index(source, "streamCtx, cancel := context.WithCancel(parentCtx)")
	idxDefer := strings.Index(source, "defer cancel()")
	if idxWriter < 0 || idxCancel < idxWriter || idxDefer < idxCancel {
		t.Fatalf("WithCancel+defer cancel must be inside BodyStreamWriter, got:\n%s", source)
	}
}

func TestRenderSSE_HTTPContractWiresServerRef(t *testing.T) {

	project := streamTestProject("Live", model.TagServerSSE)
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderHTTP(); err != nil {
		t.Fatalf("RenderHTTP: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-http.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(content), "srv") || !strings.Contains(string(content), "*Server") {
		t.Fatalf("SSE contract http wrapper must expose srv *Server:\n%s", content)
	}
}

func TestRenderHTTP_WSAcquireBeforeUpgrade(t *testing.T) {

	project := streamTestProject("Live", model.TagServerWS)
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderHTTP(); err != nil {
		t.Fatalf("RenderHTTP: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-http.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{
		"http.srv.acquireWS()",
		"http.srv.releaseWS()",
		`fiber.NewError(fiber.StatusServiceUnavailable, "too many WS connections")`,
		`ftx.Locals("wsCtx", ftx.UserContext())`,
		"ftx.Next()",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("missing %q in:\n%s", fragment, source)
		}
	}
	if strings.Contains(source, "defer http.srv.releaseWS()") {
		t.Fatal("HTTP middleware must not defer releaseWS: fasthttp Hijack returns before serveWS ends")
	}
	idxAcquire := strings.Index(source, "http.srv.acquireWS()")
	idxLocals := strings.Index(source, `ftx.Locals("wsCtx"`)
	idxNext := strings.Index(source, "ftx.Next()")
	idxRelease := strings.Index(source, "http.srv.releaseWS()")
	if idxAcquire < 0 || idxLocals < 0 || idxNext < 0 || idxRelease < 0 {
		t.Fatal("missing acquire/Locals/Next/release markers")
	}
	if idxAcquire > idxLocals || idxLocals > idxNext {
		t.Fatalf("acquire must run before Locals and Next, got order acquire=%d locals=%d next=%d", idxAcquire, idxLocals, idxNext)
	}
	if idxRelease < idxNext {
		t.Fatal("middleware releaseWS must run only after Next error path")
	}
	if !strings.Contains(source, "!websocket.IsWebSocketUpgrade(ftx)") {
		t.Fatalf("non-upgrade requests must be rejected before acquire, got:\n%s", source)
	}
	idxUpgrade := strings.Index(source, "!websocket.IsWebSocketUpgrade(ftx)")
	if idxUpgrade > idxAcquire {
		t.Fatal("IsWebSocketUpgrade check must precede acquireWS")
	}
}

func TestRenderTransportOptions_SSEHeartbeat(t *testing.T) {

	project := streamTestProject("Live", model.TagServerSSE)
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewTransportRenderer(project, dir)
	if err := renderer.RenderTransportOptions(); err != nil {
		t.Fatalf("RenderTransportOptions: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "options.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{
		"func SetSSEHeartbeat(interval time.Duration) Option",
		"srv.sseHeartbeat = interval",
		"httpSvc.srv = srv",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("missing %q in:\n%s", fragment, source)
		}
	}
}

func streamTestProject(name string, transportTag string) (project *model.Project) {

	return &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    name,
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					transportTag:        "",
					model.TagHTTPPrefix: "api/v1",
					model.TagWSPath:     "/ws/live",
				},
				Methods: []*model.Method{
					{
						Name: "Subscribe",
						Annotations: tags.DocTags{
							model.TagStream:  model.StreamModeServer,
							model.TagSSEPath: "/sse/live/subscribe",
						},
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "symbol", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "ticks", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2},
							{Name: "count", TypeID: "int"},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
		},
	}
}
