# Kafka Topic Provisioning Runbook

## Catalogue

Edit `deploy/local/kafka/topics.yaml` only through the repository's normal change process.
The current catalogue declares local and production environments and two topics:
`observa.events.v1` and `observa.events.dlq.v1`.

## Local Compose provisioning

The integrated Compose stack runs `kafka-topic-provisioner` after Kafka is healthy and requires
successful completion before ingestion starts (`deploy/local/docker-compose.yml:211-226`,
`146-172`). To inspect its result:

```sh
docker compose -f deploy/local/docker-compose.yml logs kafka-topic-provisioner
docker compose -f deploy/local/docker-compose.yml ps -a kafka-topic-provisioner
```

The container uses `OBSERVA_ENVIRONMENT=local`, `KAFKA_BROKERS=kafka:29092`, and
`KAFKA_TOPIC_CATALOG=/app/topics.yaml`.

## Manual execution

The provisioner command requires `OBSERVA_ENVIRONMENT`, `KAFKA_TOPIC_CATALOG`, and
`KAFKA_BROKERS` (`services/ingestion/cmd/kafka-topic-provisioner/main.go:25-50`).
From the repository root, with Kafka reachable:

```sh
OBSERVA_ENVIRONMENT=local \
KAFKA_TOPIC_CATALOG=deploy/local/kafka/topics.yaml \
KAFKA_BROKERS=127.0.0.1:9092 \
go run ./services/ingestion/cmd/kafka-topic-provisioner
```

For the production catalogue, set `OBSERVA_ENVIRONMENT=production` and supply production broker
addresses and any required `KAFKA_TLS`, `KAFKA_SASL_MECHANISM`, `KAFKA_USERNAME`, and
`KAFKA_PASSWORD` environment values.

## Reconciliation behavior

The command waits up to 90 seconds for Kafka availability. It creates missing topics, rejects a
replication factor greater than the available broker count, verifies partition and replica
counts, reconciles declared topic configs, and verifies them after changes
(`services/ingestion/cmd/kafka-topic-provisioner/main.go:102-123`;
`services/ingestion/internal/kafkatopics/provisioner.go:26-55`, `58-140`,
`142-208`).

Partition-count and replication-factor drift are not automatically migrated; the command
returns an error and requires an explicit migration. Review broker metadata and catalogue
values before rerunning.

## Important boundary

The DLQ topic is provisioned but no consumer or DLQ writer exists in this repository. Topic
existence does not imply active dead-letter handling.
