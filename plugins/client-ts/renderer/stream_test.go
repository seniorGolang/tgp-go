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
					model.TagHttpPrefix: "api/v1",
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
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "symbol", TypeRef: model.TypeRef{TypeID: "string"}},
						},
						Results: []*model.Variable{
							{Name: "ticks", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
					{
						Name:        "Ingest",
						Annotations: tags.DocTags{model.TagStream: model.StreamModeClient},
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "batchID", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "samples", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 1}},
						},
						Results: []*model.Variable{
							{Name: "accepted", TypeRef: model.TypeRef{TypeID: "int"}},
							{Name: "rejected", TypeRef: model.TypeRef{TypeID: "int"}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
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
					model.TagHttpPrefix: "api/v1",
					model.TagWSPath:     "/ws/live-mapping/:room",
				},
				Methods: []*model.Method{
					{
						Name: "HeaderExplicit",
						Annotations: tags.DocTags{
							model.TagStream:     model.StreamModeServer,
							model.TagSSEPath:    "/sse/live-mapping/:room/header",
							model.TagHttpHeader: "token|X-Map-Token|explicit",
						},
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "room", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "token", TypeRef: model.TypeRef{TypeID: "string"}},
							{Name: "symbol", TypeRef: model.TypeRef{TypeID: "string"}},
						},
						Results: []*model.Variable{
							{Name: "ticks", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
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
	if err := renderer.RenderJsonRPCClientClass(project.Contracts[0]); err != nil {
		t.Fatalf("RenderJsonRPCClientClass Live: %v", err)
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
	if idx := strings.Index(liveSrc, "export class"); idx >= 0 {
		importBlock = liveSrc[:idx]
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
	if err = renderer.RenderJsonRPCClientClass(project.Contracts[1]); err != nil {
		t.Fatalf("RenderJsonRPCClientClass LiveMapping: %v", err)
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
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
						},
						Results: []*model.Variable{
							{Name: "ticks", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
				},
			},
		},
	}
	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderJsonRPCClientClass(project.Contracts[0]); err != nil {
		t.Fatalf("RenderJsonRPCClientClass: %v", err)
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
					model.TagHttpPrefix: "api/v1",
				},
				Methods: []*model.Method{
					{
						Name: "Follow",
						Annotations: tags.DocTags{
							model.TagStream:  model.StreamModeServer,
							model.TagSSEPath: "/sse/feed/follow",
						},
						Args: []*model.Variable{
							{Name: "ctx", TypeRef: model.TypeRef{TypeID: "context:Context"}},
							{Name: "topic", TypeRef: model.TypeRef{TypeID: "string"}},
						},
						Results: []*model.Variable{
							{Name: "events", TypeRef: model.TypeRef{ChanOf: &model.TypeRef{TypeID: "string"}, ChanDirection: 2}},
							{Name: "err", TypeRef: model.TypeRef{TypeID: "error"}},
						},
					},
				},
			},
		},
	}
	dir := t.TempDir()
	renderer := NewClientRenderer(project, dir, false, "", true)
	if err := renderer.RenderJsonRPCClientClass(project.Contracts[0]); err != nil {
		t.Fatalf("RenderJsonRPCClientClass: %v", err)
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
	if !strings.Contains(source, "public async *follow(topic: string, signal?: AbortSignal)") {
		t.Fatalf("follow must accept optional AbortSignal:\n%s", source)
	}
	if !strings.Contains(source, "this.streamSSE<string>(streamPath, \"feed.follow\", params, query, extraHeaders, signal)") {
		t.Fatalf("follow must forward signal into streamSSE:\n%s", source)
	}
	if strings.Contains(source, "signal?: AbortSignal, ") {
		t.Fatalf("signal must be the last parameter:\n%s", source)
	}
}

func assertStreamAbortContract(t *testing.T, source string, wantWS bool, wantSSE bool) {

	t.Helper()
	if !strings.Contains(source, "signal?: AbortSignal") {
		t.Fatalf("stream methods must accept signal?: AbortSignal:\n%s", source)
	}
	if wantWS {
		if !strings.Contains(source, "query?: Record<string, string>, signal?: AbortSignal") {
			t.Fatalf("WS helpers must accept signal:\n%s", source)
		}
		if !strings.Contains(source, "signal?.addEventListener(\"abort\"") {
			t.Fatalf("WS helpers must listen for abort:\n%s", source)
		}
	}
	if !wantSSE {
		return
	}
	if !strings.Contains(source, "extraHeaders?: Record<string, string>, signal?: AbortSignal") {
		t.Fatalf("streamSSE must accept signal:\n%s", source)
	}
	if !strings.Contains(source, "        signal,\n") {
		t.Fatalf("streamSSE fetch must pass signal:\n%s", source)
	}
	if !strings.Contains(source, "await reader.cancel()") {
		t.Fatalf("streamSSE must cancel reader in finally:\n%s", source)
	}
	if !strings.Contains(source, "finally {") {
		t.Fatalf("streamSSE must use finally for reader cleanup:\n%s", source)
	}
}
