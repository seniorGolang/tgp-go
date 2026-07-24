---
name: tgp-astg-hook
description: >-
  Saves the current tgp project/contracts model into the local contracts DB after
  astg analysis. Use only when persisting contract versions for later from-db
  loads with astg-db — not during normal generate-from-sources workflows.
disable-model-invocation: true
---

# tgp-astg-hook

Manual (`/tgp-astg-hook`).

- Runs after `astg` when a project model exists; persists to local DB (project + version from git tag/branch or default).  
- **Skips save** if this run used `from-db` (avoids overwriting).  
- No user options. Requires `astg` in the pipeline.  
- Load later with skill `/tgp-astg-db` / option `from-db`.

`tg plugin doc astg-hook`
