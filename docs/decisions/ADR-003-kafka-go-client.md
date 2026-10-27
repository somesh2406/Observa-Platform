# ADR-003: Go Kafka Client

- **Status:** Selected; implementation added, target-environment verification pending
- **Date:** 2026-10-16

## Context

The ingestion service must wait for an explicit Kafka delivery result, propagate request cancellation/deadlines, configure idempotent production, and later support consumer behavior without introducing a C toolchain dependency unless a measured requirement justifies it.

## Options considered

| Client | Strengths | Trade-offs relevant to Observa |
|---|---|---|
| `franz-go` (`github.com/twmb/franz-go`) | Pure Go; broad Kafka protocol/features; idempotent and transactional producer support; avoids CGO | Rich API/configuration needs deliberate wrapper and integration tests |
| `confluent-kafka-go` | Confluent-maintained client backed by librdkafka; strong performance and mature client implementation | Uses CGO; build/runtime platform and libc/toolchain details become part of the deployment contract |
| `segmentio/kafka-go` | Idiomatic Go API, context integration, pure Go | Must validate the exact Kafka protocol/features and durability controls needed for the selected broker version; avoid assuming all newer Kafka features are supported |

## Decision

Use `franz-go` behind Observa's small `EventPublisher` interface for the first implementation, subject to a pinned-version build and integration-test gate. Do not expose franz-go types in the HTTP/domain packages.

## Rationale

- Pure Go avoids a CGO dependency in the first containerized deployment.
- Its project documentation describes a broad Kafka feature set, including idempotent/transactional production and current consumer-group capabilities.
- The adapter boundary allows the client to be replaced if benchmarks, broker compatibility, support requirements, or operational evidence justify it.

## Implementation status

An initial adapter now uses franz-go with `acks=all`, idempotent production, bounded delivery timeouts, explicit synchronous delivery results, optional TLS, and optional PLAIN/SCRAM SASL. Idempotent produce cancellation is enabled to honor the HTTP delivery deadline; cancellation can leave the broker outcome ambiguous and may result in duplicate records, so consumers must deduplicate by logical event identity. The repository pins `v1.18.1` pending the required target-environment verification below. The current execution environment cannot resolve external module downloads, so the dependency was not fetched and this adapter has not been compiled or integration-tested here.

## Required verification before production

- Pin a reviewed version and verify supported Go version, Kafka broker compatibility, license, release/security posture, and dependency vulnerability status.
- Integration-test delivery acknowledgements, timeouts, cancellation, TLS/SASL configuration, broker outage behavior, and shutdown/flush semantics.
- Configure `acks=all`, producer idempotence, bounded delivery timeout, and topic durability settings explicitly; do not rely on defaults without asserting them.
- Do not claim end-to-end exactly-once behavior from producer idempotence.

## References

- franz-go project: https://github.com/twmb/franz-go
- Confluent Go client and CGO requirements: https://github.com/confluentinc/confluent-kafka-go
- kafka-go project: https://github.com/segmentio/kafka-go
