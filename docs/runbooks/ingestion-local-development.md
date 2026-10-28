# Ingestion local-development runbook

This runbook describes the current Envoy-fronted local stack and the separate direct-host development mode. Local development uses a loopback-only unauthenticated configuration; the Compose ingestion container disables that mode and accepts identity only from Envoy-forwarded JWT claim headers.

## 1. Prerequisites and stack start

Install Go 1.27.3, Docker Compose v2, GNU Make, and `curl`. From the repository root:

```sh
cp .env.example .env
# Set OBSERVA_JWT_ISSUER, OBSERVA_JWT_AUDIENCE, OBSERVA_JWKS_URL,
# OBSERVA_JWKS_HOST and OBSERVA_JWKS_PORT to your IdP's actual values.
make local-up
make local-logs
```

The local HTTPS entry point is `https://127.0.0.1:8443`. The self-signed certificate is for local development and is not trusted by default. The local stack also starts a single-node KRaft Kafka broker, the existing-style dev Vault, Redis, and the Envoy external rate-limit service. Ingestion has no host-published port.

The `.env.example` JWT values are examples only. They allow the edge process to start, but event publishing requires a JWT signed by the configured issuer with the configured audience and string `tenant_id` and `source_id` claims.

## 2. Verify service readiness

```sh
curl -k -i https://127.0.0.1:8443/health/live
curl -k -i https://127.0.0.1:8443/health/ready
./scripts/test-integration.sh
```

`/health/live` should return `200`. `/health/ready` should return `200` only when Kafka is reachable. The integration smoke test also verifies that an event request without a JWT is rejected with `401`.

To check individual infrastructure services:

```sh
docker compose -f deploy/local/docker-compose.yml ps
docker compose -f deploy/local/docker-compose.yml exec kafka /opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server localhost:9092
docker compose -f deploy/local/docker-compose.yml exec redis redis-cli ping
curl -sS http://127.0.0.1:8200/v1/sys/health
```

## 3. Ingest a contract-valid event

Set a JWT in your shell (minted by the configured IdP for the configured audience), then submit the contract example:

```sh
curl -k --fail-with-body -i \
  -H "Authorization: Bearer $OBSERVA_TEST_JWT" \
  -H 'Content-Type: application/cloudevents+json' \
  --data-binary @examples/user-action.cloudevent.json \
  https://127.0.0.1:8443/v1/events
```

A `202 Accepted` means Kafka confirmed publication, not that any downstream system has processed the event. The ingestion service validates the event before applying the tenant-specific business quota. The edge's shared coarse global rate limit and ingestion's tenant quota serve different purposes.

## 4. Verify the Kafka record

The local Compose Kafka broker can be accessed from the host at `127.0.0.1:19092`; container clients use `kafka:29092`. Inspect the topic with a Kafka UI/client or run the CLI consumer:

```sh
docker compose -f deploy/local/docker-compose.yml exec kafka \
  /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 --topic observa-events-v1 --from-beginning
```

The record value is the original CloudEvents JSON. The Kafka key is `tenantID:sourceID` or `tenantID:sourceID:sessionID` when `data.context.session_id` is present. The event ID is intentionally not included in the key. The publisher uses confirmed delivery and a retry can still produce a duplicate if the broker accepted a record but the client did not observe confirmation; future consumers must deduplicate by CloudEvents `event.id`.

## 5. Direct host development mode

For handler/debug work that does not need the gateway, stop or leave the Compose stack as appropriate, then ensure Kafka is available at `127.0.0.1:19092` or adjust the local config. `configs/ingestion.local.json` defaults to `127.0.0.1:8080`, fixed local tenant/source values, and `development.allow_unauthenticated=true`.

```sh
make run-ingestion
```

Configuration validation permits unauthenticated mode only when `environment=development`, tenant/source values are set, and the bind address is literally `127.0.0.1` or `::1`. `localhost`, wildcard addresses, and non-loopback addresses are rejected. Never expose this mode to a LAN or production.

## 6. Delivery semantics and troubleshooting

- `400`: invalid CloudEvent or body; no publication.
- `401`: gateway rejected the request or ingestion did not receive trusted tenant/source headers.
- `413`: request body exceeded the configured limit.
- `415`: request content type is not `application/cloudevents+json`.
- `429`: edge-wide or tenant-specific rate limit was exceeded.
- `503`: Kafka publication was not confirmed or the tenant limiter could not make a reliable decision.

The ingestion boundary is at-least-once. A timeout may leave an ambiguous Kafka outcome, so retries may duplicate an event. Consumers must deduplicate by `event.id`; consumer-side deduplication is not part of this service.

Stop the local stack with `make local-down`. The command preserves named volumes; remove them only when intentionally deleting local Kafka records.
