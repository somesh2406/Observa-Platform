# Observa ingestion — High-level design

**Status:** Current ingestion/gateway implementation

## Goal

Accept one validated Observa CloudEvents 1.0 event, bind it to a trusted tenant/source identity, and publish the unchanged event body to Kafka. The HTTP acknowledgement indicates broker publication only.

## Current architecture

```text
Client -- HTTPS/JWT --> Envoy edge --> Ingestion service --> Kafka topic
                         |                 |
                         |                 +-- CloudEvents validation
                         +-- TLS            +-- tenant quota (memory/Redis)
                         +-- JWT            +-- at-least-once publish
                         +-- global rate limit
```

Envoy Gateway API resources are the production edge configuration. The local stack runs plain Envoy without a Kubernetes Gateway API controller. Both configurations require a JWT on `POST /v1/events`, validate issuer/audience and JWKS, forward `tenant_id` and `source_id` claims as `X-Observa-Tenant` / `X-Observa-Source`, terminate TLS, and use the same default coarse edge rate limit. Health is routed separately from the JWT-protected event route.

The ingestion service validates the request content type, body size, and event contract. In authenticated mode, it requires the gateway-forwarded tenant/source identity and applies its own per-tenant quota; it does not validate JWTs itself. The ingestion backend must be reachable only by the gateway. Local direct development mode is separately permitted only with `environment=development`, fixed tenant/source values, and a literal loopback bind.

Kafka receives the original CloudEvents JSON. Its key is `tenantID:sourceID` or `tenantID:sourceID:sessionID` if `data.context.session_id` is present. `event.id` is excluded from the partition key. The HTTP handler returns `202` only after producer delivery confirmation; the overall boundary is at-least-once, so consumers must deduplicate by `event.id`.

## Operational constraints

The Compose environment uses one local Kafka broker and local development TLS/Vault settings. These do not establish production durability. The tenant limiter's `memory` backend is process-local; production replicas that need a shared tenant quota must use a shared Redis backend. Redis/backend failures fail closed for the tenant quota.

## Out of Scope (Deferred)

OpenTelemetry instrumentation, Prometheus metrics, consumer processing, evidence storage, ClickHouse, and Iceberg are not implemented by this architecture increment. No downstream processing or evidence/query store is included in the request path.
