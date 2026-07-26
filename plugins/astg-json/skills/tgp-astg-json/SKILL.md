---
name: tgp-astg-json
description: >-
  Exports and interprets the resolved tgp/astg project model as JSON
  (local parse or --from-db). Use when finding which contracts exist,
  classifying HTTP/JSON-RPC/WS/SSE/kafka surfaces, tracing types via typeID,
  reading resolved annotations/tags, comparing local vs DB versions, or
  debugging wrong generation without guessing from contracts/ sources.
  Do not use for editing contracts (tgp-contracts) or generating code.
---

# tgp-astg-json

## Quick start

```bash
tg astg json -o .tg/project.json   # preferred: file, then query — don't dump whole stdout into chat
tg astg json                       # stdout / pipe
```

Prefer this model over guessing from `contracts/`: types and references are resolved, while annotations remain visible at their original levels.

## Source selection

| Need | Command |
|------|---------|
| Current local | `tg astg json` *(default)* |
| DB version | `tg astg json --from-db name@version` |
| Latest DB | `tg astg json --from-db name` |
| Interactive | `tg astg json --from-db` |

Full model always (`all-contracts` defaults to `true`). `from-db` comes from plugin `astg-db`.

## Model map

Top-level keys that matter:

- `contracts[]` — interfaces: `name`, `id`, `filePath`, `annotations`, `methods`
- `types` — map `typeID` → type (`kind`, `typeName`, `structFields`, …)
- `annotations` — project-level `@tg`
- `services[]` — optional `contractIds` linkage
- `git` / `modulePath` / `version` — provenance

Method node: `name`, `annotations`, `args`/`results` (each has `name`, `typeID`, optional annotations), `errors`, `handler`.

## Find useful contracts

Do **not** load the whole JSON into context. Query the file:

```bash
# inventory
jq -r '.contracts[] | "\(.name)\t\(.filePath)\t\(.annotations|keys|join(","))"' .tg/project.json

# by transport family (contract-level enable flags)
jq -r '.contracts[] | select(.annotations["http-server"] != null) | .name' .tg/project.json
jq -r '.contracts[] | select(.annotations["jsonRPC-server"] != null) | .name' .tg/project.json
jq -r '.contracts[] | select(.annotations["ws-server"] != null or .annotations["sse-server"] != null) | .name' .tg/project.json
jq -r '.contracts[] | select(.annotations.kafka != null) | .name' .tg/project.json

# one contract + methods
jq '.contracts[] | select(.name=="UserService") | {annotations, methods: [.methods[]|{name, annotations, args, results}]}' .tg/project.json
```

Flag annotations often serialize as JSON `true` (empty DocTags value). Presence of the key = enabled.

## Interpret annotations

Parsed annotations live in `annotations` maps — not raw `// @tg` comments.

The JSON does **not** merge inherited annotations. Compute the effective value with the same cascade as generators (highest wins): arg/result → method sub-annotation → method → contract → project.
Sub-annotations appear as keys like `token.required`, `profile.tags` on the **method**.

Transport sense (read-only cheat sheet):

| Keys on contract | Means |
|---|---|
| `http-server` | REST generate |
| `jsonRPC-server` | JSON-RPC generate |
| `ws-server` / `sse-server` | stream servers |
| `kafka` | kafka pub/sub contracts |

| Keys on method | Means |
|---|---|
| `http-method`, `http-path`, `http-prefix` | REST route |
| `http-headers` / `http-cookies` / `http-args` | mapping `arg\|key\|mode` |
| `stream` | `server\|client\|bidi` |
| `kafka-topic` (+ optional key/headers/message/codec) | kafka method |

For authoring semantics and modes → skill `tgp-contracts`. Here: **verify what astg resolved**.

## Follow types

1. From arg/result take `typeID` (and `numberOfPointers` / `isSlice` if set).
2. Resolve `types[typeID]`.
3. Structs: `structFields[]` — `name`, nested `typeID`, `tags` (go struct tags), field `annotations`.
4. Named scalar enums: `enums[]` with `name` / `value` when the type has ≥2 typed package consts.
5. Nested/composites: follow `typeID` / `arrayOfID` / `mapKey`/`mapValue` / `aliasOf` / `underlyingTypeID`.

```bash
jq --arg id "…typeID…" '.types[$id]' .tg/project.json
```

## Workflows

**Inventory / find what to generate**

1. Export → inventory jq
2. Classify by family
3. Hand off to the matching generator skill

**Debug wrong generation**

1. Export local model
2. Open the failing contract/method annotations
3. Follow `typeID` for payload shape
4. Compare with expected `@tg` rules (`tgp-contracts`) — fix sources, re-export, then regenerate

**Local vs DB**

1. `tg astg json -o .tg/local.json`
2. `tg astg json --from-db name@version -o .tg/db.json`
3. Diff contract names / method annotations / type fields

## Never

- Substitute for `tgp-contracts` or generators
- Hand-edit DB `.astg` files — load via `--from-db`, save via `astg-hook`
- Confuse with skill `tgp-astg-db` (pipeline load for generate) — this skill is inspect/analyze
- Paste entire project JSON into the chat when a `jq` slice suffices

## Dig deeper

`tg plugin doc astg-json` · `tg plugin doc astg-db` · skill `tgp-contracts`
