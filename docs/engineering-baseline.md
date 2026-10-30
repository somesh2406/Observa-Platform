# Engineering Baseline

## Repository state

The Go workspace contains `libs/eventcontract` and `services/ingestion`
(`go.work:1-8`). The ingestion module declares Go `1.27.3`
(`services/ingestion/go.mod:1-3`). The root `Makefile` provides `test`, `test-race`, `vet`,
`schema-check`, `check`, `run-ingestion`, local infrastructure, and integration-test targets.

## Current runtime components

- Ingestion HTTP service: `services/ingestion/cmd/ingestion/main.go`.
- Kafka topic provisioner: `services/ingestion/cmd/kafka-topic-provisioner/main.go`.
- Plain Envoy local gateway: `deploy/gateway/envoy.yaml`.
- Gateway API resource templates: `deploy/gateway/*.yaml.tmpl`.
- Local Compose stack: `deploy/local/docker-compose.yml`.

## Current validation commands

```sh
make test
make test-race
make vet
make schema-check
make check
```

`make check` runs `test`, `vet`, and `schema-check` (`Makefile:10-22`). Redis-backed limiter
tests require `OBSERVA_TEST_REDIS_ADDR` and skip when it is unset
(`services/ingestion/internal/ratelimit/limiter_test.go:12-18`).

## Out of Scope (Deferred)

There is no Kafka consumer, ClickHouse or Iceberg writer, object-storage integration, PostgreSQL
integration, OpenTelemetry instrumentation, or Prometheus metrics implementation in the current
source tree.
