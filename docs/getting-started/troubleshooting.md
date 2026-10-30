# Troubleshooting

| Symptom | Cause evidenced by implementation | Fix / check |
|---|---|---|
| Ingestion exits during startup with a configuration error | `config.Load` rejects missing environment, brokers, topic, invalid timeouts, or inconsistent Vault/SASL settings. | Check the exact validation error and compare values with [configuration](configuration.md). |
| Ingestion exits with `tenant rate limiter init failed` | Redis cannot be reached or Redis credentials/TLS are incorrect; bootstrap pings Redis before serving. | Check `OBSERVA_TENANT_RATE_LIMIT_REDIS_ADDRESS`, username/password env names, DB, TLS, and `redis-tenant` health. |
| `/health/live` works but `/health/ready` returns `503` | Readiness calls Kafka `Ping`; liveness does not. | Check Kafka health, broker address, topic configuration, and network reachability. |
| Event request returns `401` at Envoy | JWT missing/invalid, issuer/audience mismatch, or JWKS retrieval/verification failure. | Check token issuer/audience, `OBSERVA_JWT_*`, JWKS URL and reachability. |
| Event request returns `401 missing_trusted_identity` from ingestion | Authenticated mode did not receive `X-Observa-Tenant` or `X-Observa-Source`. | Check Envoy claim mapping and ensure the JWT contains string `tenant_id` and `source_id` claims. |
| Event request returns `415 unsupported_media_type` | Content type is not `application/cloudevents+json`. | Set the required media type. |
| Event request returns `413 request_too_large` | Body exceeds `http.max_body_bytes`. | Reduce the event size or configure a permitted limit (maximum 16 MiB). |
| Event request returns `400 invalid_event` | JSON is malformed or the event fails the strict Observa v1 validator. | Check `specversion`, `type`, `dataschema`, `data.event_name`, unknown fields, and optional-null fields. |
| Event request returns `429 tenant_rate_limit_exceeded` | Tenant fixed-window counter exceeded configured requests. | Wait for the configured window or review tenant quota values. |
| Event request returns `503 rate_limiter_unavailable` | Tenant Redis operation failed; the handler fails closed. | Restore Redis connectivity; inspect `redis-tenant` logs and credentials. |
| Event request returns `503 publication_unconfirmed` | Kafka delivery was not confirmed within the producer/publish timeouts. | Check Kafka broker health and producer logs. A timeout can have an ambiguous broker outcome; retry using the same CloudEvents `id`. |
| Envoy returns `503` or edge requests are denied when rate limiting is unavailable | Plain Envoy config sets `failure_mode_deny: true`; the edge limiter is not fail-open in this config. | Check `ratelimit`, `redis-edge`, and the loaded `observa-ingestion` domain. |
| Local Envoy fails to start on TLS | Certificate/key mount paths are invalid or the files are absent. | Set `OBSERVA_GATEWAY_TLS_CERT_FILE` and `OBSERVA_GATEWAY_TLS_KEY_FILE` to readable files. |
| Topic provisioner exits before ingestion starts | Kafka is not reachable, catalogue invalid, or actual topic partition/replication/configuration differs from the declared policy. | Inspect provisioner logs, broker count, `deploy/local/kafka/topics.yaml`, and Kafka metadata. |

Relevant implementation: `services/ingestion/internal/config/config.go`,
`services/ingestion/internal/ingestion/handler.go`,
`services/ingestion/internal/publisher/publisher.go`,
`services/ingestion/internal/ratelimit/limiter.go`,
`deploy/gateway/envoy.yaml`, and
`services/ingestion/internal/kafkatopics/provisioner.go`.
