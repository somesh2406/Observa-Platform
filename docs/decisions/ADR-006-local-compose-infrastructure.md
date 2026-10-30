# ADR-006: Local Docker Compose Infrastructure

- **Status:** Historical decision for standalone dependencies; the integrated Envoy-fronted stack is now the default end-to-end path
- **Scope:** Kafka, Vault, and optional Kafka UI used to exercise the ingestion service locally

## Decision

The original standalone Kafka and Vault Compose files remain available for dependency-only work. End-to-end ingestion testing now uses `deploy/local/docker-compose.yml`, which additionally starts Envoy as the sole public ingestion ingress, the Go ingestion image, Redis, and the external rate-limit service. Kafka remains single-node KRaft and Vault remains dev-mode for local use. Kafka UI remains an optional `ui` profile in the standalone Kafka Compose file and is reachable at loopback port 8023 when that file is used.

Kafka publishes a host listener at `localhost:9092` for the Go process running on the host and an internal listener at `kafka:29092` for other Compose services. The controller listener is not published to the host. Kafka log data is held in a named volume.

## Rationale

- The standalone Compose files allow starting only the dependency needed; the integrated Compose stack exercises the real gateway-to-ingestion request path.
- Explicit image tags make local setup more repeatable than floating `latest` tags.
- Separate advertised listeners prevent Docker containers from being told to connect to their own localhost.
- A named volume preserves local Kafka records across container recreation.
- Kafka UI is optional and does not need to run for ingestion to publish events.

## Consequences and limitations

- This is a single broker with replication factor 1, suitable only for local development.
- Vault dev mode has ephemeral in-memory storage and a known root token. It is not secure or durable.
- Kafka UI uses a floating `latest` tag in this initial local-only setup; pin a reviewed version for repeatable team CI.
- The local Kafka stack does not configure Kafka SASL/TLS. The integrated stack leaves Vault available for secret experiments but does not enable Vault-backed Kafka credentials until the broker is configured with matching authentication.
- Do not use `docker compose down -v` unless deleting the local Kafka records is intentional.


> Maintenance note: This ADR records the original local Compose choice for traceability. Routine local setup, image pulls, and machine-specific changes belong in the relevant runbook; do not create a new ADR for each local infrastructure operation.

## Implementation notes

The integrated local stack is defined in `deploy/local/docker-compose.yml:4-237` and includes
separate edge/tenant Redis instances, rate-limit configuration/service, Kafka, Vault, ingestion,
Envoy, and a topic provisioner. The root Makefile starts and stops this stack with the `local-up`
and `local-down` targets (`Makefile:41-46`).

