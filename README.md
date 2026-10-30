# Observa

Observa currently accepts one CloudEvents 1.0 structured JSON event per HTTP request, validates
the Observa v1 profile, applies a tenant quota, and publishes the original event bytes to Kafka.
The repository contains the ingestion service, an Envoy edge gateway configuration, local
infrastructure, and a one-shot Kafka topic provisioner.

## Current scope

Implemented: Envoy JWT validation and TLS termination, a shared edge rate limit, ingestion HTTP
handlers, CloudEvents validation, Redis-backed tenant quotas, Kafka publication, topic
provisioning, Vault-based Kafka credential retrieval, and local Docker Compose infrastructure.

Not implemented: Kafka consumer processing, ClickHouse, Apache Iceberg, object-storage
integration, PostgreSQL, OpenTelemetry, or Prometheus metrics.

## Request flow

The diagram depicts `deploy/local/docker-compose.yml`, `deploy/gateway/envoy.yaml`,
`services/ingestion/internal/ingestion/handler.go`, and
`services/ingestion/internal/publisher/publisher.go`.

```text
Client
  |
  | HTTPS POST /v1/events + JWT
  v
Envoy Gateway
  |-- JWT verification; tenant/source claim headers
  |-- edge rate-limit service --> redis-edge
  |
  v
Ingestion HTTP handler
  |-- CloudEvents profile validation
  |-- tenant quota --------------------> redis-tenant
  |
  v
Kafka producer
  |
  v
Kafka topic: observa.events.v1
```

The HTTP response is `202 Accepted` only after the producer receives broker delivery
confirmation. Delivery is at-least-once, not exactly-once. Any consumer added outside this
repository must deduplicate retries using the CloudEvents `id`; no consumer or deduplication
store is present here.

## Prerequisites

- Go toolchain `1.27.3` for local Go commands (`go.work`).
- Docker Engine and the Docker Compose plugin for the integrated local stack.
- `make`, `curl`, and `envsubst` for the documented workflows.
- A reachable OIDC issuer that issues JWTs with string `tenant_id` and `source_id` claims,
  plus a JWKS endpoint, when testing authenticated event ingestion through Envoy.

## Quickstart

```sh
cp .env.example .env
# Edit .env: set OBSERVA_JWT_ISSUER, OBSERVA_JWT_AUDIENCE, OBSERVA_JWKS_URL,
# OBSERVA_JWKS_HOST, OBSERVA_JWKS_PORT, and OBSERVA_GATEWAY_TLS_CERT_FILE /
# OBSERVA_GATEWAY_TLS_KEY_FILE to values valid in your environment.
make local-up
```

Set `TOKEN` to a JWT from the configured issuer, then send a valid event:

```sh
export TOKEN='replace-with-a-valid-jwt'
curl --fail-with-body --insecure \
  -X POST 'https://127.0.0.1:8443/v1/events' \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/cloudevents+json' \
  --data-binary '{
    "specversion":"1.0",
    "id":"quickstart-001",
    "source":"https://example.com/mobile",
    "type":"com.observa.user_action",
    "time":"2026-10-30T10:00:00Z",
    "datacontenttype":"application/json",
    "dataschema":"https://somesh2406.github.io/observa-event-contracts/schemas/event-data-v1.schema.json",
    "data":{"event_name":"loan.submit"}
  }'
```

A successful response has HTTP `202` and includes `status`, `id`, `ingestion_id`, and
`received_at`. The example event timestamp is illustrative; replace it with the event occurrence
time. Stop the stack with:

```sh
make local-down
```

The local smoke test (`make test-integration`) checks liveness/readiness and rejects an event
request without a JWT. It does not mint a valid JWT or prove successful authenticated publishing.

## Repository layout

- `libs/` — shared Go libraries, currently the CloudEvents profile validator.
- `services/` — Go services, currently the ingestion service and topic-provisioner command.
- `deploy/` — local Compose infrastructure and Envoy gateway resources.
- `configs/` — ingestion service configuration.
- `schemas/` — JSON Schemas for the CloudEvents envelope and Observa event data.
- `examples/` — valid producer event example.
- `scripts/` — schema validation, integration smoke test, and local Vault setup.
- `tests/` — shared event-contract fixture corpus.
- `docs/` — engineering notes, decisions, runbooks, contracts, and test plans.

## Documentation

- [Architecture](ARCHITECTURE.md)
- [Local setup](docs/getting-started/local-setup.md)
- [ADR index](docs/adr/README.md)
- [Documentation index](docs/README.md)

## Out of Scope (Deferred)

The following capabilities are not implemented in the current codebase and must not be assumed
by operators, integrators, or downstream services:

- Kafka consumer processing service
- ClickHouse integration
- Apache Iceberg / object-storage integration
- PostgreSQL integration
- OpenTelemetry tracing
- Prometheus metrics
