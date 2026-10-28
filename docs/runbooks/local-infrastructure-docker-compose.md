# Local infrastructure with Docker Compose

The recommended local setup is the integrated `deploy/local/docker-compose.yml` stack. Envoy is the only host-published HTTP ingress for ingestion; Kafka, Vault, Redis, and the edge rate-limit service are local supporting services.

## Start and stop

```sh
cp .env.example .env
# Set issuer/audience/JWKS values to an actual test IdP before publishing events.
make local-up
make local-logs
make test-integration
make local-down
```

The local Compose stack contains:

| Service | Purpose | Port / health check |
|---|---|---|
| Envoy | TLS, JWT validation, claim forwarding, coarse shared rate limit | `127.0.0.1:8443`; `/health/live` and `/health/ready` via TLS |
| Ingestion | CloudEvents validation, tenant quota, Kafka publication | private container port `8080`; Compose health check is `/health/live` |
| Kafka | Single-broker KRaft event topic | host-only `127.0.0.1:19092`; internal `kafka:29092` |
| Vault | Dev-mode Vault for local secret experiments | `127.0.0.1:8200`; `GET /v1/sys/health` |
| Redis | Shared store for Envoy's edge rate-limit service | private `redis:6379`; `redis-cli ping` returns `PONG` |
| Rate-limit service | External gRPC service used by Envoy's shared edge limit | private `ratelimit:8081`; inspect `docker compose logs ratelimit` |
| Rate-limit config renderer | Renders the same edge request/unit variables into the local rate-limit service config | one-shot container; expected exit code `0` |

## Verification commands

```sh
docker compose -f deploy/local/docker-compose.yml ps
curl -k -i https://127.0.0.1:8443/health/live
curl -k -i https://127.0.0.1:8443/health/ready
docker compose -f deploy/local/docker-compose.yml exec kafka \
  /opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server localhost:9092
docker compose -f deploy/local/docker-compose.yml exec redis redis-cli ping
curl -sS http://127.0.0.1:8200/v1/sys/health
```

A valid JWT is required for `POST /v1/events`. The example JWT issuer/JWKS values are placeholders; replace them with a real issuer that returns the configured audience and the `tenant_id` / `source_id` claims. Unauthenticated event requests should return `401`. The in-memory tenant quota and global edge limit are independent.

## Persistence and safety

`make local-down` does not delete named volumes. To delete the Kafka volume and local records intentionally:

```sh
docker compose -f deploy/local/docker-compose.yml down -v
```

Vault runs in dev mode with a known root token and ephemeral storage. It is not wired into the local plaintext Kafka credentials path. Do not expose the dev Vault, local TLS key, or local configuration outside an isolated development machine.
