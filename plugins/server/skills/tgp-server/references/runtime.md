# Server runtime patterns

## Lifecycle

1. Construct with `transport.New(log, options...)`.
2. Register each implementation through its generated contract option.
3. Add Fiber middleware and optional observation.
4. Configure separate health/metrics endpoints when required.
5. Start `srv.Fiber().Listen`.
6. On cancellation, call `srv.Shutdown()` and stop any separately retained health server.

Keep all lifecycle and business code outside the generated package.

## Observation

- `WithLog()` requires effective `log`
- `WithMetrics()` requires effective `metrics`
- `WithTrace(ctx, appName, endpoint, attributes...)` requires effective `trace`
- `ServeMetrics` exposes the generated Prometheus collectors
- Sensitive arguments/results need `log-skip` in the contract

Observation generation follows HTTP/JSON-RPC contracts; Kafka has separate generated runtime.

## Errors

For REST-capable contracts, use `srv.<Contract>().WithErrorHandler(handler)` to map or wrap application errors before transport encoding. JSON-RPC errors use their generated RPC representation.

Fix missing HTTP codes/types in contract or implementation analysis, regenerate, and test both transport representations.

## Streams

- WS uses one upgrade connection and connection-scoped header/query values
- SSE uses an HTTP request per server stream
- Browser WS clients cannot send arbitrary headers and may use query fallback
- Cancellation and channel closure are part of the generated stream protocol

## Files

- Single `io.Reader`: raw streaming request body
- Single `io.ReadCloser`: raw streaming response body
- Multiple streams or `http-multipart`: multipart
- Multipart readers are sequential; consume one part to EOF before the next
- The client owns closing returned `io.ReadCloser`

## Regeneration

Cleanup removes generated `.go`/`.ts` files carrying the Tool Gateway marker. It does not make mixed handwritten/generated packages a good ownership boundary. Keep custom code elsewhere and compile all consumers after every contract change.
