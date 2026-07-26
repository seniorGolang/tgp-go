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
tg kafka pub go -o internal/publisher/kafka
# optional: --contracts OrderEvents,AuditEvents
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
- `kafka-key`: optional string/bytes-compatible key argument
- `kafka-headers`: optional compatible header arguments
- `kafka-codec`: `json` default; built-ins include `bytes`, `msgpack`, `cbor`, `yaml`, `xml`
- `kafka-acks`: `noAck`, `leaderAck`, or `allISRAcks` (default)

Message/key/header arguments must be distinct. Extra unused arguments are not published and should be removed rather than relying on generator warnings.

## Batch behavior

A message argument shaped as `[]T` or `...T` generates batch publishing:

- empty batch is a no-op
- every record is encoded before the first send
- encoding failure prevents sending that batch
- `[]byte` remains one byte message, not a batch

## Runtime decisions

- `Brokers` is required
- `Auth` and `SASL` must be configured together
- `TLS` configures broker TLS
- `Compression`, `BatchMaxLinger`, `BatchMaxBytes`, `MaxBufferedRecords` tune production
- `Codec` registers/overrides a named codec
- `Metrics` and `Trace` exist when effective annotations enable them
- `ClientOpt` is the escape hatch for franz-go options, excluding security already handled explicitly

Use acknowledgements and buffering intentionally for the required delivery/latency trade-off; do not change them merely to silence an operational problem.

## Verify

```bash
go test ./...
```

Confirm generated constructor/options/contract adapter exist and test:

- exact topic
- encoded message shape
- key and ordered headers
- custom codec error handling
- cancellation and publisher close
- metrics/traces when enabled

## Diagnose

- `no kafka contracts` — filter/model contains no `@tg kafka` interface
- topic required/duplicate — fix method annotations; topics are globally unique
- cannot resolve message — set `kafka-message` or remove ambiguous arguments
- bytes codec rejected — message is not byte-oriented
- unknown codec — register it with `Codec`
- `go.mod not found` — move `-o` inside a Go module
- Auth/SASL mismatch — configure both

Validation details: [references/validation.md](references/validation.md).

## Never

- Hand-edit generated publisher files
- Mix Kafka and HTTP-family annotations on one contract
- Generate publisher and subscriber into the same package
- Ignore an argument that is neither message, key, nor header
- Expect Kafka operations in Swagger/server/client generators

## Dig deeper

`tg plugin doc kafka-pub-go` · skills `tgp-contracts`, `tgp-astg-json`, `tgp-kafka-sub-go`
