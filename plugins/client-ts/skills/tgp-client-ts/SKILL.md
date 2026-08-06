---
name: tgp-client-ts
description: >-
  Generates TypeScript API clients from tgp contracts (REST/JSON-RPC/streams) and
  optional npm package metadata. Use when building or updating a browser/Node SDK,
  choosing RPC vs REST accessors, wiring auth/identity, batch, WS/SSE, Blob/FormData,
  publishing npm, or diagnosing path and signature problems. Do not use for
  authoring contracts or server transport.
---

# tgp-client-ts

## Workflow

1. Prepare contracts with `tgp-contracts`.
2. Inspect the resolved HTTP/RPC/stream model with `tgp-astg-json`.
3. Run from the repository root; all output paths are resolved from that root.
4. Generate:

```bash
tg client ts --out ./client-ts
# optional: --package-json=… --contracts=… --no-doc --no-client-id
# pinned model (requires astg-db): --from-db name@version
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
| form / CSV slices | `form.ts` helpers |
| Kafka | not generated |

Use `newClient(endpoint, options)`. Bearer credentials go through `headers` (object or sync/async fn) — there is no Go-style `Auth`/`token` package.

## Gotchas

- Only `--out` (no `-o`); wrong cwd (`go generate` from `contracts/`) writes to the wrong place
- REST/SSE/WS join endpoint path with the declared route; overlapping segments collapse (keeps mounts, no doubled `http-prefix`)
- Browser WebSocket cannot set arbitrary upgrade headers; generated clients use query fallback
- `implicit` args may be hidden; supply via shared `headers`
- Without `npmName`, `--package-json` fails; omit the flag for source-only clients
- `--from-db` skips `astg-hook` save in the same run

Details: [references/usage.md](references/usage.md).

## Auth and identity

- `headers` accepts a static object or sync/async function
- `clientName` controls `X-Client-Id`; a default identity is generated otherwise
- `--no-client-id` removes identity support and the header
- `idGeneratorFn` customizes JSON-RPC IDs

## Files and streams

- One binary body/result maps to `Blob`
- Multiple binary values or `http-multipart` map to `FormData`
- WS/SSE: pass `AbortSignal` to cancel and optional `onReady` after transport is up
- REST/RPC failures throw; catch and narrow at the app boundary

## Publish npm package

1. Set package-level `npmName` and intentional version/license/author/private/registry annotations.
2. Generate with `--package-json <path>`.
3. Run the generated package build.
4. Inspect exports and package metadata.
5. Publish only after the consumer smoke test passes.

## Verify

- [ ] `npx tsc -p <out>/tsconfig.json` (or package `npm run build`) succeeds
- [ ] RPC and REST accessors match contract families
- [ ] async/static auth headers are sent
- [ ] browser WS works through query fallback where required
- [ ] `X-Client-Id` behavior matches server metrics/logging expectations
- [ ] Blob/FormData and stream `AbortSignal` cancellation work in the target runtime

## Diagnose

- Output in the wrong directory — command ran from the wrong project root
- Missing package metadata — `npmName` or package annotations are absent
- Missing accessor/method — wrong transport family, filter, or stale model
- Auth missing only on browser WS — custom upgrade headers unavailable; inspect query mapping
- Wrong result shape — `enableInlineSingle`, result tags, regenerated consumer types

## Never

- Patch generated TypeScript instead of contracts
- Mix server `transport/` output with TS client `--out`
- Publish without compiling and testing the generated package
- Expect Go `context.Context` in TypeScript method signatures
- Expect Kafka contracts in this client

## Dig deeper

`tg plugin doc client-ts` · [references/usage.md](references/usage.md) · skills `tgp-contracts`, `tgp-astg-json`, `tgp-astg-db`, `tgp-pipeline`
