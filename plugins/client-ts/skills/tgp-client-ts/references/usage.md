# TypeScript client usage

## Runtime surfaces

- `client.<service>()`: JSON-RPC methods, batch request builders, and streams
- `client.<service>HTTP()`: REST methods
- `client.batch(...)`: JSON-RPC batch only

Do not choose an accessor by naming preference; choose it from the contract transport family.

## Headers and identity

Use static headers for fixed metadata and an async function for renewable credentials. `clientName` becomes `X-Client-Id`; disabling client identity changes labels observed by generated server logs/metrics.

Mapped `implicit` arguments may not appear in a generated method. Their values must come from shared headers or other designed transport context.

## Browser and Node

Browser WebSocket constructors cannot attach arbitrary headers. Generated WS requests may duplicate mapped upgrade metadata in query parameters. SSE and ordinary HTTP use `fetch` headers normally.

Verify availability or polyfills for runtime APIs such as `fetch`, `Blob`, `FormData`, and `crypto.randomUUID` in the actual target environment.

## Streams

Consume generated async streams until completion and cancel them when their owning UI/task is disposed. For client or bidirectional streams, propagate producer errors and stop sending after cancellation.

## Binary data

- Single body/result: `Blob`
- Multiple parts: `FormData`
- Part names and content types come from contract annotations

Test upload and download in the target runtime because browser and Node implementations of these APIs differ.

## Errors

Generated calls reject/throw on RPC and HTTP errors. Catch at the application boundary and map to domain/UI errors without modifying generated sources.

## Publishing

Treat generated `package.json` and `tsconfig.json` as generated artifacts. Keep handwritten package scripts or wrappers in a clearly owned location and verify they are not lost during regeneration.
