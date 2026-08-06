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

Manual (`/tgp-astg-hook`). Post-plugin with no user options: after a local ASTG
parse produces `project`, it writes the **full** snapshot to the contracts DB.

## Install

```bash
tg pkg add https://github.com/seniorGolang/tgp-go:astg-hook
tg pkg add https://github.com/seniorGolang/tgp-go:astg-db   # needed to load later
```

Once installed, hook runs automatically on any pipeline that builds `project` from
**local** sources (no `--from-db`): `tg astg json`, `tg server`, `tg client …`,
`tg swagger`, `tg kafka …`.

## When to persist

- Publish a contract version for another repository / CI without `contracts/`
- Preserve the model used to generate a release (prefer a **Git tag** as version)
- Compare branch/tag surfaces later via `tgp-astg-json --from-db`

Do not snapshot every exploratory edit without a versioning reason.

## How to save (concrete)

1. Ensure `astg-hook` is installed in the tg package (see Install).
2. Work from a repo with resolvable Git remote or Go module path (forms the project key).
3. Prefer tagging the release commit before save when you need an immutable pin.
4. Run a **local** parse (any of these triggers hook after ASTG):

```bash
tg astg json -o .tg/project.json
# or any generator without --from-db, e.g.:
# tg server -o transport
```

5. **Acceptance = read-back**, not “generator exited 0”:

```bash
tg astg json --from-db -o .tg/saved.json          # interactive: pick what hook wrote
# once key/version known:
tg astg json --from-db <key>@<version> -o .tg/saved.json
jq -r '.contracts[] | .name' .tg/saved.json
```

6. Hand consumers `key@version` or `key:Contracts@version` (see `tgp-astg-db` refs).

## Stored identity

| Source | Project key | Version |
|--------|-------------|---------|
| Git remote | sanitized URL (`host.org.repo`) | tag → else branch → else `default` |
| Git, no usable remote | module path | same version rule |
| No Git | module path (`local.` prefix possible) | `default` |

- Same key+version **overwrites** the stored entry
- Snapshot is always the full project; generator `--contracts` does not shrink the save
- `--from-db` on the same run → hook **skips** (will not re-save the loaded model)

Details: [references/persist.md](references/persist.md).

## Silent skips and failures

Hook returns without saving when:

- request has no `project`
- request contains `from-db`
- no project key can be derived

DB root/index/write failures are logged at **debug** and may not fail the outer
command. Treat successful read-back as the only acceptance check; enable host
debug logging when the ref does not appear.

## Never

- Expect a `from-db` run to save another copy
- Treat a mutable branch ref as an immutable release
- Hand-edit DB `.astg` files or index data
- Report success before read-back verification

## Dig deeper

`tg plugin doc astg-hook` · [references/persist.md](references/persist.md) · skills `tgp-astg-db`, `tgp-astg-json`, `tgp-pipeline`
