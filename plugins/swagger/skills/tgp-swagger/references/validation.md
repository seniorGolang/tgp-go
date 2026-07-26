# OpenAPI validation

## Required inference

When no explicit `required` exists:

- Request body field: required when non-pointer and not `omitempty`
- Response field: required when not `omitempty`; pointer alone does not make it optional
- Query parameter: required when non-pointer
- Header/cookie: required only when explicitly marked

An explicit field or method sub-annotation overrides inference.

## Mapping

`explicit` and `implicit` header/cookie/query mappings become OpenAPI parameters. `body` remains payload data and does not become a separate parameter.

Path placeholders must resolve to method arguments. Inspect the resolved method annotations when a parameter is missing or duplicated.

## Errors

Responses are generated from errors discovered for the method and their HTTP codes. Standard error codes are not synthesized automatically. If an implementation error is expected but absent, verify implementation discovery in ASTG before changing Swagger generation.

## Schemas

Follow `typeID` in `tgp-astg-json` when a schema is wrong. Check:

- pointer/slice/array/map shape
- field JSON tags and `omitempty`
- `required`, `format`, `example`, `enums`, `type`
- named typed enums (`Type.Enums`) as `$ref` schemas with `enum`
- inline result/field tags
- custom marshalers, which may produce a generic object

## Security and metadata

Title, version, description, servers, and security originate at project/package level. Tags come from contract/method annotations with method values taking precedence.

## Acceptance

For JSON, parse and inspect deterministically:

```bash
jq -e '.openapi == "3.0.0" and (.paths | length > 0)' openapi.json
jq '.paths, .components.schemas, .components.securitySchemes' openapi.json
```

For YAML, use the repository's existing OpenAPI/YAML validator when available. Do not add a validator dependency merely to satisfy this skill.
