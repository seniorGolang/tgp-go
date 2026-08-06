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

3. Generate (from repo root):

```bash
tg server -o transport
# optional: --contracts-dir=contracts --contracts=UserService,OrderService
# pinned model (requires astg-db): --from-db name@version
```

4. Implement exported contract interfaces **outside** `transport/`.
5. Wire implementations:

```go
srv := transport.New(log, transport.UserService(impl))
srv.WithLog()     // requires @tg log on an HTTP/JSON-RPC contract
srv.WithMetrics() // requires @tg metrics
if err := srv.Fiber().Listen(":8080"); err != nil {
    return err
}
```

There is no generated `contracts.NewUserService` wrapper; the option accepts the interface implementation.

6. Compile the module and smoke-test the selected transports.

## Choose generated surface

| Contract annotations | Generated behavior |
|----------------------|--------------------|
| `http-server` | REST handlers and routes |
| `jsonRPC-server` | JSON-RPC endpoint and batch (`POST /` + prefix mounts) |
| both | shared implementation over REST and RPC |
| `ws-server` + `stream` | WebSocket stream endpoint |
| `sse-server` + `stream=server` | SSE endpoint |
| `kafka` | nothing; use Kafka generator skills |

If a contract produces no handlers, inspect its resolved annotations before changing generator code.

## Gotchas

- Own lifecycle/business code outside `--out`; never patch generated files
- `SetFiberCfg` always forces `Immutable` (request strings safe past the handler)
- Header/query parse is **lenient** (failed parse → zero); add `Parse(string)` for strict types
- `implicit` args still need a wire source on the server (middleware/context)
- Multipart / multiple `io.Reader`: consume parts **in order** to EOF
- `WithLog` / `WithMetrics` / `WithTrace` follow HTTP/JSON-RPC observation annotations — not pure WS/SSE/Kafka alone
- JSON-RPC batch: headers/cookies come from batch/method context overlay, not a foreign Fiber Ctx
- `--from-db` skips `astg-hook` save in the same run

Full option catalog (Auth, limits, health, buckets, streams): [references/runtime.md](references/runtime.md) · `tg plugin doc server`.

## Verify

```bash
go test ./...
```

- [ ] Contract option exists (`transport.UserService`)
- [ ] Expected REST path or JSON-RPC name (`userService.getUser`) is registered
- [ ] Handwritten implementation still satisfies the contract
- [ ] Stream cancel/close behaves correctly
- [ ] Callers close file `io.ReadCloser` responses
- [ ] Health/metrics stop during `Shutdown` / `ShutdownWithTimeout`

## Diagnose

- Missing contract option/route — wrong family flag, filter, stale model, or missing `--from-db` pin
- Interface mismatch — update handwritten implementation after contract change
- Zero value from header/query — lenient conversion failed
- Missing observability method — effective `log`/`metrics`/`trace` or transport family absent
- Wrong body/header shape — mapping mode in `tgp-astg-json`

## Never

- Hand-edit generated files under `--out` / `-o`
- Put handwritten implementations in the generated package
- Patch transport instead of fixing contracts and regenerating
- Expect Kafka contracts in Fiber transport
- Expect handlers without transport enable flags

## Dig deeper

`tg plugin doc server` · [references/runtime.md](references/runtime.md) · skills `tgp-contracts`, `tgp-astg-json`, `tgp-astg-db`, `tgp-pipeline`
