# Event contract v1 acceptance tests

## Implemented local validator checks

The standalone Go package `internal/eventcontract` currently verifies:

- Valid example event and all three supported Observa event categories.
- Required envelope/profile values, timestamp parsing, event-name pattern, supported outcome values, and trace/span identifier shape.
- Unknown envelope and nested fields, explicit `null` for optional typed fields, multiple JSON values, and attribute-count limits.
- The repository example `examples/user-action.cloudevent.json` decodes successfully.

Locally verified with Go 1.23.2: `go test ./...`, `go test -cover ./...`, and `go vet ./...` pass. This is a local compatibility test, not the final production toolchain/CI pin. JSON Schema validation of the example also passes with a Draft 2020-12 validator and format checking enabled.

The Go validator is currently a handwritten profile validator; keep it aligned with both JSON Schema files and add conformance cases whenever the schemas change.

## Schema-level cases

- Accept one valid CloudEvents 1.0 structured JSON event for each allowed `type`.
- Reject missing `specversion`, `id`, `source`, `type`, `datacontenttype`, `dataschema`, or `data`.
- Reject unsupported CloudEvents `specversion`, unsupported Observa `type`, unsupported `dataschema`, wrong content type value, malformed timestamp, malformed trace ID/span ID, missing `data.event_name`, and unknown properties in the strict v1 envelope/data structures.
- Accept optional context fields being absent.
- Accept `context.user_id` and `context.business_application_id`; reject legacy `user_ref` and `business_application_ref` keys.
- Verify the producer payload has no `user_name` or `applicant_name` fields; names are resolved through an authorized lookup rather than replicated into event telemetry.
- Verify `tenant_id`, `source_id`, and `received_at` are not producer fields in the envelope or data payload.

## Runtime/integration cases (not proven by JSON Schema)

- Reject a CloudEvents `source` not authorized for the authenticated source registration.
- Resolve tenant and stable source identity from credentials, never from payload data.
- Reject oversized bodies before unbounded buffering; enforce nesting/depth and attribute limits.
- Duplicate key with same content is recognized as duplicate; same key with different content raises an identity conflict.
- Preserve event `time` and separately assign server `received_at`.
- Invalid/missing `traceparent` is handled per W3C Trace Context rules; do not trust arbitrary trace IDs for authorization.
- Sensitive values are redacted/rejected according to policy and are not written to ordinary logs.
- Correctly handle event-time skew and producer retry after timeout.

Schema validation alone does not prove authentication, durable acceptance, Kafka delivery, or downstream exactly-once processing.

## JSON Schema validation

- `schemas/producer-event-v1.schema.json` is the one entry point for complete producer-event validation.
- The producer schema composes the CloudEvents profile and payload schema using `$ref`; do not duplicate field definitions in the composed schema.
- Run `python scripts/validate_event_schemas.py` after any schema/example change. The script checks schema metaschemas, the valid example, and representative invalid cases.
- Run the Go tests separately; JSON Schema validation does not prove the Go implementation is fully equivalent.


## Cross-validator conformance gate

- The Go validator and JSON Schema validator consume the same fixture corpus at `tests/fixtures/event-contract-cases.json`.
- Every fixture has an explicit expected validity result. Both validators must match that result.
- The corpus covers required fields, supported event categories, event-name syntax, timestamps, URI references, legacy fields, unexpected fields, explicit nulls, trace identifiers, identifier lengths, attribute limits, and nested JSON attributes.
- Run `make check` before merging contract changes. Passing this corpus demonstrates agreement for these representative cases only; it does not prove equivalence for every possible input.
