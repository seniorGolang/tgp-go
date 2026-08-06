# Annotation quick reference

Full list and edge cases: `tg plugin doc astg`.

## Contents

- Package
- Interface
- Method (HTTP / RPC)
- Method (Kafka)
- Field / parameter
- Minimal REST example

## Package

`log`, `trace`, `metrics`, `http-prefix=`, `packageJSON=`, `uuidPackage=`, `swaggerTags=`, `security=`, `servers=`, `version=`, `title=`, `desc=`, `author=`, `npmRegistry=`, `npmName=`, `npmPrivate=`, `license=`

**`security=`** (app2app, no URLs): `http:bearer[:JWT][|jwks=/path/:sub/…]`, `apiKey:<in>:<name>`, `oauth2:tokenExchange`, `oauth2:clientCredentials[:jwt-bearer]`. `openId` is rejected. AS/API endpoints are runtime options, not annotation values.

## Interface

| Annotation | Role |
|------------|------|
| `http-server` | REST handlers |
| `jsonRPC-server` | JSON-RPC 2.0 (+ batch) |
| `ws-server` | WebSocket streams |
| `sse-server` | SSE server streams |
| `kafka` | Kafka events (kafka-pub-go / kafka-sub-go) |
| `http-prefix=`, `log`, `trace`, `metrics`, `swaggerTags=`, `tagDesc.<tag>=`, `tagOmitemptyAll`, `desc=` | shared |

## Method (HTTP / RPC)

`http-method=`, `http-path=`, `http-success=`, `http-args=`, `http-headers=`, `http-cookies=`, `http-response=`, `handler=`, `requestContentType=`, `responseContentType=`, `http-multipart`, `http-part-name=`, `http-part-content=`, `enableInlineSingle`, `log-skip=`, `deprecated`, `summary=`, `desc=`, `requestBodyDesc=`, `swaggerTags=`, `stream=`, `ws-path=`, `sse-path=`

**Mapping modes** on `http-headers` / `http-cookies` / `http-args`: `arg|key` or `arg|key|mode` where `mode` is `explicit` | `implicit` | `body` (default **`body`**). Details: `tg plugin doc astg` / swagger.

**HTTP errors:** numeric keys `// @tg 404=pkg:Type` (codes 400–599).

## Method (Kafka)

`kafka-topic=`, `kafka-key=`, `kafka-headers=`, `kafka-message=`, `kafka-codec=`, `kafka-acks=`

## Field / parameter

`desc=`, `type=`, `enums=`, `format=`, `required`, `example=`, `http-part-name=`, `http-part-content=`, `log-skip`

Prefer named typed enums (`type Role string` + typed consts) over bare `@tg enums=` when the type is shared.

## Minimal REST example

Interfaces in `contracts/`; DTO in `contracts/dto` (referenced as `dto.Type`).

```go
package contracts

import "<module>/contracts/dto"

// @tg http-server
// @tg http-prefix=api/v1
type UserService interface {
	// @tg http-method=GET
	// @tg http-path=/users/:id
	// @tg http-args=id|userId|explicit
	// @tg 404=errs:NotFound
	GetUser(ctx context.Context, id string) (user dto.User, err error)
}
```
