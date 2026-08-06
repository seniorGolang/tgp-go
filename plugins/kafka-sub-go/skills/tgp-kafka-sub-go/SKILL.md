---
name: tgp-kafka-sub-go
description: >-
  Generates and wires a franz-go Kafka subscriber from @tg kafka contracts.
  Use when choosing plain/Meta/Slice/Batch handlers, configuring consumer group,
  offset reset and commit policy, TLS/SASL/codecs/metrics/tracing, or diagnosing
  decode, handler, poll, and commit failures. Do not use for publishers or HTTP.
---

# tgp-kafka-sub-go

## Workflow

1. Author a dedicated `@tg kafka` contract with `tgp-contracts`.
2. Inspect resolved topics/message/codecs with `tgp-astg-json`.
3. Generate into a package inside the target Go module:

```bash
tg kafka sub go --out internal/subscriber/kafka
# optional: --contracts OrderEvents,AuditEvents
# pinned model (requires astg-db): --from-db name@version
```

4. Implement one generated handler form per contract.
5. Construct and run:

```go
subscriber, err := kafka.New(log,
    kafka.Brokers("127.0.0.1:9092"),
    kafka.Group("orders-worker"),
    kafka.OrderEventsMeta(handler),
)
if err != nil {
    return err
}
defer subscriber.Close()

if err = subscriber.Run(ctx); err != nil {
    return err
}
```

6. Test decode, handler errors, cancellation, and offset policy with Kafka.

## Choose one handler form

| Option | Input | Use when |
|--------|-------|----------|
| `<Contract>` | one decoded event | metadata is irrelevant |
| `<Contract>Meta` | event + `Meta` | default choice; key/header/topic/partition/offset/timestamp matter |
| `<Contract>Slice` | `[]T` | process decoded values as one unit |
| `<Contract>Batch` | `Batch[T]` | preserve metadata for every record |

Registering none, nil, or more than one form for a contract is an error. Details: [references/handlers.md](references/handlers.md).

## Consumer decisions

- `Brokers` and `Group` are required
- `ResetOffset(AtStart|AtEnd)` applies only when no committed offset exists; default `AtStart`
- `CommitAfterBatch` (default, at-least-once) vs `CommitAuto` (at-most-once) — mutually exclusive
- `ShutdownTimeout` limits how long `Close` waits for `Run` (default 30s)
- `MaxPollRecords`, `FetchMinBytes`, `FetchMaxWait` tune polling
- `Codec` registers/overrides decoding
- `Auth` + `SASL` together; `TLS` independent
- `Metrics` / `LagInterval` / `Trace` when generated

Choose commit policy from processing semantics — do not use auto-commit to hide handler failures.

## Processing semantics

- `Run(ctx)` has one active loop; a concurrent second run is rejected
- Context cancellation stops polling
- Decode, handler, or commit error stops `Run`
- Under default commit-after-batch, offsets commit only after successful dispatch
- `Close()` stops the loop/client and observation workers

## Verify

```bash
go test ./...
```

- [ ] Handlers/options/subscriber exist
- [ ] Topic-to-method dispatch and codec behavior
- [ ] Metadata key/headers/offset for Meta/Batch
- [ ] Handler error prevents successful commit (default policy)
- [ ] Cancellation and `Close`
- [ ] Lag metrics and traces when enabled

## Diagnose

- `no kafka contracts` — filter/model contains no Kafka family
- handler required/conflict/nil — register exactly one valid form
- brokers/group required — configure both
- codec required — selected codec not registered/generated
- commit modes conflict — choose one policy
- Auth/SASL mismatch — configure both
- repeated `Run` — keep one lifecycle owner

## Never

- Hand-edit generated subscriber files
- Register multiple handler forms for one contract
- Swallow handler errors and assume offsets were committed
- Generate publisher and subscriber into the same package
- Mix Kafka and HTTP-family annotations

## Dig deeper

`tg plugin doc kafka-sub-go` · [references/handlers.md](references/handlers.md) · skills `tgp-contracts`, `tgp-astg-json`, `tgp-astg-db`, `tgp-kafka-pub-go`, `tgp-pipeline`
