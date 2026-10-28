# Ingestion Service — Low-Level Design v0.1

**Status:** Current implementation with Envoy edge and tenant quotas

## 1. Scope

Implemented behavior includes `POST /v1/events`, body-size/content-type enforcement, strict CloudEvents JSON decoding, Observa contract validation, gateway-forwarded tenant/source identity, a tenant-specific memory or Redis limiter, Kafka publication with delivery confirmation, health endpoints, environment-based configuration, and optional Vault KV v2 retrieval of Kafka SASL credentials.

## 2. Request path

```text
HTTPS request
  -> Envoy TLS + JWT validation + edge global limit
  -> forward verified tenant_id/source_id as X-Observa-Tenant / X-Observa-Source
  -> ingestion content type/body limit
  -> decode and validate CloudEvent
  -> resolve trusted identity (JWT claim headers, or configured loopback dev identity)
  -> enforce per-tenant business quota
  -> publish original CloudEvents bytes to Kafka
  -> return 202 only after delivery confirmation
```

The gateway owns JWT validation; ingestion does not parse or validate JWTs. In authenticated mode, requests without both trusted identity headers receive `401`. The health route is separate from the event route and is not JWT-protected. Keep the ingestion service private and restrict network access to the gateway.

## 3. HTTP contract

`POST /v1/events` requires `Content-Type: application/cloudevents+json` and one valid event. The handler enforces `http.max_body_bytes` before strict decoding and validates the event using `libs/eventcontract`. It does not modify the event body before publishing. The `202 Accepted` body includes the event `id`, a server-generated `ingestion_id`, `received_at`, and status `accepted`.

The handler resolves tenant/source identity only after contract validation. In authenticated mode, `X-Observa-Tenant` and `X-Observa-Source` are trusted only because the gateway overwrites them from verified JWT string claims and the backend network is private. Local unauthenticated mode uses the configured development identity and is accepted only for `environment=development`, with non-empty tenant/source and a literal `127.0.0.1` or `::1` listener bind.

## 4. Tenant rate limit

The business-layer limiter runs after event validation and tenant resolution, so invalid events do not consume tenant quota. `tenant_rate_limit.backend=memory` uses a per-process fixed window and is appropriate for local development only. `backend=redis` uses a Redis atomic counter keyed by a SHA-256 digest of tenant ID and a fixed TTL; replicas sharing the Redis database share a quota. Limiter errors fail closed with `503`; exceeded quotas return `429`.

This is separate from Envoy's global coarse rate limit. The edge filter protects the shared service from bursts; the ingestion limiter applies the business quota per verified tenant.

## 5. Configuration and security

Viper reads `configs/ingestion.local.json` or `--config` / `OBSERVA_CONFIG_FILE`; `OBSERVA_*` environment variables override those values. Unknown config fields and invalid effective values fail startup. SASL `PLAIN` is rejected unless `kafka.tls=true`. When Vault is enabled, its KV v2 path is relative to the `secret` mount and it supplies configured Kafka credentials.

For a production deployment, keep JWT verification and TLS at Envoy, supply a real issuer/audience/JWKS configuration, set tenant rate limiting to a shared Redis backend, and make the ingestion Service reachable only by the gateway. The service does not independently authenticate gateway headers.

## 6. Kafka publication semantics

- **Value:** original, validated CloudEvents JSON bytes.
- **Key:** `tenantID:sourceID` if no session ID is present, otherwise `tenantID:sourceID:sessionID` from `data.context.session_id`.
- **Headers:** trusted tenant/source IDs, ingestion ID, server receive time, and contract version.
- **Acknowledgement:** producer waits for a Kafka delivery result; no `202` is returned when delivery remains unconfirmed.
- **Delivery contract:** at-least-once. HTTP deadline cancellation can leave the broker result ambiguous and retry can duplicate a logical event. Downstream consumers must deduplicate by CloudEvents `event.id`; the event ID is not used as the Kafka key.

## 7. Implemented error behavior

| Condition | HTTP status |
|---|---:|
| Invalid content type | `415` |
| Oversized body | `413` |
| Empty, malformed, or invalid CloudEvent | `400` |
| Missing gateway tenant/source identity | `401` |
| Tenant quota exceeded | `429` |
| Tenant limiter failure | `503` |
| Kafka delivery unconfirmed | `503` |
| Kafka readiness failure | `503` |

## 8. Out of Scope (Deferred)

OpenTelemetry instrumentation, Prometheus metrics, consumer processing, evidence storage, ClickHouse, and Iceberg are not implemented by this service increment. Human-user login/query authorization, a producer-registration UI, downstream deduplication, and external persistence are also outside this ingestion service's current scope.
