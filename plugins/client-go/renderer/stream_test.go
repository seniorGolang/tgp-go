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

func TestRenderServiceClient_GeneratesWebSocketStream(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Live",
				PkgPath: "example/live",
				Annotations: tags.DocTags{
					model.TagServerWS: "",
				},
				Methods: []*model.Method{
					{
						Name:        "Subscribe",
						Annotations: tags.DocTags{model.TagStream: model.StreamModeServer},
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "symbol", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "ticks", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2},
							{Name: "err", TypeID: "error"},
						},
					},
					{
						Name:        "Ingest",
						Annotations: tags.DocTags{model.TagStream: model.StreamModeClient},
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "samples", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 1},
						},
						Results: []*model.Variable{
							{Name: "accepted", TypeID: "int"},
							{Name: "err", TypeID: "error"},
						},
					},
					{
						Name:        "Chat",
						Annotations: tags.DocTags{model.TagStream: model.StreamModeBidi},
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "in", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 1},
						},
						Results: []*model.Variable{
							{Name: "out", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, "example", "client")

	if err := renderer.RenderPathHelpers(); err != nil {
		t.Fatalf("RenderPathHelpers: %v", err)
	}
	if err := renderer.RenderStreamHelpers(); err != nil {
		t.Fatalf("RenderStreamHelpers: %v", err)
	}
	if err := renderer.RenderServiceClient(project.Contracts[0]); err != nil {
		t.Fatalf("RenderServiceClient: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-client.go"))
	if err != nil {
		t.Fatalf("read generated client: %v", err)
	}
	source := string(content)
	for _, fragment := range []string{"cli.webSocketDialer.DialContext", `$/stream`, "func (cli *ClientLive) Subscribe"} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("generated stream client does not contain %q:\n%s", fragment, source)
		}
	}
	if strings.Contains(source, "websocket.DefaultDialer.DialContext") {
		t.Fatalf("generated stream client must not dial through global default:\n%s", source)
	}
	for _, fragment := range []string{
		"var dialResp *http.Response",
		"conn, dialResp, err = cli.webSocketDialer.DialContext",
		"io.Copy(io.Discard, io.LimitReader(dialResp.Body, 1048576))",
		"dialResp.Body.Close()",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("WS Dial must drain and close handshake response on error, missing %q:\n%s", fragment, source)
		}
	}
	if strings.Contains(source, "conn, _, err = cli.webSocketDialer.DialContext") {
		t.Fatalf("WS Dial must not discard handshake response:\n%s", source)
	}
	for _, fragment := range []string{
		"case <-ctx.Done():",
		"case item, ok := <-samples:",
		"streamCtx, cancelStream := context.WithCancel(ctx)",
		"case item, ok := <-in:",
		"cancelStream()",
	} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("generated stream cancellation missing %q:\n%s", fragment, source)
		}
	}
	if _, err = os.Stat(filepath.Join(dir, "stream.go")); err != nil {
		t.Fatalf("expected package stream helpers: %v", err)
	}
	helpers, err := os.ReadFile(filepath.Join(dir, "stream.go"))
	if err != nil {
		t.Fatalf("read stream.go: %v", err)
	}
	helperSrc := string(helpers)
	if !strings.Contains(helperSrc, "func (cli *Client) httpURL(path string, query url.Values) (resolved string)") {
		t.Fatalf("stream helpers must define httpURL:\n%s", helperSrc)
	}
	if !strings.Contains(helperSrc, "parsed.Path = joinEndpointPath(parsed.Path, path)") {
		t.Fatalf("httpURL must join endpoint path with stream route:\n%s", helperSrc)
	}
	if !strings.Contains(helperSrc, "cli.httpURL(streamPath, query)") {
		t.Fatalf("wsURL must delegate path join to httpURL:\n%s", helperSrc)
	}
}

func TestRenderServiceClient_SkipsImplicitDialAndDropsSyncResults(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Live",
				PkgPath: "example/live",
				Annotations: tags.DocTags{
					model.TagServerWS:   "",
					model.TagServerSSE:  "",
					model.TagHTTPPrefix: "api/v1",
					model.TagWSPath:     "/ws/live",
				},
				Methods: []*model.Method{
					{
						Name: "Subscribe",
						Annotations: tags.DocTags{
							model.TagStream:     model.StreamModeServer,
							model.TagSSEPath:    "/sse/live/subscribe",
							model.TagHTTPHeader: "locale|X-Locale|implicit",
						},
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "locale", TypeID: "string"},
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
	dir := filepath.Join(t.TempDir(), "client")
	renderer := NewClientRenderer(project, dir, "example", "client")
	if err := renderer.RenderPathHelpers(); err != nil {
		t.Fatalf("RenderPathHelpers: %v", err)
	}
	if err := renderer.RenderStreamHelpers(); err != nil {
		t.Fatalf("RenderStreamHelpers: %v", err)
	}
	if err := renderer.RenderServiceClient(project.Contracts[0]); err != nil {
		t.Fatalf("RenderServiceClient: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "live-client.go"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, "func (cli *ClientLive) Subscribe(ctx context.Context, symbol string) (ticks <-chan string, err error)") {
		t.Fatalf("expected client signature without locale/count, got:\n%s", source)
	}
	if strings.Contains(source, "locale") {
		t.Fatalf("implicit locale must not appear in client stream method:\n%s", source)
	}
	if strings.Contains(source, "dialHeader.Set(\"X-Locale\"") {
		t.Fatal("implicit header must not be dialed")
	}
	if !strings.Contains(source, `streamPath := "/api/v1/sse/live/subscribe"`) {
		t.Fatalf("SSE path must keep http-prefix absolute route:\n%s", source)
	}
	if !strings.Contains(source, "cli.httpURL(streamPath, query)") {
		t.Fatalf("SSE must resolve URL via httpURL:\n%s", source)
	}
	if strings.Contains(source, "cli.endpoint+streamPath") || strings.Contains(source, "cli.endpoint + streamPath") {
		t.Fatalf("SSE must not concatenate endpoint with streamPath:\n%s", source)
	}
	if !strings.Contains(source, "scanner.Buffer(nil, cli.maxSSEEventSize)") || !strings.Contains(source, "scanner.Err()") {
		t.Fatalf("SSE scanner must enforce the configured limit and check errors:\n%s", source)
	}
	if err = renderer.RenderClient(); err != nil {
		t.Fatalf("RenderClient: %v", err)
	}
	if err = renderer.RenderClientOptions(); err != nil {
		t.Fatalf("RenderClientOptions: %v", err)
	}
	clientSource := mustRead(t, filepath.Join(dir, "client.go"))
	optionsSource := mustRead(t, filepath.Join(dir, "options.go"))
	for _, fragment := range []string{"defaultWebSocketDialer := *websocket.DefaultDialer", "maxSSEEventSize: 1048576"} {
		if !strings.Contains(clientSource, fragment) {
			t.Fatalf("client defaults missing %q:\n%s", fragment, clientSource)
		}
	}
	for _, fragment := range []string{"func WebSocketDialer(", "func MaxSSEEventSize(", "cli.webSocketDialer.TLSClientConfig = tlsConfig"} {
		if !strings.Contains(optionsSource, fragment) {
			t.Fatalf("client options missing %q:\n%s", fragment, optionsSource)
		}
	}
}
