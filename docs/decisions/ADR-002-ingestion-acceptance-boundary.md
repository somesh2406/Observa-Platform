# ADR-002: Ingestion Acceptance Boundary

- **Status:** Accepted
- **Date:** 2026-10-16; revised for edge/tenant isolation on 2026-10-27
- **Decision owners:** Observa engineering

## Context

Ingestion must acknowledge events without falsely claiming downstream processing or storage completion. A synchronous cross-system dual write would introduce partial-success cases. Kafka producer idempotence does not deduplicate arbitrary repeated HTTP requests or provide exactly-once guarantees across external systems.

## Decision

`202 Accepted` means the Kafka producer returned a successful delivery result for the unchanged, validated CloudEvents body. It does not mean the event was processed downstream or is query-visible. The ingestion boundary is **at-least-once**: a timeout can leave the broker outcome ambiguous, and a client retry can publish the same logical event more than once. Consumers must deduplicate using CloudEvents `event.id`.

The Kafka record key is `tenantID:sourceID` when no session ID is present and `tenantID:sourceID:sessionID` when `data.context.session_id` exists. `event.id` is intentionally excluded from the key so events from the same tenant/source/session share a partition key rather than being partitioned by event identity.

The local stack uses one Kafka broker. Its replication and availability characteristics are for development only and do not establish production durability.

## Implementation notes

- Envoy is the single public HTTP ingress and owns TLS termination, JWT issuer/audience/JWKS validation, and the coarse shared edge limit.
- Envoy forwards verified `tenant_id` and `source_id` claims as `X-Observa-Tenant` and `X-Observa-Source`.
- Ingestion validates the event contract and applies a separate tenant-specific quota before publishing.
- A non-success HTTP status is returned when tenant limiting fails or Kafka delivery is not confirmed.

## Consequences

The HTTP acceptance boundary is explicit and testable. A `202` confirms broker publication only. Retry behavior is necessarily at-least-once, and there is no synchronous duplicate/conflict ledger in this service. A downstream consumer must define deduplication behavior for duplicate `event.id` values and should not treat Kafka key partitioning as deduplication.

## Out of Scope (Deferred)

OpenTelemetry instrumentation, Prometheus metrics, consumer processing, evidence storage, ClickHouse, and Iceberg are not implemented by this change. No archival or query-storage write occurs in the ingestion request path.
