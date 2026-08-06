# Contract DB refs

## Ref forms

| Ref | Meaning |
|-----|---------|
| `` (empty `--from-db`) | Interactive project (and possibly contract) pick |
| `key` | Latest index entry for that project (mutable “latest”) |
| `key@version` | Pinned version |
| `key:ContractA,ContractB@version` | Pin + filter contracts in the loaded model |
| `key@version:ContractA,ContractB` | Same filter, alternate order |

Examples: `myapi@v1.0.1`, `github.com.acme.api@main`,
`github.com.acme.api:UserService,OrderService@v1.2.0`.

## Project key

Not an arbitrary short name. It is whatever `astg-hook` wrote:

1. Prefer sanitized Git remote (`https://github.com/org/repo.git` → `github.com.org.repo`)
2. Else module path (slashes → dots; single-segment keys may become `local.<name>`)

If a ref “is not found”, list interactively (`tg astg json --from-db`) or read back
after a known save — do not invent the key.

## Version identity

Hook version priority: **Git tag → branch → `default`**.

A branch name is mutable: each successful local save on that branch overwrites the
same `(key, version)` entry. Prefer a Git tag for immutable release pins.

## Filter layers

1. **Contracts in the ref** — `astg-db` filters the loaded snapshot (`FilterProject`).
   Related `services` follow; the `types` map is **not** trimmed.
2. **`--contracts` on a generator** — suppresses interactive DB contract UI; the
   command plugin filters its **copy** for generation. It does not rewrite the DB file.
3. **`all-contracts=true`** — keep full loaded model (default on `tg astg json`).

## Related

- Persist: skill `tgp-astg-hook`
- Inspect: `tg astg json --from-db …` / skill `tgp-astg-json`
