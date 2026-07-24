---
name: tgp-swagger
description: >-
  Generates OpenAPI/Swagger from tgp contracts and can serve Swagger UI. Use when
  updating API specs, openapi.json/yaml, documenting endpoints, or running swagger
  serve after contract or annotation changes (title, security, servers, swaggerTags).
---

# tgp-swagger

## Do

1. Ensure OpenAPI-facing annotations on contracts (`title`, `version`, `security`, `servers`, `swaggerTags`, method `summary`/`desc`, field `required`/`format`/…).
2. Generate:

```bash
tgp swagger -o ./openapi/openapi.yaml
# optional: --contracts=… --serve=:8080
```

3. After contract changes → regenerate the spec.

## Never

- Hand-edit generated OpenAPI as source of truth — fix contracts, then regen  
- Expect rich docs without annotations (skill `tgp-contracts`)  

## Dig deeper

`tg plugin doc swagger` · `tg plugin doc astg`
