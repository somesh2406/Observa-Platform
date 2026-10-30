# Kafka

## Provisioned topics

The topic catalogue is `deploy/local/kafka/topics.yaml`. It defines common defaults:
`cleanup.policy=delete`, `compression.type=producer`, and `max.message.bytes=1048576`
(lines 4-7). `LoadCatalogue` also sets `min.insync.replicas`, `retention.ms`, and
`retention.bytes` from each environment's values
(`services/ingestion/internal/kafkatopics/catalogue.go:124-136`).

| Environment | Topic | Partitions | Replication factor | Min ISR | Retention | Retention bytes |
|---|---|---:|---:|---:|---:|---:|
| local | `observa.events.v1` | 1 | 1 | 1 | 259200000 ms (3 days) | 53687091200 (50 GiB) |
| local | `observa.events.dlq.v1` | 1 | 1 | 1 | 259200000 ms (3 days) | 10737418240 (10 GiB) |
| production catalogue | `observa.events.v1` | 12 | 3 | 2 | 604800000 ms (7 days) | 107374182400 (100 GiB) |
| production catalogue | `observa.events.dlq.v1` | 3 | 3 | 2 | 1209600000 ms (14 days) | 21474836480 (20 GiB) |

Source: `deploy/local/kafka/topics.yaml:4-36`. The `observa.events.dlq.v1` topic is provisioned,
but this repository contains no consumer or DLQ publisher/handler that writes to it.

## Local broker

The integrated Compose stack uses `apache/kafka:4.3.1` in single-node KRaft mode. Host clients
connect to `127.0.0.1:9092`; containers connect to `kafka:29092`
(`deploy/local/docker-compose.yml:96-126`). The standalone Kafka Compose file also defines an
optional `kafka-ui` profile on `127.0.0.1:8023`
(`deploy/local/kafka/docker-compose.yml:42-58`); it is separate from the integrated stack.

## Partition key

The producer key is:

- `tenantID:sourceID` when `data.context.session_id` is absent or empty.
- `tenantID:sourceID:sessionID` when a session ID is present.

The event ID is deliberately excluded
(`services/ingestion/internal/publisher/publisher.go:77-80`, `100-108`). The producer writes
the original event body and attaches metadata headers
(`services/ingestion/internal/publisher/publisher.go:83-94`).

## Producer settings

`publisher.NewKafka` configures:

| Setting | Implementation |
|---|---|
| Acknowledgements | `kgo.RequiredAcks(kgo.AllISRAcks())` |
| Record retries | `kgo.RecordRetries(10)` |
| Delivery timeout | `kgo.RecordDeliveryTimeout(o.DeliveryTimeout)` |
| Produce request timeout | `kgo.ProduceRequestTimeout(o.DeliveryTimeout)` |
| Dial timeout | `kgo.DialTimeout(3 * time.Second)` |
| Cancellation | `kgo.AllowIdempotentProduceCancellation()` |
| TLS | Optional; minimum TLS 1.2 when enabled |
| SASL | Optional PLAIN, SCRAM-SHA-256, or SCRAM-SHA-512 |

Source: `services/ingestion/internal/publisher/publisher.go:36-74`. The code comments explicitly
define the application delivery boundary as at-least-once; do not describe it as exactly-once.
The code does not explicitly call a separate idempotence-enable option.

## Provisioning

The one-shot command is `services/ingestion/cmd/kafka-topic-provisioner/main.go`. It reads
`OBSERVA_ENVIRONMENT`, `KAFKA_TOPIC_CATALOG`, and comma-separated `KAFKA_BROKERS`, loads the
matching catalogue, waits for Kafka, then reconciles topics
(`services/ingestion/cmd/kafka-topic-provisioner/main.go:25-124`).

Provisioning creates missing topics, refuses impossible replication factors, checks partition
and replica counts, reconciles the declared topic configs, and verifies the resulting configs
(`services/ingestion/internal/kafkatopics/provisioner.go:26-55`, `58-140`,
`142-208`). Partition-count and replication-factor drift require an explicit migration rather
than automatic reassignment.

## Local/production parity

The catalogue uses the same topic names and common cleanup/compression/message-size defaults in
both environments. The configured environment differences include replication factor, min ISR,
partition counts, retention durations, and retention byte limits
(`deploy/local/kafka/topics.yaml:4-36`). Therefore, it is not accurate to claim that only
replication factor and broker count differ between local and production. Local and production
catalogue values differ in additional fields listed above.
