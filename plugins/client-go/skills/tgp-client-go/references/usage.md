# Go client usage

## RPC batch

Build requests with generated `Req<Method>` functions, attach callbacks where results are needed, then call `Client.Batch`. Batch support exists only when the selected model contains JSON-RPC contracts.

## Mapping modes

- `explicit`: remains an explicit client argument and is sent through the mapped HTTP element
- `implicit`: may be removed from the method signature; provide it through shared headers/context
- `body`: remains in request/response JSON rather than becoming a distinct parameter

When generated signatures surprise you, inspect method annotations and their inheritance with `tgp-astg-json`.

## Errors

Use separate decoders:

- `DecodeError` for JSON-RPC error objects
- `DecodeHTTPError` for non-successful REST responses

Keep domain error decoding in handwritten consumer code; regeneration must not erase it.

## Streams

- Server stream: receive from the returned channel until close or context cancellation
- Client stream: send values through the generated input channel and obtain the final result
- Bidirectional stream: coordinate both channel directions and cancellation
- If both WS and SSE are generated, use the `SSE`-suffixed method to force SSE

Connection-scoped WS headers are established during dial. SSE metadata belongs to its HTTP request.

## Files

- Close every returned `io.ReadCloser`
- Treat multiple stream parts as ordered multipart data
- Do not read the next multipart stream before consuming the current one to EOF
- Propagate context cancellation during long upload/download operations

## Metrics

`WithMetrics` uses a client-owned Prometheus registry. Expose `GetMetricsRegistry()` explicitly or combine it with other gatherers; do not assume registration in the global registry.
