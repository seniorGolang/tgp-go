# Kafka publisher validation

## Contract family

- Use `kafka`; legacy `kafka-consumer` and `kafka-publisher` are rejected
- Do not combine with `http-server`, `jsonRPC-server`, `ws-server`, `sse-server`, or `stream`
- Topic names must be unique across the project

## Method

- First argument: `context.Context`
- Results: only `error`
- `kafka-topic`: required
- Message: explicitly selected or exactly one unambiguous remaining argument
- Message must not be a pointer

## Key and headers

Key/header arguments must be string/byte-compatible according to the generator model and cannot also be the message. Inspect `args` and `typeID` in `tgp-astg-json` when validation reports a mismatch.

## Codec

Codec resolution cascades method → contract → project/default. `json` is the default. `bytes` requires `[]byte` for one message or byte-oriented batch forms.

Custom codecs must be registered before `New` validates required codecs.

## Acknowledgements

Acks resolution cascades method → contract → default `allISRAcks`. Generated runtime creates only the clients required by selected acknowledgement policies.

## Output

The output must be inside a Go module. Regeneration cleans files carrying the generated marker. Expected core artifacts include constructor/options, contract adapters, codecs, producer helpers, security, and version; observation files appear only when enabled.
