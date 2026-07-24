---
name: tgp-contracts
description: >-
  Writes and reviews Go tgp contracts with @tg annotations (levels, HTTP/JSON-RPC/stream
  mapping, sub-annotations, kafka). Use when editing contracts/, adding API methods,
  fixing @tg annotations, http-headers/cookies/args modes, stream/ws/sse, or before
  regenerating server/clients/swagger/kafka.
---

# tgp-contracts

## Goal

Correct `@tg` contracts in `contracts/*.go` (flat dir only — no nested folders). Generators read this model; bad annotations → bad code.

## Workflow

1. Edit only contracts (and domain types they reference) — not generated `transport/` / client / kafka packages.
2. Put annotations in comments: `// @tg name[=value] …`
3. Values with spaces/special chars → backticks: `` // @tg http-path=`/users/:id` ``
4. Long OpenAPI text → `file:docs/api.md` or `file:docs/api.md#Section`
5. Before generate: ensure each HTTP iface has `http-server` and/or `jsonRPC-server` (and `ws-server`/`sse-server` if streaming); kafka ifaces use only kafka annotations.
6. Then run the matching generator skill / command.

## Annotation levels (priority high → low)

1. Field / parameter comment  
2. Method sub-annotation `arg.required`, `result.tags=…`  
3. Method  
4. Interface  
5. Package  

## Mapping modes (`http-headers` / `http-cookies` / `http-args`)

Format: `arg|key` or `arg|key|mode`. Default mode = `body`.

| Mode | Meaning |
|------|---------|
| `explicit` | Value via header/cookie/path/query only — not duplicated in body |
| `implicit` | Like explicit for HTTP; often filled from middleware/context; may be hidden in clients |
| `body` | Body is primary; HTTP overlay may override (e.g. JSON-RPC context) |

Example:

```go
// @tg http-headers=token|Authorization|explicit
// @tg token.required
GetProfile(ctx context.Context, token string) (profile Profile, err error)
```

## Streams

- Method: `@tg stream=server|client|bidi`
- Interface: `@tg ws-server` and/or `@tg sse-server` (SSE is server-stream only)
- Optional: `ws-path=…`, `sse-path=…` (support `:param` with `http-args`)

## Kafka (separate contracts)

- `@tg kafka` only (not with http/jsonrpc/ws/sse/stream)
- Legacy `@tg kafka-consumer` / `@tg kafka-publisher` → validation error (migrate to `@tg kafka`)
- Method: `kafka-topic=…` required; optional `kafka-key=`, `kafka-headers=`, `kafka-message=`, `kafka-codec=`, `kafka-acks=`
- Methods return only `error`; first arg is `context.Context`
- Generate: `tg kafka pub go -o …` and/or `tg kafka sub go -o …`

## Sub-annotations on methods

`// @tg <paramOrResult>.<key>[=value]` — keys: `required`, `desc`, `format`, `example`, `enums`, `type`, `tags`, `log-skip`, `http-part-name`, `http-part-content`.

Useful `tags`: `json:inline`, `json:name,omitempty`, `form:name`, `dumper:hide`.

## Checklist before generate

- [ ] Interface has the right transport enable flags  
- [ ] Paths/methods set for REST (`http-method`, `http-path`, `http-prefix`)  
- [ ] Mapping modes intentional (not accidental default `body`)  
- [ ] Secrets marked `log-skip`  
- [ ] No edits planned inside generated trees  

## Dig deeper

- Compact tables: [references/annotations.md](references/annotations.md)
- Full catalog: `tg plugin doc astg`
