# Observa High-Level Design — Initial Boundary

**Status:** Initial logical view with the first local-development ingestion boundary implemented; production runtime architecture remains incomplete.

## Product views and shared event model

The first product release has four primary investigation views:

1. **User action timeline:** events attributed to a tenant-scoped user identity across connected sources.
2. **Application journey:** events associated with a business application (for example, a loan application), potentially across users and sessions.
3. **Session action trace:** events associated with a specific session, including technical events only when the source emits them.
4. **Technical incidents:** technical signals and incidents derived by explicit detection/correlation rules, correlated to affected sessions, users, business applications, and traces when supported by event context.

These views should be built from a common canonical event representation and shared correlation semantics. Incident detection/grouping is a derived capability over technical signals; it must not be conflated with event ingestion or assumed to be available before detection rules are designed. Do not build a separate ingestion pipeline or independently authoritative event history for each view. A single event may reference a user, business application, session, source, and request/trace, but these references are not always present. Unknown relationships remain unknown; the platform must not guess them.

Terminology rule: a **business application** is the loan/application record being traced; a **source application** is the software emitting events. These must not be conflated in schemas or APIs.

## Product flow

```text
Instrumented application
        |
        v
Authenticated ingestion boundary
        |
        v
Durable event transport
        |
        v
Processing / normalization
        |                    \
        v                     v
Original evidence       Query-optimized data
        \                     /
         v                   v
          Authorized Query API
                   |
                   v
         Investigation clients
```

## First ingestion implementation increment

The initial Go ingestion service implements the synchronous HTTP boundary and publishes the unchanged, validated CloudEvents JSON to Kafka. It waits for a Kafka delivery result before returning `202 Accepted`; that acknowledgement means broker publication only, not evidence archival or downstream query visibility. Trusted tenant/source development identity and ingestion metadata are carried in Kafka headers and a stable composite record key, not injected into the producer contract.

Authentication and authorization are intentionally not implemented in this increment. Unauthenticated mode is explicitly limited to `environment=development`, configured development tenant/source values, and a loopback HTTP bind. Production exposure remains blocked until a real producer authentication and tenant/source resolution adapter is implemented and tested. Kafka consumption and downstream storage remain the separate processing service's responsibility.

## Boundary principles

- The ingestion path must define what durable acceptance means before returning success.
- The event transport decouples producer availability from downstream processing but does not by itself guarantee end-to-end exactly-once writes.
- Processing must support bounded batches, retry policy, poison-event handling, replay, and destination reconciliation.
- Original evidence and normalized/query-optimized data have different access, retention, integrity, and lifecycle requirements; they may share infrastructure only if the guarantees remain clear.
- The query boundary enforces authentication, tenant authorization, query limits, and data minimization.
- Technical-event capture and incident detection are separate concerns: source instrumentation determines which signals exist, while explicit rules determine whether signals become grouped incidents.
- Iceberg, ClickHouse, RustFS, and Kafka are candidates/currently known technologies, not unconditional commitments for every future component. Their roles and necessity must be validated against product workloads and operational cost.

## Decisions intentionally deferred

- Exact service count and deployment topology.
- Whether Iceberg and ClickHouse are both required.
- Kafka partition count, replication factor, retention, and producer/consumer tuning.
- Event ordering and deduplication semantics.
- Multi-region topology.
- Exact API and event schema versions.
- Incident grouping, severity, deduplication, status lifecycle, and alerting semantics.

These require product workload, failure, recovery, and security requirements first.
