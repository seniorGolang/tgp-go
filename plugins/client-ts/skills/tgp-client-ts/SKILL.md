---
name: tgp-client-ts
description: >-
  Generates TypeScript API clients from tgp contracts (REST/JSON-RPC/streams) and
  optional npm package metadata. Use when building or updating a browser/Node SDK,
  choosing RPC vs REST accessors, wiring auth/identity, batch, WS/SSE, Blob/FormData,
  publishing npm, or diagnosing path and signature problems.
---

# tgp-client-ts

## Workflow

1. Prepare contracts with `tgp-contracts`.
2. Inspect the resolved HTTP/RPC/stream model with `tgp-astg-json`.
3. Run from the repository root; all output paths are resolved from that root.
4. Generate:

```bash
tg client ts -o ./client-ts
# optional: --package-json=… --contracts=… --no-doc --no-client-id
```

When `go generate` starts in `contracts/`, return to module root:

```go
//go:generate sh -c "cd .. && tg client ts --out web --package-json web/package.json"
```

5. Compile the generated TypeScript and smoke-test the intended runtime.

## Choose generated API

| Need | API |
|------|-----|
| JSON-RPC / stream | `client.userService()` |
| REST | `client.userServiceHTTP()` |
| JSON-RPC batch | `client.batch(...)` + generated `req<Method>` |
| WS/SSE | generated async stream helpers |
| Kafka | not generated |

Use `newClient(endpoint, options)` and prefer generated methods over ad-hoc `fetch`.

## Auth and identity

- `headers` accepts a static object or sync/async function
- `clientName` controls `X-Client-Id`; a default identity is generated otherwise
- `--no-client-id` removes identity support and the header
- `idGeneratorFn` customizes JSON-RPC IDs
- `implicit` contract arguments may be hidden and supplied through shared headers

Browser WebSocket APIs cannot send arbitrary upgrade headers. Generated WS clients use query fallback for values the server also reads during upgrade.

## Files and streams

- One binary body/result maps to `Blob`
- Multiple binary values or `http-multipart` map to `FormData`
- WS/SSE streams use generated asynchronous helpers; pass `AbortSignal` to cancel (SSE: fetch + reader.cancel, WS: socket.close)
- REST/RPC failures are thrown; catch and narrow generated/custom error types

Detailed usage: [references/usage.md](references/usage.md).

## Publish npm package

1. Set package-level `npmName` and intentional version/license/author/private/registry annotations.
2. Generate with `--package-json <path>`.
3. Run the generated package build.
4. Inspect exports, generated declaration behavior, and package metadata.
5. Publish only after the consumer smoke test passes.

Without `npmName`, `--package-json` fails. For an internal source-only client, omit `--package-json`.

## Verify

- generated `tsconfig.json` compiles
- RPC and REST accessors match contract families
- async/static auth headers are sent
- browser WS works through query fallback where required
- `X-Client-Id` behavior matches server metrics/logging expectations
- Blob/FormData and stream `AbortSignal` cancellation work in the target browser/Node runtime

## Diagnose

- Output in the wrong directory — command ran from the wrong project root
- Missing package metadata — `npmName` or package annotations are absent
- Missing accessor/method — wrong transport family, filter, or stale model
- Auth missing only on browser WS — custom upgrade headers are unavailable; inspect query mapping
- Wrong result shape — check `enableInlineSingle`, result tags, and regenerated consumer types

## Never

- Patch generated TypeScript instead of contracts
- Mix server `transport/` output with TS client `-o`
- Publish without compiling and testing the generated package
- Expect Go `context.Context` in TypeScript method signatures
- Expect Kafka contracts in this client

## Dig deeper

`tg plugin doc client-ts` · skills `tgp-contracts`, `tgp-astg-json`, `tgp-server`
