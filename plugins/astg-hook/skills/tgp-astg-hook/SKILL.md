---
name: tgp-astg-hook
description: >-
  Persists the current ASTG project model in the local contracts DB for later
  pinned generation with astg-db. Use when publishing a contract snapshot,
  preserving a branch/tag model, or preparing consumers that do not have source
  contracts. Do not use for loading or editing stored models.
disable-model-invocation: true
---

# tgp-astg-hook

Manual (`/tgp-astg-hook`). This is a post-plugin with no user options.

## When to persist

- Publish a contract version for another repository
- Preserve the model used to generate a release
- Compare branch/tag contract surfaces
- Re-run generators later without cloning source contracts

Do not snapshot every exploratory edit without a versioning reason.

## Stored identity

- Version priority: Git tag → branch → `default`
- Without Git: module path + `default`
- Project key: normalized remote URL; fallback to module path
- Reusing the same project key/version updates that stored entry; a branch name is not an immutable release identifier

## Workflow

1. Build the project model from local sources (not `--from-db`).
2. Confirm its contract inventory with `tgp-astg-json`.
3. Run a pipeline/package configuration that includes `astg-hook`; it saves after ASTG.
4. Verify the save explicitly:

```bash
tg astg json --from-db project@version -o .tg/saved.json
```

5. Compare contract names, methods, annotations, types, and Git provenance with the local export.
6. Give consumers the explicit `project@version` or `project:Contracts@version` ref.

## Silent skips and failures

The hook deliberately returns without saving when:

- request has no `project`
- request contains `from-db`
- no project key can be derived

DB root/index/write failures are logged at debug level and may not fail the outer command. Treat successful read-back through `astg-json --from-db` as the acceptance check; enable debug logging when the ref does not appear.

## Never

- Expect a `from-db` run to save another copy
- Treat a mutable branch ref as an immutable release
- Hand-edit DB `.astg` files or index data
- Report success before read-back verification

Load with `tgp-astg-db`; inspect with `tgp-astg-json`.

`tg plugin doc astg-hook`
