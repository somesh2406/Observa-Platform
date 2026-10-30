# Ingestion HTTP API

## Event ingestion

| Property | Value |
|---|---|
| Method | `POST` |
| Path | `/v1/events` |
| External transport | HTTPS through Envoy |
| Required content type | `application/cloudevents+json` |
| Body | One Observa CloudEvents 1.0 structured JSON event |
| Maximum body | `http.max_body_bytes`, default 1 MiB, maximum 16 MiB |

The route is registered at `services/ingestion/internal/ingestion/router.go:12-15`.
The handler enforces media type and body size at
`services/ingestion/internal/ingestion/handler.go:69-100`.

## Headers

- `Content-Type: application/cloudevents+json` is required.
- `Authorization: Bearer <JWT>` is required at Envoy for `/v1/events`.
- Envoy maps verified JWT claim `tenant_id` to `X-Observa-Tenant`.
- Envoy maps verified JWT claim `source_id` to `X-Observa-Source`.
- The ingestion handler requires both identity headers when
  `development.allow_unauthenticated=false`.

The mapping is in `deploy/gateway/envoy.yaml:43-67`; handler use is in
`services/ingestion/internal/ingestion/handler.go:103-116`. The Gateway API mapping is in
`deploy/gateway/security-policy.yaml.tmpl:10-22`.

## Responses

| HTTP status | Exact `code` | Exact `message` | Trigger |
|---:|---|---|---|
| `202` | — | — | Kafka confirms delivery. Body includes `status: accepted`, event `id`, `ingestion_id`, `received_at`. |
| `400` | `invalid_request_body` | `Request body could not be read` | Body read error other than max-bytes error. |
| `400` | `invalid_event` | `Request body must contain one event` | Empty/whitespace body. |
| `400` | `invalid_event` | `Event does not conform to the Observa event contract` | Event decode or validation error. |
| `401` | `missing_trusted_identity` | `Verified tenant and source identity are required` | Tenant or source identity header missing in authenticated mode. |
| `413` | `request_too_large` | `Request body exceeds the configured limit` | Declared or actual body exceeds configured limit. |
| `415` | `unsupported_media_type` | `Content-Type must be application/cloudevents+json` | Missing, malformed, or unsupported media type. |
| `429` | `tenant_rate_limit_exceeded` | `Tenant ingestion quota exceeded` | Tenant quota exceeded. |
| `503` | `rate_limiter_unavailable` | `Tenant quota could not be checked` | Limiter missing or Redis operation failed. |
| `503` | `publication_unconfirmed` | `Event publication was not confirmed; retry with the same event id` | Kafka delivery not confirmed. |
| `500` | `internal_error` | `The request could not be processed` | Ingestion ID generation failed. |

Exact response strings are in `services/ingestion/internal/ingestion/handler.go:58-169`.
Gateway-level JWT and edge-rate-limit responses are generated before the ingestion handler and
are not assigned the handler's JSON error codes.

## Health endpoints

| Method | Path | Success behavior |
|---|---|---|
| `GET` | `/health/live` | `200` with `{"status":"live"}`; does not check Kafka. |
| `GET` | `/health/ready` | `200` with `{"status":"ready"}` only when Kafka `Ping` succeeds; otherwise `503 not_ready`. |

Routes: `services/ingestion/internal/ingestion/router.go:17-23`.
Readiness response: `services/ingestion/internal/ingestion/handler.go:58-67`.

## Delivery boundary

A `202` means the Kafka producer received delivery confirmation. It does not mean a consumer
processed the event or that a downstream store/query projection is available. Delivery is
at-least-once; clients should retry an unconfirmed event using the same CloudEvents `id`.
