---
name: tgp-astg-db
description: >-
  Loads a pinned tgp project model from the local contracts DB via --from-db for
  server, client, Swagger, or Kafka generation without reparsing sources. Use when
  generating from a saved snapshot, reproducing an older contract version, CI
  without contracts/, or when the user mentions from-db / pinned model. Do not use
  for editing contracts or for inspection-only exports (use tgp-astg-json).
---

# tgp-astg-db

Pre-plugin: supplies `project` before ASTG source parsing when `--from-db` is set
and no `project` is already in the request. Requires package `astg-db` installed.

## Choose a ref

| Ref | Meaning |
|-----|---------|
| `` | Interactive project pick |
| `name` / `name@version` | Project (+ version or latest) |
| `name:ContractA,ContractB@version` | Fixed contracts, no interactive pick |
| `name@version:ContractA,ContractB` | Equivalent fixed form |

Use an explicit `name@version` for reproducible scripts/CI. A bare project name
selects the latest index entry and is **not** a stable release pin.

Project key is whatever `astg-hook` stored (sanitized git remote, else module path) —
see [references/refs.md](references/refs.md).

## Workflow

1. Ensure the model was saved previously (`tgp-astg-hook`).
2. Inspect the ref before generation:

```bash
tg astg json --from-db name@version -o .tg/db.json
```

3. Pass the same `--from-db` to the generator (requires `astg-db` in the package):

```bash
tg server -o transport --from-db name@version
tg client go --out ./client --from-db name@version
tg client ts --out ./client-ts --from-db name@version
tg swagger --out ./openapi/openapi.yaml --from-db name@version
tg kafka pub go --out internal/publisher/kafka --from-db name@version
tg kafka sub go --out internal/subscriber/kafka --from-db name@version
```

4. Continue with the matching generator skill for wiring/verify.
5. In the same run, `astg-hook` will **not** overwrite the DB (skip on `from-db`).

## Selection semantics

- Empty `--from-db` selects project/version interactively.
- Contracts embedded in the ref filter the loaded DB model immediately.
- `--contracts` suppresses DB interactive selection; command-level filtering belongs
  to the target command — not identical to contracts embedded in the ref.
- `all-contracts=true` skips contract selection and keeps the full model (`astg-json`).
- If a project is already present in the request, `astg-db` does nothing.

## Diagnose

- Empty DB / cancelled choice — persist a source model first or supply a valid ref
- Unknown project/version — interactive list; check hook-derived key/version
- Expected contract missing — unfiltered saved model, then ref contracts / `--contracts`
- Local sources used instead — confirm `--from-db` reached the pipeline
- DB model ignored — another plugin already supplied `project`

## Never

- Use an unpinned “latest” ref for release generation
- Hand-edit `.astg` files or the DB index
- Expect this skill to change annotations or source contracts
- Confuse DB loading with inspection-only: use `tgp-astg-json` when no generator runs

## Dig deeper

`tg plugin doc astg-db` · [references/refs.md](references/refs.md) · skills `tgp-astg-hook`, `tgp-astg-json`, `tgp-pipeline`
