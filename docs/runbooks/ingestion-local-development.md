# Ingestion Service — Local Development Runbook

## Scope and warning

This runbook is for an isolated developer machine. The sample configuration enables unauthenticated ingestion using a fixed development tenant/source and binds HTTP to `127.0.0.1`. Do not expose this mode to a LAN, shared environment, or production. Production startup intentionally fails until a producer authentication/identity adapter is implemented.

The local Vault Compose file uses Vault development mode, an in-memory store, and a known root token. It is not a production deployment.

## Prerequisites

- Go 1.27.3 (or a newer supported patch release), matching the repository `go.mod` toolchain directive.
- A running local Kafka broker reachable at `127.0.0.1:9092`.
- A Kafka topic named `observa.events.v1`.
- Docker Compose only if using the optional local Vault setup.

Run the Go checks on a machine with the required toolchain and dependency access. Unit tests do not require Kafka; the integration test and live API checks below do.

## 1. Verify the toolchain and modules

```powershell
go version
# Expected: go1.27.3 or a newer supported patch release

Push-Location libs/eventcontract
go mod tidy
Pop-Location

Push-Location services/ingestion
go mod tidy
Pop-Location

go test ./libs/eventcontract/... ./services/ingestion/...
go test -race ./libs/eventcontract/... ./services/ingestion/...
go vet ./libs/eventcontract/... ./services/ingestion/...
make check
```

Review dependency/license/vulnerability posture for `github.com/twmb/franz-go` before production release. Do not bypass a failed dependency download or test by treating it as a pass.

## 2. Prepare Kafka

Create the topic using the topic-management utility provided by your local Kafka distribution. Example for a Kafka installation that exposes `kafka-topics.sh`:

```powershell
docker exec observa-kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic observa.events.v1 --partitions 1 --replication-factor 1
```

The single-broker replication factor is for local development only. It does not meet the production durability target in ADR-002. If the topic already exists, inspect its configuration instead of recreating it.

## 3. Start the service without Vault (plaintext local Kafka)

Review `configs/ingestion.local.json`. It intentionally uses loopback HTTP and explicit development tenant/source values. Keep `vault.enabled` false when the local broker does not use SASL credentials.

```powershell
$env:OBSERVA_CONFIG_FILE = "configs/ingestion.local.json"
go run ./services/ingestion/cmd/ingestion
```

The process fails startup if the local-development identity mode is not explicitly enabled or the required Kafka configuration is invalid.

## 4. Configuration and environment overrides

Viper reads the JSON file, registers the supported configuration keys for environment lookup, and applies `OBSERVA_*` overrides. This explicit binding also permits an environment variable to supply a value when that key is absent from the JSON file. Environment values override file values. For list settings such as Kafka brokers, use a comma-separated value. Examples in PowerShell:

```powershell
$env:OBSERVA_HTTP_ADDRESS = "127.0.0.1:8080"
$env:OBSERVA_KAFKA_BROKERS = "127.0.0.1:9092"
$env:OBSERVA_KAFKA_TOPIC = "observa.events.v1"
```

Unset an override with `Remove-Item Env:OBSERVA_HTTP_ADDRESS` (or the relevant variable). Do not put secrets into the JSON file or commit local secret files. Vault is for Kafka credentials when SASL is actually enabled, not for ordinary listener/topic/timeouts.

## 5. Optional: run local Vault

In another terminal:

```sh
docker compose -f deploy/local/vault/docker-compose.yml up -d
./scripts/vault-local-setup.sh
```

The script writes placeholder credentials to KV v2 at `secret/observa/ingestion`. Those values do not authenticate to a Kafka broker unless you deliberately configure matching local broker credentials. Do not enable SASL with placeholders and expect a successful connection.

To use Vault-backed Kafka credentials, configure the local Kafka broker for a supported SASL mechanism, then edit `configs/ingestion.local.json`:

- Set `kafka.sasl_mechanism` to `PLAIN`, `SCRAM-SHA-256`, or `SCRAM-SHA-512` to match the broker.
- Set `vault.enabled` to `true`.
- Keep `vault.kv_v2_path` as `observa/ingestion`. This path is relative to the `secret` KV v2 mount; the Vault API client constructs `/v1/secret/data/observa/ingestion`.
- Keep the credential keys as `kafka_username` and `kafka_password` or update them to match the secret.
- Set `kafka.tls` according to the broker configuration.

Then run in PowerShell (only after Kafka has been configured for matching SASL credentials):

```powershell
$env:VAULT_TOKEN = "local-dev-only-root-token"
$env:OBSERVA_VAULT_ENABLED = "true"
$env:OBSERVA_KAFKA_SASL_MECHANISM = "PLAIN"
$env:OBSERVA_CONFIG_FILE = "configs/ingestion.local.json"
go run ./services/ingestion/cmd/ingestion
```

These overrides will not work against the current plaintext Kafka container: it has no SASL listener or matching broker credentials. For the current end-to-end smoke test, keep `vault.enabled=false` and `kafka.sasl_mechanism=""`. Use `vault kv get` to verify the secret exists, and test the Go Vault adapter with its automated HTTP test. Do not enable SASL with placeholder credentials.

Vault is read at startup. Missing secrets or a Vault outage cause startup failure. Credential refresh without restart is not implemented. The root token and dev-mode server must never be used outside local development.

## 6. Submit an event

From the repository root:

```powershell
curl.exe --fail-with-body -i -H "Content-Type: application/cloudevents+json" --data-binary "@examples/user-action.cloudevent.json" http://127.0.0.1:8080/v1/events
```

Expected result: `202 Accepted` only after Kafka confirms delivery. The response includes the event ID, an ingestion ID, and server receive time. It does not assert that downstream processing or storage has completed.

## 7. Verify health endpoints

```sh
curl -i http://127.0.0.1:8080/health/live
curl -i http://127.0.0.1:8080/health/ready
```

Liveness is independent of Kafka availability. Readiness pings Kafka and returns `503` when Kafka is unreachable.

## 8. Verify the Kafka record

Consume from `observa.events.v1` using the consumer utility supplied by your Kafka distribution. Verify:

- The record value is the original CloudEvents JSON, unchanged.
- The record key is a JSON tuple `[tenant_id, source_id, event_id]`.
- Headers include `observa_tenant_id`, `observa_source_id`, `observa_ingestion_id`, `observa_received_at`, and `observa_contract_version`.

## 9. Automated Kafka integration test

The publisher integration test is opt-in so normal unit tests do not require a broker. The dedicated `make test-kafka-integration` target fails immediately if either required environment variable is missing. Ensure the topic exists, then run from the repository root in PowerShell:

```powershell
$env:OBSERVA_KAFKA_INTEGRATION_BROKERS = "127.0.0.1:9092"
$env:OBSERVA_KAFKA_INTEGRATION_TOPIC = "observa.events.v1"
make test-kafka-integration
```

The test verifies broker readiness and confirmed publication. It does not validate the event contract or exercise the HTTP endpoint; those are covered by handler/contract tests and the manual Postman/Kafka UI smoke test below.

## 10. Negative-path checks

- Submit malformed JSON or a schema-invalid event: expect `400` and no Kafka publication.
- Omit or change the content type: expect `415`.
- Exceed the configured body size: expect `413`.
- Stop Kafka, then submit a valid event: expect a non-`202` response (`503`); the outcome may be ambiguous if the broker accepted the record but confirmation was lost.
- Stop Vault while `vault.enabled=true`: service startup must fail.

## Known limitations before production

- Authentication and authorization are not implemented. The local identity is fixed configuration, not producer identity.
- No rate limiter or metrics exporter is included in this initial increment.
- Run the Go test, race, vet, and live integration checks on the target machine before treating this increment as verified. Vault/Kafka authentication testing requires a broker configured with matching SASL credentials.
- `202` confirms Kafka publication only, not raw evidence archival, ClickHouse insertion, Iceberg commit, or query visibility.
