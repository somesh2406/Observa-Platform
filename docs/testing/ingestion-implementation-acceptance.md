# Ingestion Service — Implementation Acceptance Record

**Status:** Source changes are present; automated runtime verification depends on the target Go toolchain and Docker environment.

## Automated test coverage in the source tree

- Contract validator tests and the shared JSON Schema acceptance corpus.
- HTTP handler tests for confirmed publication (`202`), invalid event rejection, publication failure (`503`), oversized body (`413`), missing content type (`415`), empty body rejection, readiness failure, and liveness independent of Kafka.
- Authenticated-mode handler tests for missing trusted identity, gateway-forwarded tenant/source claims, session ID propagation, and tenant-limit enforcement after event validation.
- Configuration tests for explicit loopback-only unauthenticated mode, gateway-authenticated production config, strict configuration decoding, environment overrides, invalid Vault paths, SASL/Vault consistency, explicit rejection of SASL `PLAIN` without TLS, and tenant limiter selection.
- Memory limiter tests for per-tenant isolation, fixed-window reset, and blank tenant rejection.
- Vault adapter tests using an HTTP test server.
- Kafka publisher constructor tests and an opt-in Kafka integration test (`TestKafkaPublishIntegration`).
- Compose smoke test checks liveness/readiness through Envoy and verifies JWT rejection when an event request has no token.

## Commands to run on a configured development machine

```sh
go test ./...
go vet ./...
python scripts/validate_event_schemas.py
make local-up
make test-integration
make local-down
```

Use Go 1.27.3 as declared by `go.work` and the ingestion module. Configure a real test JWT issuer/JWKS pair before testing successful event publishing. The integration smoke test does not claim to validate a valid JWT unless a separate authenticated request is made with a token minted by that issuer.

## Verification limitations for this archive build

The current execution environment cannot download the required Go 1.27.3 toolchain from `proxy.golang.org`, and Docker is not available. Therefore, a full Go build/test, Docker image build, Compose health test, or real Kafka/JWT end-to-end test cannot be reported as passing from this environment. Formatting, static file/config checks, schema validation, manifest hashing, and ZIP preservation checks are reported separately in the final revision output.
