# Local Infrastructure Runbook: Kafka, Vault and Kafka UI

## Purpose and boundaries

This runbook starts the local infrastructure required for the Observa ingestion service. It does not start a Kafka consumer or storage processors.

- Kafka: `apache/kafka:4.3.1`, single-node KRaft mode, for local development.
- Vault: `hashicorp/vault:2.1.2`, dev mode, for learning and testing secret retrieval.
- Kafka UI: optional profile in the Kafka Compose file, reachable at `http://127.0.0.1:8023`.

These configurations are not production deployments. Kafka has one broker and replication factor 1. Vault dev mode uses in-memory storage and a known root token. Do not expose these services to a LAN or public network.

## Docker Compose concepts used here

- **Image**: packaged software downloaded from a registry (`docker pull`).
- **Container**: a running instance created from an image.
- **Compose file**: a YAML declaration of services, ports, environment, networks, health checks and volumes.
- **Port mapping**: `127.0.0.1:9092:9092` means host port 9092 forwards to container port 9092, bound to loopback only.
- **Environment**: settings passed into the container. They configure Kafka/Vault; they do not rebuild images.
- **Network/service DNS**: containers on the Compose network reach Kafka as `kafka:29092`. Host processes reach it as `127.0.0.1:9092`.
- **Volume**: `observa_kafka_data` stores Kafka log data outside the container's writable layer so it survives container recreation.
- **Health check**: a readiness probe. A container can be running before the service is ready.
- **Profile**: Kafka UI is optional and starts only when `--profile ui` is supplied.
- **`up -d`**: create/start services in the background.
- **`logs -f`**: follow service logs.
- **`down`**: stop and remove containers and the Compose network; named volumes are retained unless `-v` is supplied.
- **`down -v`**: also deletes named volumes. For Kafka this deletes local broker data. Do not use unless you intentionally want to reset local Kafka.

## 1. Check Docker is working

Run in PowerShell or a terminal:

```powershell
docker version
docker compose version
```

If either command fails, start Docker Desktop (or your Docker Engine) first.

## 2. Pull the images

From any directory:

```powershell
docker pull apache/kafka:4.3.1
docker pull hashicorp/vault:2.1.2
```

Kafka UI is not required yet. When ready for the UI step, pull it with:

```powershell
docker pull provectuslabs/kafka-ui:latest
```

`latest` is convenient for local experimentation but is mutable. For repeatable team/CI setups, choose and pin a reviewed Kafka UI version.

## Existing containers started with `docker run`

If Kafka and Kafka UI are already running from the manual `docker run` commands, **do not run Compose `up` for Kafka or Kafka UI yet**. The Compose file uses the same explicit container names (`observa-kafka` and `observa-kafka-ui`), so Docker will report a name conflict. Keep using the running containers for the ingestion smoke test.

Verify them first:

```powershell
docker ps --filter "name=observa-kafka"
docker ps --filter "name=observa-kafka-ui"
docker exec observa-kafka /opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server localhost:9092
```

Your manually started Kafka advertises `host.docker.internal:9092`. The host-based Go service should be able to use `127.0.0.1:9092` as its bootstrap address on Docker Desktop; if Kafka metadata connectivity fails, inspect the broker logs and advertised listener configuration. Your manually started Kafka UI is configured for `host.docker.internal:9092`, which matches that manual broker configuration.

The Compose setup below is the reproducible alternative for a future clean local setup. Do not remove existing containers or volumes blindly; a manually started Kafka container without a named volume may lose its local records when removed.

## 3. Start Kafka (only if you are using Compose rather than the already-running manual container)

Run these commands from the repository root (the directory containing `go.work`):

```powershell
docker compose -f deploy/local/kafka/docker-compose.yml up -d kafka
docker compose -f deploy/local/kafka/docker-compose.yml ps
docker compose -f deploy/local/kafka/docker-compose.yml logs --tail 100 kafka
```

Wait until `docker compose ... ps` reports Kafka as healthy. If it is unhealthy, inspect logs before continuing.

### Why Kafka has two client listeners

Kafka gives clients broker addresses in its metadata. If a container advertises `localhost:9092`, another container interprets `localhost` as itself, not as Kafka. This Compose setup therefore has two listeners:

- Host application (Go ingestion): `127.0.0.1:9092` / advertised as `localhost:9092`.
- Other containers (Kafka UI): `kafka:29092` using Docker's internal DNS.

The controller listener on 9093 is internal to Kafka and is not published to the host. This is a single-node KRaft setup; no ZooKeeper is required.

## 4. Create and verify the Observa topic

Create the topic if it does not already exist:

```powershell
docker exec observa-kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic observa.events.v1 --partitions 1 --replication-factor 1
```

List topics:

```powershell
docker exec observa-kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list
```

Describe the topic:

```powershell
docker exec observa-kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --describe --topic observa.events.v1
```

For local development, one partition and replication factor 1 are sufficient. These settings do not demonstrate production availability or replication guarantees.

## 5. Start Vault

Vault is optional while the local Kafka broker uses plaintext and no SASL credentials. If the Vault container is already running, do not create a second instance; verify it and continue. Start it only if it is not running:

```powershell
docker ps --filter "name=observa-vault-dev"
docker compose -f deploy/local/vault/docker-compose.yml up -d
docker compose -f deploy/local/vault/docker-compose.yml ps
docker compose -f deploy/local/vault/docker-compose.yml logs --tail 100 vault
```
If `observa-vault-dev` is already running, the `docker ps` check is sufficient; the Compose command is only needed if it is not running or is already managed by that Compose project.

Set the development token for the setup script in the current PowerShell session:

```powershell
$env:VAULT_ADDR = "http://127.0.0.1:8200"
$env:VAULT_TOKEN = "local-dev-only-root-token"
```

Run the repository setup script using Git Bash/WSL or another POSIX shell:

```sh
./scripts/vault-local-setup.sh
```

The script creates/uses the KV v2 `secret/` mount and writes placeholder Kafka username/password values. These placeholders do not authenticate to the plaintext local Kafka broker. Do not turn on SASL in ingestion config until Kafka is actually configured with matching credentials.

Vault dev mode is deliberately insecure and ephemeral. Its known root token must never be used outside isolated local development.

## 6. Start Kafka UI

If you already started `observa-kafka-ui` with `docker run`, keep using it; do not start the Compose UI profile because it uses the same container name and Compose Kafka service. Your manual UI configuration should point to `host.docker.internal:9092`, matching the manually started broker.

For a clean Compose-based setup (where Kafka is also managed by that Compose project), pull and start the optional profile:

```powershell
docker pull provectuslabs/kafka-ui:latest
docker compose -f deploy/local/kafka/docker-compose.yml --profile ui up -d kafka-ui
docker compose -f deploy/local/kafka/docker-compose.yml --profile ui ps
```

Open `http://127.0.0.1:8023`. In the Compose setup, UI connects to Kafka using `kafka:29092` on the shared Compose network. Do not change this to `localhost:9092` inside the UI container.

To stop only Kafka UI:

```powershell
docker compose -f deploy/local/kafka/docker-compose.yml --profile ui stop kafka-ui
```

## 7. Ingestion configuration and first event

The current `configs/ingestion.local.json` is intended for the Go process running on the host:

- Kafka broker: `127.0.0.1:9092`.
- Topic: `observa.events.v1`.
- Vault disabled for plaintext Kafka.
- Unauthenticated development mode enabled, fixed development tenant/source, and HTTP bound to loopback.

After installing the repository's required Go toolchain and resolving dependencies, run from the repository root:

```powershell
$env:OBSERVA_CONFIG_FILE = "configs/ingestion.local.json"
go run ./services/ingestion/cmd/ingestion
```

In a second terminal, submit the existing contract-valid example:

```powershell
curl.exe --fail-with-body -i -H "Content-Type: application/cloudevents+json" --data-binary "@examples/user-action.cloudevent.json" http://127.0.0.1:8080/v1/events
```

Expected response: `202 Accepted` only after Kafka confirms publication. This does not mean the event has been consumed or persisted to downstream stores.

## 8. Inspect the event in Kafka

Open Kafka UI and select the `observa.events.v1` topic, or use the CLI consumer:

```powershell
docker exec -it observa-kafka /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic observa.events.v1 --from-beginning
```

The CLI consumer will continue waiting for messages. Stop it with Ctrl+C. Verify that the published value is the original CloudEvents JSON and inspect the Kafka headers/key using Kafka UI or an appropriate client.

## 9. Useful everyday commands

```powershell
# Status
docker compose -f deploy/local/kafka/docker-compose.yml ps
docker compose -f deploy/local/vault/docker-compose.yml ps

# Follow logs
docker compose -f deploy/local/kafka/docker-compose.yml logs -f kafka
docker compose -f deploy/local/vault/docker-compose.yml logs -f vault

# Stop services but preserve Kafka's named volume
docker compose -f deploy/local/kafka/docker-compose.yml --profile ui down
docker compose -f deploy/local/vault/docker-compose.yml down

# WARNING: deletes Kafka's named volume and all locally stored Kafka records
docker compose -f deploy/local/kafka/docker-compose.yml down -v
```

## Troubleshooting

- **Port 9092 is already in use**: identify the existing broker/container before stopping it. Do not blindly remove a container that may hold data.
- **Kafka is running but not healthy**: inspect `logs kafka`; verify no old incompatible data directory was reused with a different Kafka version.
- **Kafka UI cannot connect**: verify Kafka is healthy and UI bootstrap server is `kafka:29092`, not localhost.
- **Ingestion cannot connect**: host configuration should use `127.0.0.1:9092`; check the broker is healthy and the topic exists.
- **Vault secret setup fails**: confirm Vault is healthy, the local token is set, and the setup script is run in a POSIX shell.
- **`202` not returned**: inspect ingestion logs and Kafka availability. Ingestion must not claim acceptance without broker delivery confirmation.

## Production gaps

These Compose files are strictly local development. Production requires a multi-broker Kafka deployment, suitable replication/min-ISR settings, authentication/TLS, a production Vault storage and authentication model, durable backups, access control, monitoring and operational runbooks.
