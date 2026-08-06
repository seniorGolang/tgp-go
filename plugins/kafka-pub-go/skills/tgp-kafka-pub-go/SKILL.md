---
name: tgp-kafka-pub-go
description: >-
  Generates and wires a franz-go Kafka publisher from @tg kafka contracts.
  Use when publishing typed events, choosing message/key/headers, codecs and
  acknowledgements, configuring TLS/SASL/metrics/tracing, or diagnosing Kafka
  contract and generated adapter failures. Do not use for subscribers or HTTP.
---

# tgp-kafka-pub-go

## Workflow

1. Author a dedicated `@tg kafka` contract with `tgp-contracts`.
2. Inspect resolved Kafka methods before generation:

```bash
tg astg json -o .tg/project.json
```

3. Generate into a package inside the target Go module:

```bash
tg kafka pub go --out internal/publisher/kafka
# optional: --contracts OrderEvents,AuditEvents
# pinned model (requires astg-db): --from-db name@version
```

4. Construct, use, and close the publisher:

```go
publisher, err := kafka.New(log, kafka.Brokers("127.0.0.1:9092"))
if err != nil {
    return err
}
defer publisher.Close()

if err = publisher.OrderEvents().OrderCreated(ctx, orderID, event); err != nil {
    return err
}
```

5. Compile the module and integration-test record topic/key/headers/value.

## Contract decisions

- Interface: `@tg kafka` only; never combine with HTTP/JSON-RPC/WS/SSE/stream
- Method: unique `kafka-topic`, first argument `context.Context`, only `error` result
- `kafka-message`: message argument; otherwise it must be unambiguous
- `kafka-key` / `kafka-headers`: optional compatible arguments (distinct from message)
- `kafka-codec`: `json` default; built-ins include `bytes`, `msgpack`, `cbor`, `yaml`, `xml`
- `kafka-acks`: `noAck`, `leaderAck`, or `allISRAcks` (default)

Extra unused arguments are not published — remove them rather than relying on warnings.

## Batch behavior

Message `[]T` or `...T` → batch publish: empty batch is a no-op; encode all records before first send; encoding failure prevents the batch; `[]byte` is one byte message, not a batch.

## Runtime decisions

- `Brokers` is required
- `Auth` and `SASL` must be configured together; `TLS` is independent
- `Compression`, `BatchMaxLinger`, `BatchMaxBytes`, `MaxBufferedRecords` tune production
- `ShutdownTimeout` limits wait for in-flight publishes on `Close` (default **30s**)
- `Codec` registers/overrides a named codec
- `Metrics` / `Trace` when effective annotations enable them
- `ClientOpt` escape hatch for franz-go options (excluding security handled explicitly)

## Verify

```bash
go test ./...
```

- [ ] Constructor/options/contract adapter exist
- [ ] Exact topic, encoded message, key, ordered headers
- [ ] Custom codec error handling
- [ ] Cancellation and publisher close (`ShutdownTimeout`)
- [ ] Metrics/traces when enabled

## Diagnose

- `no kafka contracts` — filter/model has no `@tg kafka` interface
- topic required/duplicate — fix method annotations; topics are globally unique
- cannot resolve message — set `kafka-message` or remove ambiguous arguments
- bytes codec rejected — message is not byte-oriented
- unknown codec — register with `Codec`
- `go.mod not found` — move `--out` inside a Go module
- Auth/SASL mismatch — configure both

Validation details: [references/validation.md](references/validation.md).

## Never

- Hand-edit generated publisher files
- Mix Kafka and HTTP-family annotations on one contract
- Generate publisher and subscriber into the same package
- Ignore an argument that is neither message, key, nor header
- Expect Kafka operations in Swagger/server/client generators

## Dig deeper

`tg plugin doc kafka-pub-go` · skills `tgp-contracts`, `tgp-astg-json`, `tgp-astg-db`, `tgp-kafka-sub-go`, `tgp-pipeline`
