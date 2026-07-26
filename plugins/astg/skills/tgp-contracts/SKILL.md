---
name: tgp-contracts
description: >-
  Writes and reviews Go tgp contracts with @tg annotations (levels, HTTP/JSON-RPC/stream
  mapping, sub-annotations, kafka) and OpenAPI-facing docs (summary, desc, tagDesc,
  requestBodyDesc, file: refs, field examples). Use when editing contracts/ or
  contracts/dto, adding API methods, fixing @tg annotations, writing or improving API
  descriptions for Swagger/OpenAPI, http-headers/cookies/args modes, stream/ws/sse,
  or before regenerating server/clients/swagger/kafka.
---

# tgp-contracts

## Goal

Correct `@tg` contracts in `contracts/`. Generators read this model; bad annotations → bad code.

## Layout (required)

Separate the API contract from its DTO types:

- `contracts/*.go` — **only** `@tg` interfaces (+ package-level annotations). Flat: nested folders are **not** scanned for contracts.
- `contracts/dto/*.go` — DTO types (`package dto`); reference them in signatures as `dto.Type`.
- Do **not** put payload structs in `package contracts`.
- Do **not** put `@tg` interfaces in `contracts/dto` or any nested folder — they won't be discovered.

DTO types are resolved via imports/`go/types`, so `contracts/dto` (and other imported packages) work fine — only the interface files must stay flat in `contracts/`.

## Workflow

1. Inspect the existing resolved model with `tgp-astg-json`; do not infer current wire behavior from comments alone.
2. Edit interfaces in `contracts/` and DTO in `contracts/dto` — not generated server/client/kafka packages.
3. Put annotations in comments: `// @tg name[=value] …`
4. Values with spaces/special chars → backticks: `` // @tg http-path=`/users/:id` ``
5. Document public surface: package `title`/`version`/`desc`, method `summary`/`desc`, DTO/`arg` `desc` — see [references/documentation.md](references/documentation.md). Long OpenAPI text → `file:docs/api.md` or `file:docs/api.md#Section`.
6. Export the model again and verify contract family, method annotations, arguments/results, referenced types, and resolved descriptions.
7. Run every affected generator, then compile its consumers.

## Choose transport

| Need | Interface | Method |
|------|-----------|--------|
| REST | `http-server` | `http-method`, `http-path` |
| JSON-RPC / batch | `jsonRPC-server` | ordinary named signature |
| WebSocket stream | `ws-server` | `stream=server|client|bidi` |
| SSE stream | `sse-server` | `stream=server` |
| Kafka | `kafka` only | `kafka-topic` |

REST and JSON-RPC may share a contract. Kafka must use a separate contract and cannot mix with HTTP/JSON-RPC/WS/SSE/stream annotations.

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
GetProfile(ctx context.Context, token string) (profile dto.Profile, err error)
```

## Streams

- Method: `@tg stream=server|client|bidi`
- Interface: `@tg ws-server` and/or `@tg sse-server` (SSE is server-stream only)
- Optional: `ws-path=…`, `sse-path=…` (support `:param` with `http-args`)
- Stream signatures use channels in the direction required by the selected mode
- Do not add `http-method` to stream methods

## Kafka (separate contracts)

- `@tg kafka` only (not with http/jsonrpc/ws/sse/stream)
- Legacy `@tg kafka-consumer` / `@tg kafka-publisher` → validation error (migrate to `@tg kafka`)
- Method: `kafka-topic=…` required; optional `kafka-key=`, `kafka-headers=`, `kafka-message=`, `kafka-codec=`, `kafka-acks=`
- Methods return only `error`; first arg is `context.Context`
- Generate: `tg kafka pub go -o …` and/or `tg kafka sub go -o …`

## Sub-annotations on methods

`// @tg <paramOrResult>.<key>[=value]` — keys: `required`, `desc`, `format`, `example`, `enums`, `type`, `tags`, `log-skip`, `http-part-name`, `http-part-content`.

Useful `tags`: `json:inline`, `json:name,omitempty`, `form:name`, `dumper:hide`.

## Wire-shape review

Before generating, answer explicitly:

1. Which values are path/query/header/cookie/body?
2. Which `implicit` values disappear from generated client signatures?
3. Is a single result wrapped or changed by `enableInlineSingle`?
4. Are optional request/response fields represented intentionally (`pointer`, `omitempty`, `required`)?
5. Do secrets carry `log-skip` at the effective level?
6. Do file methods require raw body or multipart?

For `application/x-www-form-urlencoded`, body arguments need explicit `form:<name>` tags.

## Checklist before generate

- [ ] DTO live in `contracts/dto` (`package dto`), not alongside interfaces
- [ ] Interface is exported, flat in `contracts/`, and marked with `@tg`
- [ ] Interface has the right transport enable flags
- [ ] Paths/methods set for REST (`http-method`, `http-path`, `http-prefix`)
- [ ] Mapping modes intentional (not accidental default `body`)
- [ ] Stream channel directions and SSE limitations are valid
- [ ] Kafka topics are unique and message/key/header arguments resolve
- [ ] Secrets marked `log-skip`
- [ ] Public HTTP/JSON-RPC methods have `summary`; public fields/params have `desc` where OpenAPI is published
- [ ] `tg astg json -o .tg/project.json` shows the intended model (including resolved `file:` descriptions)
- [ ] No edits planned inside generated trees

Generator validation is the final authority. If validation fails, fix the source contract, re-export the model, and run the same generator again.

## Never

- Put contract interfaces in nested directories or DTO packages
- Use unexported or embedded interfaces as contracts
- Mix Kafka and HTTP-family annotations on one interface
- Patch generated code to compensate for a bad signature or annotation
- Assume an inherited annotation is physically copied into every JSON node

## Dig deeper

- Compact tables: [references/annotations.md](references/annotations.md)
- OpenAPI/API documentation text: [references/documentation.md](references/documentation.md)
- Validation and discovery failures: [references/validation.md](references/validation.md)
- Full catalog: `tg plugin doc astg`
- Inspect resolved model (local or DB): skill `tgp-astg-json` / `tg astg json`
