# Ingestion Service Low-Level Design

## Package layout

- `services/ingestion/cmd/ingestion` — executable and config-path selection.
- `services/ingestion/internal/config` — Viper loading, defaults, and validation.
- `services/ingestion/internal/ingestion` — HTTP routes, request validation, identity, and response mapping.
- `services/ingestion/internal/publisher` — Kafka producer construction, readiness, and record creation.
- `services/ingestion/internal/ratelimit` — Redis-backed tenant fixed-window limiter.
- `services/ingestion/internal/secrets` — Vault KV v2 credential retrieval.
- `services/ingestion/internal/platform` — structured logger.
- `services/ingestion/internal/platform/bootstrap` — dependency construction, HTTP server, and shutdown lifecycle.
- `libs/eventcontract` — shared CloudEvents v1 profile decoder and validator.

## Startup and dependencies

`cmd/ingestion/main.go` accepts `--config`, falls back to `OBSERVA_CONFIG_FILE`, and calls
`bootstrap.Build` (`services/ingestion/cmd/ingestion/main.go:11-29`). `Build` loads config,
optionally fetches Kafka SASL credentials from Vault, creates the Kafka producer, creates the
Redis tenant limiter, registers handlers, and configures the HTTP server
(`services/ingestion/internal/platform/bootstrap/build.go:20-128`).

The tenant limiter is always Redis-backed in the current bootstrap. Redis is pinged during
initialization; startup fails if the ping fails
(`services/ingestion/internal/platform/bootstrap/build.go:81-106`;
`services/ingestion/internal/ratelimit/limiter.go:91-105`).

## HTTP routes and handler flow

Routes are registered in `services/ingestion/internal/ingestion/router.go:9-24`:

- `POST /v1/events` — event ingestion.
- `GET /health/live` — process liveness.
- `GET /health/ready` — Kafka connectivity readiness.

The event handler performs these operations in order
(`services/ingestion/internal/ingestion/handler.go:69-169`):

1. Parse media type and require `application/cloudevents+json`.
2. Reject a declared or streamed body larger than `max_body_bytes`.
3. Reject empty bodies and decode one strict JSON CloudEvent through `eventcontract.Decode`.
4. Resolve tenant/source from configured development identity when unauthenticated development
   mode is enabled, otherwise from `X-Observa-Tenant` and `X-Observa-Source`.
5. Reject missing trusted identity.
6. Require a configured limiter; check the tenant quota.
7. Generate a random 16-byte hexadecimal ingestion ID and UTC receive timestamp.
8. Extract `data.context.session_id` when present and publish the unchanged request bytes.
9. Return `202` after producer confirmation.

## HTTP status and error mapping

| Status | Error code | Condition |
|---|---|---|
| `202` | — | Broker delivery confirmed; response includes `status`, `id`, `ingestion_id`, `received_at`. |
| `400` | `invalid_request_body` | Request body read failure other than size overflow. |
| `400` | `invalid_event` | Empty body or event decode/validation failure. |
| `401` | `missing_trusted_identity` | Authenticated mode lacks tenant or source header. |
| `413` | `request_too_large` | Body exceeds configured maximum. |
| `415` | `unsupported_media_type` | Content type is absent, malformed, or not the required media type. |
| `429` | `tenant_rate_limit_exceeded` | Tenant counter exceeds configured requests. |
| `503` | `rate_limiter_unavailable` | Limiter is nil or Redis quota check errors. |
| `503` | `publication_unconfirmed` | Kafka delivery was not confirmed. |
| `500` | `internal_error` | Ingestion ID generation fails. |

The exact response strings are emitted in `services/ingestion/internal/ingestion/handler.go:58-169`.
Gateway-level JWT failures occur before the request reaches this handler.

## Kafka record

`publisher.Kafka.Publish` sets the key to `tenantID:sourceID` or
`tenantID:sourceID:sessionID` and attaches `observa_tenant_id`, `observa_source_id`,
`observa_ingestion_id`, `observa_received_at`, and `observa_contract_version` headers
(`services/ingestion/internal/publisher/publisher.go:77-94`, `100-108`).

## Configuration knobs

The service-specific knobs are `http.*`, `development.*`, `kafka.*`, `vault.*`,
`tenant_rate_limit.*`, and `logger.*`. Their types, defaults, environment mapping, and validation
are listed in [configuration](../getting-started/configuration.md).

## Testing strategy

Unit tests cover handler status behavior and identity propagation
(`services/ingestion/internal/ingestion/handler_test.go`), configuration validation
(`services/ingestion/internal/config/config_test.go`), limiter behavior
(`services/ingestion/internal/ratelimit/limiter_test.go`), Vault behavior
(`services/ingestion/internal/secrets/vault_test.go`), and publisher construction/key behavior
(`services/ingestion/internal/publisher/publisher_test.go`). Kafka broker integration is opt-in
through `TestKafkaPublishIntegration` in
`services/ingestion/internal/publisher/kafka_integration_test.go`.
