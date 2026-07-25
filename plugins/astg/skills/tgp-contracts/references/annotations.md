# Annotation quick reference

Full list and edge cases: `tg plugin doc astg`.

## Package

`log`, `trace`, `metrics`, `http-prefix=`, `packageJSON=`, `uuidPackage=`, `swaggerTags=`, `security=`, `servers=`, `version=`, `title=`, `author=`, `npmRegistry=`, `npmName=`, `npmPrivate=`, `license=`

## Interface

| Annotation | Role |
|------------|------|
| `http-server` | REST handlers |
| `jsonRPC-server` | JSON-RPC 2.0 (+ batch) |
| `ws-server` | WebSocket streams |
| `sse-server` | SSE server streams |
| `kafka` | Контракт событий Kafka (плагины kafka-pub-go / kafka-sub-go) |
| `http-prefix=`, `log`, `trace`, `metrics`, `swaggerTags=`, `desc=` | shared |

## Method (HTTP / RPC)

`http-method=`, `http-path=`, `http-success=`, `http-args=`, `http-headers=`, `http-cookies=`, `http-response=`, `handler=`, `requestContentType=`, `responseContentType=`, `http-multipart`, `http-part-name=`, `http-part-content=`, `enableInlineSingle`, `log-skip=`, `deprecated`, `summary=`, `desc=`, `requestBodyDesc=`, `swaggerTags=`, `stream=`, `ws-path=`, `sse-path=`

## Method (Kafka)

`kafka-topic=`, `kafka-key=`, `kafka-headers=`, `kafka-message=`, `kafka-codec=`, `kafka-acks=`

## Field / parameter

`desc=`, `type=`, `enums=`, `format=`, `required`, `example=`, `http-part-name=`, `http-part-content=`, `log-skip`

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
	// @tg http-args=id|userId
	GetUser(ctx context.Context, id string) (user dto.User, err error)
}
```
