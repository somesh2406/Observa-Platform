# Observa Product Contract

## Current implemented boundary

The repository implements the ingestion boundary only: a producer submits one CloudEvents 1.0
structured JSON event; the ingestion service validates the Observa v1 profile, applies a
tenant-specific Redis quota, and publishes the original event bytes to Kafka. See
[Event Contract](architecture/event-contract.md), [API](reference/api.md), and
[Architecture](architecture/hld.md).

## Event categories and views

The accepted CloudEvents `type` values are `com.observa.user_action`,
`com.observa.business_state`, and `com.observa.technical_signal`. The validator accepts a
syntactic `data.event_name`; it does not maintain an event-name registry.

The payload may carry optional `context.user_id`, `business_application_id`, `session_id`,
`request_id`, `trace_id`, and `span_id` fields. Their presence is not guaranteed by the validator
and does not mean the repository currently constructs a user timeline, application journey,
session view, or incident view.

## Trust boundary

In authenticated gateway mode, tenant and source identities are obtained from JWT claims mapped
by Envoy to `X-Observa-Tenant` and `X-Observa-Source`. The event payload is not the source of
trusted tenant identity. The handler validates the event before applying the tenant quota.

## Current guarantees

- One event per HTTP request.
- Strict CloudEvents 1.0 envelope/profile validation.
- Request body limit configured through `http.max_body_bytes`.
- Tenant quota through a Redis fixed-window counter.
- Kafka publication of original request bytes with server metadata in Kafka record headers.
- `202 Accepted` only after broker delivery confirmation.
- At-least-once delivery boundary; consumers must deduplicate using CloudEvents `id`.

## Out of Scope (Deferred)

The following product capabilities are not implemented in this repository:

- User timeline query and UI.
- Application journey construction and visualization.
- Session action reconstruction and query APIs.
- Technical incident grouping, severity, lifecycle, or root-cause analysis.
- Kafka consumer processing and deduplication.
- ClickHouse, Iceberg, object storage, and PostgreSQL integrations.
- OpenTelemetry tracing and Prometheus metrics.

The payload fields are event inputs only; their presence does not imply these derived product
views or storage/query components exist.
