---
name: tgp-init-go
description: >-
  Bootstraps a new Go service module with tgp contracts, DTO, service stubs,
  generated Fiber transport, OpenAPI, configuration, health endpoints, and
  error types. Use when starting a new repository and choosing REST, JSON-RPC,
  or both. Do not use to add contracts to an existing service.
disable-model-invocation: true
---

# tgp-init-go

Manual skill (`/tgp-init-go`). One-time bootstrap, not a daily generator.

## Choose the scaffold

| Need | Options | Result |
|------|---------|--------|
| JSON-RPC only | `--json-rpc users,orders` | one `Do` stub per contract |
| REST only | `--rest users,orders` | CRUD contracts, DTO, and service stubs |
| Both | both options | both transport surfaces |
| Minimal default | neither option | one JSON-RPC contract named `Some` |

Names become PascalCase Go interfaces and snake_case files (`siteNova` → `SiteNova` / `site_nova.go`).

## Preflight

1. Use a new directory or one containing no `.go` files (non-Go files are OK).
2. Choose the final Go module path; changing it after generation causes unnecessary import rewrites.
3. Do not point `--out` at an existing service repository.

## Generate

```bash
tg init go --module <module> [--out <dir>] [--json-rpc a,b] [--rest c,d]
```

The command writes the scaffold, then runs `go generate ./...` and `go mod tidy` itself. Do not repeat those as mandatory initialization steps.

## Ownership after generation

- `contracts/` and `contracts/dto/` — API source of truth; edit with skill `tgp-contracts`
- `internal/services/` — handwritten implementations; replace `NotImplemented` stubs
- `internal/transport/` and `api/swagger.json` — generated; regenerate, never patch
- `cmd/<module>/`, `internal/config/`, `pkg/errs/` — application/runtime code
- `.golangci.yml`, `.gitignore` — project tooling defaults; adjust locally if needed

`contracts/dto/` is created only for REST scaffolds. Kafka is **not** part of init — add later via kafka skills.

## Verify

From the generated module:

```bash
go build ./...
go test ./...
```

Confirm that `go.mod`, `.golangci.yml`, `contracts/tg.go`, `internal/services/`, `internal/transport/`, and `api/swagger.json` exist. Expected `NotImplemented` errors belong only to unimplemented stubs. Health uses `transport.ServeHealth`; process lifecycle uses `signal.NotifyContext` and `srv.ShutdownWithTimeout(cfg.ShutdownTimeout)` (`SHUTDOWN_TIMEOUT`, default `10s`).

## Continue

1. Design real API methods and DTO with `tgp-contracts`.
2. Implement interfaces in `internal/services/`.
3. Regenerate server/OpenAPI after contract changes (`tgp-pipeline`).
4. Add `tgp-client-go` / `tgp-client-ts` when consumers need an SDK.
5. Add Kafka pub/sub skills only when event contracts exist.

## Troubleshoot

- `directory is not empty` — rejects directories that already contain `.go` files (message may say “not empty” even when only non-Go files exist)
- generation failure — inspect nested `go generate` error and plugin availability
- build failure in a service — update handwritten implementation; do not patch transport

## Never

- Run over an existing service to “add one contract”
- Hand-edit generated transport or OpenAPI
- Treat placeholder CRUD/`Do` methods as the final domain API

## Dig deeper

`tg plugin doc init-go` · skills `tgp-contracts`, `tgp-pipeline`
