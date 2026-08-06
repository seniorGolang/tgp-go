# Go client auth and token helpers

Generated only when package `@tg security` includes `http:bearer` or `oauth2:`.

## Options

| Option | Role |
|--------|------|
| `Auth(...)` | app2app / Resource tokens |
| `TokenSource(src)` | ready Bearer via `oauth2.Transport`; disables default Auth transport |
| `Auth(TokenURL(...), Assertion(...))` | jwt-bearer + tokenExchange profile → default Resource transport |
| `Auth(ResourceTokens(...))` | override obtain function |
| `Auth(TokenRetry(...))` | IdP/metadata transient retries (`MaxAttempts`, `Backoff`, `Disabled`) |

## Behavior

- Lifetime skew defaults (~10% TTL): `SkewFraction` / `SkewMin` / `SkewMax` / `EarlyExpiry`
- On API **401**: one re-obtain + replay when the body is rewindable
- Non-rewindable body → `token.ErrBodyNotRewindable`
- TokenRetry covers IdP/metadata 429/5xx/network — **not** resource API 5xx
- `ToCurl` / `LogRequest` redact `Authorization` as `Bearer ***`

## Helpers in generated `token` package

`JWTBearer` (RFC 7523), `Exchange` (RFC 8693), `Resource` / `WellKnownURL` / `FetchMetadata` (RFC 9728).

## Example

```go
cli := client.New(apiURL,
    client.Auth(
        client.TokenURL(tokenURL),
        client.Assertion(sub, kid, pem),
        client.TokenRetry(client.MaxAttempts(8), client.Backoff(50*time.Millisecond, 2*time.Second)),
    ),
)
```

Without matching `security` annotations, do not look for `Auth` or `token/` in generated options.
