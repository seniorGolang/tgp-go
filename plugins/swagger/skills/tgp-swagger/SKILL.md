---
name: tgp-swagger
description: >-
  Generates OpenAPI/Swagger from tgp contracts and can serve Swagger UI. Use when
  updating or reviewing OpenAPI JSON/YAML, serving Swagger UI, diagnosing missing
  paths/parameters/schemas/errors/security/empty summaries or descriptions, or
  checking API compatibility after contract changes. Do not use for Kafka, writing
  contract docs (use tgp-contracts), or as the source of contract edits.
---

# tgp-swagger

## Workflow

1. Prepare transport and documentation annotations with `tgp-contracts` (API prose: that skill's `references/documentation.md`).
2. Inspect the resolved HTTP-family model:

```bash
tg astg json -o .tg/project.json
```

3. Choose output mode:

```bash
tg swagger --out ./openapi/openapi.yaml
tg swagger --serve :8080
tg swagger --out ./openapi/openapi.json --serve :8080
```

Format follows `.json`, `.yaml`, or `.yml`. With neither `--out` nor `--serve`, Swagger UI starts on `:8080`.

4. Validate the generated document and review its diff.

## Included surface

Only HTTP-family contracts appear:

- `http-server`: REST operations
- `jsonRPC-server`: JSON-RPC POST operations
- `ws-server`: `x-websocket`
- `sse-server`: `text/event-stream`

Kafka contracts are intentionally omitted. Contract filters support explicit names and `!Name` exclusions.

## Review the artifact

Check:

- `openapi: 3.0.0`
- expected `paths` and HTTP methods
- stable, unique operation IDs
- request/response schemas and `components.schemas`
- path/query/header/cookie placement
- required fields and nullable/optional intent
- success and application error responses
- package-level title/version/servers/security
- operation tags, summaries, descriptions, deprecation
- non-empty `info.description`, operation `summary`/`description`, and schema/parameter descriptions for the published surface
- WS/SSE markers and content types

Empty or missing prose → fix contract sources via `tgp-contracts` documentation rules, not the generated file.

Detailed rules: [references/validation.md](references/validation.md).

## Diagnose

- Empty/missing path — contract lacks HTTP-family marker, was filtered, or is Kafka-only
- Missing package metadata — inspect project-level annotations (`title`/`version`/`desc`)
- Empty operation summary/description or schema descriptions — add `@tg summary`/`desc` (or `file:`) in contracts; see `tgp-contracts` documentation reference
- Header/cookie/query shown in body — effective mapping mode is `body`
- Wrong `required` — review request/response/query/header inference, not only explicit `required`
- Generic object schema — type has custom marshaling or no representable field model
- Missing error response — ASTG method errors do not contain that HTTP code/type
- Abstract success response — method uses custom `http-response`

Fix contract sources or implementation-visible error types, export ASTG again, then regenerate.

## Compatibility workflow

1. Generate to the committed canonical path.
2. Diff paths, parameters, request/response schemas, and security.
3. Classify removals, stricter requirements, and incompatible type changes as breaking.
4. Regenerate clients/server after accepted contract changes.
5. Import into Postman/Insomnia only after the canonical spec passes review.

## Never

- Hand-edit generated OpenAPI as source of truth
- Expect Kafka topics in OpenAPI
- Invent default 400/401/404/500 responses not present in the model
- Treat Swagger UI rendering alone as semantic validation
- Duplicate the full annotation catalog or documentation style guide here; use `tgp-contracts`

## Dig deeper

`tg plugin doc swagger` · skills `tgp-contracts` (incl. documentation reference), `tgp-astg-json`
