# Observa ingestion platform

Observa is a fintech application-event ingestion service. The current implementation accepts one Observa CloudEvents 1.0 structured JSON event per request, validates it using `libs/eventcontract`, applies a tenant-scoped request quota, and publishes the unchanged event body to Kafka. Envoy is the only public HTTP entry point: it terminates TLS, validates JWTs for event publishing, forwards verified tenant/source claims, and applies a coarse shared edge rate limit.

## Request flow

```text
Client
  | HTTPS + Authorization: Bearer <JWT>
  v
Envoy edge gateway
  | TLS termination; JWT issuer/audience/JWKS validation
  | edge-wide rate limit; forwards verified tenant_id/source_id claims
  v
Observa ingestion service
  | validates CloudEvents contract and trusted identity
  | applies per-tenant business quota
  | publishes original event body, with tenant/source/session Kafka key
  v
Kafka topic: observa-events-v1
```

`202 Accepted` means the Kafka producer confirmed broker delivery. The ingestion boundary is **at-least-once**: a timeout can leave an ambiguous publish outcome, so clients may retry and an event may be published more than once. Future consumers must deduplicate using CloudEvents `event.id`; the ingestion service does not claim exactly-once processing.

## Repository layout

- `services/ingestion/` — HTTP handler, gateway-identity consumption, tenant limiter, and Kafka publisher.
- `libs/eventcontract/` — CloudEvents 1.0 and Observa event validation.
- `deploy/gateway/` — Envoy Gateway API resources and the plain-Envoy fallback.
- `deploy/local/docker-compose.yml` — single local stack for Envoy, ingestion, Kafka, Vault, Redis and the edge rate-limit service.
- `schemas/`, `tests/fixtures/`, and `docs/contracts/` — event schema, shared fixtures, and contract documentation.
- `docs/adr/ADR-006-edge-gateway-and-tenant-isolation.md` — edge/tenant isolation decision.

## Prerequisites

- Go **1.27.3** (matches `go.work` and the ingestion module).
- Docker Engine or Docker Desktop with the Docker Compose v2 plugin.
- GNU Make.
- `curl` for `make test-integration`; Python and `requirements-dev.txt` are needed for schema validation.

## Run the complete local stack

```sh
cp .env.example .env
# Edit the JWT issuer, audience, JWKS URL, and JWKS host to match an actual IdP.
make local-up
make local-logs
make test-integration
make local-down
```

The edge is published only on `https://127.0.0.1:8443`; the local self-signed certificate is under `deploy/gateway/local/` and is for local development only. Vault is available on `http://127.0.0.1:8200`. Ingestion, Redis, and the edge rate-limit service have no host-published ports. Kafka's broker port is bound to `127.0.0.1:19092`; container-to-container traffic uses `kafka:29092`.

The default issuer/JWKS values in `.env.example` are placeholders, not a working identity provider. The stack can start and its health endpoints can be checked with those defaults; event submission requires a JWT minted by the configured issuer and containing string claims `tenant_id` and `source_id`, plus the configured audience. Do not use the local certificate or development Vault token outside local development.

To stop the stack without deleting persisted Kafka records, run `make local-down`. `make local-down` does not remove named volumes. Use `docker compose -f deploy/local/docker-compose.yml down -v` only when intentionally deleting local Kafka data.

## Tests and verification

```sh
# From the workspace root: test every package in both workspace modules.
go test ./libs/eventcontract/... ./services/ingestion/...
# The same Go wildcard command also works from an individual module root:
(cd services/ingestion && go test ./...)
make test-integration
make check
make test-race
```

`make test-integration` starts the Compose stack and checks liveness, readiness through Envoy, and JWT rejection for an unauthenticated event request. It does not validate a real IdP token; configure a valid JWT issuer/JWKS pair and test token separately. `make check` runs Go tests, `go vet`, and JSON Schema checks; `make test-race` runs Go tests with the race detector.

For manual host-based ingestion development, `configs/ingestion.local.json` keeps unauthenticated mode enabled and binds only to `127.0.0.1:8080`; use `make run-ingestion` with local Kafka available at the address in that config. Startup refuses this mode if the bind address is not literal `127.0.0.1` or `::1`. The local Compose process runs with unauthenticated mode disabled and is not published directly to the host.

## Configuration overview

The ingestion service reads `configs/ingestion.local.json` by default or the file given by `--config` / `OBSERVA_CONFIG_FILE`. `OBSERVA_*` variables override the matching config fields. The important groups are:

- `HTTP_*` — bind address, request/body limits and timeouts.
- `DEVELOPMENT_ALLOW_UNAUTHENTICATED`, `DEVELOPMENT_TENANT_ID`, `DEVELOPMENT_SOURCE_ID` — loopback-only local mode and its fixed identity.
- `KAFKA_*` — brokers, topic, TLS, SASL mechanism and Vault secret key names. SASL `PLAIN` is rejected unless Kafka TLS is enabled.
- `VAULT_*` — optional Vault KV v2 retrieval of Kafka credentials. The KV path is relative to the `secret` mount, e.g. `observa/ingestion`.
- `TENANT_RATE_LIMIT_*` — the business-layer quota backend (`memory` or `redis`), requests/window and Redis connection settings. Local development defaults to memory; production replicas should use a shared Redis instance.
- The Compose-level `OBSERVA_JWT_*`, `OBSERVA_JWKS_*`, `OBSERVA_EDGE_RATE_LIMIT_*`, and `OBSERVA_GATEWAY_*` variables configure the edge layer. `OBSERVA_GATEWAY_RATE_LIMIT_POLICY_PATH` points to the Gateway API edge rate-limit resource template.

The edge and ingestion limits are complementary: Envoy's shared coarse limit protects the gateway/backend from bursts; the ingestion limiter applies the per-tenant business quota after event validation and trusted-identity resolution. The ingestion backend must be private and reachable only through the gateway; trusted identity headers are not a replacement for network isolation.

## Current scope

- JWT validation and TLS termination at Envoy.
- Shared coarse rate limiting at the edge.
- CloudEvents validation and body-size enforcement.
- Trusted `tenant_id` and `source_id` ingestion identity from gateway-verified claims, with a loopback-only unauthenticated development mode.
- In-memory and Redis-backed per-tenant rate limiting.
- Kafka publication with a key of `tenantID:sourceID` or `tenantID:sourceID:sessionID`, when a session ID exists.
- Liveness/readiness endpoints and tests for the current ingestion behavior.

## Out of Scope (Deferred)

OpenTelemetry instrumentation, Prometheus metrics, consumer processing, evidence storage, ClickHouse, and Iceberg are not implemented by this revision and are not represented as implemented. A consumer that provides deduplication by `event.id` is also outside this ingestion-only change.

## Documentation

- [Gateway configuration and policy boundary](deploy/gateway/README.md)
- [Local Compose services and verification](deploy/local/README.md)
- [Ingestion service README](services/ingestion/README.md)
- [ADR-006: edge gateway and tenant isolation](docs/adr/ADR-006-edge-gateway-and-tenant-isolation.md)
- [Event contract v1](docs/contracts/event-contract-v1.md)
- [Ingestion low-level design](docs/architecture/ingestion-lld-v0.1.md)
- [Local ingestion runbook](docs/runbooks/ingestion-local-development.md)
