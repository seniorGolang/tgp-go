---
name: tgp-astg-db
description: >-
  Loads a previously saved tgp project model from the local contracts DB via
  from-db (ref like project@v1 or interactive). Use only when generating from a
  stored contracts database instead of parsing local contracts/ sources.
disable-model-invocation: true
---

# tgp-astg-db

Manual (`/tgp-astg-db`). Pipeline option: `from-db`.

| Ref | Meaning |
|-----|---------|
| `` | Interactive project pick |
| `name` / `name@version` | Project (+ version or latest) |
| `name:ContractA,ContractB@version` | Fixed contracts, no interactive pick |

- Does nothing if project model already in the request or `from-db` omitted.  
- Pair with `astg-hook` which **saves** models; this plugin **loads**.  
- After load, run generators as usual (`tgp-server`, clients, …).

`tg plugin doc astg-db`
