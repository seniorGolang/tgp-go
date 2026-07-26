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
tg client go -o ./client
# optional: --contracts=… --doc-file=… --no-doc
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

## Runtime options

- `Name` sets client identity / `X-Client-Id`
- `Headers` copies selected values from context
- `ConfigTLS`, `ClientHTTP`, `Transport` control networking
- `BeforeRequest`, `AfterRequest` add hooks
- `DecodeError` and `DecodeHTTPError` customize RPC and REST errors separately
- `LogRequest` and `LogOnError` control logging
- `WithMetrics` creates a dedicated registry available through `GetMetricsRegistry`

Do not enable full request logging around secrets without reviewing `log-skip` and payload exposure.

## Batch, streams, and files

- JSON-RPC batch uses generated `Req<Method>` values and callbacks
- WS/SSE stream signatures use channels; handle cancellation and channel closure
- A returned `io.ReadCloser` must always be closed by the caller
- Multiple readers/results or multipart contracts use generated multipart/schema support

Examples and caveats: [references/usage.md](references/usage.md).

## Verify

```bash
go test ./...
```

Confirm:

- expected contract accessor and method exist
- signatures match intended `explicit`/`implicit` mapping
- REST path/header/body or RPC method is correct
- custom error decoder receives the expected payload
- stream and file resources close on success, error, and cancellation

## Diagnose

- `go.mod not found` — move `-o` under the target module
- Missing contract/method — wrong family flag/filter or stale ASTG model
- Argument “disappeared” — effective mode is `implicit`
- Header/query remains in body — mode is `body`
- Response shape changed — check `enableInlineSingle` and result tags
- Caller no longer compiles — regenerate, then update caller; never patch client sources

## Never

- Reimplement HTTP/RPC manually when a generated client exists
- Point `-o` at server `transport/`
- Put handwritten code in the generated client package
- Forget to close `io.ReadCloser`
- Expect Kafka contracts in this client

## Dig deeper

`tg plugin doc client-go` · skills `tgp-contracts`, `tgp-astg-json`, `tgp-server`
