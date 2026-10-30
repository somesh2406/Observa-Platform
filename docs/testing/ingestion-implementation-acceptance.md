# Ingestion Service — Implementation Acceptance Record

## Test coverage present in source

- Event contract unit tests: `libs/eventcontract/event_test.go`.
- HTTP handler tests: `services/ingestion/internal/ingestion/handler_test.go`.
- Configuration tests: `services/ingestion/internal/config/config_test.go`.
- Redis limiter tests: `services/ingestion/internal/ratelimit/limiter_test.go`.
- Vault adapter tests: `services/ingestion/internal/secrets/vault_test.go`.
- Kafka publisher tests and opt-in broker integration test:
  `services/ingestion/internal/publisher/publisher_test.go` and
  `services/ingestion/internal/publisher/kafka_integration_test.go`.
- Topic catalogue tests: `services/ingestion/internal/kafkatopics/catalogue_test.go`.
- Compose smoke test: `scripts/test-integration.sh`.

The limiter tests are Redis integration tests and skip unless `OBSERVA_TEST_REDIS_ADDR` is set.
There is no in-memory limiter implementation.

## Commands

```sh
make test
make vet
make schema-check
make local-up
make test-integration
make local-down
```

The integration smoke test verifies liveness, readiness, and unauthenticated JWT rejection. It
does not mint a valid token or prove successful authenticated publishing.

## Toolchain and environment

The workspace and ingestion module declare Go `1.27.3` (`go.work:1-3`,
`services/ingestion/go.mod:1-3`). A successful execution of these commands depends on the local
Go toolchain, Docker, and external infrastructure being available; this record does not claim
that those commands were executed as part of this documentation revision.
