// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package tsg

import (
	"strings"
	"testing"
)

func TestAsyncMethod_writesReadyReturnType(t *testing.T) {

	params := NewStatement()
	params.Params(func(pg *Group) {
		pg.Add(NewStatement().ID("topic").Add(TypeFromString(": string")))
		pg.Add(NewStatement().ID("signal").Optional().Add(TypeFromString(": AbortSignal")))
	})
	stmt := NewStatement()
	stmt.Public()
	stmt.AsyncMethod("ingest", params, NewStatement().ID("Promise").Generic("number"), func(mg *Group) {
		mg.Return(NewStatement().This().Dot("streamWebSocketResult").Generic("number").Call(
			NewStatement().ID("streamPath"),
			NewStatement().Lit("live.ingest"),
			NewStatement().ID("params"),
			NewStatement().ID("samples"),
			NewStatement().ID("query"),
			NewStatement().ID("signal"),
			NewStatement().ID("onReady"),
		))
	})
	got := stmt.String()
	wantPrefix := "public async ingest(topic: string, signal?: AbortSignal): Promise<number> {"
	if !strings.HasPrefix(strings.TrimLeft(got, " \t"), "public async ingest") {
		t.Fatalf("expected public async method, got:\n%s", got)
	}
	if !strings.Contains(got, wantPrefix) {
		t.Fatalf("expected signature %q in:\n%s", wantPrefix, got)
	}
	if !strings.Contains(got, "return this.streamWebSocketResult<number>(streamPath, \"live.ingest\", params, samples, query, signal, onReady);") {
		t.Fatalf("expected return call in:\n%s", got)
	}
	if strings.Contains(got, "Promise<Promise<") {
		t.Fatalf("must not double-wrap Promise:\n%s", got)
	}
}

func TestAsyncGeneratorMethod_forAwaitYield(t *testing.T) {

	params := NewStatement()
	params.Params(func(pg *Group) {
		pg.Add(NewStatement().ID("topic").Add(TypeFromString(": string")))
	})
	stmt := NewStatement()
	stmt.Public()
	stmt.AsyncGeneratorMethod("follow", params, NewStatement().ID("AsyncGenerator").Generic("string"), func(mg *Group) {
		mg.ForAwait("item", NewStatement().This().Dot("streamSSE").Generic("string").Call(
			NewStatement().ID("streamPath"),
			NewStatement().Lit("feed.follow"),
			NewStatement().ID("params"),
			NewStatement().ID("query"),
			NewStatement().ID("extraHeaders"),
			NewStatement().ID("signal"),
			NewStatement().ID("onReady"),
		), func(fg *Group) {
			fg.Yield(NewStatement().ID("item"))
		})
	})
	got := stmt.String()
	if !strings.Contains(got, "public async *follow(topic: string): AsyncGenerator<string> {") {
		t.Fatalf("expected async generator signature in:\n%s", got)
	}
	if !strings.Contains(got, "for await (const item of this.streamSSE<string>(streamPath, \"feed.follow\", params, query, extraHeaders, signal, onReady)) {") {
		t.Fatalf("expected for await in:\n%s", got)
	}
	if !strings.Contains(got, "yield item;") {
		t.Fatalf("expected yield in:\n%s", got)
	}
}

func TestAsyncMethod_nilParamsAndReturnType(t *testing.T) {

	stmt := NewStatement()
	stmt.AsyncMethod("noop", nil, nil, func(mg *Group) {
		mg.Return()
	})
	got := stmt.String()
	if !strings.Contains(got, "async noop() {") {
		t.Fatalf("expected empty params without return type:\n%s", got)
	}
	if strings.Contains(got, ": ") {
		t.Fatalf("nil returnType must omit colon:\n%s", got)
	}
}

func TestForAwait_andYield_onStatement(t *testing.T) {

	stmt := NewStatement()
	stmt.ForAwait("item", NewStatement().ID("source"), func(bg *Group) {
		bg.Add(NewStatement().Yield(NewStatement().ID("item")))
	})
	got := stmt.String()
	if !strings.Contains(got, "for await (const item of source) {") {
		t.Fatalf("expected for await:\n%s", got)
	}
	if !strings.Contains(got, "yield item;") {
		t.Fatalf("expected yield:\n%s", got)
	}
}
