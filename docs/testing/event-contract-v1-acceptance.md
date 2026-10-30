# Event Contract v1 Acceptance Tests

## Automated checks in the repository

- Go profile validator tests: `libs/eventcontract/event_test.go`.
- Shared fixture corpus: `tests/fixtures/event-contract-cases.json`.
- JSON Schema validation script: `scripts/validate_event_schemas.py`.
- Schema and example files: `schemas/` and `examples/user-action.cloudevent.json`.

Run:

```sh
make schema-check
go test ./libs/eventcontract
```

The Go validator is implemented in `libs/eventcontract/event.go`; the HTTP handler maps any
decode/validation error to `400 invalid_event`. See
[Event Contract](../architecture/event-contract.md) for exact runtime constraints.

## Runtime behaviors not asserted by schema validation

The JSON Schemas do not prove JWT authentication, Redis quota behavior, Kafka delivery, or
downstream deduplication. The current implementation does not enforce producer source
registration, privacy allowlists, event-time skew, or traceparent propagation.

## Cross-validator fixture note

The Python script validates the three checked-in Draft 2020-12 schemas against the shared
fixture corpus (`scripts/validate_event_schemas.py:35-67`). Go validator tests are separate; the
repository does not demonstrate exhaustive equivalence between JSON Schema and Go behavior.
