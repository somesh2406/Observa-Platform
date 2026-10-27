# Ingestion Service — Implementation Acceptance Record

**Status:** Implementation and tests updated; target-toolchain build and real Kafka/Vault verification remain pending.

## Automated tests included

- Contract validator tests and the shared JSON Schema acceptance corpus.
- HTTP handler tests for confirmed publication (`202`), invalid event rejection, publication failure (`503`), oversized body (`413`), missing content type (`415`), empty body rejection, readiness failure, and liveness independent of Kafka.
- Configuration tests for explicit loopback development mode, blocking non-development startup until authentication exists, loopback-only unauthenticated binding, unknown configuration fields, environment overrides, invalid Vault paths, and SASL/Vault consistency.
- Vault adapter tests using an HTTP test server for KV v2 path, token forwarding, successful credential retrieval, invalid mount-prefixed paths, missing credentials, and required client settings.
- Kafka publisher constructor tests for missing broker/topic, partial SASL credentials, and unsupported SASL mechanisms.
- Application lifecycle tests for cleanup after listener startup failure and idempotent cleanup.
- Opt-in Kafka integration test (`TestKafkaPublishIntegration`) that verifies readiness and confirmed publication when explicit test environment variables are provided.

## Verification completed for this revision

- Go files formatted with `gofmt`.
- JSON Schema validator passed all 27 shared contract cases across 3 schemas.
- The shared `libs/eventcontract` unit tests passed under Go 1.23.2 in a compatibility copy; the delivered workspace still retains the agreed Go 1.27.3 directive.
- ZIP integrity and preservation manifests are checked when the final archive is assembled.

## Not verified in this environment

- Full compile/test of the ingestion module against the real Chi, Viper, Vault API and franz-go dependencies. External Go module downloads are blocked here.
- `go mod tidy` and generation/verification of missing checksums for the newly direct dependencies.
- `go test -race`, `go vet`, and all package tests under the required Go 1.27.3 toolchain.
- End-to-end HTTP -> Kafka publication against the user's running broker.
- Kafka SASL/TLS, which is not configured on the current local broker.
- The Go Vault adapter against the user's live Vault instance. Its automated test uses a local HTTP test server; live secret retrieval can be checked with the runbook's Vault CLI commands.

Follow `docs/runbooks/ingestion-local-development.md` on the developer machine. Do not interpret formatting, schema tests, or archive creation as proof that the real Kafka/Vault integration has passed.
