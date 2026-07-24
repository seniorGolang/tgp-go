---
name: tgp-client-ts
description: >-
  Generates TypeScript API clients from tgp contracts (REST/JSON-RPC/streams) and
  optional package.json via @tg npmName. Use when building/updating a TS/JS SDK,
  npm package metadata, or regenerating client-ts output after contract changes.
---

# tgp-client-ts

## Do

1. Set package metadata on contracts when publishing npm: `@tg npmName=…` (and related npm* / author / license annotations).
2. Generate:

```bash
tgp client ts -o ./client-ts
# optional: --package-json=… --contracts=… --no-doc --no-client-id
```

3. Prefer generated client methods over ad-hoc `fetch`.  
4. After contract edits → regenerate; do not patch generated TS by hand.

## Never

- Forget `npmName` when `--package-json` is required  
- Mix server `transport/` output with TS client `-o`  

## Dig deeper

`tg plugin doc client-ts` · skill `tgp-contracts`
