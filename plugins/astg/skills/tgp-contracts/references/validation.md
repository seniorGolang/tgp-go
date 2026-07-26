# Contract validation

Use this reference when a contract is missing from the model or a generator rejects it.

## Discovery

A contract interface must:

- be exported
- live in a flat `.go` file directly under `contracts-dir`
- carry an `@tg` annotation
- not rely on interface embedding

DTO and imported types may live in nested packages because type resolution uses `go/types`.

If a contract is missing, check `contracts-dir`, filename location, interface visibility, and annotations before changing generator code. Use `--no-cache` only when diagnosing stale ASTG input; normally cache invalidation follows relevant Go files and module metadata.

## Signatures

- First argument is `context.Context`
- All arguments and results are named except `context` and `error`
- `error` is the last result
- `io.Reader` / `io.ReadCloser` are HTTP-only
- Form-urlencoded body values require `form:<name>` tags

## HTTP

- `http-method`: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, or `OPTIONS`
- `http-success`: positive integer
- Path placeholders must map to existing arguments
- Header/cookie/query mappings must reference existing arguments/results
- `handler` and `http-response` targets must resolve

## Streams

- `stream=server`: server output channel; valid for WS or SSE
- `stream=client`: client input channel; WS only
- `stream=bidi`: input and output channels; WS only
- Stream methods must not use `http-method`
- `sse-server` supports only server streams

## Kafka

- Use `kafka`; legacy `kafka-consumer` / `kafka-publisher` are rejected
- Do not combine with HTTP/JSON-RPC/WS/SSE/stream annotations
- Every method needs a globally unique `kafka-topic`
- Methods return only `error`
- Message/key/header annotations must reference compatible arguments
- `kafka-codec=bytes` requires byte-oriented messages

## Feedback loop

1. Fix the source interface or DTO.
2. Export with `tg astg json -o .tg/project.json`.
3. Inspect the affected contract, method, and `typeID` chain.
4. Run the matching generator.
5. Repeat until validation and consumer compilation pass.
