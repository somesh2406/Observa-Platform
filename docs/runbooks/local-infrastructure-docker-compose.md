# Local Infrastructure with Docker Compose

The authoritative local stack is `deploy/local/docker-compose.yml`. Its current services are
`redis-edge`, `redis-tenant`, `ratelimit-config`, `ratelimit`, `kafka`, `vault`, `ingestion`,
`envoy`, and `kafka-topic-provisioner`.

Use the [Local Setup guide](../getting-started/local-setup.md) and
[Local Stack Runbook](../operations/runbook-local.md) for current ports, startup, verification,
logs, and teardown. This record replaces the earlier Compose inventory, which referred to a
single Redis service and an in-memory tenant limiter.
