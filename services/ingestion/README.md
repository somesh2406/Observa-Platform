# Observa Ingestion Service

This Go service accepts one CloudEvents 1.0 structured JSON event, validates the Observa contract through `libs/eventcontract`, enforces the tenant-specific business quota, and publishes the original event bytes to Kafka. The public HTTP route is Envoy's responsibility; do not expose the ingestion listener directly to untrusted clients.

## Request handling

1. Require `Content-Type: application/cloudevents+json` and enforce the configured request-body limit.
2. Parse and validate the CloudEvent. Invalid events are rejected before any tenant quota is consumed.
3. In gateway-authenticated mode, read `X-Observa-Tenant` and `X-Observa-Source` from headers that Envoy overwrites using verified JWT claims `tenant_id` and `source_id`. Requests missing either value receive `401`. Configure network policy/firewall rules so only the gateway can reach the ingestion service; these headers are trusted metadata, not credentials.
4. Apply the tenant-specific limiter. `memory` is process-local and intended for local development; `redis` uses an atomic Redis counter so application replicas share a fixed-window tenant quota. Limiter backend errors fail closed with `503`; exceeded quotas return `429`.
5. Publish the unchanged event body and return `202 Accepted` only after Kafka delivery confirmation.

The Kafka key is `tenantID:sourceID` when the event has no `data.context.session_id`, otherwise `tenantID:sourceID:sessionID`. `event.id` is deliberately excluded from the key so it does not split records from the same session across partitions.

## Delivery semantics

The ingestion boundary is **at-least-once**, not exactly-once. A timeout can leave the broker result ambiguous and retries can publish a duplicate. Any downstream consumer must deduplicate by CloudEvents `event.id`; no consumer or deduplication store is implemented in this service.

## Configuration

The service loads `configs/ingestion.local.json` or `--config` / `OBSERVA_CONFIG_FILE`; `OBSERVA_*` environment variables override config values.

- `environment` and `development.allow_unauthenticated`: unauthenticated mode is accepted only with `environment=development`, a fixed tenant/source, and a literal loopback bind of `127.0.0.1` or `::1`.
- `tenant_rate_limit.backend`: `memory` or `redis`; set `tenant_rate_limit.requests` and `tenant_rate_limit.window`. Redis mode also uses `redis_address`, `redis_password_env`, `redis_db`, `redis_tls`, and `dial_timeout`.
- `kafka.sasl_mechanism`: `PLAIN`, `SCRAM-SHA-256`, or `SCRAM-SHA-512`; `PLAIN` requires `kafka.tls=true`.
- `vault`: optional KV v2 secret lookup for Kafka SASL credentials. The path is relative to the KV mount.

## Run locally

The full local stack, including the edge gateway, is started with `make local-up` and stopped with `make local-down`. See [`deploy/local/README.md`](../../deploy/local/README.md). For direct host-only development, `make run-ingestion` reads the loopback-only local config and requires Kafka to be available at the configured address.
