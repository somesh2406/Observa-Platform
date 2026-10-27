# Observa Backend — Go Workspace

Observa is maintained as a Git monorepo with a Go workspace. Independently deployable Go services have their own modules; shared, stable contracts live in explicit shared modules. The workspace does not combine service processes: ingestion publishes to Kafka, and a separate processor will consume events in a later increment.

## Workspace layout

- `go.work` — local workspace for the modules below.
- `libs/eventcontract/` — shared CloudEvents 1.0 / Observa event validator.
- `services/ingestion/` — HTTP ingestion service and Kafka producer.
- `schemas/` — canonical JSON Schema files, retained at their original paths.
- `tests/fixtures/event-contract-cases.json` — shared acceptance corpus.
- `docs/` — architecture, ADRs, contracts, testing, and runbooks.
- `migration-preservation/source-archives/` — byte-for-byte source archives and SHA-256 manifest for traceable migration.

See [Go workspace migration record](docs/architecture/go-workspace-migration.md) and [ADR-005](docs/decisions/ADR-005-go-workspace-and-module-boundaries.md).

## Event contract

The existing CloudEvents 1.0 structured JSON profile and Observa payload contract are preserved. The canonical JSON Schema entry point is `schemas/producer-event-v1.schema.json`, which composes `schemas/cloudevent-v1.schema.json` and `schemas/event-data-v1.schema.json`. The Go validator lives in `libs/eventcontract`. The shared fixture corpus is consumed by both Go contract tests and the JSON Schema validator.

Published schema URLs:

- Producer event: https://somesh2406.github.io/observa-event-contracts/schemas/producer-event-v1.schema.json
- CloudEvents profile: https://somesh2406.github.io/observa-event-contracts/schemas/cloudevent-v1.schema.json
- Event data: https://somesh2406.github.io/observa-event-contracts/schemas/event-data-v1.schema.json

## Ingestion behavior

The ingestion service accepts CloudEvents 1.0 structured JSON, validates it through `libs/eventcontract`, and publishes the unchanged event body to Kafka. It returns `202 Accepted` only after Kafka delivery confirmation. This confirms broker publication only, not evidence archival, ClickHouse/Iceberg writes, or UI visibility. Authentication and authorization remain deferred; unauthenticated operation is permitted only in explicit development mode with a loopback bind and configured development tenant/source context. Do not expose this mode to untrusted networks.

## Toolchain and verification

The repository targets Go 1.27.3. The environment used to prepare this archive has Go 1.23.2, so full Go workspace compilation could not be run here. External module downloads and Docker were also unavailable; real Kafka/Vault integration remains unverified. Do not interpret archive creation or source-level checks as production verification.

Install the target Go toolchain and Python development dependencies, ensure local Kafka is running, then run from the repository root:

```sh
python -m pip install -r requirements-dev.txt
make check
make test-race
make run-ingestion
```

For Vault setup and end-to-end checks, follow [`docs/runbooks/ingestion-local-development.md`](docs/runbooks/ingestion-local-development.md).

## Documentation

- [Product contract](docs/product-contract.md)
- [Engineering baseline](docs/engineering-baseline.md)
- [High-level design](docs/architecture/hld-v0.1.md)
- [Ingestion low-level design](docs/architecture/ingestion-lld-v0.1.md)
- [Workspace migration record](docs/architecture/go-workspace-migration.md)
- [Event contract v1](docs/contracts/event-contract-v1.md)
- [Schema hosting rules](docs/contracts/schema-hosting.md)
- [ADR-001: repository/service boundaries](docs/decisions/ADR-001-repository-and-service-boundaries.md)
- [ADR-002: ingestion acceptance boundary](docs/decisions/ADR-002-ingestion-acceptance-boundary.md)
- [ADR-003: Kafka Go client](docs/decisions/ADR-003-kafka-go-client.md)
- [ADR-004: Vault secrets](docs/decisions/ADR-004-vault-secret-management.md)
- [ADR-005: Go workspace/module boundaries](docs/decisions/ADR-005-go-workspace-and-module-boundaries.md)
- [Event contract acceptance](docs/testing/event-contract-v1-acceptance.md)
- [Ingestion acceptance record](docs/testing/ingestion-implementation-acceptance.md)

## Local infrastructure: Kafka and Vault

The local Docker Compose setup and step-by-step explanation are documented in
[`docs/runbooks/local-infrastructure-docker-compose.md`](docs/runbooks/local-infrastructure-docker-compose.md).

Kafka Compose file: `deploy/local/kafka/docker-compose.yml`  
Vault Compose file: `deploy/local/vault/docker-compose.yml`

The Kafka UI is an optional `ui` profile in the Kafka Compose file. These are local-development configurations, not production deployments. If you already started Kafka/Kafka UI manually with `docker run` using the same container names, keep using those containers; do not start duplicate Compose services until you intentionally migrate the setup.
