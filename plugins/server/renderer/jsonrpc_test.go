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

func TestRenderJsonRPC_ServiceBatchUsesScopedMethodMap(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Rpc",
				PkgPath: "example/contracts",
				ID:      "Rpc",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
					model.TagHTTPPrefix:    "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Echo",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "msg", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "out", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")

	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderJSONRPC(); err != nil {
		t.Fatalf("RenderJSONRPC: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "rpc-jsonrpc.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)

	if !strings.Contains(source, `methods := http.srv.jsonRPCMethodMaps["/api/v1/rpc"]`) {
		t.Fatalf("expected scoped methods lookup for /api/v1/rpc, got:\n%s", source)
	}
	if strings.Contains(source, `jsonRPCMethodMaps["/"]`) {
		t.Fatalf("service serveBatch must not use root method map, got:\n%s", source)
	}
	if !strings.Contains(source, "http.srv.doSingleBatch(") {
		t.Fatalf("expected srv.doSingleBatch for single request, got:\n%s", source)
	}
	if !strings.Contains(source, "http.srv.doBatch(") {
		t.Fatalf("expected srv.doBatch for batch request, got:\n%s", source)
	}
	if !strings.Contains(source, "stream.DecodeEnvelope[baseJSONRPC](") {
		t.Fatalf("expected stream.DecodeEnvelope for batch body, got:\n%s", source)
	}
	if !strings.Contains(source, "stream.ErrBatchTooLarge") {
		t.Fatalf("expected ErrBatchTooLarge handling, got:\n%s", source)
	}
	if strings.Contains(source, "Decode(&requests)") {
		t.Fatalf("full batch Decode(&requests) must not be generated, got:\n%s", source)
	}
	if strings.Contains(source, "func (http *httpRpc) doSingleBatch(") {
		t.Fatalf("local doSingleBatch must be removed, got:\n%s", source)
	}
}

func jsonRPCOverlayTestProject() (project *model.Project) {

	return &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Abac",
				PkgPath: "example/contracts",
				ID:      "Abac",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
					model.TagHTTPPrefix:    "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "CheckAccess",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "userID", TypeID: "string"},
							{Name: "key", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "ok", TypeID: "bool"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPHeader:  "userID|X-User-Id",
							model.TagHTTPCookies: "key|x-supplier-id",
						},
					},
				},
			},
		},
	}
}

func TestRenderTransportJsonRPC_SnapshotsOverlayWithoutGetter(t *testing.T) {

	project := jsonRPCOverlayTestProject()
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
	if strings.Contains(source, "requestOverlayGetter") {
		t.Fatalf("getter must not be generated:\n%s", source)
	}
	if strings.Contains(source, "requestOverlayMiddleware") {
		t.Fatalf("overlay middleware must not be generated:\n%s", source)
	}
	if !strings.Contains(source, "func requestOverlayFromFiber(ftx *fiber.Ctx) (overlay requestOverlay)") {
		t.Fatalf("expected named overlay snapshot from Fiber:\n%s", source)
	}
	if !strings.Contains(source, `ftx.Get("X-User-Id")`) {
		t.Fatalf("expected Get for header overlay:\n%s", source)
	}
	if strings.Contains(source, `Header.Peek("X-User-Id")`) {
		t.Fatalf("header overlay must not use Peek:\n%s", source)
	}
	if strings.Contains(source, "strings.Clone") {
		t.Fatalf("overlay must not Clone under Immutable:\n%s", source)
	}
	if !strings.Contains(source, `Cookies("x-supplier-id")`) {
		t.Fatalf("expected Cookies for cookie overlay:\n%s", source)
	}
	if !strings.Contains(source, `ftx.SetUserContext(context.WithValue(ftx.UserContext(), keyRequestOverlay, requestOverlayFromFiber(ftx)))`) {
		t.Fatalf("serveBatch must snapshot overlay before workers:\n%s", source)
	}
	if strings.Count(source, "requestOverlayFromFiber(ftx)") != 1 {
		t.Fatalf("overlay must be snapshotted once at serveBatch, got %d:\n%s", strings.Count(source, "requestOverlayFromFiber(ftx)"), source)
	}
	doBatchStart := strings.Index(source, "func (srv *Server) doBatch(")
	if doBatchStart < 0 {
		t.Fatalf("expected doBatch:\n%s", source)
	}
	if strings.Contains(source[doBatchStart:], "requestOverlayFromFiber") {
		t.Fatalf("doBatch must not read Fiber headers:\n%s", source[doBatchStart:])
	}
	if !strings.Contains(source[doBatchStart:], `makeErrorResponseJsonRPC(requests[i].ID, invalidRequestError, "request context cancelled", nil)`) {
		t.Fatalf("doBatch must emit cancelled error for missing results:\n%s", source[doBatchStart:])
	}
	if !strings.Contains(source[doBatchStart:], "recover()") {
		t.Fatalf("doBatch workers must recover panics:\n%s", source[doBatchStart:])
	}
	for _, fragment := range []string{
		"resultsCh := make(chan batchResult, len(requests))",
		"for result := range resultsCh",
		"case <-batchCtx.Done():",
		"wg.Wait()",
		`makeErrorResponseJsonRPC(requests[idx].ID, internalError, "internal error", nil)`,
	} {
		if !strings.Contains(source[doBatchStart:], fragment) {
			t.Fatalf("doBatch must contain %q:\n%s", fragment, source[doBatchStart:])
		}
	}
	if strings.Contains(source[doBatchStart:], "remaining := expectedCount") {
		t.Fatal("doBatch must wait for workers instead of timeout collectLoop")
	}
	if !strings.Contains(source, "func sendJSONRPCResponse(") {
		t.Fatalf("expected sendJSONRPCResponse helper:\n%s", source)
	}
	if !strings.Contains(source, "sendJSONRPCResponse(ftx, requests[0].ID,") {
		t.Fatalf("single JSON-RPC must use sendJSONRPCResponse for notifications:\n%s", source)
	}
	if !strings.Contains(source, "stream.DecodeEnvelope[baseJSONRPC](") {
		t.Fatalf("expected stream.DecodeEnvelope in shared serveBatch:\n%s", source)
	}
	if strings.Contains(source, "Decode(&requests)") {
		t.Fatalf("full batch Decode(&requests) must not be generated:\n%s", source)
	}
	if strings.Contains(source, "readUntilFirstNonWhitespace") || strings.Contains(source, "maxPeekBytes") {
		t.Fatalf("peek helper must be removed from generated jsonrpc:\n%s", source)
	}
}

func TestRenderJsonRPC_ReadsOverlayStructFromContext(t *testing.T) {

	project := jsonRPCOverlayTestProject()
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderJSONRPC(); err != nil {
		t.Fatalf("RenderJSONRPC: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "abac-jsonrpc.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)
	if strings.Contains(source, "requestOverlayGetter") {
		t.Fatalf("handler must not use overlay getter:\n%s", source)
	}
	if !strings.Contains(source, "if overlay, ok := ctx.Value(keyRequestOverlay).(requestOverlay); ok") {
		t.Fatalf("handler must assert requestOverlay from context:\n%s", source)
	}
	if !strings.Contains(source, `overlay.Get("X-User-Id")`) {
		t.Fatalf("handler must read header from overlay:\n%s", source)
	}
	if !strings.Contains(source, `overlay.Get("x-supplier-id")`) {
		t.Fatalf("handler must read cookie from overlay:\n%s", source)
	}
	if strings.Count(source, "requestOverlayFromFiber(ftx)") != 2 {
		t.Fatalf("serveBatch and _serveMethod must snapshot overlay, got %d:\n%s", strings.Count(source, "requestOverlayFromFiber(ftx)"), source)
	}
	if !strings.Contains(source, "func (http *httpAbac) _serveMethod(") {
		t.Fatalf("expected _serveMethod:\n%s", source)
	}
	serveMethod := source[strings.Index(source, "func (http *httpAbac) _serveMethod("):] //nolint:gocritic // Index guarded by Contains above
	if !strings.Contains(serveMethod, "requestOverlayFromFiber(ftx)") {
		t.Fatalf("_serveMethod must snapshot overlay:\n%s", serveMethod)
	}
}

func TestRenderJsonRPC_SkipsOverlayAssertWithoutHeaderCookie(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Rpc",
				PkgPath: "example/contracts",
				ID:      "Rpc",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
					model.TagHTTPPrefix:    "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Echo",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "msg", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "out", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	renderer := NewContractRenderer(project, project.Contracts[0], dir)
	if err := renderer.RenderJSONRPC(); err != nil {
		t.Fatalf("RenderJSONRPC: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "rpc-jsonrpc.go"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	source := string(content)
	if strings.Contains(source, "ctx.Value(keyRequestOverlay)") {
		t.Fatalf("method without header/cookie must not read overlay:\n%s", source)
	}
	if strings.Contains(source, "requestOverlayFromFiber") {
		t.Fatalf("entries without overlay keys must not snapshot overlay:\n%s", source)
	}
	if strings.Contains(source, "keyRequestOverlay") {
		t.Fatalf("entries without overlay keys must not reference overlay key:\n%s", source)
	}
}

func TestRenderTransportJsonRPC_SkipsOverlayTypesWithoutKeys(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Rpc",
				PkgPath: "example/contracts",
				ID:      "Rpc",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
					model.TagHTTPPrefix:    "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Echo",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "msg", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "out", TypeID: "string"},
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
	for _, fragment := range []string{
		"type requestOverlay ",
		"keyRequestOverlay",
		"requestOverlayFromFiber",
		"SetUserContext(context.WithValue",
	} {
		if strings.Contains(source, fragment) {
			t.Fatalf("transport jsonrpc without overlay keys must not contain %q:\n%s", fragment, source)
		}
	}
}

func TestRenderJsonRPC_ContractWithoutKeysSkipsSnapshotWhenSiblingHasKeys(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Abac",
				PkgPath: "example/contracts",
				ID:      "Abac",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
					model.TagHTTPPrefix:    "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "CheckAccess",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "userID", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "ok", TypeID: "bool"},
							{Name: "err", TypeID: "error"},
						},
						Annotations: tags.DocTags{
							model.TagHTTPHeader: "userID|X-User-Id",
						},
					},
				},
			},
			{
				Name:    "Rpc",
				PkgPath: "example/contracts",
				ID:      "Rpc",
				Annotations: tags.DocTags{
					model.TagServerJSONRPC: "",
					model.TagHTTPPrefix:    "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Echo",
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "msg", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "out", TypeID: "string"},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
		},
	}
	dir := filepath.Join(t.TempDir(), "transport")
	transport := NewTransportRenderer(project, dir)
	if err := transport.RenderTransportJSONRPC(); err != nil {
		t.Fatalf("RenderTransportJSONRPC: %v", err)
	}
	rpc := NewContractRenderer(project, project.Contracts[1], dir)
	if err := rpc.RenderJSONRPC(); err != nil {
		t.Fatalf("RenderJSONRPC Rpc: %v", err)
	}
	abac := NewContractRenderer(project, project.Contracts[0], dir)
	if err := abac.RenderJSONRPC(); err != nil {
		t.Fatalf("RenderJSONRPC Abac: %v", err)
	}

	transportSrc, err := os.ReadFile(filepath.Join(dir, "jsonrpc.go"))
	if err != nil {
		t.Fatalf("read transport jsonrpc: %v", err)
	}
	if !strings.Contains(string(transportSrc), "requestOverlayFromFiber") {
		t.Fatalf("shared serveBatch must snapshot when any contract needs overlay:\n%s", transportSrc)
	}
	if !strings.Contains(string(transportSrc), "SetUserContext(context.WithValue") {
		t.Fatalf("shared serveBatch must WithValue overlay when keys exist:\n%s", transportSrc)
	}

	rpcSrc, err := os.ReadFile(filepath.Join(dir, "rpc-jsonrpc.go"))
	if err != nil {
		t.Fatalf("read rpc jsonrpc: %v", err)
	}
	if strings.Contains(string(rpcSrc), "requestOverlayFromFiber") {
		t.Fatalf("contract without keys must not snapshot overlay:\n%s", rpcSrc)
	}

	abacSrc, err := os.ReadFile(filepath.Join(dir, "abac-jsonrpc.go"))
	if err != nil {
		t.Fatalf("read abac jsonrpc: %v", err)
	}
	if strings.Count(string(abacSrc), "requestOverlayFromFiber(ftx)") != 2 {
		t.Fatalf("contract with keys must snapshot in serveBatch and _serveMethod, got %d:\n%s",
			strings.Count(string(abacSrc), "requestOverlayFromFiber(ftx)"), abacSrc)
	}
}
