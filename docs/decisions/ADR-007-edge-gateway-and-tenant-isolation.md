# ADR-006: Envoy edge gateway and tenant isolation

- **Status:** Accepted for the current ingestion increment
- **Date:** 2026-10-27

## Context

The ingestion service previously exposed its HTTP listener directly and used a fixed development tenant/source identity. Production-style use needs one public ingress, a clear edge security boundary, and per-tenant business quotas without embedding a full producer-authentication implementation in the ingestion service.

## Decision

Use Envoy as the single public HTTP ingress. The production configuration uses Envoy Gateway API resources (`GatewayClass`, `Gateway`, `HTTPRoute`, `SecurityPolicy`) and a global `BackendTrafficPolicy`. The local Compose stack uses the plain-Envoy `deploy/gateway/envoy.yaml` fallback with the same event route, JWT issuer/audience/JWKS inputs, tenant/source claim mappings, TLS termination, and shared coarse edge rate-limit intent.

Envoy validates the JWT on `POST /v1/events`, overwrites `X-Observa-Tenant` and `X-Observa-Source` from the verified `tenant_id` and `source_id` claims, and applies the shared coarse rate limit before forwarding requests. TLS is terminated at Envoy. Health routing is separate from the JWT-protected event route.

The ingestion handler remains responsible for request size checks, CloudEvents contract validation, requiring the forwarded tenant/source identity in authenticated mode, and applying tenant-specific business quotas. A memory limiter is available for local development; a Redis-backed fixed-window counter allows production ingestion replicas to share tenant quotas. Configure the ingestion Service/network policy so untrusted callers cannot bypass Envoy; the identity headers are trusted only because they come through the controlled gateway network path.

The Kafka record key is `tenantID:sourceID` without a session ID and `tenantID:sourceID:sessionID` when the event includes `data.context.session_id`. The event ID is not part of the key, preserving same-session partition affinity.

## Delivery contract

Ingestion returns `202 Accepted` only after Kafka confirms delivery. This boundary is at-least-once: timeouts may be ambiguous and retries may duplicate events. Future consumers must deduplicate by CloudEvents `event.id`; that responsibility is not implemented in this change.

## Consequences

The edge owns TLS, JWT validation and coarse global rate limiting. Ingestion owns event/schema validation and tenant-level quotas. The limits are complementary and not duplicated: edge limiting protects the shared service from bursts, while tenant limiting applies business-specific admission after the tenant identity is known. Local and production configurations share the same route/claims/limit intent; the local environment uses plain Envoy because it does not require a Kubernetes Gateway API controller.

## Out of Scope (Deferred)

OpenTelemetry instrumentation, Prometheus metrics, consumer processing, evidence storage, ClickHouse, and Iceberg remain out of scope and are not implemented by this decision. No downstream consumer or deduplication store is added.

## Implementation notes

The local Envoy configuration validates JWTs for `/v1/events`, maps `tenant_id` and `source_id`
to trusted headers, and invokes the external edge rate-limit service
(`deploy/gateway/envoy.yaml:43-77`). The ingestion handler consumes those headers when
unauthenticated development mode is disabled (`services/ingestion/internal/ingestion/handler.go:103-116`).
The current bootstrap always constructs a Redis tenant limiter; there is no in-memory mode
(`services/ingestion/internal/platform/bootstrap/build.go:81-106`).

