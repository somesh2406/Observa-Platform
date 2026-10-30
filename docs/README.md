# Observa Documentation

Documentation for the current Observa ingestion and gateway implementation.

## Getting started

- [Local setup](getting-started/local-setup.md) — prerequisites, Compose startup, ports, and verification.
- [Configuration](getting-started/configuration.md) — ingestion fields, defaults, environment overrides, and validation.
- [Troubleshooting](getting-started/troubleshooting.md) — observed failure symptoms and actionable checks.

## Architecture

- [High-level design](architecture/hld.md) — components, runtime boundaries, and request flow.
- [Ingestion low-level design](architecture/lld-ingestion.md) — HTTP handler, validation, quota, and Kafka publication.
- [Gateway low-level design](architecture/lld-gateway.md) — Envoy JWT, TLS, and edge rate-limit configuration.
- [Event contract](architecture/event-contract.md) — accepted CloudEvents profile and validator behavior.
- [Kafka](architecture/kafka.md) — topics, keys, producer settings, and provisioning.

## Operations

- [Local runbook](operations/runbook-local.md) — start, verify, inspect, and stop the local stack.
- [Topic provisioning runbook](operations/runbook-topic-provisioning.md) — catalogue-driven topic reconciliation.
- [Redis runbook](operations/runbook-redis.md) — separate edge and tenant Redis responsibilities.

## ADRs

- [ADR index](adr/README.md) — status and implementation links for the repository's decisions.

## Reference

- [Ingestion API](reference/api.md) — endpoint, headers, status codes, and error codes.
- [Glossary](reference/glossary.md) — platform vocabulary.

## Existing engineering records

- [Engineering baseline](engineering-baseline.md)
- [Product contract](product-contract.md) — product-level model and current implementation boundary.
- [Go workspace migration record](architecture/go-workspace-migration.md)
- [Earlier HLD record](architecture/hld-v0.1.md)
- [Earlier ingestion LLD record](architecture/ingestion-lld-v0.1.md)
- [Ingestion local-development runbook](runbooks/ingestion-local-development.md)
- [Local Compose infrastructure record](runbooks/local-infrastructure-docker-compose.md)
- [CloudEvents v1 contract record](contracts/event-contract-v1.md)
- [Schema hosting record](contracts/schema-hosting.md)
- [Event contract acceptance tests](testing/event-contract-v1-acceptance.md)
- [Increment 0 acceptance record](testing/increment-0-acceptance.md)
- [Ingestion implementation acceptance record](testing/ingestion-implementation-acceptance.md)
- [Postman ingestion test plan](testing/postman-ingestion-test-plan.md)
- [Gateway configuration notes](../deploy/gateway/README.md)
- [Local Compose notes](../deploy/local/README.md)
- [Ingestion service notes](../services/ingestion/README.md)

## Out of Scope (Deferred)

The following capabilities are not implemented in the current codebase:

- Kafka consumer processing service
- ClickHouse integration
- Apache Iceberg / object-storage integration
- PostgreSQL integration
- OpenTelemetry tracing
- Prometheus metrics
