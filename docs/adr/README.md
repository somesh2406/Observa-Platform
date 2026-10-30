# Architecture Decision Records

The accepted/rejected decisions remain at their original paths under `docs/decisions/`.
Implementation notes are added to the records where the file is Markdown. The extensionless
ADR-008 record is preserved at its original path and is not renamed because the source archive
contains it without a `.md` extension.

| ADR | Decision | Status | Record |
|---|---|---|---|
| ADR-001 | Repository and service boundaries | Superseded in part; runtime boundary principles remain accepted | [ADR-001](../decisions/ADR-001-repository-and-service-boundaries.md) |
| ADR-002 | Ingestion acceptance boundary | Accepted | [ADR-002](../decisions/ADR-002-ingestion-acceptance-boundary.md) |
| ADR-003 | Go Kafka client | Selected; implementation added, target-environment verification pending | [ADR-003](../decisions/ADR-003-kafka-go-client.md) |
| ADR-004 | Vault-backed runtime secrets | Accepted | [ADR-004](../decisions/ADR-004-vault-secret-management.md) |
| ADR-005 | Go workspace and module boundaries | Accepted | [ADR-005](../decisions/ADR-005-go-workspace-and-module-boundaries.md) |
| ADR-006 | Local Docker Compose infrastructure | Historical; integrated Envoy-fronted stack is the default end-to-end path | [ADR-006](../decisions/ADR-006-local-compose-infrastructure.md) |
| ADR-007 | Envoy edge gateway and tenant isolation | Accepted | [ADR-007](../decisions/ADR-007-edge-gateway-and-tenant-isolation.md) |
| ADR-008 | Kafka topic provisioning strategy | Accepted | [ADR-008](../decisions/ADR-008-kafka-topic-provisioning-strategy) |
| ADR-009 | Separate Redis instances for edge and tenant quota | Accepted | [ADR-009](../decisions/ADR-009-redis-isolation.md) |

## Implementation note

The records are decision history. Current runtime behavior is documented in
[architecture HLD](../architecture/hld.md), [Kafka](../architecture/kafka.md),
[Gateway LLD](../architecture/lld-gateway.md), and [Redis runbook](../operations/runbook-redis.md).
