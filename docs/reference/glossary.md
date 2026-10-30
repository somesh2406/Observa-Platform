# Glossary

- **Tenant** — the customer/organization identity derived from the verified JWT claim
  `tenant_id` in gateway-authenticated mode. Tenant identity is used for quota isolation.
- **Source** — the producer identity derived from the verified JWT claim `source_id` in
  gateway-authenticated mode. It is forwarded as `X-Observa-Source`.
- **Event** — one CloudEvents 1.0 structured JSON object submitted to `POST /v1/events`.
- **Namespace** — in this implementation, a syntactic segment in `data.event_name`; the
  validator applies a lowercase dotted/colon/hyphen/underscore pattern but does not check a
  namespace registry.
- **Category** — one of the three accepted CloudEvents `type` values:
  `com.observa.user_action`, `com.observa.business_state`, or
  `com.observa.technical_signal`.
- **Partition** — a Kafka topic subdivision to which records are assigned. The producer key
  groups the same tenant/source/session tuple.
- **Consumer group** — a Kafka consumer coordination identity. No consumer or consumer group
  implementation exists in this repository.
- **Edge rate limit** — the shared coarse limit applied by Envoy through its external
  rate-limit service and `redis-edge`.
- **Tenant quota** — the per-tenant fixed-window request counter applied by ingestion through
  `redis-tenant`.
- **Ingestion ID** — a server-generated random 16-byte hexadecimal identifier returned in the
  accepted response and written to Kafka header `observa_ingestion_id`.
- **Received timestamp** — server-generated UTC timestamp recorded in Kafka header
  `observa_received_at` and returned as `received_at`.
