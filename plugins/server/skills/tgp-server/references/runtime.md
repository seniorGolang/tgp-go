# Server runtime patterns

## Contents

- Lifecycle
- Construction options
- Auth
- Observation
- Errors
- Streams
- Files
- Regeneration

## Lifecycle

1. Construct with `transport.New(log, options...)`.
2. Register each implementation through its generated contract option.
3. Add Fiber middleware and optional observation.
4. Configure separate health/metrics endpoints when required. Use `ServeHealth(..., WithCheck(...))` for readiness probes that must fail with `503` when dependencies are down; use `WithResponse(...)` when the success body should not be the default `"ok"`.
5. Start `srv.Fiber().Listen`.
6. On cancellation, call `srv.ShutdownWithTimeout(timeout)` (or `srv.Shutdown()` for the default 30s budget) and stop any separately retained health server.

Keep all lifecycle and business code outside the generated package.

## Construction options

| Option | Role |
|--------|------|
| `{Contract}(impl)` | register implementation |
| `Service(svc)` | arbitrary `ServiceRoute` |
| `SetFiberCfg` / buffer / body / timeout options | Fiber tuning (`Immutable` forced) |
| `MaxBatchSize` / `MaxBatchWorkers` | JSON-RPC batch |
| `MaxOpenStreams` | WS multiplex streams per connection (default 64) |
| `MaxWSConnections` / `MaxSSEConnections` | concurrent stream connections (default 512; `0` = off) |
| `MaxWSMessageSize` / `StreamWriteTimeout` / `SetSSEHeartbeat` | stream bounds |
| `MaxClientIDs` | cardinality of normalized `client_id` (default 64; `0` = off; overflow → `other`) |
| `DurationBuckets` / `LatencyBuckets` | histogram buckets **before** `WithMetrics` |
| `WithRequestID` / `WithHeader` / `Use` | request metadata / middleware |
| `Auth(...)` | bearer JWT (see Auth) |

Defaults and edge cases: `tg plugin doc server`.

## Auth

Requires package `@tg security` with `http:bearer`:

- `AuthorizationServer` / `Resource` / `Audience` required for a working verify path
- Optional: `ClientHTTP`, `ConfigTLS`, `Bind`, `Allow`, `Verify`
- If both `ClientHTTP` and `ConfigTLS` are set, `ClientHTTP` wins (put TLS on the custom client)
- Incomplete AS/Audience fails closed on `Verify` (401), not on `New`
- Subject via `srvctx.GetSubject`; RFC 9728 well-known
- JWKS: RFC 8414 from AS, or `|jwks=` path with `:name` from JWT claims/`kid`
- On `@tg metrics`, denials increment `service_auth_denied_total{reason}`

## Observation

- `WithLog()` requires effective `log`
- `WithMetrics()` requires effective `metrics`
- `WithTrace(ctx, appName, endpoint, attributes...)` requires effective `trace`
- `ServeMetrics` exposes the generated Prometheus collectors
- Sensitive arguments/results need `log-skip` in the contract
- Override histogram buckets with `DurationBuckets` / `LatencyBuckets` before `WithMetrics`
- Bound metric cardinality for `X-Client-Id` with `MaxClientIDs` (default 64; `0` = off)

Observation generation follows HTTP/JSON-RPC contracts; Kafka has separate generated runtime.

## Errors

For REST-capable contracts, use `srv.<Contract>().WithErrorHandler(handler)` to map or wrap application errors before transport encoding. JSON-RPC errors use their generated RPC representation.

Fix missing HTTP codes/types in contract or implementation analysis, regenerate, and test both transport representations.

## Streams

- **WS**: one upgrade connection; header/query values are connection-scoped for the multiplex session. `MaxWSConnections` (default 512; `0` = unlimited) rejects with `503` when saturated. `MaxOpenStreams` (default 64) caps concurrent RPC streams per connection; `MaxWSMessageSize` and `StreamWriteTimeout` bound frame size and write deadlines.
- **SSE**: one HTTP request per server stream. `MaxSSEConnections` (default 512; `0` = unlimited) rejects with `503` when saturated. `SetSSEHeartbeat` keeps the response alive; write deadlines follow `StreamWriteTimeout`.
- Browser WS clients cannot send arbitrary headers and may use query fallback (server reads query during upgrade).
- Cancellation and channel closure are part of the generated stream protocol.
- Duplicate JSON-RPC stream `id` among open streams → error; per-stream buffer overflow closes that stream only.

## Files

- Single `io.Reader`: raw streaming request body
- Single `io.ReadCloser`: raw streaming response body
- Multiple streams or `http-multipart`: multipart
- Multipart readers are sequential; consume one part to EOF before the next
- The client owns closing returned `io.ReadCloser`
- `io.Reader` / `io.ReadCloser` only with `http-server`

## Regeneration

Cleanup removes generated `.go`/`.ts` files carrying the Tool Gateway marker. It does not make mixed handwritten/generated packages a good ownership boundary. Keep custom code elsewhere and compile all consumers after every contract change.
