---
name: tgp-server
description: >-
  Generates and wires Fiber HTTP/JSON-RPC/WS/SSE server code from tgp contracts.
  Use when generating or regenerating transport/, implementing service interfaces,
  wiring Fiber, errors, health, metrics, tracing, streams, or diagnosing missing
  routes and signature mismatches. Do not use when only editing contracts.
---

# tgp-server

## Workflow

1. Prepare contracts with `tgp-contracts`.
2. Export the resolved model and confirm the intended HTTP family:

```bash
tg astg json -o .tg/project.json
```

3. Generate:

```bash
tg server -o transport
# optional: --contracts-dir=contracts --contracts=UserService,OrderService
```

4. Implement exported contract interfaces outside `transport/`.
5. Wire implementations directly:

```go
srv := transport.New(log, transport.UserService(impl))
srv.WithLog()     // requires @tg log on an HTTP/JSON-RPC contract
srv.WithMetrics() // requires @tg metrics
if err := srv.Fiber().Listen(":8080"); err != nil {
    return err
}
```

There is no generated `contracts.NewUserService` wrapper; the option accepts the interface implementation.

6. Compile the whole module and smoke-test the selected transports.

## Choose generated surface

| Contract annotations | Generated behavior |
|----------------------|--------------------|
| `http-server` | REST handlers and routes |
| `jsonRPC-server` | JSON-RPC endpoint and batch support |
| both | shared implementation exposed over REST and RPC |
| `ws-server` + `stream` | WebSocket stream endpoint |
| `sse-server` + `stream=server` | SSE endpoint |
| `kafka` | nothing; use Kafka generator skills |

If a contract produces no handlers, inspect its resolved annotations before changing generator code.

## Runtime integration

- `SetFiberCfg`, `Use`, buffer/body/timeout options configure Fiber
- `MaxBatchSize` / `MaxBatchWorkers` apply to JSON-RPC batch
- `WithRequestID` / `WithHeader` integrate request metadata
- `srv.<Contract>().WithErrorHandler(...)` customizes REST error handling
- `ServeHealth` and `ServeMetrics` run separate endpoints
- `Shutdown()` performs graceful shutdown
- `WithTrace(...)` requires `@tg trace`

`WithLog`, `WithMetrics`, and `WithTrace` are generated for observable HTTP/JSON-RPC contracts. Do not assume pure WS/SSE or Kafka annotations activate the same Fiber hooks.

## Mapping and files

- `explicit` values are transport-only; `implicit` values are commonly supplied by middleware/context; `body` remains in the payload
- Header/cookie/query conversion is lenient; define a supported `Parse(string)` function when invalid values must fail explicitly
- One `io.Reader` / `io.ReadCloser` uses a streaming body
- Multiple streams or `http-multipart` use multipart; consume parts in order

Details: [references/runtime.md](references/runtime.md).

## Verify

```bash
go test ./...
```

Then verify:

- expected contract option exists (`transport.UserService`)
- expected REST path or JSON-RPC name (`userService.getUser`) is registered
- implementation still satisfies the contract after regeneration
- stream connection and cancellation close correctly
- file responses are closed by callers
- health/metrics servers stop during shutdown

## Diagnose

- Missing contract option/route — wrong family flag, filter, or stale model
- Interface mismatch — update handwritten implementation after contract change
- Zero value from header/query — parsing failed under lenient conversion
- Missing observability method — effective `log`/`metrics`/`trace` annotation or transport family is absent
- Wrong body/header shape — inspect mapping mode in `tgp-astg-json`

## Never

- Hand-edit generated files under `-o`
- Put handwritten implementations in the generated package
- Patch transport instead of fixing contracts and regenerating
- Expect Kafka contracts to appear in Fiber transport
- Expect handlers without transport enable flags

## Dig deeper

`tg plugin doc server` · skills `tgp-contracts`, `tgp-astg-json`
