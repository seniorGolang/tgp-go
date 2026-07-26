# Kafka subscriber handlers

## Plain

Generated method shape:

```go
Method(ctx context.Context, event T) (err error)
```

Use only when processing does not need record identity, headers, partition, offset, or timestamp.

## Meta

Generated method shape:

```go
Method(ctx context.Context, event T, meta Meta) (err error)
```

`Meta` exposes key, headers, topic, partition, offset, and timestamp. Prefer this form for most event handlers because it preserves operational context without forcing batch processing.

## Slice

Generated method shape:

```go
Method(ctx context.Context, events []T) (err error)
```

Use for value-oriented batch work where per-record metadata is unnecessary. Only successfully decoded values are passed to the handler.

## Batch

Generated method shape:

```go
Method(ctx context.Context, batch Batch[T]) (err error)
```

`Batch[T]` contains records with both decoded value and `Meta`. Use when batching and per-record metadata are both required.

## Selection

All four interfaces/options are generated, but `New` accepts exactly one selected form per contract. Multiple forms create a configuration conflict; missing or nil handlers fail validation.

## Error behavior

Returning an error aborts dispatch and returns from `Run`. With default commit-after-batch, the failed batch is not successfully committed. Design idempotency and retry/dead-letter behavior in application infrastructure; do not hide failures inside generated code.

## Testing

Unit-test handler logic independently, then integration-test:

- decoding into `T`
- key/header extraction
- topic-to-contract dispatch
- batch boundaries
- handler error and offset behavior
