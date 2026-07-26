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
tg kafka sub go -o internal/subscriber/kafka
# optional: --contracts OrderEvents,AuditEvents
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
- `ResetOffset(AtStart|AtEnd)` applies only when no committed offset exists; default is `AtStart`
- Default behavior commits after a successfully dispatched batch
- `CommitAuto` enables franz-go auto-commit instead
- `CommitAfterBatch` and `CommitAuto` are mutually exclusive
- `MaxPollRecords`, `FetchMinBytes`, `FetchMaxWait` tune polling
- `Codec` registers/overrides message decoding
- `Auth` and `SASL` must be configured together; `TLS` is independent
- `Metrics` enables consumer metrics; `LagInterval` controls lag refresh
- `Trace` enables handler spans when generated

Choose commit policy from processing semantics. Do not use auto-commit merely to hide handler or commit failures.

## Processing semantics

- `Run(ctx)` has one active loop and rejects a concurrent second run
- Context cancellation stops polling; handle `context.Canceled` as expected shutdown where appropriate
- Decode or handler error stops `Run`
- Under default commit-after-batch, offsets are committed only after successful dispatch
- Commit failure stops `Run`
- `Close()` stops the loop/client and observation workers

## Verify

```bash
go test ./...
```

Confirm generated handlers/options/subscriber exist and test:

- topic-to-method dispatch
- codec success/failure
- metadata key/headers/offset
- selected plain/Slice/Batch semantics
- handler error prevents successful commit
- cancellation and `Close`
- consumer lag metrics and traces when enabled

## Diagnose

- `no kafka contracts` — filter/model contains no Kafka family
- handler required/conflict/nil — register exactly one valid form
- brokers/group required — configure both
- codec required — selected contract codec is not registered/generated
- commit modes conflict — choose one policy
- Auth/SASL mismatch — configure both
- repeated `Run` — keep one lifecycle owner
- Kafka contract validation — fix via `tgp-contracts`, inspect via `tgp-astg-json`

## Never

- Hand-edit generated subscriber files
- Register multiple handler forms for one contract
- Swallow handler errors and assume offsets were committed
- Generate publisher and subscriber into the same package
- Mix Kafka and HTTP-family annotations

## Dig deeper

`tg plugin doc kafka-sub-go` · skills `tgp-contracts`, `tgp-astg-json`, `tgp-kafka-pub-go`
