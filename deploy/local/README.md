# Local Observa Compose stack

Run from the repository root. This stack publishes ingestion HTTP traffic only through Envoy.

```sh
cp .env.example .env
# Replace the example JWT issuer/audience/JWKS values with a real test IdP.
make local-up
make local-logs
make test-integration
make local-down
```

## Services and ports

| Service | Container role | Host access | Health/verification |
|---|---|---|---|
| `envoy` | Single public HTTP ingress; TLS, JWT validation, shared edge rate limit | `https://127.0.0.1:8443` (loopback only) | `docker compose -f deploy/local/docker-compose.yml ps`; admin readiness is checked inside its container on `127.0.0.1:9901/ready`; `curl -k https://127.0.0.1:8443/health/live` |
| `ingestion` | Go service image built from the repository `Dockerfile`; validates CloudEvents, applies tenant quota and publishes to Kafka | No host-published port; internal `ingestion:8080` only | Compose health check calls `/health/live` inside the container; verify through Envoy at `/health/ready` |
| `kafka` | Single-node KRaft broker for local development | `127.0.0.1:19092`; container clients use `kafka:29092` | `docker compose -f deploy/local/docker-compose.yml exec kafka /opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server localhost:9092` |
| `vault` | Existing-style dev Vault for local secret experiments; ingestion is not configured to fetch Kafka credentials in this plaintext Kafka stack | `http://127.0.0.1:8200` | `curl -sS http://127.0.0.1:8200/v1/sys/health`; dev token is `local-dev-only-root-token` |
| `redis` | Redis backing the edge global rate-limit service | No host-published port; internal `redis:6379` | `docker compose -f deploy/local/docker-compose.yml exec redis redis-cli ping` should return `PONG` |
| `ratelimit-config` | One-shot renderer that writes the edge limit policy into the shared runtime volume | No port; successful exit code `0` is expected | `docker compose -f deploy/local/docker-compose.yml ps -a ratelimit-config` and `docker compose -f deploy/local/docker-compose.yml logs ratelimit-config` |
| `ratelimit` | Envoy external rate-limit gRPC service with the shared coarse edge policy | No host-published port; internal `ratelimit:8081` | Inspect `docker compose -f deploy/local/docker-compose.yml logs --tail=100 ratelimit` for the loaded `observa-ingestion` domain; verify requests through Envoy and check `429` after exhausting the configured bucket |

## Readiness and expected behavior

- `GET /health/live` through Envoy returns `200` without a JWT.
- `GET /health/ready` returns `200` only when ingestion can reach Kafka.
- `POST /v1/events` without a valid JWT returns `401` at Envoy before ingestion receives it.
- A valid token must have the configured issuer and audience, plus string `tenant_id` and `source_id` claims. The values are forwarded as `X-Observa-Tenant` and `X-Observa-Source`.
- After the JWT check, ingestion validates the event and consumes the per-tenant business quota. Local Compose defaults to the in-memory tenant limiter; the edge-wide limit uses Redis independently. Its service configuration is rendered from `deploy/local/rate-limit/config.yaml.tmpl` into the Compose-managed runtime volume.

The `.env.example` issuer/JWKS values are placeholders, so the stack can become healthy without being able to accept a valid test JWT. Configure these values before testing authenticated event publishing. The `ratelimit-config` renderer is expected to exit successfully after writing its configuration; it is not a long-running service. The rate-limit service does not have a Compose healthcheck, so verify its startup logs and the enforced `429` behavior. The mounted TLS private key and the Vault dev root token are local-only and must not be used in a production deployment.

## Logs and teardown

```sh
make local-logs
make local-down
```

Teardown preserves the named Kafka volume. `docker compose -f deploy/local/docker-compose.yml down -v` deletes that volume and local Kafka records. The stack is a single-broker development topology, not a production Kafka deployment.
