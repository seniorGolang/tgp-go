---
name: tgp-pipeline
description: >-
  Routes tgp multi-step work across contracts, model inspection, and generators
  (server, clients, swagger, kafka, init, DB snapshot/load). Use when the user
  asks to regenerate after contract changes, update all generated artifacts,
  choose which tgp skill/plugin to run, or combine several generators in one
  flow. Do not use for editing a single annotation in isolation when
  tgp-contracts alone suffices.
---

# tgp-pipeline

Pick the skill for the task, then follow that skill’s workflow. Do not invent
generator flags — each skill and `tg plugin doc <plugin>` is the source of truth.

## Route by intent

| Intent | Skill |
|--------|-------|
| Write / review `@tg` contracts and OpenAPI prose | `tgp-contracts` |
| Inspect resolved model (`jq`, local or DB) | `tgp-astg-json` |
| Bootstrap a **new** Go service repo | `tgp-init-go` (manual) |
| Fiber transport | `tgp-server` |
| Go SDK | `tgp-client-go` |
| TypeScript SDK / npm package | `tgp-client-ts` |
| OpenAPI file or Swagger UI | `tgp-swagger` |
| Kafka publisher | `tgp-kafka-pub-go` |
| Kafka subscriber | `tgp-kafka-sub-go` |
| Persist model to local DB | `tgp-astg-hook` (manual) |
| Generate from pinned DB model | `tgp-astg-db` |

## After contract edits (default loop)

Copy and track:

```
- [ ] 1. Edit contracts/DTO with tgp-contracts
- [ ] 2. tg astg json -o .tg/project.json — confirm family, mapping, types
- [ ] 3. Regenerate every affected surface (table below)
- [ ] 4. Compile / tsc / smoke-test consumers
- [ ] 5. Optional: persist snapshot (tgp-astg-hook) if a pin is required
```

### What to regenerate

| Contract change | Regenerate |
|-----------------|------------|
| `http-server` / paths / mapping / multipart | `tgp-server`, clients, `tgp-swagger` |
| `jsonRPC-server` / RPC shape | `tgp-server`, clients, `tgp-swagger` |
| `ws-server` / `sse-server` / `stream` | `tgp-server`, clients, `tgp-swagger` |
| `log` / `metrics` / `trace` | `tgp-server` and/or clients (observation options) |
| `security` | `tgp-server`, clients, `tgp-swagger` |
| docs only (`summary` / `desc` / `file:`) | `tgp-swagger` (clients if they embed docs) |
| `npm*` package meta | `tgp-client-ts` with `--package-json` |
| `@tg kafka` / topic / codec / acks | `tgp-kafka-pub-go` and/or `tgp-kafka-sub-go` |

Never hand-edit generated trees to “fix” a contract mistake — fix sources and re-run.

## Common combined flows

**New HTTP service from scratch**

1. `tgp-init-go` → 2. `tgp-contracts` → 3. `tgp-astg-json` → 4. `tgp-server` + `tgp-swagger` → 5. clients if needed.

**SDK-only consumer of existing contracts**

1. `tgp-astg-json` (or `tgp-astg-db` + `--from-db`) → 2. `tgp-client-go` or `tgp-client-ts`.

**Pinned generate without local contracts/**

1. Ensure snapshot exists (`tgp-astg-hook`) → 2. `tgp-astg-db` → 3. target generator with `--from-db key@version`.

**Wrong generation**

1. `tgp-astg-json` on the failing method/typeID → 2. fix via `tgp-contracts` → 3. regenerate the same plugin only → 4. broaden if wire is shared.

## Guardrails

- Prefer `tg astg json -o .tg/…` + `jq` over pasting full model JSON into chat
- `--from-db` skips hook save in the same run
- Kafka and HTTP-family annotations never share one interface
- Generator `--out` / `-o` rules differ by plugin (`tg plugin doc`)

## Dig deeper

`tg plugin doc astg` · skills listed above · `readme.md` scenarios in tgp-go
