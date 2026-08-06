# Persisting ASTG models

## Pipeline behavior

`astg-hook` is a **post** plugin (`Kind: post`, depends on `astg`). After a command
pipeline produces `project` from local analysis, the hook upserts `(projectKey, version)`
and writes the full snapshot under the contracts DB root (`@tg/astg/db`).

No CLI flags belong to the hook. Enablement is **package install**:

```bash
tg pkg add https://github.com/seniorGolang/tgp-go:astg-hook
```

## Commands that trigger a save

Any installed command that runs ASTG without `--from-db`, for example:

- `tg astg json -o .tg/project.json`
- `tg server -o …`
- `tg client go --out …` / `tg client ts --out …`
- `tg swagger --out …`
- `tg kafka pub go --out …` / `tg kafka sub go --out …`

## Commands that do not save

- Any invocation with `--from-db` (intentional skip)
- Pipelines where `project` was never set
- Runs where project key cannot be derived (empty remote and empty module path)

## Acceptance checklist

- [ ] `astg-hook` present in `tg pkg` / package scope
- [ ] Local parse completed without `--from-db`
- [ ] `tg astg json --from-db <key>@<version> -o .tg/saved.json` succeeds
- [ ] Contract names in the read-back match the intended surface
- [ ] Consumers received the exact ref string

## Failure modes

| Symptom | Likely cause |
|---------|----------------|
| Ref missing after “successful” generate | DB write failed at debug; or hook not installed |
| Wrong key guessed | Used short name instead of sanitized remote / module key |
| Branch pin changed underfoot | Same branch version overwritten on later save |
| Expected filtered snapshot in DB | `--contracts` never shrinks what hook stores |

Load with skill `tgp-astg-db`; inspect with `tgp-astg-json`.
