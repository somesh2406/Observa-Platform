# Redis Operations Runbook

## Two independent Redis roles

| Instance | Caller | Purpose | Local persistence / memory policy |
|---|---|---|---|
| `redis-edge` | Envoy external rate-limit service | Shared coarse edge limit | RDB/AOF disabled; `256mb`; `allkeys-lru` |
| `redis-tenant` | Ingestion Go limiter | Per-tenant fixed-window quota | AOF enabled with `everysec`; `512mb`; `noeviction` |

Both use `redis:7.4-alpine` in local Compose. Their ports are loopback-only: edge `6379`, tenant
`6380` (`deploy/local/docker-compose.yml:6-56`).

## Verify connectivity

```sh
docker compose -f deploy/local/docker-compose.yml exec redis-edge redis-cli ping
docker compose -f deploy/local/docker-compose.yml exec redis-tenant redis-cli ping
```

Expected output for each command:

```text
PONG
```

## Edge limiter

Envoy calls the rate-limit service at `ratelimit:8081`. The service connects to
`redis-edge:6379` and loads a generated policy under the `observa-ingestion` domain
(`deploy/local/docker-compose.yml:70-94`; `deploy/gateway/envoy.yaml:68-77`).
The plain Envoy configuration sets `failure_mode_deny: true`: this is fail-closed behavior
when the external rate-limit check fails. The edge Redis instance is configured to evict
least-recently-used counters under memory pressure; counters are intentionally ephemeral.

## Tenant limiter

The ingestion service connects to `redis-tenant:6379` in Compose. Bootstrap always constructs
`RedisLimiter`; startup pings Redis and fails if it is unavailable. Runtime Redis errors are
returned to the handler, which responds with `503 rate_limiter_unavailable`
(`services/ingestion/internal/platform/bootstrap/build.go:81-106`;
`services/ingestion/internal/ratelimit/limiter.go:91-134`;
`services/ingestion/internal/ingestion/handler.go:118-136`).

The limiter uses an atomic Lua increment and sets the key expiry only on the first increment.
The Redis key is `observa:ingestion:tenant-limit:v1:` plus the lowercase hexadecimal SHA-256
digest of the tenant ID (`services/ingestion/internal/ratelimit/limiter.go:42-51`,
`119-134`).

## Failure diagnosis

```sh
docker compose -f deploy/local/docker-compose.yml logs --tail=100 ratelimit
docker compose -f deploy/local/docker-compose.yml logs --tail=100 redis-edge
docker compose -f deploy/local/docker-compose.yml logs --tail=100 redis-tenant
docker compose -f deploy/local/docker-compose.yml logs --tail=100 ingestion
```

A tenant Redis outage prevents ingestion startup or causes tenant quota checks to return `503`.
An edge rate-limit service failure is denied by the local Envoy filter configuration. The
repository does not define a Redis failover or multi-region procedure.
