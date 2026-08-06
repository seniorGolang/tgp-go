// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import "tgp/plugins/client-ts/tsg"

func renderStreamHelperWebSocketURL() (stmt *tsg.Statement) {

	params := tsg.NewStatement().Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("streamPath").Add(tsg.TypeFromString(": string")))
		pg.Add(tsg.NewStatement().ID("query").Optional().Add(tsg.TypeFromString(": Record<string, string>")))
	})
	stmt = tsg.NewStatement()
	stmt.Private()
	stmt.MethodTyped("websocketURL", params, tsg.NewStatement().ID(typeString), func(mg *tsg.Group) {
		mg.Add(tsg.NewStatement().Const("url").Op("=").New("URL").Call(
			tsg.NewStatement().This().Dot("baseClient").Dot("resolveURL").Call(tsg.NewStatement().ID("streamPath"), tsg.NewStatement().ID("query")),
		).Semicolon())
		mg.Add(tsg.NewStatement().ID("url").Dot("protocol").Op("=").ID("url").Dot("protocol").Op("===").Lit("https:").Op("?").Lit("wss:").Op(":").Lit("ws:").Semicolon())
		mg.Return(tsg.NewStatement().ID("url").Dot("toString").Call())
	})
	return stmt
}

func renderStreamHelperOpenWebSocket() (stmt *tsg.Statement) {

	params := tsg.NewStatement().Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("streamPath").Add(tsg.TypeFromString(": string")))
		pg.Add(tsg.NewStatement().ID("query").Optional().Add(tsg.TypeFromString(": Record<string, string>")))
		pg.Add(tsg.NewStatement().ID("signal").Optional().Add(tsg.TypeFromString(": AbortSignal")))
	})
	executor := tsg.NewStatement()
	executor.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("resolve"))
		pg.Add(tsg.NewStatement().ID("reject"))
	})
	executor.Op("=>")
	executor.Block(func(bg *tsg.Group) {
		bg.If(tsg.NewStatement().ID("signal").OptionalChain("aborted"), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().ID("reject").Call(tsg.NewStatement().ID("signal").Dot("reason").Op("instanceof").ID("Error").Op("?").ID("signal").Dot("reason").Op(":").New("DOMException").Call(
				tsg.NewStatement().Lit("This operation was aborted"), tsg.NewStatement().Lit("AbortError"),
			)).Semicolon())
			ig.Add(tsg.NewStatement().Return())
		})
		bg.Add(tsg.NewStatement().Const("socket").Op("=").New("WebSocket").Call(
			tsg.NewStatement().This().Dot("websocketURL").Call(tsg.NewStatement().ID("streamPath"), tsg.NewStatement().ID("query")),
		).Semicolon())
		bg.Add(tsg.NewStatement().Const("onAbort").Op("=").ArrowFunc().Block(func(ag *tsg.Group) {
			ag.Add(tsg.NewStatement().ID("socket").Dot("close").Call().Semicolon())
			ag.Add(tsg.NewStatement().ID("reject").Call(tsg.NewStatement().ID("signal").OptionalChain("reason").Op("instanceof").ID("Error").Op("?").ID("signal").Dot("reason").Op(":").New("DOMException").Call(
				tsg.NewStatement().Lit("This operation was aborted"), tsg.NewStatement().Lit("AbortError"),
			)).Semicolon())
		}).Semicolon())
		bg.Add(signalAbortListener())
		bg.Add(tsg.NewStatement().ID("socket").Dot("addEventListener").Call(
			tsg.NewStatement().Lit("open"),
			tsg.NewStatement().ArrowFunc().Block(func(og *tsg.Group) {
				og.Add(tsg.NewStatement().ID("signal").OptionalChain("removeEventListener").Call(tsg.NewStatement().Lit("abort"), tsg.NewStatement().ID("onAbort")).Semicolon())
				og.Add(tsg.NewStatement().ID("resolve").Call(tsg.NewStatement().ID("socket")).Semicolon())
			}),
			tsg.NewStatement().ObjectLiteral(func(og *tsg.Group) { og.Add(tsg.NewStatement().ID("once").Colon().Lit(true)) }),
		).Semicolon())
		bg.Add(tsg.NewStatement().ID("socket").Dot("addEventListener").Call(
			tsg.NewStatement().Lit("error"),
			tsg.NewStatement().ArrowFunc().Block(func(eg *tsg.Group) {
				eg.Add(tsg.NewStatement().ID("signal").OptionalChain("removeEventListener").Call(tsg.NewStatement().Lit("abort"), tsg.NewStatement().ID("onAbort")).Semicolon())
				eg.Add(tsg.NewStatement().ID("reject").Call(tsg.NewStatement().New("Error").Call(tsg.NewStatement().Lit("WebSocket connection failed"))).Semicolon())
			}),
			tsg.NewStatement().ObjectLiteral(func(og *tsg.Group) { og.Add(tsg.NewStatement().ID("once").Colon().Lit(true)) }),
		).Semicolon())
	})
	stmt = tsg.NewStatement()
	stmt.Private()
	stmt.MethodTyped("openWebSocket", params, tsg.NewStatement().ID("Promise").Generic("WebSocket"), func(mg *tsg.Group) {
		mg.Return(tsg.NewStatement().New("Promise").Call(executor))
	})
	return stmt
}

func renderStreamHelperWebSocket() (stmt *tsg.Statement) {

	params := streamHelperCommonParams(true)
	stmt = tsg.NewStatement()
	stmt.Private()
	stmt.AsyncGeneratorMethodGeneric("streamWebSocket", "T", params, tsg.NewStatement().ID("AsyncGenerator").Generic("T"), func(mg *tsg.Group) {
		addStreamAbortGuard(mg)
		mg.Add(tsg.NewStatement().Const("socket").Op("=").Await(tsg.NewStatement().This().Dot("openWebSocket").Call(
			tsg.NewStatement().ID("streamPath"), tsg.NewStatement().ID("query"), tsg.NewStatement().ID("signal"),
		)).Semicolon())
		mg.Add(tsg.NewStatement().ID("onReady").OptionalChain("").Call().Semicolon())
		mg.Add(tsg.NewStatement().Const("id").Op("=").ID("crypto").Dot("randomUUID").Call().Semicolon())
		mg.Add(tsg.NewStatement().Const("maxBufferedMessages").Op("=").Lit(32).Semicolon())
		mg.Add(tsg.NewStatement().Const("messages").Colon().ID("unknown").Index(nil).Op("=").Index(nil).Semicolon())
		mg.Add(tsg.NewStatement().Var("done").Op("=").Lit(false).Semicolon())
		mg.Add(tsg.NewStatement().Var("closed").Op("=").Lit(false).Semicolon())
		mg.Add(tsg.NewStatement().Var("overflowed").Op("=").Lit(false).Semicolon())
		mg.Add(tsg.NewStatement().Var("wake").Colon().Add(tsg.TypeFromString("(() => void) | undefined")).Semicolon())
		mg.Add(tsg.NewStatement().Const("onAbort").Op("=").ArrowFunc().Block(func(bg *tsg.Group) {
			bg.Add(tsg.NewStatement().ID("socket").Dot("close").Call().Semicolon())
			bg.Add(tsg.NewStatement().ID("wake").OptionalChain("").Call().Semicolon())
		}).Semicolon())
		mg.Add(signalAbortListener())
		mg.Add(tsg.NewStatement().Const("onSocketClosed").Op("=").ArrowFunc().Block(func(bg *tsg.Group) {
			bg.Add(tsg.NewStatement().ID("closed").Op("=").Lit(true).Semicolon())
			bg.Add(tsg.NewStatement().ID("wake").OptionalChain("").Call().Semicolon())
		}).Semicolon())
		mg.Add(tsg.NewStatement().ID("socket").Dot("addEventListener").Call(
			tsg.NewStatement().Lit("close"), tsg.NewStatement().ID("onSocketClosed"),
		).Semicolon())
		mg.Add(tsg.NewStatement().ID("socket").Dot("addEventListener").Call(
			tsg.NewStatement().Lit("error"), tsg.NewStatement().ID("onSocketClosed"),
		).Semicolon())
		mg.Add(tsg.NewStatement().ID("socket").Dot("addEventListener").Call(
			tsg.NewStatement().Lit("message"),
			tsg.NewStatement().Params(func(pg *tsg.Group) { pg.Add(tsg.NewStatement().ID("event")) }).Op("=>").Block(func(bg *tsg.Group) {
				bg.If(tsg.NewStatement().ID("messages").Dot("length").Op(">=").ID("maxBufferedMessages"), func(ig *tsg.Group) {
					ig.Add(tsg.NewStatement().ID("overflowed").Op("=").Lit(true).Semicolon())
					ig.Add(tsg.NewStatement().ID("socket").Dot("close").Call().Semicolon())
					ig.Add(tsg.NewStatement().ID("wake").OptionalChain("").Call().Semicolon())
					ig.Add(tsg.NewStatement().Return())
				})
				bg.Add(tsg.NewStatement().ID("messages").Dot("push").Call(
					tsg.NewStatement().ID("JSON").Dot("parse").Call(tsg.NewStatement().ID("String").Call(tsg.NewStatement().ID("event").Dot("data"))),
				).Semicolon())
				bg.Add(tsg.NewStatement().ID("wake").OptionalChain("").Call().Semicolon())
			}),
		).Semicolon())
		mg.Add(socketSendJSONRPC())
		mg.If(tsg.NewStatement().ID("input"), func(ig *tsg.Group) {
			ig.Add(voidAsyncInputPump(true))
		})
		mg.Add(tsg.NewStatement().TryFinally(func(tg *tsg.Group) {
			tg.While(tsg.NewStatement().Bang().ID("done"), func(wg *tsg.Group) {
				addStreamAbortGuard(wg)
				wg.If(tsg.NewStatement().ID("messages").Dot("length").Op("===").Lit(0), func(ig *tsg.Group) {
					ig.If(tsg.NewStatement().ID("closed"), func(cg *tsg.Group) {
						cg.Throw(tsg.NewStatement().New("Error").Call(
							tsg.NewStatement().ID("overflowed").Op("?").Lit("WebSocket incoming buffer overflow").Op(":").Lit("WebSocket closed"),
						))
					})
					ig.Add(tsg.NewStatement().Await(tsg.NewStatement().New("Promise").Generic("void").Call(
						tsg.NewStatement().Params(func(pg *tsg.Group) { pg.Add(tsg.NewStatement().ID("resolve")) }).Op("=>").Block(func(bg *tsg.Group) {
							bg.Add(tsg.NewStatement().ID("wake").Op("=").ID("resolve").Semicolon())
							bg.If(tsg.NewStatement().ID("messages").Dot("length").Op(">").Lit(0).Op("||").ID("closed"), func(rg *tsg.Group) {
								rg.Add(tsg.NewStatement().ID("resolve").Call().Semicolon())
							})
						}),
					)).Semicolon())
					ig.Add(tsg.NewStatement().ID("wake").Op("=").ID(typeUndefined).Semicolon())
					ig.Add(tsg.NewStatement().Continue())
				})
				wg.Add(tsg.NewStatement().ForOf("message", "messages.splice(0)", func(fg *tsg.Group) {
					fg.Add(tsg.NewStatement().Const("rpc").Op("=").ID("message").Op("as").Add(tsg.TypeFromString("{ id?: string; method?: string; params?: { id?: string; item?: T }; error?: { code: number; message: string } }")).Semicolon())
					fg.If(tsg.NewStatement().ID("rpc").Dot("method").Op("===").Lit("$/stream").Op("&&").ID("rpc").Dot("params").OptionalChain("id").Op("===").ID("id"), func(ig *tsg.Group) {
						ig.Add(tsg.NewStatement().Yield(tsg.NewStatement().ID("rpc").Dot("params").Dot("item").Op("as").ID("T")))
						ig.Add(tsg.NewStatement().Continue())
					})
					fg.If(tsg.NewStatement().ID("rpc").Dot("id").Op("===").ID("id"), func(ig *tsg.Group) {
						ig.If(tsg.NewStatement().ID("rpc").Dot(typeError), func(eg *tsg.Group) {
							eg.Throw(tsg.NewStatement().This().Dot("baseClient").Dot("decodeRPCError").Call(tsg.NewStatement().ID("rpc").Dot(typeError)))
						})
						ig.Add(tsg.NewStatement().ID("done").Op("=").Lit(true).Semicolon())
					})
				}))
			})
		}, func(fg *tsg.Group) {
			fg.Add(tsg.NewStatement().ID("signal").OptionalChain("removeEventListener").Call(tsg.NewStatement().Lit("abort"), tsg.NewStatement().ID("onAbort")).Semicolon())
			fg.Add(tsg.NewStatement().ID("socket").Dot("close").Call().Semicolon())
		}))
	})
	return stmt
}

func renderStreamHelperWebSocketResult() (stmt *tsg.Statement) {

	params := streamHelperCommonParams(false)
	stmt = tsg.NewStatement()
	stmt.Private()
	stmt.AsyncMethodGeneric("streamWebSocketResult", "TResult", params, tsg.NewStatement().ID("Promise").Generic("TResult"), func(mg *tsg.Group) {
		addStreamAbortGuard(mg)
		mg.Add(tsg.NewStatement().Const("socket").Op("=").Await(tsg.NewStatement().This().Dot("openWebSocket").Call(
			tsg.NewStatement().ID("streamPath"), tsg.NewStatement().ID("query"), tsg.NewStatement().ID("signal"),
		)).Semicolon())
		mg.Add(tsg.NewStatement().ID("onReady").OptionalChain("").Call().Semicolon())
		mg.Add(tsg.NewStatement().Const("id").Op("=").ID("crypto").Dot("randomUUID").Call().Semicolon())
		executor := tsg.NewStatement()
		executor.Params(func(pg *tsg.Group) {
			pg.Add(tsg.NewStatement().ID("resolve"))
			pg.Add(tsg.NewStatement().ID("reject"))
		})
		executor.Op("=>")
		executor.Block(func(bg *tsg.Group) {
			bg.Add(tsg.NewStatement().Var("settled").Op("=").Lit(false).Semicolon())
			bg.Add(tsg.NewStatement().Var("onAbort").Colon().Add(tsg.TypeFromString("(() => void) | undefined")).Semicolon())
			bg.Add(tsg.NewStatement().Var("failClosed").Colon().Add(tsg.TypeFromString("(() => void) | undefined")).Semicolon())
			bg.Add(tsg.NewStatement().Const("settle").Op("=").Params(func(pg *tsg.Group) {
				pg.Add(tsg.NewStatement().ID("action").Add(tsg.TypeFromString(": () => void")))
			}).Op("=>").Block(func(sg *tsg.Group) {
				sg.If(tsg.NewStatement().ID("settled"), func(ig *tsg.Group) {
					ig.Add(tsg.NewStatement().Return())
				})
				sg.Add(tsg.NewStatement().ID("settled").Op("=").Lit(true).Semicolon())
				sg.Add(tsg.NewStatement().ID("signal").OptionalChain("removeEventListener").Call(tsg.NewStatement().Lit("abort"), tsg.NewStatement().ID("onAbort")).Semicolon())
				sg.Add(tsg.NewStatement().ID("socket").Dot("removeEventListener").Call(tsg.NewStatement().Lit("close"), tsg.NewStatement().ID("failClosed")).Semicolon())
				sg.Add(tsg.NewStatement().ID("socket").Dot("removeEventListener").Call(tsg.NewStatement().Lit("error"), tsg.NewStatement().ID("failClosed")).Semicolon())
				sg.Add(tsg.NewStatement().ID("action").Call().Semicolon())
			}).Semicolon())
			bg.Add(tsg.NewStatement().ID("onAbort").Op("=").ArrowFunc().Block(func(ag *tsg.Group) {
				ag.Add(tsg.NewStatement().ID("settle").Call(tsg.NewStatement().ArrowFunc().Block(func(sg *tsg.Group) {
					sg.Add(tsg.NewStatement().ID("socket").Dot("close").Call().Semicolon())
					sg.Add(tsg.NewStatement().ID("reject").Call(tsg.NewStatement().ID("signal").OptionalChain("reason").Op("instanceof").ID("Error").Op("?").ID("signal").Dot("reason").Op(":").New("DOMException").Call(
						tsg.NewStatement().Lit("This operation was aborted"), tsg.NewStatement().Lit("AbortError"),
					)).Semicolon())
				})).Semicolon())
			}).Semicolon())
			bg.Add(signalAbortListener())
			bg.Add(tsg.NewStatement().ID("failClosed").Op("=").ArrowFunc().Block(func(ag *tsg.Group) {
				ag.Add(tsg.NewStatement().ID("settle").Call(tsg.NewStatement().ArrowFunc().Block(func(sg *tsg.Group) {
					sg.Add(tsg.NewStatement().ID("reject").Call(tsg.NewStatement().New("Error").Call(tsg.NewStatement().Lit("WebSocket closed"))).Semicolon())
				})).Semicolon())
			}).Semicolon())
			bg.Add(tsg.NewStatement().ID("socket").Dot("addEventListener").Call(
				tsg.NewStatement().Lit("close"), tsg.NewStatement().ID("failClosed"),
			).Semicolon())
			bg.Add(tsg.NewStatement().ID("socket").Dot("addEventListener").Call(
				tsg.NewStatement().Lit("error"), tsg.NewStatement().ID("failClosed"),
			).Semicolon())
			bg.Add(tsg.NewStatement().ID("socket").Dot("addEventListener").Call(
				tsg.NewStatement().Lit("message"),
				tsg.NewStatement().Params(func(pg *tsg.Group) { pg.Add(tsg.NewStatement().ID("event")) }).Op("=>").Block(func(mg2 *tsg.Group) {
					mg2.Add(tsg.NewStatement().Const("rpc").Op("=").ID("JSON").Dot("parse").Call(tsg.NewStatement().ID("String").Call(tsg.NewStatement().ID("event").Dot("data"))).Op("as").Add(tsg.TypeFromString("{ id?: string; result?: TResult; error?: { code: number; message: string } }")).Semicolon())
					mg2.If(tsg.NewStatement().ID("rpc").Dot("id").Op("!==").ID("id"), func(ig *tsg.Group) { ig.Return() })
					mg2.Add(tsg.NewStatement().ID("settle").Call(tsg.NewStatement().ArrowFunc().Block(func(sg *tsg.Group) {
						sg.Add(tsg.NewStatement().ID("socket").Dot("close").Call().Semicolon())
						sg.If(tsg.NewStatement().ID("rpc").Dot(typeError), func(ig *tsg.Group) {
							ig.Add(tsg.NewStatement().ID("reject").Call(tsg.NewStatement().This().Dot("baseClient").Dot("decodeRPCError").Call(tsg.NewStatement().ID("rpc").Dot(typeError))).Semicolon())
							ig.Return()
						})
						sg.Add(tsg.NewStatement().ID("resolve").Call(tsg.NewStatement().ID("rpc").Dot(nameResult).Op("as").ID("TResult")).Semicolon())
					})).Semicolon())
				}),
			).Semicolon())
			bg.Add(socketSendJSONRPC())
			bg.Add(voidAsyncInputPump(false))
		})
		mg.Return(tsg.NewStatement().New("Promise").Generic("TResult").Call(executor))
	})
	return stmt
}

func renderStreamHelperSSE() (stmt *tsg.Statement) {

	params := tsg.NewStatement().Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("streamPath").Add(tsg.TypeFromString(": string")))
		pg.Add(tsg.NewStatement().ID("method").Add(tsg.TypeFromString(": string")))
		pg.Add(tsg.NewStatement().ID("params").Add(tsg.TypeFromString(": unknown")))
		pg.Add(tsg.NewStatement().ID("query").Optional().Add(tsg.TypeFromString(": Record<string, string>")))
		pg.Add(tsg.NewStatement().ID("extraHeaders").Optional().Add(tsg.TypeFromString(": Record<string, string>")))
		pg.Add(tsg.NewStatement().ID("signal").Optional().Add(tsg.TypeFromString(": AbortSignal")))
		pg.Add(tsg.NewStatement().ID("onReady").Optional().Add(tsg.TypeFromString(": () => void")))
	})
	stmt = tsg.NewStatement()
	stmt.Private()
	stmt.AsyncGeneratorMethodGeneric("streamSSE", "T", params, tsg.NewStatement().ID("AsyncGenerator").Generic("T"), func(mg *tsg.Group) {
		fetchOpts := tsg.NewStatement().ObjectLiteral(func(og *tsg.Group) {
			og.Add(tsg.NewStatement().ID("method").Colon().Lit("POST"))
			og.Add(tsg.NewStatement().ID("headers").Colon().ObjectLiteral(func(hg *tsg.Group) {
				hg.Add(tsg.NewStatement().Spread(tsg.NewStatement().Paren(tsg.NewStatement().Await(tsg.NewStatement().This().Dot("baseClient").Dot("getHeaders").Call()))))
				hg.Add(tsg.NewStatement().ID("Accept").Colon().Lit("text/event-stream"))
				hg.Add(tsg.NewStatement().ObjectField("Content-Type", tsg.NewStatement().Lit("application/json")))
				hg.Add(tsg.NewStatement().Spread(tsg.NewStatement().Paren(tsg.NewStatement().ID("extraHeaders").Op("??").ObjectLiteral(nil))))
			}))
			og.Add(tsg.NewStatement().ID("body").Colon().ID("JSON").Dot("stringify").Call(tsg.NewStatement().ObjectLiteral(func(bg *tsg.Group) {
				bg.Add(tsg.NewStatement().ID("id").Colon().ID("crypto").Dot("randomUUID").Call())
				bg.Add(tsg.NewStatement().ID("jsonrpc").Colon().Lit("2.0"))
				bg.Add(tsg.NewStatement().ID("method").Colon().ID("method"))
				bg.Add(tsg.NewStatement().ID("params").Colon().ID("params"))
			})))
			og.Add(tsg.NewStatement().ID("signal").Colon().ID("signal"))
		})
		mg.Add(tsg.NewStatement().Const("response").Op("=").Await(tsg.NewStatement().ID("fetch").Call(
			tsg.NewStatement().This().Dot("baseClient").Dot("resolveURL").Call(tsg.NewStatement().ID("streamPath"), tsg.NewStatement().ID("query")),
			fetchOpts,
		)).Semicolon())
		mg.If(tsg.NewStatement().Bang().ID("response").Dot("ok").Op("||").Bang().ID("response").Dot("body"), func(ig *tsg.Group) {
			ig.Add(tsg.NewStatement().Await(tsg.NewStatement().ID("response").Dot("body").OptionalChain("cancel").Call()).Semicolon())
			ig.Throw(tsg.NewStatement().New("Error").Call(tsg.NewStatement().Lit("SSE request failed: ").Op("+").ID("String").Call(tsg.NewStatement().ID("response").Dot("status"))))
		})
		mg.Add(tsg.NewStatement().ID("onReady").OptionalChain("").Call().Semicolon())
		mg.Add(tsg.NewStatement().Const("reader").Op("=").ID("response").Dot("body").Dot("getReader").Call().Semicolon())
		mg.Add(tsg.NewStatement().Const("decoder").Op("=").New("TextDecoder").Call().Semicolon())
		mg.Add(tsg.NewStatement().Var("buffer").Op("=").Lit("").Semicolon())
		mg.Add(tsg.NewStatement().TryFinally(func(tg *tsg.Group) {
			tg.While(tsg.NewStatement().Lit(true), func(wg *tsg.Group) {
				wg.Add(tsg.NewStatement().Const("next").Op("=").Await(tsg.NewStatement().ID("reader").Dot("read").Call()).Semicolon())
				wg.If(tsg.NewStatement().ID("next").Dot("done"), func(ig *tsg.Group) { ig.Return() })
				wg.Add(tsg.NewStatement().ID("buffer").Op("+=").ID("decoder").Dot("decode").Call(
					tsg.NewStatement().ID("next").Dot("value"),
					tsg.NewStatement().ObjectLiteral(func(og *tsg.Group) { og.Add(tsg.NewStatement().ID("stream").Colon().Lit(true)) }),
				).Semicolon())
				wg.If(tsg.NewStatement().ID("buffer").Dot("length").Op(">").Lit(1048576), func(ig *tsg.Group) {
					ig.Add(tsg.NewStatement().Await(tsg.NewStatement().ID("reader").Dot("cancel").Call()).Semicolon())
					ig.Throw(tsg.NewStatement().New("Error").Call(tsg.NewStatement().Lit("SSE event too large")))
				})
				wg.Add(tsg.NewStatement().Const("lines").Op("=").ID("buffer").Dot("split").Call(tsg.NewStatement().Lit("\n")).Semicolon())
				wg.Add(tsg.NewStatement().ID("buffer").Op("=").ID("lines").Dot("pop").Call().Op("??").Lit("").Semicolon())
				wg.Add(tsg.NewStatement().ForOf("line", "lines", func(fg *tsg.Group) {
					fg.If(tsg.NewStatement().Bang().ID("line").Dot("startsWith").Call(tsg.NewStatement().Lit("data:")), func(ig *tsg.Group) {
						ig.Add(tsg.NewStatement().Continue())
					})
					fg.Add(tsg.NewStatement().Const("rpc").Op("=").ID("JSON").Dot("parse").Call(
						tsg.NewStatement().ID("line").Dot("slice").Call(tsg.NewStatement().Lit(5)).Dot("trim").Call(),
					).Op("as").Add(tsg.TypeFromString("{ method?: string; params?: { item?: T }; error?: { code: number; message: string } }")).Semicolon())
					fg.If(tsg.NewStatement().ID("rpc").Dot(typeError), func(ig *tsg.Group) {
						ig.Throw(tsg.NewStatement().This().Dot("baseClient").Dot("decodeRPCError").Call(tsg.NewStatement().ID("rpc").Dot(typeError)))
					})
					fg.If(tsg.NewStatement().ID("rpc").Dot("method").Op("===").Lit("$/stream"), func(ig *tsg.Group) {
						ig.Add(tsg.NewStatement().Yield(tsg.NewStatement().ID("rpc").Dot("params").OptionalChain("item").Op("as").ID("T")))
					})
				}))
			})
		}, func(fg *tsg.Group) {
			fg.Add(tsg.NewStatement().TryCatch(func(tg *tsg.Group) {
				tg.Add(tsg.NewStatement().Await(tsg.NewStatement().ID("reader").Dot("cancel").Call()).Semicolon())
			}, func(_ *tsg.Group) {}, ""))
		}))
	})
	return stmt
}

func streamHelperCommonParams(inputOptional bool) (params *tsg.Statement) {

	params = tsg.NewStatement()
	params.Params(func(pg *tsg.Group) {
		pg.Add(tsg.NewStatement().ID("streamPath").Add(tsg.TypeFromString(": string")))
		pg.Add(tsg.NewStatement().ID("method").Add(tsg.TypeFromString(": string")))
		pg.Add(tsg.NewStatement().ID("params").Add(tsg.TypeFromString(": unknown")))
		input := tsg.NewStatement().ID("input")
		if inputOptional {
			input.Optional()
		}
		input.Add(tsg.TypeFromString(": AsyncIterable<unknown>"))
		pg.Add(input)
		pg.Add(tsg.NewStatement().ID("query").Optional().Add(tsg.TypeFromString(": Record<string, string>")))
		pg.Add(tsg.NewStatement().ID("signal").Optional().Add(tsg.TypeFromString(": AbortSignal")))
		pg.Add(tsg.NewStatement().ID("onReady").Optional().Add(tsg.TypeFromString(": () => void")))
	})
	return params
}

func addStreamAbortGuard(mg *tsg.Group) {

	mg.If(tsg.NewStatement().ID("signal").OptionalChain("aborted"), func(ig *tsg.Group) {
		ig.Throw(tsg.NewStatement().ID("signal").Dot("reason").Op("instanceof").ID("Error").Op("?").ID("signal").Dot("reason").Op(":").New("DOMException").Call(
			tsg.NewStatement().Lit("This operation was aborted"), tsg.NewStatement().Lit("AbortError"),
		))
	})
}

func signalAbortListener() (stmt *tsg.Statement) {

	return tsg.NewStatement().ID("signal").OptionalChain("addEventListener").Call(
		tsg.NewStatement().Lit("abort"),
		tsg.NewStatement().ID("onAbort"),
		tsg.NewStatement().ObjectLiteral(func(og *tsg.Group) { og.Add(tsg.NewStatement().ID("once").Colon().Lit(true)) }),
	).Semicolon()
}

func socketSendJSONRPC() (stmt *tsg.Statement) {

	return tsg.NewStatement().ID("socket").Dot("send").Call(
		tsg.NewStatement().ID("JSON").Dot("stringify").Call(tsg.NewStatement().ObjectLiteral(func(og *tsg.Group) {
			og.Add(tsg.NewStatement().ID("id").Colon().ID("id"))
			og.Add(tsg.NewStatement().ID("jsonrpc").Colon().Lit("2.0"))
			og.Add(tsg.NewStatement().ID("method").Colon().ID("method"))
			og.Add(tsg.NewStatement().ID("params").Colon().ID("params"))
		})),
	).Semicolon()
}

func streamSocketAbortCond() (cond *tsg.Statement) {

	return tsg.NewStatement().ID("signal").OptionalChain("aborted").Op("||").ID("socket").Dot("readyState").Op("!==").ID("WebSocket").Dot("OPEN")
}

func voidAsyncInputPump(ignoreCatch bool) (stmt *tsg.Statement) {

	body := tsg.NewStatement()
	body.Async()
	body.ArrowFunc()
	body.Block(func(ag *tsg.Group) {
		pump := func(tg *tsg.Group) {
			tg.ForAwait("item", tsg.NewStatement().ID("input"), func(fg *tsg.Group) {
				fg.If(streamSocketAbortCond(), func(sg *tsg.Group) { sg.Return() })
				fg.Add(tsg.NewStatement().ID("socket").Dot("send").Call(
					tsg.NewStatement().ID("JSON").Dot("stringify").Call(tsg.NewStatement().ObjectLiteral(func(og *tsg.Group) {
						og.Add(tsg.NewStatement().ID("jsonrpc").Colon().Lit("2.0"))
						og.Add(tsg.NewStatement().ID("method").Colon().Lit("$/stream"))
						og.Add(tsg.NewStatement().ID("params").Colon().ObjectLiteral(func(pg *tsg.Group) {
							pg.Add(tsg.NewStatement().ID("id").Colon().ID("id"))
							pg.Add(tsg.NewStatement().ID("item").Colon().ID("item"))
						}))
					})),
				).Semicolon())
			})
			tg.If(streamSocketAbortCond(), func(sg *tsg.Group) { sg.Return() })
			tg.Add(tsg.NewStatement().ID("socket").Dot("send").Call(
				tsg.NewStatement().ID("JSON").Dot("stringify").Call(tsg.NewStatement().ObjectLiteral(func(og *tsg.Group) {
					og.Add(tsg.NewStatement().ID("jsonrpc").Colon().Lit("2.0"))
					og.Add(tsg.NewStatement().ID("method").Colon().Lit("$/stream.end"))
					og.Add(tsg.NewStatement().ID("params").Colon().ObjectLiteral(func(pg *tsg.Group) {
						pg.Add(tsg.NewStatement().ID("id").Colon().ID("id"))
					}))
				})),
			).Semicolon())
		}
		if ignoreCatch {
			pump(ag)
			return
		}
		ag.Add(tsg.NewStatement().TryCatch(pump, func(cg *tsg.Group) {
			cg.Add(tsg.NewStatement().ID("settle").Call(tsg.NewStatement().ArrowFunc().Block(func(sg *tsg.Group) {
				sg.Add(tsg.NewStatement().ID("socket").Dot("close").Call().Semicolon())
				sg.Add(tsg.NewStatement().ID("reject").Call(tsg.NewStatement().ID(typeError)).Semicolon())
			})).Semicolon())
		}, typeError))
	})
	return tsg.NewStatement().VoidExpr(tsg.NewStatement().Paren(body).Call())
}
