# Local Observa Compose Stack

Run from the repository root. Copy `.env.example` to `.env`, configure a valid test JWT issuer,
audience, JWKS endpoint and TLS files, then start the stack:

```sh
cp .env.example .env
make local-up
make local-logs
make test-integration
make local-down
```

## Services

| Service | Image / build | Ports | Dependencies | Health check |
|---|---|---|---|---|
| `redis-edge` | `redis:7.4-alpine` | `127.0.0.1:6379:6379` | none | `redis-cli ping` |
| `redis-tenant` | `redis:7.4-alpine` | `127.0.0.1:6380:6379` | none | `redis-cli ping` |
| `ratelimit-config` | build `deploy/local/rate-limit/Dockerfile` | none | none | none; one-shot successful exit |
| `ratelimit` | `envoyproxy/ratelimit:v1.4.0` | internal `8081` gRPC, `8080` HTTP | healthy `redis-edge`; successful `ratelimit-config` | none |
| `kafka` | `apache/kafka:4.3.1` | `127.0.0.1:9092:9092` | none | broker API versions command |
| `vault` | `hashicorp/vault:2.1.2` | `127.0.0.1:8200:8200` | none | `vault status` |
| `ingestion` | root `Dockerfile`, target `ingestion` | internal `8080`; not host-published | healthy Kafka, healthy `redis-tenant`, successful topic provisioner | `GET /health/live` |
| `envoy` | `deploy/gateway/Dockerfile` | `127.0.0.1:8443:8443` | healthy ingestion; started rate-limit service | Envoy admin `/ready` |
| `kafka-topic-provisioner` | root `Dockerfile`, target `topic-provisioner` | none | healthy Kafka | none; one-shot successful exit |

Source: `deploy/local/docker-compose.yml:6-237`. Ports and health checks are container-level
where shown; the rate-limit service and one-shot jobs have no Compose health check.

## Important behavior

The integrated Compose environment overrides the sample config to use authenticated mode,
binds ingestion to `0.0.0.0:8080` inside the private Compose network, and connects it to
`redis-tenant:6379`. The host only reaches ingestion through Envoy on loopback port `8443`.

The `make test-integration` smoke test checks both health endpoints and verifies that an event
request without a JWT receives `401`. It does not verify successful publishing with a valid JWT.

## Logs and teardown

```sh
make local-logs
make local-down
```

`make local-down` preserves named volumes. Use
`docker compose -f deploy/local/docker-compose.yml down --volumes --remove-orphans` only when
intentionally deleting local Kafka and tenant Redis data.
