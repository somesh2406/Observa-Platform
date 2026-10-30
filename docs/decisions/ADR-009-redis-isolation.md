# ADR-009: Separate Redis Instances for Edge Rate Limiting and Tenant Quotas

## Status

Accepted

## Context

Observa enforces rate limits at two layers:

1. **Edge (Envoy Gateway)** — coarse per-client abuse protection. High
   throughput, counters are ephemeral, failure can be tolerated.
2. **Tenant quotas (Ingestion service)** — per-tenant throughput limits
   defined by contract. Lower throughput, counters are business-critical,
   failure must not exceed quota.

Sharing a single Redis instance couples the two workloads' failure modes.
An edge attack saturates the shared Redis CPU, which causes the tenant
limiter (which fails closed) to reject all tenants' valid ingestion.

## Decision

Use two physically separate Redis instances:

- `redis-edge`: `maxmemory-policy=allkeys-lru`, no persistence, ephemeral.
- `redis-tenant`: `maxmemory-policy=noeviction`, AOF enabled, durable.

Both the local Docker Compose stack and production use this two-instance
topology. Only replication and management differ.

### Failure postures

- **Edge limiter: fail-open.** On Redis outage, requests pass through. Safe
  because the tenant limiter still enforces business quotas.
- **Tenant limiter: fail-closed.** On Redis outage, requests are rejected
  with 503. Fintech requirement — never exceed quota, even during outage.

### The pairing is what makes fail-open at the edge safe

If both layers shared a Redis instance, a Redis outage would disable both,
and fail-open at the edge would mean no protection at all. The isolation
of the backing stores is what makes the asymmetric postures correct.

## Out of scope (deferred)

- Prometheus metrics / alerting for each Redis instance
- Automated failover in production (managed-service feature)
- Multi-region replication for tenant Redis

## Consequences

- Two Redis instances to operate in production, with distinct secrets,
  backup policies, and alert routes.
- Edge attacks cannot cascade into tenant quota failures.
- Tenant quota counters survive Redis restart (AOF); edge counters do not
  (they are ephemeral by design).
- Cost: two small Redis instances instead of one large one — typically
  lower total cost because each is sized for its workload.
