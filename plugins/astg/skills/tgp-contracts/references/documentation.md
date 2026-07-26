# Contract documentation (OpenAPI text)

Author API descriptions in contract sources. Swagger only renders the resolved model — do not hand-edit generated OpenAPI.

Full annotation catalog: `tg plugin doc swagger` · `tg plugin doc astg`.

## Where text comes from

| Source | OpenAPI target |
|--------|----------------|
| Package `@tg title` / `version` / `desc` | `info.title` / `info.version` / `info.description` |
| Package `@tg servers` / `security` | `servers[]` / security schemes |
| Interface `@tg swaggerTags` | `operation.tags`, root `tags[]` |
| Interface `@tg tagDesc.<tag>` | `tags[].description` |
| Interface `@tg desc` | tag description when `tagDesc` is absent |
| Method `@tg summary` | `operation.summary` |
| Method `@tg desc` | `operation.description` |
| Method `@tg requestBodyDesc` | `requestBody.description` |
| Method `@tg deprecated` | `operation.deprecated` |
| Field / arg / result `@tg desc` (or `<name>.desc`) | `schema.description` / parameter description |
| Field / arg `@tg format` / `example` / `enums` / `type` | schema facets |
| Named typed enum (`type Role string` + typed consts) | `Type.Enums` → OpenAPI `$ref` schema.enum, client typed unions/consts |

Priority for a description value: explicit `@tg desc` wins. If absent, non-`@tg` godoc lines on the same declaration are used. Prefer `@tg desc` for anything that must appear in OpenAPI.

## What to write where

| Need | Annotation | Rule |
|------|------------|------|
| Operation title in Swagger UI | `summary` | One short phrase; required for every public HTTP/JSON-RPC method |
| Operation details | `desc` | Behavior, constraints, side effects; markdown OK |
| Request body prose | `requestBodyDesc` | Only when body needs text beyond the schema |
| Field / parameter meaning | `desc` on DTO field or `<arg>.desc` | Public request/response fields and path/query/header params |
| Scalar shape hints | `format`, `example`, `enums` | UUIDs, emails, closed sets on bare scalars, illustrative values |
| Typed closed set | named type + typed `const` | Prefer `type Role string` with `const RoleAdmin Role = "admin"`; ASTG fills `Type.Enums` |
| Tag grouping | `swaggerTags` + `tagDesc.<tag>` | Stable tag ids; describe each tag once on the interface |
| API product blurb | package `title`, `version`, `desc` | Once per contracts package |

Do **not** put transport wiring (`http-path`, mapping modes, codecs) into prose — that belongs in transport annotations.

## Long text: `file:` refs

Any annotation value may be `file:path` or `file:path#Section` (path relative to project root). Resolved by ASTG before generators run.

```go
// @tg desc=`file:docs/user-api.md#GetUser`
```

Use `file:` when the text is multi-paragraph or shared across methods. Keep short `summary` / field `desc` inline.

## Placement

1. Package docs — top of a `contracts/*.go` file (`package` comment annotations).
2. Interface docs — on the contract interface.
3. Method docs — on the method; arg/result sub-keys as `<name>.desc=…` when the name is only in the signature.
4. DTO field docs — on the struct field in `contracts/dto` (preferred over repeating the same text on every method).

## Minimal public HTTP method

```go
// @tg http-method=GET
// @tg http-path=/users/:id
// @tg http-args=id|userId
// @tg summary=`Get user by id`
// @tg desc=`Returns the full user profile.`
// @tg id.desc=`User id`
GetUser(ctx context.Context, id string) (user dto.User, err error)
```

```go
type User struct {
	// @tg desc=`User id`
	// @tg format=uuid
	// @tg example=4a1b2c3d-0000-0000-0000-000000000000
	ID string `json:"id"`

	// @tg desc=`Display name`
	Name string `json:"name"`
}
```

## Checklist

- [ ] Package has `title`, `version`, and a non-empty `desc` when OpenAPI is published
- [ ] Each public HTTP/JSON-RPC method has `summary`
- [ ] Non-trivial methods have `desc` (or `file:`)
- [ ] Public DTO fields and wire parameters have `desc`
- [ ] Closed scalars use typed named enums (`type X string`/`int` + typed consts) or field `@tg enums` for bare scalars; ids/emails use `format` / `example` where useful
- [ ] Tags have `tagDesc` (or interface `desc` as fallback)
- [ ] Deprecated operations use `@tg deprecated`
- [ ] `tg astg json` shows intended `desc`/`summary` after `file:` resolution
- [ ] `tg swagger` output has non-empty `info`, operation `summary`/`description`, and schema descriptions

## Never

- Hand-edit generated OpenAPI to add descriptions
- Duplicate the same long prose on every method when a DTO field or `file:` section suffices
- Rely on godoc alone when consumers depend on OpenAPI text — set `@tg desc` explicitly
- Document Kafka contracts via Swagger (Kafka is omitted from OpenAPI)
