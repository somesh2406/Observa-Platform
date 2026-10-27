# ADR-002: Ingestion Acceptance Boundary

- **Status:** Accepted and implemented in initial code; target-environment integration verification pending
- **Date:** 2026-10-16
- **Decision owners:** Observa engineering

## Context

Ingestion must acknowledge events without falsely claiming that all downstream stores are updated. Writing to Kafka and object storage synchronously would introduce a dual-write problem: one write may succeed while the other fails. Kafka producer idempotence also does not deduplicate arbitrary repeated HTTP requests across producer sessions or make writes to external systems exactly-once.

## Decision

For the first vertical slice, define `202 Accepted` to mean that Kafka has acknowledged the event under the configured Kafka durability policy. The response does not mean the event is already archived in RustFS, committed to Iceberg, inserted into ClickHouse, or visible through query APIs.

Production-oriented Kafka configuration target:

- Topic replication factor: `3`.
- Producer `acks=all`.
- Producer idempotence enabled.
- Topic/broker `min.insync.replicas=2`.
- Do not acknowledge success when the producer reports a delivery error or times out without confirmation.
- Retain events long enough to cover the agreed recovery and archival outage window; set the actual duration only after RPO/RTO and throughput are quantified.

The local development environment may use a single broker, but it must be labeled as a weaker durability profile and must not be used to claim production durability.

The producer event's stable identity is `(tenant_id, source_id, event_id)`. Kafka record keys will initially use this identity so repeats of the same logical identity are routed to the same partition. This does not establish per-session or per-journey ordering. Downstream processing must detect repeated identities and surface conflicting content; duplicate delivery is not solved solely by Kafka producer idempotence.

Raw evidence archival to RustFS is asynchronous downstream work. Its lag and failures must be observable, and Kafka retention/replay must be sufficient to recover within the agreed recovery window. This decision must be revisited if product requirements demand that the HTTP acknowledgement guarantee durable object-store evidence before returning.

## Implementation notes

The HTTP handler validates the existing CloudEvents body and publishes the original event bytes unchanged. Trusted development tenant/source identity and ingestion metadata are carried in Kafka headers; the Kafka key is a JSON tuple of tenant ID, source ID and CloudEvents event ID. This avoids modifying the producer event contract. The local implementation still requires a production authentication adapter before external exposure.

## Consequences

### Positive

- Avoids synchronous dual writes across Kafka and RustFS in the request path.
- Gives the API a precise, testable acceptance boundary.
- Decouples producer latency from analytical-store writes.
- Allows downstream processing and archival to recover from temporary outages while the Kafka record remains retained.

### Costs and limitations

- `202` does not mean query visibility or RustFS archival has completed.
- Durability depends on Kafka replication, ISR policy, broker storage, retention, and operational correctness.
- Eventual downstream visibility must be measured and communicated.
- End-to-end logical deduplication/conflict detection remains a processing responsibility.
- The system must alert on archival/processing lag before Kafka retention is exhausted.

## Evidence

Apache Kafka's producer configuration documentation explains that `acks=all` waits for in-sync replicas and that idempotent production requires compatible acknowledgement/retry/in-flight settings. Idempotence protects producer retries; it is not a general exactly-once guarantee for HTTP retries or external database writes.

- https://kafka.apache.org/40/configuration/producer-configs/

## Revisit when

- Required acceptance semantics change to include synchronous object-store durability.
- Throughput, availability, retention, RPO, or RTO targets are established and invalidate this policy.
- A persistent idempotency ledger or transactional architecture is justified by measured requirements.
