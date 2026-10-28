# Ingestion API — Postman Test Plan

## Preconditions

This collection targets the **direct loopback-only development mode** of ingestion. For the gateway-fronted local stack, use `https://127.0.0.1:8443`, a JWT from the configured test issuer, and `make test-integration` as documented in `deploy/local/README.md`.

- Start Kafka and ensure topic `observa-events-v1` exists.
- Start ingestion with `configs/ingestion.local.json`; the sample binds only to `127.0.0.1:8080` and uses fixed local development tenant/source identity.
- Import `docs/testing/postman/Observa-Ingestion-Local.postman_collection.json` into Postman.
- Set collection variable `baseUrl` to `http://127.0.0.1:8080`.
- Before and after the run, keep a Kafka consumer/UI open on `observa-events-v1` so accepted and rejected requests can be correlated with actual records.

## Expected behavior

| ID | Request | Expected HTTP result | Kafka expectation |
|---|---|---|---|
| P01 | Liveness | `200` | No record |
| P02 | Readiness while Kafka is healthy | `200` | No record |
| P03 | Valid user-action event without a session ID | `202` | One record with the original JSON value and Kafka key `tenantID:sourceID` |
| P04 | Valid business-state event | `202` | One record |
| P05 | Valid technical-signal event | `202` | One record |
| P06 | Malformed JSON | `400 invalid_event` | No record |
| P07 | Missing required CloudEvents field | `400 invalid_event` | No record |
| P08 | Unsupported CloudEvents type | `400 invalid_event` | No record |
| P09 | Unknown envelope field | `400 invalid_event` | No record |
| P10 | Missing `Content-Type` | `415 unsupported_media_type` | No record |
| P11 | `Content-Type: application/json` | `415 unsupported_media_type` | No record |
| P12 | Empty/whitespace body with correct content type | `400 invalid_event` | No record |
| P13 | Body larger than configured 1 MiB limit | `413 request_too_large` | No record |
| P14 | Same event ID submitted twice | Both calls currently return `202` if Kafka is healthy | Expect two Kafka records today, with the same tenant/source/session key and distinct ingestion IDs. This verifies at-least-once behavior, not deduplication. |
| P15 | Valid event while Kafka is unavailable | `503 publication_unconfirmed` after publish timeout | Outcome can be ambiguous if Kafka accepted the record but acknowledgement was lost; do not assume zero records solely from the HTTP response |
| P16 | Readiness while Kafka is unavailable | `503` | No record |

## How to run the negative infrastructure cases

### Oversized body (P13)

Use a local file larger than 1 MiB and select **Body → binary** in Postman, or temporarily set `http.max_body_bytes` to a small value such as `1024`, restart the service, and send a valid event larger than that limit. Restore the configuration afterward. Do not commit the temporary setting.

### Kafka unavailable (P15–P16)

Stop only the local Kafka broker/container, keep ingestion running, execute P15 and P16, then restart Kafka. Allow up to the configured publish timeout for P15 to return. Re-check service readiness after Kafka has recovered.

## Evidence to record

For each request, record the Postman status/body and whether a corresponding Kafka record appeared. For accepted events, inspect the unchanged event JSON and headers `observa_tenant_id`, `observa_source_id`, `observa_ingestion_id`, `observa_received_at`, and `observa_contract_version`. The Kafka key is `tenantID:sourceID` when `data.context.session_id` is absent and `tenantID:sourceID:sessionID` when it is present; `event.id` is never part of the key. Rejected requests should not produce a record, except that a broker acknowledgement timeout has an explicitly ambiguous outcome.

## Current boundary

The local development identity is fixed by configuration and is not authenticated producer identity. The ingestion service does not deduplicate event IDs. A `202` confirms Kafka publication only; it does not mean that downstream archival or query projections have completed.
