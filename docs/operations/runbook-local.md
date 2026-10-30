# Local Stack Runbook

## Start

From the repository root:

```sh
cp .env.example .env
```

Edit `.env` with a valid test issuer/audience/JWKS endpoint and local TLS certificate/key paths,
then:

```sh
make local-up
docker compose -f deploy/local/docker-compose.yml ps
```

## Verify

```sh
curl --insecure --fail https://127.0.0.1:8443/health/live
curl --insecure --fail https://127.0.0.1:8443/health/ready
make test-integration
```

The integration script expects `200` from both health endpoints and `401` for an event request
without a JWT (`scripts/test-integration.sh:29-34`).

## Send an authenticated event

```sh
export TOKEN='replace-with-a-valid-jwt'
curl --insecure --fail-with-body \
  -X POST https://127.0.0.1:8443/v1/events \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/cloudevents+json' \
  --data-binary @examples/user-action.cloudevent.json
```

The token must match configured issuer/audience and include string `tenant_id` and `source_id`
claims. The example file must validate against the currently checked-in event contract.

## Inspect logs and services

```sh
docker compose -f deploy/local/docker-compose.yml ps
make local-logs
docker compose -f deploy/local/docker-compose.yml logs --tail=100 ingestion
docker compose -f deploy/local/docker-compose.yml logs --tail=100 envoy
docker compose -f deploy/local/docker-compose.yml logs --tail=100 ratelimit
docker compose -f deploy/local/docker-compose.yml logs --tail=100 kafka-topic-provisioner
```

Check Redis independently:

```sh
docker compose -f deploy/local/docker-compose.yml exec redis-edge redis-cli ping
docker compose -f deploy/local/docker-compose.yml exec redis-tenant redis-cli ping
```

Each successful Redis ping returns `PONG`.

## Stop and preserve data

```sh
make local-down
```

This removes Compose containers/networks but preserves named volumes. To intentionally remove
local Kafka and tenant Redis data, use the destructive command below:

```sh
docker compose -f deploy/local/docker-compose.yml down --volumes --remove-orphans
```

## Run tests

```sh
make check
make test-integration
```

`make check` runs Go tests, `go vet`, and JSON Schema validation. `make test-integration` starts
the stack and runs the smoke test (`Makefile:19-22`, `51-52`).
