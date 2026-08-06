// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package tsg

import (
	"strings"
	"testing"
)

func TestWhile_andTryFinally(t *testing.T) {

	stmt := NewStatement()
	stmt.While(NewStatement().Bang().ID("done"), func(wg *Group) {
		wg.Add(NewStatement().Continue())
	})
	got := stmt.String()
	if !strings.Contains(got, "while (!done) {") {
		t.Fatalf("expected while:\n%s", got)
	}
	if !strings.Contains(got, "continue;") {
		t.Fatalf("expected continue:\n%s", got)
	}

	stmt = NewStatement()
	stmt.TryFinally(func(tg *Group) {
		tg.Add(NewStatement().ID("work").Call().Semicolon())
	}, func(fg *Group) {
		fg.Add(NewStatement().ID("cleanup").Call().Semicolon())
	})
	got = stmt.String()
	if !strings.Contains(got, "try {") || !strings.Contains(got, "} finally {") {
		t.Fatalf("expected try/finally:\n%s", got)
	}
}

func TestTryCatch_emptyBinding(t *testing.T) {

	stmt := NewStatement()
	stmt.TryCatch(func(tg *Group) {
		tg.Add(NewStatement().Await(NewStatement().ID("reader").Dot("cancel").Call()).Semicolon())
	}, func(_ *Group) {}, "")
	got := stmt.String()
	if !strings.Contains(got, "} catch {") {
		t.Fatalf("expected empty catch binding:\n%s", got)
	}
	if strings.Contains(got, "catch (") {
		t.Fatalf("empty catchParam must omit binding:\n%s", got)
	}
}

func TestClassDecl_extendsImplements(t *testing.T) {

	stmt := NewStatement().Export()
	stmt.ClassDecl("ResponseError")
	stmt.Extends("Error")
	stmt.Implements("WithErrorCode")
	stmt.Block(func(g *Group) {
		g.Add(NewStatement().ID("statusCode").Colon().ID("number").Semicolon())
	})
	got := stmt.String()
	if !strings.Contains(got, "export class ResponseError extends Error implements WithErrorCode {") {
		t.Fatalf("expected class decl:\n%s", got)
	}
}

func TestElseIf_chainsAfterIf(t *testing.T) {

	stmt := NewStatement()
	stmt.Block(func(g *Group) {
		g.If(NewStatement().ID("a"), func(ig *Group) {
			ig.Return(NewStatement().Lit(1))
		})
		g.ElseIf(NewStatement().ID("b"), func(ig *Group) {
			ig.Return(NewStatement().Lit(2))
		})
		g.Else(func(ig *Group) {
			ig.Return(NewStatement().Lit(3))
		})
	})
	got := stmt.String()
	if !strings.Contains(got, "} else if (b) {") {
		t.Fatalf("expected else if:\n%s", got)
	}
	if !strings.Contains(got, "} else {") {
		t.Fatalf("expected else:\n%s", got)
	}
}

func TestAsyncMethodGeneric_insertsTypeParams(t *testing.T) {

	stmt := NewStatement()
	stmt.Private()
	stmt.AsyncMethodGeneric("streamWebSocketResult", "TResult", nil, NewStatement().ID("Promise").Generic("TResult"), func(mg *Group) {
		mg.Return()
	})
	got := stmt.String()
	if !strings.Contains(got, "private async streamWebSocketResult<TResult>(): Promise<TResult> {") {
		t.Fatalf("expected generic async method:\n%s", got)
	}
}

func TestVoidExpr_andBang(t *testing.T) {

	stmt := NewStatement().VoidExpr(NewStatement().Paren(NewStatement().ID("fn").Call()).Call())
	got := stmt.String()
	if !strings.Contains(got, "void (fn())();") {
		t.Fatalf("expected void expr:\n%s", got)
	}
	bang := NewStatement().ID("req").Bang().Call(NewStatement().Lit("node:os"))
	if bang.String() != `req!("node:os")` {
		t.Fatalf("expected non-null call, got %q", bang.String())
	}
}

func TestArrayLit(t *testing.T) {

	got := NewStatement().ArrayLit(
		NewStatement().Lit("a"),
		NewStatement().ID("encodeURIComponent").Call(NewStatement().ID("x")),
	).Dot("join").Call(NewStatement().Lit("; ")).String()
	want := `["a", encodeURIComponent(x)].join("; ")`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if NewStatement().ArrayLit().String() != "[]" {
		t.Fatalf("empty array lit")
	}
}
