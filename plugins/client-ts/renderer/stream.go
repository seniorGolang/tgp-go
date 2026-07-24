// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package renderer

import (
	"fmt"
	"strings"

	"tgp/internal/model"
	"tgp/plugins/client-ts/tsg"
)

func (r *ClientRenderer) renderStreamSupport(grp *tsg.Group, contract *model.Contract) {

	needWSOut, needWSResult, needSSE := r.streamHelperNeeds(contract)
	if !needWSOut && !needWSResult && !needSSE {
		return
	}
	parts := make([]string, 0, 4)
	if needWSOut || needWSResult {
		parts = append(parts, streamHelperWebSocketURL, streamHelperOpenWebSocket)
	}
	if needWSOut {
		parts = append(parts, streamHelperWebSocket)
	}
	if needWSResult {
		parts = append(parts, streamHelperWebSocketResult)
	}
	if needSSE {
		parts = append(parts, streamHelperSSE)
	}
	grp.Add(tsg.TypeFromString(strings.Join(parts, "\n\n")))
	grp.Line()
}

func (r *ClientRenderer) streamHelperNeeds(contract *model.Contract) (needWSOut bool, needWSResult bool, needSSE bool) {

	for _, method := range contract.Methods {
		if model.MethodIsSSE(r.project, contract, method) {
			needSSE = true
		}
		if !model.MethodIsWS(r.project, contract, method) {
			continue
		}
		switch model.MethodStreamMode(r.project, contract, method) {
		case model.StreamModeClient:
			needWSResult = true
		case model.StreamModeServer, model.StreamModeBidi:
			needWSOut = true
		}
	}
	return
}

// appendStreamExchangeTypes добавляет Response* exchange-типы, на которые ссылается stream-клиент.
func (r *ClientRenderer) appendStreamExchangeTypes(contract *model.Contract, method *model.Method, exchangeTypes *[]string, seenTypes map[string]bool) {

	if !model.MethodIsStream(r.project, contract, method) {
		return
	}
	if model.MethodStreamMode(r.project, contract, method) != model.StreamModeClient {
		return
	}
	if len(r.resultsWithoutChannel(method)) <= 1 {
		return
	}
	responseType := r.responseTypeName(contract, method)
	if seenTypes[responseType] {
		return
	}
	*exchangeTypes = append(*exchangeTypes, responseType)
	seenTypes[responseType] = true
}

func (r *ClientRenderer) renderStreamMethods(grp *tsg.Group, contract *model.Contract, method *model.Method) {

	if model.MethodIsWS(r.project, contract, method) {
		grp.Add(tsg.TypeFromString(r.streamMethodSource(contract, method, method.Name, true)))
		grp.Line()
	}
	if model.MethodIsSSE(r.project, contract, method) {
		name := method.Name
		if model.MethodIsWS(r.project, contract, method) {
			name += "SSE"
		}
		grp.Add(tsg.TypeFromString(r.streamMethodSource(contract, method, name, false)))
		grp.Line()
	}
}

func (r *ClientRenderer) streamMethodSource(contract *model.Contract, method *model.Method, name string, websocket bool) (source string) {

	mode := model.MethodStreamMode(r.project, contract, method)
	clientArgs := r.streamClientArgs(contract, method)
	bodyArgs := r.argsForExchangeRequest(contract, method)
	params := r.streamTSParams(contract, method, clientArgs, mode)
	paramsObject := r.streamTSParamsObject(bodyArgs)
	methodName := model.JsonRPCWireMethod(contract.Name, method.Name)
	rawPath := model.MethodSSEPath(r.project, contract, method)
	if websocket {
		rawPath = model.ContractWSPath(r.project, contract)
	}
	pathExpr := r.streamTSPathExpr(contract, method, rawPath)
	queryExpr := r.streamTSQueryExpr(contract, method, websocket)
	headersExpr := r.streamTSHeadersExpr(contract, method)
	methodNameTS := r.lcName(name)

	if mode == model.StreamModeClient {
		in, _, _ := model.MethodStreamInChan(r.project, method)
		resultType := r.streamTSResultType(contract, method)
		return fmt.Sprintf("public async %s(%s): Promise<%s> {\n    const params = %s;\n    const streamPath = %s;\n    const query = %s;\n    return this.streamWebSocketResult<%s>(streamPath, %q, params, %s, query);\n}",
			methodNameTS, params, resultType, paramsObject, pathExpr, queryExpr, resultType, methodName, tsSafeName(in.Name))
	}

	_, element, _ := model.MethodStreamOutChan(r.project, method)
	outType := r.streamTSType(contract, method, element, false)
	inputArg := "undefined"
	if mode == model.StreamModeBidi {
		in, _, _ := model.MethodStreamInChan(r.project, method)
		inputArg = tsSafeName(in.Name)
	}
	if websocket {
		return fmt.Sprintf("public async *%s(%s): AsyncGenerator<%s> {\n    const params = %s;\n    const streamPath = %s;\n    const query = %s;\n    for await (const item of this.streamWebSocket<%s>(streamPath, %q, params, %s, query)) {\n        yield item;\n    }\n}",
			methodNameTS, params, outType, paramsObject, pathExpr, queryExpr, outType, methodName, inputArg)
	}
	return fmt.Sprintf("public async *%s(%s): AsyncGenerator<%s> {\n    const params = %s;\n    const streamPath = %s;\n    const query = %s;\n    const extraHeaders = %s;\n    for await (const item of this.streamSSE<%s>(streamPath, %q, params, query, extraHeaders)) {\n        yield item;\n    }\n}",
		methodNameTS, params, outType, paramsObject, pathExpr, queryExpr, headersExpr, outType, methodName)
}

func (r *ClientRenderer) streamClientArgs(contract *model.Contract, method *model.Method) (out []*model.Variable) {

	for _, arg := range r.argsForClient(contract, method) {
		if model.TypeRefIsChan(r.project, &arg.TypeRef) {
			continue
		}
		out = append(out, arg)
	}
	return out
}

func (r *ClientRenderer) streamTSPathExpr(contract *model.Contract, method *model.Method, rawPath string) (expr string) {

	expr = fmt.Sprintf("%q", rawPath)
	for argName, segment := range model.StreamPathParamArgMap(r.project, contract, method) {
		expr = fmt.Sprintf("(%s).split(%q).join(encodeURIComponent(String(%s)))", expr, ":"+segment, tsSafeName(argName))
	}
	return expr
}

func (r *ClientRenderer) streamTSQueryExpr(contract *model.Contract, method *model.Method, websocket bool) (expr string) {

	implicit := model.HTTPImplicitArgSet(model.BuildHTTPArgMappings(r.project, contract, method))
	parts := make([]string, 0)
	for argName, key := range model.HTTPArgQueryMapForRequest(r.project, contract, method) {
		if _, skip := implicit[argName]; skip {
			continue
		}
		if r.clientArgByName(contract, method, argName) == nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("%q: String(%s)", key, tsSafeName(argName)))
	}
	if websocket {
		// Browser WebSocket cannot set custom headers/cookies; mirror them as query for WS upgrade fallback.
		headerMap := model.HTTPHeaderArgMapForRequest(r.project, contract, method)
		for argName, headerName := range headerMap {
			if _, skip := implicit[argName]; skip {
				continue
			}
			if r.clientArgByName(contract, method, argName) == nil {
				continue
			}
			parts = append(parts, fmt.Sprintf("%q: String(%s)", headerName, tsSafeName(argName)))
		}
		cookieMap := model.HTTPCookieArgMapForRequest(r.project, contract, method)
		for argName, cookieName := range cookieMap {
			if _, skip := implicit[argName]; skip {
				continue
			}
			if r.clientArgByName(contract, method, argName) == nil {
				continue
			}
			parts = append(parts, fmt.Sprintf("%q: String(%s)", cookieName, tsSafeName(argName)))
		}
	}
	if len(parts) == 0 {
		return "undefined"
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func (r *ClientRenderer) streamTSHeadersExpr(contract *model.Contract, method *model.Method) (expr string) {

	implicit := model.HTTPImplicitArgSet(model.BuildHTTPArgMappings(r.project, contract, method))
	parts := make([]string, 0)
	headerMap := model.HTTPHeaderArgMapForRequest(r.project, contract, method)
	for argName, headerName := range headerMap {
		if _, skip := implicit[argName]; skip {
			continue
		}
		if r.clientArgByName(contract, method, argName) == nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("%q: String(%s)", headerName, tsSafeName(argName)))
	}
	cookieMap := model.HTTPCookieArgMapForRequest(r.project, contract, method)
	cookieParts := make([]string, 0)
	for argName, cookieName := range cookieMap {
		if _, skip := implicit[argName]; skip {
			continue
		}
		if r.clientArgByName(contract, method, argName) == nil {
			continue
		}
		cookieParts = append(cookieParts, fmt.Sprintf("%q + \"=\" + encodeURIComponent(String(%s))", cookieName, tsSafeName(argName)))
	}
	if len(cookieParts) > 0 {
		parts = append(parts, fmt.Sprintf("%q: [%s].join(\"; \")", "Cookie", strings.Join(cookieParts, ", ")))
	}
	if len(parts) == 0 {
		return "undefined"
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func (r *ClientRenderer) streamTSParams(contract *model.Contract, method *model.Method, args []*model.Variable, mode string) (params string) {

	parts := make([]string, 0, len(args)+1)
	for _, arg := range args {
		parts = append(parts, fmt.Sprintf("%s: %s", tsSafeName(arg.Name), r.streamTSType(contract, method, &arg.TypeRef, true)))
	}
	if mode == model.StreamModeClient || mode == model.StreamModeBidi {
		in, element, _ := model.MethodStreamInChan(r.project, method)
		parts = append(parts, fmt.Sprintf("%s: AsyncIterable<%s>", tsSafeName(in.Name), r.streamTSType(contract, method, element, true)))
	}
	return strings.Join(parts, ", ")
}

func (r *ClientRenderer) streamTSParamsObject(args []*model.Variable) (object string) {

	if len(args) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, fmt.Sprintf("%s: %s", tsSafeName(arg.Name), tsSafeName(arg.Name)))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func (r *ClientRenderer) streamTSType(contract *model.Contract, method *model.Method, typeRef *model.TypeRef, input bool) (typeName string) {

	if typeRef == nil {
		return "unknown"
	}
	variable := &model.Variable{Name: "item", TypeRef: *typeRef}
	return r.walkVariable(variable.Name, contract.PkgPath, variable, method.Annotations, input).typeLink()
}

func (r *ClientRenderer) streamTSResultType(contract *model.Contract, method *model.Method) (typeName string) {

	results := r.resultsWithoutChannel(method)
	if len(results) == 0 {
		return "void"
	}
	if len(results) == 1 {
		return r.streamTSType(contract, method, &results[0].TypeRef, false)
	}
	return r.responseTypeName(contract, method)
}

func (r *ClientRenderer) resultsWithoutChannel(method *model.Method) (out []*model.Variable) {

	for _, result := range r.resultsWithoutError(method) {
		if !model.TypeRefIsChan(r.project, &result.TypeRef) {
			out = append(out, result)
		}
	}
	return
}

func (r *ClientRenderer) clientArgByName(contract *model.Contract, method *model.Method, argName string) (variable *model.Variable) {

	for _, arg := range r.argsForClient(contract, method) {
		if arg.Name == argName {
			return arg
		}
	}
	return nil
}

const streamHelperWebSocketURL = `private websocketURL(streamPath: string, query?: Record<string, string>): string {
    const url = new URL(this.baseClient.getEndpoint());
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    url.pathname = streamPath;
    url.search = "";
    if (query) {
        for (const [key, value] of Object.entries(query)) {
            url.searchParams.set(key, value);
        }
    }
    return url.toString();
}`

const streamHelperOpenWebSocket = `private openWebSocket(streamPath: string, query?: Record<string, string>): Promise<WebSocket> {
    return new Promise((resolve, reject) => {
        const socket = new WebSocket(this.websocketURL(streamPath, query));
        socket.addEventListener("open", () => resolve(socket), { once: true });
        socket.addEventListener("error", () => reject(new Error("WebSocket connection failed")), { once: true });
    });
}`

const streamHelperWebSocket = `private async *streamWebSocket<T>(streamPath: string, method: string, params: unknown, input?: AsyncIterable<unknown>, query?: Record<string, string>): AsyncGenerator<T> {
    const socket = await this.openWebSocket(streamPath, query);
    const id = crypto.randomUUID();
    const messages: unknown[] = [];
    let done = false;
    let wake: (() => void) | undefined;
    socket.addEventListener("message", (event) => {
        messages.push(JSON.parse(String(event.data)));
        wake?.();
    });
    socket.send(JSON.stringify({ id, jsonrpc: "2.0", method, params }));
    if (input) {
        void (async () => {
            for await (const item of input) {
                socket.send(JSON.stringify({ jsonrpc: "2.0", method: "$/stream", params: { id, item } }));
            }
            socket.send(JSON.stringify({ jsonrpc: "2.0", method: "$/stream.end", params: { id } }));
        })();
    }
    try {
        while (!done) {
            if (messages.length === 0) {
                await new Promise<void>((resolve) => { wake = resolve; });
                wake = undefined;
            }
            for (const message of messages.splice(0)) {
                const rpc = message as { id?: string; method?: string; params?: { id?: string; item?: T }; error?: { code: number; message: string } };
                if (rpc.method === "$/stream" && rpc.params?.id === id) {
                    yield rpc.params.item as T;
                    continue;
                }
                if (rpc.id === id) {
                    if (rpc.error) {
                        throw this.baseClient.decodeRPCError(rpc.error);
                    }
                    done = true;
                }
            }
        }
    } finally {
        socket.close();
    }
}`

const streamHelperWebSocketResult = `private async streamWebSocketResult<TResult>(streamPath: string, method: string, params: unknown, input: AsyncIterable<unknown>, query?: Record<string, string>): Promise<TResult> {
    const socket = await this.openWebSocket(streamPath, query);
    const id = crypto.randomUUID();
    return new Promise<TResult>((resolve, reject) => {
        socket.addEventListener("message", (event) => {
            const rpc = JSON.parse(String(event.data)) as { id?: string; result?: TResult; error?: { code: number; message: string } };
            if (rpc.id !== id) {
                return;
            }
            socket.close();
            if (rpc.error) {
                reject(this.baseClient.decodeRPCError(rpc.error));
                return;
            }
            resolve(rpc.result as TResult);
        });
        socket.send(JSON.stringify({ id, jsonrpc: "2.0", method, params }));
        void (async () => {
            try {
                for await (const item of input) {
                    socket.send(JSON.stringify({ jsonrpc: "2.0", method: "$/stream", params: { id, item } }));
                }
                socket.send(JSON.stringify({ jsonrpc: "2.0", method: "$/stream.end", params: { id } }));
            } catch (error) {
                socket.close();
                reject(error);
            }
        })();
    });
}`

const streamHelperSSE = `private async *streamSSE<T>(streamPath: string, method: string, params: unknown, query?: Record<string, string>, extraHeaders?: Record<string, string>): AsyncGenerator<T> {
    let path = streamPath;
    if (query && Object.keys(query).length > 0) {
        const search = new URLSearchParams(query);
        path = streamPath + "?" + search.toString();
    }
    const response = await fetch(this.baseClient.getEndpoint().replace(/\/$/, "") + path, {
        method: "POST",
        headers: { ...(await this.baseClient.getHeaders()), Accept: "text/event-stream", "Content-Type": "application/json", ...(extraHeaders ?? {}) },
        body: JSON.stringify({ id: crypto.randomUUID(), jsonrpc: "2.0", method, params }),
    });
    if (!response.ok || !response.body) {
        throw new Error("SSE request failed: " + String(response.status));
    }
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    for (;;) {
        const next = await reader.read();
        if (next.done) {
            return;
        }
        buffer += decoder.decode(next.value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() ?? "";
        for (const line of lines) {
            if (!line.startsWith("data:")) {
                continue;
            }
            const rpc = JSON.parse(line.slice(5).trim()) as { method?: string; params?: { item?: T }; error?: { code: number; message: string } };
            if (rpc.error) {
                throw this.baseClient.decodeRPCError(rpc.error);
            }
            if (rpc.method === "$/stream") {
                yield rpc.params?.item as T;
            }
        }
    }
}`
