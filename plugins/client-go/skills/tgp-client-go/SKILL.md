---
name: tgp-client-go
description: >-
  Generates typed Go API clients (JSON-RPC, REST, WS/SSE streams) from tgp contracts.
  Use when creating or updating a Go SDK, wiring endpoint/options/errors/metrics,
  using RPC batch, streams, upload/download, or diagnosing missing methods and
  changed signatures. Do not use for authoring contracts or server transport.
---

# tgp-client-go

## Workflow

1. Prepare contracts with `tgp-contracts`.
2. Inspect the resolved model and confirm HTTP/RPC/stream family:

```bash
tg astg json -o .tg/project.json
```

3. Choose `out` inside the target Go module; generation fails when no containing `go.mod` exists.
4. Generate:

```bash
tg client go --out ./client
# optional: --contracts=… --doc-file=… --no-doc
# pinned model (requires astg-db): --from-db name@version
```

5. Use the actual generated constructor:

```go
c := client.New("https://api.example.com", client.Name("orders"))
user, err := c.UserService().GetUser(ctx, id)
```

6. Compile every consumer and smoke-test the selected protocol.

## Choose client surface

| Contract capability | Generated use |
|--------------------|---------------|
| `jsonRPC-server` | methods and `Req*` builders on `c.<Contract>()` |
| `http-server` | typed REST methods on the same contract accessor |
| JSON-RPC batch | `c.Batch(ctx, requests...)` |
| `ws-server` | channel-based WebSocket stream methods |
| `sse-server` | server-stream method, with `*SSE` variant when both transports exist |
| `kafka` | no client methods |

`implicit` mapped arguments may be absent from public client signatures. Supply those values through configured headers/context as designed.

## Gotchas

- Only `--out` (no `-o`); path must sit under a `go.mod`
- REST/SSE/WS join `endpoint` path with the declared route (`http-prefix` + `http-path`); overlapping segments collapse (keeps mounts like `/cli`, no `/api/v1/api/v1`)
- `New(endpoint)` JSON-RPC calls hit the endpoint URL itself; REST/SSE/WS clear endpoint query then join paths
- Auth/`token` package appears only when `@tg security` includes `http:bearer` or `oauth2:` — see [references/auth.md](references/auth.md)
- Always close returned `io.ReadCloser`; multipart parts are ordered
- `--from-db` skips `astg-hook` save in the same run

Runtime options overview and batch/streams/files: [references/usage.md](references/usage.md) · `tg plugin doc client-go`.

## Verify

```bash
go test ./...
```

- [ ] Expected contract accessor and method exist
- [ ] Signatures match intended `explicit`/`implicit` mapping
- [ ] REST path/header/body or RPC method is correct
- [ ] Custom error decoder receives the expected payload
- [ ] Stream and file resources close on success, error, and cancellation

## Diagnose

- `go.mod not found` — move `--out` under the target module
- Missing contract/method — wrong family flag/filter or stale ASTG model
- Argument “disappeared” — effective mode is `implicit`
- Header/query remains in body — mode is `body`
- Doubled or wrong REST URL — check endpoint mount vs declared `http-prefix`/`http-path` overlap join
- Response shape changed — check `enableInlineSingle` and result tags
- Caller no longer compiles — regenerate, then update caller; never patch client sources

## Never

- Reimplement HTTP/RPC manually when a generated client exists
- Point `--out` at server `transport/`
- Put handwritten code in the generated client package
- Forget to close `io.ReadCloser`
- Expect Kafka contracts in this client

## Dig deeper

`tg plugin doc client-go` · [references/usage.md](references/usage.md) · [references/auth.md](references/auth.md) · skills `tgp-contracts`, `tgp-astg-json`, `tgp-astg-db`, `tgp-pipeline`
