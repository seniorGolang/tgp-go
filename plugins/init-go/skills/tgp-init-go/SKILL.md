---
name: tgp-init-go
description: >-
  Scaffolds a new Go module with tgp contracts and service stubs via tg init go.
  Use only when bootstrapping a new service repository from scratch — not for
  day-to-day contract or generator work.
disable-model-invocation: true
---

# tgp-init-go

Manual skill (`/tgp-init-go`). Do not use inside an existing app unless the user asks to scaffold a new module.

## Command

```bash
tg init go --module <module> [--out <dir>] [--json-rpc a,b] [--rest c,d]
```

- `--out` must be empty of `.go` files (or not exist).  
- Then in the project: `go generate ./...` for transport/OpenAPI as the scaffold documents.  
- Next: edit `contracts/`, use skill `tgp-contracts`, generate with `tgp-server` / clients / swagger.

## Dig deeper

`tg plugin doc init-go`
