# Observa Event Contract v1 — CloudEvents profile

**Status:** Proposed contract decision, ready for review before ingestion implementation. **No ingestion code should be built until this contract and its open policy decisions are accepted.**

## Decision

Use **CloudEvents 1.0 structured JSON** as the producer-facing envelope and an Observa-owned JSON schema for the `data` payload. The CloudEvents envelope provides interoperable event metadata (`specversion`, `id`, `source`, `type`, `time`, content type and schema reference). Observa owns the business semantics of the event data and its versioning.

This is not a claim that CloudEvents defines Observa's domain model. It does not. CloudEvents is the envelope/interoperability contract; the Observa data schema is the product contract.

The first endpoint accepts one structured CloudEvent per HTTP request with `Content-Type: application/cloudevents+json`. Do not implement binary-mode or batch-mode ingestion in the first slice unless a concrete producer requirement justifies it.

## Schema organization and canonical validation entry point

The earlier custom root-level envelope was replaced with CloudEvents. The file `schemas/producer-event-v1.schema.json` is the **single entry point** consumers and CI should use to validate a complete producer event. It composes `schemas/cloudevent-v1.schema.json` with `$ref`; the CloudEvents profile in turn references `schemas/event-data-v1.schema.json` for the Observa payload. The component schemas are not competing contracts and should not duplicate each other’s field definitions.

Run `python scripts/validate_event_schemas.py` to validate all schema documents and the sample plus representative invalid cases. The Go decoder/validator is a separate implementation and must be kept behaviorally aligned through tests.

## Envelope vs payload

| CloudEvents field | Meaning in Observa |
|---|---|
| `specversion` | CloudEvents protocol/spec version; fixed to `1.0` in this profile. It is **not** Observa schema version. |
| `id` | Producer event ID, stable across retries of the same logical event. CloudEvents does not require UUID specifically; Observa recommends UUIDv4/UUIDv7 or another collision-resistant ID. |
| `source` | URI-reference identifying the event source context. Ingestion must verify it is authorized for the authenticated producer registration; never trust arbitrary source values. |
| `type` | One of `com.observa.user_action`, `com.observa.business_state`, `com.observa.technical_signal`. This is a bounded semantic category. |
| `time` | Producer-reported event occurrence time in RFC 3339. It can be inaccurate and is not the receive time. |
| `datacontenttype` | Fixed to `application/json` in the initial profile. |
| `dataschema` | Versioned URI for the Observa data schema. Current `.example` URI is a placeholder; register a real stable URI before external rollout. |
| `data` | Observa-specific event payload defined in `schemas/event-data-v1.schema.json` and validated as a complete event through `schemas/producer-event-v1.schema.json`. |

## Observa data payload

- `event_name` (required): stable low-cardinality name such as `loan.submit`, `screen.open`, or `api.request.failed`. Do not embed user/application IDs in the name.
- `outcome` (optional): `succeeded`, `failed`, `denied`, `cancelled`, or `unknown`. Omission means not supplied, not inferred.
- `context.user_id`: stable, tenant-scoped identifier for the actor/user associated with the event. It may originate from the source application login identity, but producers must not send raw credentials or unnecessary direct identifiers. Prefer a stable pseudonymous/tokenized value when the source-system identifier is sensitive.
- `context.business_application_id`: stable identifier for the business case being acted upon (for example, a loan application ID), not the mobile/web software application that emitted the event. Treat it as tenant-scoped and avoid exposing sensitive source-system keys when a tokenized reference is available.
- Do not include `user_name` or `applicant_name` in the canonical event by default. Names are direct personal data, can become stale, and would be replicated into Kafka, evidence storage, and analytical stores. For display, resolve names through an authorized, tenant-scoped lookup against the authoritative identity/business system, or use a separately controlled enrichment service with explicit access, retention, and audit policies. The logged-in actor (`user_id`) and the applicant associated with a business application may be different people.
- `context.session_id`: producer-defined session identifier.
- `context.request_id`: request/correlation ID when available.
- `context.trace_id` and `context.span_id`: W3C-compatible identifiers when present. A trace ID is not the same thing as an Observa event ID. If the producer has a `traceparent` header, parse it according to W3C Trace Context rather than inventing identifiers; span correlation fields alone do not replace propagation of trace context.
- `attributes`: structured, event-specific attributes; runtime validation must enforce depth, encoded byte size, key/value limits, and field allowlists/privacy policy. JSON Schema alone does not prevent deeply nested or huge payloads.
- `error`: optional sanitized error code/category/message. No secrets, tokens, raw exception dumps, or unredacted sensitive customer data by default.
- `producer_version`: instrumentation/SDK version.

## Four product views

1. **User timeline:** primarily `user_id`, event `time`, `event_name`, `outcome`, and selected attributes. A UI may display a name only after an authorized lookup.
2. **Application journey:** `business_application_id` plus event sequence and explicitly modeled business-state events. Applicant identity/name is a separate concern and must not be inferred from the logged-in actor. Do not infer a business transition solely from a UI click.
3. **Session trace:** `session_id`, optionally correlated with request and W3C trace context.
4. **Technical incidents:** `technical_signal` events may be inputs to a separate incident model. An error event is not automatically an incident; grouping, severity, deduplication, lifecycle and root-cause attribution are derived behavior.

These references are optional because real instrumentation may not know every identifier. Missing correlation must remain unknown; never guess across users or tenants.

## Trust boundary and canonical internal record

The producer envelope is untrusted input. After authentication and validation, ingestion creates an internal canonical record with at least:

- the validated CloudEvent envelope and payload (preserved, not silently rewritten);
- `tenant_id` derived from authenticated credentials/configuration, never from the request body;
- stable internal `source_id` derived from the producer registration, not accepted as a producer-supplied tenant/source identity;
- `received_at` assigned by the ingestion service in UTC;
- `ingestion_request_id`/server correlation metadata;
- payload digest and validation/schema-version metadata as needed for deduplication and forensics.

Do not inject `tenant_id` into the producer's event or pretend it was producer-supplied. Keep trusted ingestion metadata separate or clearly namespaced in the internal envelope.

Recommended idempotency key: `(tenant_id, authenticated_source_id, CloudEvents.source, CloudEvents.id)`. The registration must authorize the CloudEvents `source`. Same key + same canonical producer content is a duplicate delivery; same key + different content is an identity conflict that must be recorded/ surfaced, not silently overwritten. Credential rotation must not change the stable internal source registration ID.

## Time and tracing

Keep CloudEvents `time` (event occurrence time) separate from server `received_at` (observed time). Enforce a configurable future-clock-skew policy; do not silently rewrite the producer time. W3C Trace Context uses `traceparent` propagation and defines non-zero trace/span IDs; the payload fields are correlation helpers, not proof that a trace exists.

## Validation and security requirements

JSON Schema is only one layer. Ingestion must additionally enforce:

- structured CloudEvents media type and `specversion` support;
- authentication and authorization mapping credentials to tenant/source registration;
- allowed `source` values for that registration;
- byte limit before full body buffering, JSON nesting/depth and collection limits;
- schema and content type validation;
- rate/concurrency quotas and request timeouts;
- privacy/attribute policy and redaction;
- safe errors and logs (never log full raw payloads by default);
- bounded clock skew and duplicate/conflict semantics.

The initial maximum request size is a provisional safety limit, not a target payload size. Validate it against representative producer events before finalizing. The schema `$id`, `$ref`, and `dataschema` values use the published GitHub Pages URLs listed in `docs/contracts/schema-hosting.md`. Verify all URLs return JSON after each Pages deployment. Treat released v1 schemas as immutable; introduce a new version for breaking changes. `producer-event-v1.schema.json` is the canonical validation entry point; the other two schemas define the envelope profile and payload without duplicating definitions.

## Versioning

`specversion: "1.0"` versions the CloudEvents envelope specification. `dataschema` versions Observa's event payload schema. These are different version axes. Backward-compatible optional additions can be introduced under a documented compatibility policy; breaking changes require a new data schema version and an explicit producer migration/deprecation plan. Do not create a new CloudEvents `specversion` to version Observa's payload.

## Contract decisions for the first release

- CloudEvents 1.0 structured JSON only; one event per HTTP request.
- `type` is the bounded Observa category; `data.event_name` is the specific occurrence name.
- Tenant identity and stable internal source identity come from authentication/registration.
- `time` is event time; `received_at` is server-observed time.
- Event identity uses stable `id` and authorized source context; retries reuse the same ID.
- Incidents are derived entities, not producer event types.

## Must be resolved before ingestion implementation

1. The real schema URL/domain and whether schemas are served publicly or versioned in a registry.
2. Producer authentication mechanism and source-registration model.
3. Actual producer sources: web/mobile SDKs, backend services, or all of them; source URI naming convention.
4. Representative event samples and the final body/attribute/depth limits.
5. Allowed event attributes and customer-specific sensitive-data classification/redaction policy.
6. Clock-skew policy and handling of missing/late event time.
7. Whether clients can supply trace context through W3C `traceparent`, and where it is propagated.
8. Exact response and retry behavior after Kafka acknowledgement is implemented.

## Standards references

- CloudEvents 1.0 core specification: https://github.com/cloudevents/spec/blob/v1.0.2/cloudevents/spec.md
- CloudEvents JSON format: https://github.com/cloudevents/spec/blob/v1.0.2/cloudevents/formats/json-format.md
- W3C Trace Context: https://www.w3.org/TR/trace-context/
- OpenTelemetry event semantic conventions: https://opentelemetry.io/docs/specs/semconv/general/events/
