---
name: tgp-client-go
description: >-
  Generates typed Go API clients (JSON-RPC, REST, WS/SSE streams) from tgp contracts.
  Use when creating/updating a Go SDK, calling generated Client methods, Batch RPC,
  or regenerating client code after contract changes.
---

# tgp-client-go

## Do

1. Contracts marked with `http-server` and/or `jsonRPC-server` (streams need `ws-server`/`sse-server`).
2. Generate (clears previous generated files in `out`):

```bash
tgp client go -o ./client
# optional: --contracts=… --doc-file=… --no-doc
```

3. Use:

```go
c := client.New(
    client.URL("https://api.example.com"),
    // client.LogRequest(), client.WithMetrics(), …
)
user, err := c.UserService().GetUser(ctx, id)
```

JSON-RPC batch: `c.Batch(…)` when JSON-RPC contracts exist.

4. Contract change → regenerate client; do not hand-edit `client/` generated sources.

## Never

- Reimplement HTTP/RPC manually when a generated client exists  
- Point `-o` at `transport/` (server package)  

## Dig deeper

`tg plugin doc client-go` · skill `tgp-contracts`
