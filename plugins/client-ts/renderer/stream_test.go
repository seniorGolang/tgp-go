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

func TestRenderJsonRPCClient_StreamImportsResponseAndSkipsUnusedHelper(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Version:    "1.0.0",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Live",
				PkgPath: "example/contracts",
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
							model.TagStream:  model.StreamModeServer,
							model.TagSSEPath: "/sse/live/subscribe",
						},
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
							{Name: "batchID", TypeID: "string"},
							{Name: "samples", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 1},
						},
						Results: []*model.Variable{
							{Name: "accepted", TypeID: "int"},
							{Name: "rejected", TypeID: "int"},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
			{
				Name:    "LiveMapping",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerWS:   "",
					model.TagServerSSE:  "",
					model.TagHTTPPrefix: "api/v1",
					model.TagWSPath:     "/ws/live-mapping/:room",
				},
				Methods: []*model.Method{
					{
						Name: "HeaderExplicit",
						Annotations: tags.DocTags{
							model.TagStream:     model.StreamModeServer,
							model.TagSSEPath:    "/sse/live-mapping/:room/header",
							model.TagHTTPHeader: "token|X-Map-Token|explicit",
						},
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "room", TypeID: "string"},
							{Name: "token", TypeID: "string"},
							{Name: "symbol", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "ticks", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
		},
	}

	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)

	if err := renderer.RenderExchangeTypes(project.Contracts[0]); err != nil {
		t.Fatalf("RenderExchangeTypes Live: %v", err)
	}
	if err := renderer.RenderJSONRPCClientClass(project.Contracts[0]); err != nil {
		t.Fatalf("RenderJSONRPCClientClass Live: %v", err)
	}
	live, err := os.ReadFile(filepath.Join(dir, "live.ts"))
	if err != nil {
		t.Fatalf("read live.ts: %v", err)
	}
	liveSrc := string(live)
	if !strings.Contains(liveSrc, "ResponseLiveIngest") {
		t.Fatalf("live.ts must reference ResponseLiveIngest:\n%s", liveSrc)
	}
	if !strings.Contains(liveSrc, "ResponseLiveIngest") || !strings.Contains(liveSrc, "from './live-exchange'") {
		t.Fatalf("live.ts must import ResponseLiveIngest from exchange:\n%s", liveSrc)
	}
	if !strings.Contains(liveSrc, "type ResponseLiveIngest") && !strings.Contains(liveSrc, "ResponseLiveIngest }") && !strings.Contains(liveSrc, "ResponseLiveIngest,") {
		// ImportType typically emits: import {type ResponseLiveIngest} from ...
		if !strings.Contains(liveSrc, "ResponseLiveIngest") || !strings.Contains(strings.ToLower(liveSrc), "import") {
			t.Fatalf("expected typed import of ResponseLiveIngest:\n%s", liveSrc)
		}
	}
	importBlock := liveSrc
	if before, _, ok := strings.Cut(liveSrc, "export class"); ok {
		importBlock = before
	}
	if !strings.Contains(importBlock, "ResponseLiveIngest") {
		t.Fatalf("ResponseLiveIngest must be imported, not only used:\n%s", importBlock)
	}
	if !strings.Contains(liveSrc, "streamWebSocketResult") {
		t.Fatalf("Live with client-stream must include streamWebSocketResult")
	}
	if !strings.Contains(liveSrc, "this.baseClient.decodeRPCError") {
		t.Fatalf("stream helpers must decode RPC errors via baseClient.decodeRPCError:\n%s", liveSrc)
	}
	if strings.Contains(liveSrc, "new Error(String(rpc.error.code)") {
		t.Fatalf("stream helpers must not invent Error from rpc.error fields:\n%s", liveSrc)
	}
	assertStreamAbortContract(t, liveSrc, true, true)

	if err = renderer.RenderExchangeTypes(project.Contracts[1]); err != nil {
		t.Fatalf("RenderExchangeTypes LiveMapping: %v", err)
	}
	if err = renderer.RenderJSONRPCClientClass(project.Contracts[1]); err != nil {
		t.Fatalf("RenderJSONRPCClientClass LiveMapping: %v", err)
	}
	mapping, err := os.ReadFile(filepath.Join(dir, "live-mapping.ts"))
	if err != nil {
		t.Fatalf("read live-mapping.ts: %v", err)
	}
	mappingSrc := string(mapping)
	if strings.Contains(mappingSrc, "streamWebSocketResult") {
		t.Fatalf("server-stream-only contract must not emit streamWebSocketResult:\n%s", mappingSrc)
	}
	if !strings.Contains(mappingSrc, "streamWebSocket") {
		t.Fatalf("server-stream WS must emit streamWebSocket")
	}
	if !strings.Contains(mappingSrc, "streamSSE") {
		t.Fatalf("SSE contract must emit streamSSE")
	}
	assertStreamAbortContract(t, mappingSrc, true, true)
}

func TestRenderJsonRPCClient_StreamServerOnlyOmitsResultHelper(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Version:    "1.0.0",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Ticks",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerWS: "",
					model.TagWSPath:   "/ws/ticks",
				},
				Methods: []*model.Method{
					{
						Name:        "Subscribe",
						Annotations: tags.DocTags{model.TagStream: model.StreamModeServer},
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
						},
						Results: []*model.Variable{
							{Name: "ticks", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
		},
	}
	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderJSONRPCClientClass(project.Contracts[0]); err != nil {
		t.Fatalf("RenderJSONRPCClientClass: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "ticks.ts"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	if strings.Contains(source, "streamWebSocketResult") {
		t.Fatal("WS server-only must not generate streamWebSocketResult")
	}
	if strings.Contains(source, "streamSSE") {
		t.Fatal("WS-only contract must not generate streamSSE")
	}
	if !strings.Contains(source, "streamWebSocket") {
		t.Fatal("expected streamWebSocket helper")
	}
	assertStreamAbortContract(t, source, true, false)
}

func TestRenderJsonRPCClient_StreamSSEAbortSignal(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Version:    "1.0.0",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{
			{
				Name:    "Feed",
				PkgPath: "example/contracts",
				Annotations: tags.DocTags{
					model.TagServerSSE:  "",
					model.TagHTTPPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Follow",
						Annotations: tags.DocTags{
							model.TagStream:  model.StreamModeServer,
							model.TagSSEPath: "/sse/feed/follow",
						},
						Args: []*model.Variable{
							{Name: "ctx", TypeID: "context:Context"},
							{Name: "topic", TypeID: "string"},
						},
						Results: []*model.Variable{
							{Name: "events", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2},
							{Name: "err", TypeID: "error"},
						},
					},
				},
			},
		},
	}
	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderJSONRPCClientClass(project.Contracts[0]); err != nil {
		t.Fatalf("RenderJSONRPCClientClass: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "feed.ts"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	source := string(content)
	if strings.Contains(source, "streamWebSocket") {
		t.Fatal("SSE-only contract must not generate streamWebSocket")
	}
	assertStreamAbortContract(t, source, false, true)
	if !strings.Contains(source, "public async *follow(topic: string, signal?: AbortSignal, onReady?: () => void)") {
		t.Fatalf("follow must accept optional AbortSignal and onReady:\n%s", source)
	}
	if !strings.Contains(source, "this.streamSSE<string>(streamPath, \"feed.follow\", params, query, extraHeaders, signal, onReady)") {
		t.Fatalf("follow must forward signal and onReady into streamSSE:\n%s", source)
	}
	if !strings.Contains(source, `const streamPath = "/api/v1/sse/feed/follow"`) {
		t.Fatalf("SSE path must keep http-prefix absolute route:\n%s", source)
	}
	if !strings.Contains(source, "this.baseClient.resolveURL(streamPath, query)") {
		t.Fatalf("streamSSE must resolve URL via baseClient.resolveURL:\n%s", source)
	}
	if strings.Contains(source, "getEndpoint().replace") || strings.Contains(source, `getEndpoint() +`) {
		t.Fatalf("streamSSE must not concatenate endpoint with path:\n%s", source)
	}
	if !strings.Contains(source, "onReady?.()") {
		t.Fatalf("streamSSE must invoke onReady after successful fetch:\n%s", source)
	}
	if idxFail, idxReady := strings.Index(source, "SSE request failed"), strings.Index(source, "onReady?.()"); idxFail < 0 || idxReady < 0 || idxReady < idxFail {
		t.Fatalf("onReady must be called only after successful SSE response check:\n%s", source)
	}
}

func TestStreamTSPathExpr_replacesPathParams(t *testing.T) {

	project := &model.Project{
		ModulePath: "example",
		Version:    "1.0.0",
		Types:      map[string]*model.Type{},
		Contracts: []*model.Contract{{
			Name:    "LiveMapping",
			PkgPath: "example/contracts",
			Annotations: tags.DocTags{
				model.TagServerWS:   "",
				model.TagHTTPPrefix: "api/v1",
				model.TagWSPath:     "/ws/live-mapping/:room",
			},
			Methods: []*model.Method{{
				Name: "HeaderExplicit",
				Annotations: tags.DocTags{
					model.TagStream:     model.StreamModeServer,
					model.TagHTTPHeader: "token|X-Map-Token|explicit",
				},
				Args: []*model.Variable{
					{Name: "ctx", TypeID: "context:Context"},
					{Name: "room", TypeID: "string"},
					{Name: "token", TypeID: "string"},
				},
				Results: []*model.Variable{
					{Name: "ticks", ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2},
					{Name: "err", TypeID: "error"},
				},
			}},
		}},
	}
	renderer := NewClientRenderer(project, t.TempDir(), false, "", true)
	contract := project.Contracts[0]
	method := contract.Methods[0]
	pathExpr := renderer.streamTSPathExpr(contract, method, model.ContractWSPath(project, contract)).String()
	if !strings.Contains(pathExpr, `"/ws/live-mapping/:room"`) && !strings.Contains(pathExpr, "live-mapping") {
		t.Fatalf("path must start from contract ws path, got %s", pathExpr)
	}
	if !strings.Contains(pathExpr, `.split(":room").join(encodeURIComponent(String(room)))`) {
		t.Fatalf("path must substitute :room, got %s", pathExpr)
	}
	queryExpr := renderer.streamTSQueryExpr(contract, method, true).String()
	if !strings.Contains(queryExpr, "X-Map-Token") || !strings.Contains(queryExpr, "String(token)") {
		t.Fatalf("ws query must mirror explicit header, got %s", queryExpr)
	}
	if renderer.streamTSQueryExpr(contract, method, false).String() != "undefined" {
		t.Fatalf("sse query without query args must be undefined")
	}
}

func assertStreamAbortContract(t *testing.T, source string, wantWS bool, wantSSE bool) {

	t.Helper()
	if !strings.Contains(source, "signal?: AbortSignal, onReady?: () => void") {
		t.Fatalf("stream methods must accept signal?: AbortSignal, onReady?: () => void:\n%s", source)
	}
	if wantWS {
		if !strings.Contains(source, "signal?: AbortSignal, onReady?: () => void): AsyncGenerator") &&
			!strings.Contains(source, "signal?: AbortSignal, onReady?: () => void): Promise") {
			t.Fatalf("WS helpers must accept signal and onReady:\n%s", source)
		}
		if !strings.Contains(source, "signal?.addEventListener(\"abort\"") {
			t.Fatalf("WS helpers must listen for abort:\n%s", source)
		}
		if !strings.Contains(source, "this.baseClient.resolveURL(streamPath, query)") {
			t.Fatalf("websocketURL must resolve via baseClient.resolveURL:\n%s", source)
		}
		openIdx := strings.Index(source, "const socket = await this.openWebSocket(streamPath, query, signal);")
		readyIdx := strings.Index(source, "onReady?.()")
		if openIdx < 0 || readyIdx < 0 || readyIdx < openIdx {
			t.Fatalf("WS helpers must call onReady immediately after open:\n%s", source)
		}
		if !strings.Contains(source, `addEventListener("close"`) {
			t.Fatalf("WS helpers must listen for socket close:\n%s", source)
		}
		if !strings.Contains(source, "messages.length > 0 || closed") && !strings.Contains(source, "messages.length>0||closed") {
			t.Fatalf("streamWebSocket wait-loop must re-check queue after registering wake:\n%s", source)
		}
		for _, fragment := range []string{
			"const maxBufferedMessages = 32",
			"messages.length >= maxBufferedMessages",
			"overflowed = true",
			"WebSocket incoming buffer overflow",
		} {
			if !strings.Contains(source, fragment) {
				t.Fatalf("streamWebSocket must bound incoming buffer, missing %q:\n%s", fragment, source)
			}
		}
		if !strings.Contains(source, `"WebSocket closed"`) && !strings.Contains(source, "reject(new Error(\"WebSocket closed\"))") {
			t.Fatalf("WS helpers must fail when socket closes:\n%s", source)
		}
		if strings.Contains(source, "streamWebSocketResult") {
			if !strings.Contains(source, "var settled = false") && !strings.Contains(source, "let settled = false") {
				t.Fatalf("streamWebSocketResult must guard Promise settle:\n%s", source)
			}
			if !strings.Contains(source, `removeEventListener("close"`) {
				t.Fatalf("streamWebSocketResult must detach close listener before settle:\n%s", source)
			}
		}
	}
	if !wantSSE {
		return
	}
	if !strings.Contains(source, "signal?: AbortSignal, onReady?: () => void): AsyncGenerator") {
		t.Fatalf("streamSSE must accept signal and onReady:\n%s", source)
	}
	if !strings.Contains(source, "this.baseClient.resolveURL(streamPath, query)") {
		t.Fatalf("streamSSE must resolve via baseClient.resolveURL:\n%s", source)
	}
	if !strings.Contains(source, "signal:signal") && !strings.Contains(source, "signal: signal") && !strings.Contains(source, "signal,\n") {
		t.Fatalf("streamSSE fetch must pass signal:\n%s", source)
	}
	if !strings.Contains(source, "onReady?.()") {
		t.Fatalf("streamSSE must call onReady after OK:\n%s", source)
	}
	if !strings.Contains(source, "await reader.cancel()") {
		t.Fatalf("streamSSE must cancel reader in finally:\n%s", source)
	}
	if !strings.Contains(source, "await response.body?.cancel()") {
		t.Fatalf("streamSSE must cancel a non-OK response body:\n%s", source)
	}
	if !strings.Contains(source, "finally {") {
		t.Fatalf("streamSSE must use finally for reader cleanup:\n%s", source)
	}
}
