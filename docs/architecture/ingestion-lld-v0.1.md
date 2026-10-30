# Ingestion Service — Low-Level Design v0.1

This earlier LLD record is superseded by the current
[Ingestion Low-Level Design](lld-ingestion.md).

The current implementation always constructs a Redis-backed tenant limiter; there is no
in-memory limiter selection in the service bootstrap. The handler validates the CloudEvents
profile, resolves trusted tenant/source identity, applies the tenant quota, and publishes the
original event bytes to Kafka.

See [configuration](../getting-started/configuration.md),
[API reference](../reference/api.md), and [event contract](event-contract.md).
