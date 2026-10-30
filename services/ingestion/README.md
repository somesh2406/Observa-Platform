# Observa Ingestion Service

The Go service accepts one CloudEvents 1.0 structured JSON event, validates the Observa v1
profile, enforces a Redis-backed tenant quota, and publishes the original event bytes to Kafka.
The public HTTP entrypoint is Envoy; the ingestion listener is not published to the host in the
integrated Compose stack.

## Routes

- `POST /v1/events`
- `GET /health/live`
- `GET /health/ready`

## Request behavior

1. Require `Content-Type: application/cloudevents+json` and enforce the configured body limit.
2. Decode one JSON event and validate it using `libs/eventcontract`.
3. In authenticated mode, require `X-Observa-Tenant` and `X-Observa-Source`.
4. Apply the tenant quota using Redis. Limiter errors fail closed with `503`; exceeded quotas
   return `429`.
5. Publish original request bytes to Kafka and return `202` only after broker confirmation.

## Kafka key and delivery

The key is `tenantID:sourceID` without a session ID and
`tenantID:sourceID:sessionID` when `data.context.session_id` is present. The event ID is not part
of the key.

The boundary is at-least-once, not exactly-once. An ambiguous timeout can lead to duplicate
publication. Consumers outside this repository must deduplicate by CloudEvents `event.id`.

## Configuration and local execution

The service reads the JSON config file selected by `--config` or `OBSERVA_CONFIG_FILE`; otherwise
Viper searches `configs/ingestion.local.json`. Environment variables with the `OBSERVA_` prefix
override config values. The service always creates a Redis tenant limiter; no in-memory backend
is implemented.

See [configuration](../../docs/getting-started/configuration.md),
[API reference](../../docs/reference/api.md), and
[local setup](../../docs/getting-started/local-setup.md).
