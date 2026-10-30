# Observa Ingestion — High-Level Design v0.1

This earlier HLD record is superseded by the current
[High-Level Design](hld.md) and root [Architecture](../../ARCHITECTURE.md).

The current runtime consists of Envoy, the ingestion HTTP service, two separate Redis roles,
Kafka, a one-shot Kafka topic provisioner, and optional Vault credential retrieval. No consumer
or analytical storage/query component is implemented.

See [local topology](../getting-started/local-setup.md),
[Kafka](kafka.md), and [Redis operations](../operations/runbook-redis.md) for the current
configuration.
