# ADR-006: Local Docker Compose Infrastructure

- **Status:** Historical local-development record; operational instructions are maintained in the runbook
- **Scope:** Kafka, Vault, and optional Kafka UI used to exercise the ingestion service locally

## Decision

Use separate Compose files for Kafka and Vault. Kafka uses the pinned `apache/kafka:4.3.1` image in single-node KRaft mode. Vault uses the pinned `hashicorp/vault:2.1.2` image in dev mode. Kafka UI is an optional `ui` profile in the Kafka Compose file and is reachable at loopback port 8023.

Kafka publishes a host listener at `localhost:9092` for the Go process running on the host and an internal listener at `kafka:29092` for other Compose services. The controller listener is not published to the host. Kafka log data is held in a named volume.

## Rationale

- Separate Compose files allow starting only the dependency needed.
- Explicit image tags make local setup more repeatable than floating `latest` tags.
- Separate advertised listeners prevent Docker containers from being told to connect to their own localhost.
- A named volume preserves local Kafka records across container recreation.
- Kafka UI is optional and does not need to run for ingestion to publish events.

## Consequences and limitations

- This is a single broker with replication factor 1, suitable only for local development.
- Vault dev mode has ephemeral in-memory storage and a known root token. It is not secure or durable.
- Kafka UI uses a floating `latest` tag in this initial local-only setup; pin a reviewed version for repeatable team CI.
- The local Compose files do not configure Kafka SASL/TLS. Vault-backed Kafka credentials cannot be used until the broker is configured with matching authentication.
- Do not use `docker compose down -v` unless deleting the local Kafka records is intentional.


> Maintenance note: This ADR records the original local Compose choice for traceability. Routine local setup, image pulls, and machine-specific changes belong in the relevant runbook; do not create a new ADR for each local infrastructure operation.
