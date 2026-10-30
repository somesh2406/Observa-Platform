# Observa Event Contract v1 — CloudEvents Profile

The current Go validator is the source of runtime behavior. This record is superseded by the
[Event Contract reference](../architecture/event-contract.md), which documents accepted fields,
constraints, exact validation behavior, and delivery semantics from
`libs/eventcontract/event.go`.

The checked-in JSON Schemas are in `schemas/`. Schema validation is run by
`python scripts/validate_event_schemas.py`; the HTTP handler separately invokes the Go validator.
The schema and Go validator are separate implementations and should not be assumed to have
identical behavior for every possible JSON input.

The implementation does not enforce source-registration authorization, a namespace registry,
privacy allowlists, JSON depth limits, clock-skew policy, or W3C traceparent propagation.
