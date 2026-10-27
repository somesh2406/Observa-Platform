# Ingestion Service — Low-Level Design v0.1

**Status:** Implemented initial local-development vertical slice; production hardening and environment verification remain open.

## 1. Goal

Accept one producer event, validate it, bind it to a trusted tenant/source identity, publish it to Kafka, and return success only after Kafka confirms publication under the configured durability profile.

## 2. Scope of the first slice

### Included

- `POST /v1/events`.
- Bounded request-body reading.
- Explicit local-development identity context; real producer authentication and authorization are deferred.
- Strict JSON decoding and event-contract validation.
- Server-assigned ingestion ID and `received_at` in the HTTP response and Kafka headers; the producer CloudEvent body remains unchanged.
- Kafka publication with explicit acknowledgement, idempotent producer settings, timeout and delivery-error handling.
- Viper-based JSON configuration with explicit environment overrides, strict decoding, validation and optional Vault KV v2 credential retrieval.
- Stable safe error responses and structured logs without raw payloads.
- Unit tests and HTTP tests with a fake publisher; integration test against Kafka in a separate test profile.

### Excluded

- RustFS archival, processing workers, Iceberg, ClickHouse, query APIs, incident detection, and UI.
- Human-user login and query authorization.
- A database-backed producer credential management UI.
- Promise of exactly-once delivery across Kafka and downstream destinations.

## 3. Request path

```text
HTTP request
  -> body size limit
  -> resolve configured development tenant_id + source_id (local-only; producer auth is deferred)
  -> strict JSON decode + contract validation
  -> keep CloudEvents body unchanged; attach trusted ingestion metadata to Kafka key/headers
  -> publish Kafka record and wait for broker acknowledgement
  -> return 202 only on confirmed publish
```

Any failure before confirmed publication returns a non-success response. If a timeout occurs after Kafka may have accepted the record but before the client receives confirmation, the result is ambiguous: the client may retry using the same CloudEvents `id`. This is expected distributed-systems behavior; the downstream identity/deduplication contract must handle it.

## 4. HTTP contract

### `POST /v1/events`

- Content type: `application/cloudevents+json` (CloudEvents structured JSON).
- Authentication: intentionally absent in this increment. Startup is allowed only when `environment=development` and `development.allow_unauthenticated=true`; configured development tenant/source values are required, and the HTTP listener must bind to loopback. All non-development environments fail startup until producer authentication and trusted identity resolution are implemented.
- Request body: one CloudEvents 1.0 structured JSON event validated against the canonical composed schema `schemas/producer-event-v1.schema.json` (which references `schemas/cloudevent-v1.schema.json` and `schemas/event-data-v1.schema.json`).
- Response on confirmed Kafka publication: `202 Accepted` with CloudEvents `id`, server-generated `ingestion_id`, `received_at`, and status `accepted`.
- The response does not imply RustFS archival, normalization, Iceberg commit, ClickHouse insertion, or query visibility.
- Error body: stable machine-readable `code` plus safe `message`; no request-body echo.

The handler returns stable machine-readable error codes for invalid events, unsupported content types, oversized bodies, publication failures, and internal failures. Keep those mappings covered by tests. Do not expose credentials or raw payload data; operational errors should be logged without secret material.

## 5. Configuration

Viper loads the JSON configuration file and applies explicitly bound `OBSERVA_*` environment overrides. Unknown configuration keys and invalid effective values fail startup. Non-secret operational settings remain in the configuration file or deployment environment. Vault is used for Kafka SASL credentials when SASL is enabled; its KV v2 path is relative to the `secret` mount (for example, `observa/ingestion`).

## 6. Trust and security boundaries

- Current development mode derives `tenant_id` and `source_id` from explicit server configuration. A later authentication adapter must derive them from verified producer identity.
- Producer JSON is validated by the existing event-contract validator and is not augmented or rewritten. Tenant/source and ingestion metadata are placed in Kafka record headers; the key is a JSON tuple of trusted tenant ID, source ID and CloudEvents ID.
- Credentials are never logged. Production credential storage, rotation, revocation, and audit controls require a separate security design before external deployment.
- Enforce request-size and read-time limits before decoding.
- Do not log event payloads or event attributes. Log operational errors only after ensuring they do not contain credentials or sensitive payload fragments.
- Return generic authentication failure responses that do not reveal whether a tenant/source exists.

## 7. Kafka publication semantics

- Record value: original validated CloudEvents JSON bytes, unchanged.
- Record key: stable JSON tuple `[tenant_id, source_id, event_id]`.
- Record headers: trusted tenant/source identity, ingestion ID, server `received_at`, and contract version.
- Producer must wait for a delivery result; fire-and-forget publication is prohibited.
- Production target: replication factor 3, `acks=all`, idempotence enabled, `min.insync.replicas=2`.
- If the publisher reports an error or times out without confirmation, do not return `202`. The producer allows idempotent in-flight cancellation to honor the request deadline; a canceled delivery can have an ambiguous broker outcome and may later be retried as a duplicate, so downstream deduplication is required.
- Kafka producer idempotence handles certain producer retry duplicates; it does not deduplicate repeated HTTP requests as a general logical-event guarantee.
- Event ordering is not guaranteed across a session or business application in v1. Consumers/query models use `occurred_at` plus `received_at` and a deterministic tie-breaker. Strict per-session ordering is deferred until a requirement justifies the partition-key trade-off.

## 8. Proposed error mapping

| Condition                                       | HTTP result | Notes                                                                        |
| ----------------------------------------------- | ----------: | ---------------------------------------------------------------------------- |
| Malformed JSON / schema violation               |       `400` | Stable validation code; no raw payload echo                                  |
| Missing/invalid producer credential             |       `401` | Generic message                                                              |
| Authenticated producer not permitted to publish |       `403` | No cross-tenant details                                                      |
| Request body exceeds limit                      |       `413` | Reject before full decode                                                    |
| Unsupported content type                        |       `415` | Only JSON in v1                                                              |
| Rate/concurrency quota exceeded                 |       `429` | Add retry guidance only when safe and meaningful                             |
| Kafka unavailable or delivery unconfirmed       |       `503` | Client may retry with the same event ID; outcome can be ambiguous on timeout |
| Unexpected internal failure                     |       `500` | Safe message; internal details only in protected logs                        |

This mapping is proposed and must be locked by tests before release. A duplicate/conflicting identity may be detected downstream; v1 does not promise synchronous conflict detection in the HTTP response.

## 9. Interfaces

Keep the HTTP handler independent of Kafka implementation details. Minimum boundaries:

- `Authenticator`: validates credential and returns trusted `tenant_id` and `source_id`.
- `EventValidator`: parses and validates the producer event.
- `Clock`: provides server UTC time, allowing deterministic tests.
- `EventPublisher`: publishes a canonical event and returns only after confirmed delivery or a defined error.
- `IDGenerator`: generates an `ingestion_id` for request tracing.

Use context cancellation and explicit timeouts for publisher calls. Do not build a generic plugin framework; these interfaces exist to test trust and infrastructure boundaries in isolation.

## 10. Acceptance tests

### Unit/handler tests

1. Valid event + authenticated producer + successful publisher returns `202` and includes server `received_at`.
2. Missing/invalid credential returns `401`; publisher is not called.
3. Credential resolves tenant/source; payload cannot override trusted identity.
4. Invalid JSON/schema returns `400`; publisher is not called.
5. Oversized request returns `413` without unbounded read.
6. Unsupported content type returns `415`.
7. Publisher returns an error -> response is non-success; never `202`.
8. Publisher timeout -> response is non-success and retry uses the same event ID; document possible ambiguous outcome.
9. Error response and logs do not contain payload contents or credentials.
10. Unknown top-level fields are rejected.
11. Server `received_at` is generated by the server, not accepted from the producer.

### Kafka integration tests

1. A valid event can be consumed from the configured topic after the API returns `202`.
2. The record key is stable for the same tenant/source/event identity.
3. Broker unavailability causes a non-success response.
4. The integration test asserts the configured durability profile; a single-broker local profile must not be labeled production-equivalent.

## 11. Operational requirements

Minimum signals for this slice:

- Request count by outcome and safe error code.
- Request latency histogram.
- Payload-size histogram without payload contents.
- Publish latency and publish failure/timeout counters.
- Active/in-flight request count and rate-limit rejections.
- Structured request/ingestion ID correlation.

Readiness must fail if the service cannot meet its declared ability to publish; liveness must not fail merely because Kafka has a transient outage, or orchestrators may create restart storms. The exact probe semantics and degraded behavior must be tested.

## 12. Implementation status and open gates

Implemented in this increment: HTTP ingestion, strict request limits and content type, existing event validation, explicit local-development identity, Kafka producer adapter, Kafka readiness check, optional Vault KV v2 read, unit-level handler tests, and local Vault setup files. The current review environment cannot run the repository's Go 1.27.3 toolchain because it only has Go 1.23.2 and cannot download the required toolchain. Runtime verification must therefore be recorded from the target developer machine using the runbook checks.

- Install and verify Go 1.27.3 or a newer supported patch release in the developer environment and CI. The module is aligned to Go 1.27.3; run the tests and integration checks with that toolchain before release.
- Verify the pinned franz-go version against the target Go version, broker compatibility, license, vulnerability status, and required delivery/security behavior (see ADR-003).
- Decide the production credential source and lifecycle; local test credentials are not a production authentication system.
- Confirm representative payload sizes and finalize the 1 MiB proposed body ceiling.
- Define actual Kafka topic retention, replication, ISR, and recovery windows from workload/RPO/RTO requirements.
