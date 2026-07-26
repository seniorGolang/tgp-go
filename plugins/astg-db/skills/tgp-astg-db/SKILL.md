---
name: tgp-astg-db
description: >-
  Loads a pinned or interactively selected tgp project model from the local
  contracts DB for server, client, Swagger, or Kafka generation. Use when local
  contract sources are unavailable, when reproducing an older contract version,
  or when a pipeline must generate from a saved model. Do not use for editing or
  merely inspecting contracts.
disable-model-invocation: true
---

# tgp-astg-db

Manual (`/tgp-astg-db`). This is a pre-plugin: it supplies `project` before ASTG source parsing.

## Choose a ref

| Ref | Meaning |
|-----|---------|
| `` | Interactive project pick |
| `name` / `name@version` | Project (+ version or latest) |
| `name:ContractA,ContractB@version` | Fixed contracts, no interactive pick |
| `name@version:ContractA,ContractB` | Equivalent fixed form |

Use interactive selection for local exploration. Use an explicit version and contract list for reproducible scripts/CI. A bare project name selects the latest index entry and is therefore not a stable release pin.

## Workflow

1. Ensure the model was saved previously by `tgp-astg-hook`.
2. Choose a deterministic ref when the result will be committed or published.
3. Pass `--from-db <ref>` to the target generator command.
4. Before generation, inspect the same ref:

```bash
tg astg json --from-db name@version -o .tg/db.json
```

5. Verify provenance, contract names, methods, and key type/annotation changes.
6. Run the matching `tgp-server`, client, Swagger, or Kafka workflow.

## Selection semantics

- Empty `--from-db` selects project/version interactively.
- A ref without contracts may trigger contract selection.
- Contracts embedded in the ref filter the loaded DB model immediately.
- `--contracts` suppresses DB interactive selection, but command-level filtering belongs to the target command; do not treat it as identical to contracts embedded in the ref.
- `all-contracts=true` skips contract selection and keeps the full model (used by `astg-json`).
- If a project is already present in the request, `astg-db` does nothing.

## Diagnose

- Empty DB / cancelled choice — first persist a source model or supply a valid explicit ref
- Unknown project/version — inspect the interactive list; check how hook derived project key and version
- Expected contract missing — inspect the unfiltered saved model, then check ref contracts and target `--contracts`
- Local changes unexpectedly used — confirm the command actually received `--from-db`
- DB model unexpectedly ignored — another plugin already supplied `project`

## Never

- Use an unpinned “latest” ref for reproducible release generation
- Hand-edit `.astg` files or the DB index
- Expect this skill to change annotations or source contracts
- Confuse DB loading with inspection: use `tgp-astg-json` when no generator is needed

`tg plugin doc astg-db`
