---
name: tgp-server
description: >-
  Generates and wires Fiber HTTP/JSON-RPC/WS/SSE server code from tgp contracts.
  Use when running tgp server, regenerating transport/, implementing service stubs,
  WithLog/WithMetrics/WithTrace, or fixing generated server integration — never when
  only editing contracts (use tgp-contracts first).
---

# tgp-server

## Do

1. Contracts ready (`http-server` / `jsonRPC-server` / `ws-server` / `sse-server` as needed) — see skill `tgp-contracts`.
2. Generate:

```bash
tgp server -o transport
# optional: --contracts-dir=contracts --contracts=UserService,OrderService
```

3. Implement services (your code) and wire:

```go
srv := transport.New(log, transport.UserService(contracts.NewUserService(impl)))
srv.WithLog()       // if @tg log
srv.WithMetrics()   // if @tg metrics
// srv.WithTrace(...)
_ = srv.Fiber().Listen(":8080")
```

4. After any contract change → **regenerate**, do not patch generated files.

## Never

- Edit files under the `-o` package (e.g. `transport/`) by hand  
- Put Kafka adapters into separate dirs (use `tgp-kafka-pub-go` / `tgp-kafka-sub-go`)  
- Expect handlers without `http-server` / `jsonRPC-server` on the interface  

## Mapping reminder

`http-headers` / `http-cookies` / `http-args`: `arg|key|mode` with `explicit|implicit|body` (default `body`). Details in `tgp-contracts`.

## Streams

- WS: iface `@tg ws-server` + method `@tg stream=…`  
- SSE: iface `@tg sse-server` + `@tg stream=server`  
- Headers/cookies/args apply: SSE from request; WS from upgrade (connection-scoped)

## Dig deeper

`tg plugin doc server` · `tg plugin doc astg`
