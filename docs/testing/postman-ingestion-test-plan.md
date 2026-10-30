# Ingestion API — Postman Test Plan

## Preconditions

For direct local development, `configs/ingestion.local.json` enables unauthenticated development
mode and binds to `127.0.0.1:8080`; the configured Kafka broker is `127.0.0.1:9092` and tenant
Redis is `127.0.0.1:6380`. Start those dependencies separately and run `make run-ingestion`.

For the integrated gateway stack, use `https://127.0.0.1:8443`, a JWT from the configured test
issuer, and the current [Local Setup guide](../getting-started/local-setup.md).

## Expected behavior

| Request | Expected result |
|---|---|
| `GET /health/live` | `200` with `{"status":"live"}` |
| `GET /health/ready` when Kafka is reachable | `200` |
| Valid event with correct media type and identity | `202` after broker delivery confirmation |
| Malformed/invalid event | `400 invalid_event` |
| Missing/unsupported content type | `415 unsupported_media_type` |
| Empty body | `400 invalid_event` |
| Body larger than configured limit | `413 request_too_large` |
| Missing gateway identity in authenticated mode | `401 missing_trusted_identity` |
| Tenant quota exceeded | `429 tenant_rate_limit_exceeded` |
| Tenant Redis unavailable during check | `503 rate_limiter_unavailable` |
| Kafka delivery not confirmed | `503 publication_unconfirmed` |
| `GET /health/ready` when Kafka is unavailable | `503 not_ready` |

See [API Reference](../reference/api.md) for exact messages and response details.

## Duplicate submission

There is no ingestion-side deduplication. Submitting the same event ID twice can produce two Kafka
records. A `503 publication_unconfirmed` can also have an ambiguous broker outcome. Reuse the
same CloudEvents `id` when retrying; downstream deduplication is not implemented in this repo.

## Kafka evidence

Inspect record values and metadata headers:
`observa_tenant_id`, `observa_source_id`, `observa_ingestion_id`,
`observa_received_at`, and `observa_contract_version`. The key is
`tenantID:sourceID` or `tenantID:sourceID:sessionID` when session ID is present.
