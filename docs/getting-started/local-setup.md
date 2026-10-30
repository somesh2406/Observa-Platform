# Local Setup

## Prerequisites

- Go `1.27.3` for host-side Go commands (`go.work:1-3`).
- Docker Engine with Docker Compose plugin.
- `make`, `curl`, and `envsubst`.
- A JWT issuer and JWKS endpoint for successful event submission through Envoy.

## Start the integrated stack

Run these commands from the repository root:

```sh
cp .env.example .env
```

Edit `.env` to use an issuer, audience, JWKS URL/host/port, and certificate/key paths valid
for the environment. Then run:

```sh
make local-up
docker compose -f deploy/local/docker-compose.yml ps
```

`make local-up` executes `docker compose -f deploy/local/docker-compose.yml up -d --build`
(`Makefile:41-43`).

## Local ports

| Component | Host binding | Purpose |
|---|---|---|
| Envoy | `127.0.0.1:8443` | HTTPS API entrypoint |
| Kafka | `127.0.0.1:9092` | Host Kafka client access |
| Edge Redis | `127.0.0.1:6379` | Envoy edge rate-limit counters |
| Tenant Redis | `127.0.0.1:6380` | Ingestion tenant quota |
| Vault | `127.0.0.1:8200` | Local dev Vault |

The ingestion container listens on `0.0.0.0:8080` inside the Compose network and does not publish
a host port. The rate-limit service uses internal gRPC port `8081`. See
`deploy/local/docker-compose.yml:21-22`, `46-47`, `85-86`, `137-138`, `155-165`, and `196-198`.

## Verify

```sh
curl --insecure --fail https://127.0.0.1:8443/health/live
curl --insecure --fail https://127.0.0.1:8443/health/ready
make test-integration
```

Liveness returns `200` without checking Kafka. Readiness returns `200` only when the Kafka
producer's `Ping` succeeds (`services/ingestion/internal/ingestion/handler.go:58-67`).
The integration script checks both endpoints and verifies that an event request without a JWT
returns `401` (`scripts/test-integration.sh:29-34`). It does not verify a valid-token publish.

For a valid publish, set `TOKEN` to a JWT from the configured issuer and use the event example
and request shown in the root [README](../../README.md).

## Direct host development

The sample `configs/ingestion.local.json` binds to `127.0.0.1:8080`, enables unauthenticated
development identity, connects to Kafka at `127.0.0.1:9092`, and uses tenant Redis at
`127.0.0.1:6380`. Start the required Kafka and Redis services separately, then run:

```sh
make run-ingestion
```

The service still requires Redis; there is no in-memory tenant limiter in the current bootstrap.

## Stop

```sh
make local-down
```

This runs Compose `down --remove-orphans` and preserves named volumes. It does not remove Kafka
or tenant Redis data volumes (`Makefile:45-46`; `deploy/local/docker-compose.yml:228-232`).
