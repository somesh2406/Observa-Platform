# Observa Ingestion Service

This independently buildable Go module accepts CloudEvents 1.0 structured JSON, validates it through the shared `libs/eventcontract` module, and returns `202 Accepted` only after Kafka confirms publication. It does not consume Kafka or write downstream storage.

## Configuration

Viper loads `configs/ingestion.local.json` (or the path supplied by `--config` / `OBSERVA_CONFIG_FILE`). Supported `OBSERVA_*` environment variables override file values and may supply keys absent from the JSON file. Non-secret settings belong in the file or deployment environment; Vault supplies Kafka SASL credentials when SASL is enabled.

The Vault KV v2 path is relative to the `secret` mount, for example `observa/ingestion`. Do not include `secret/data/` in the configured path.

## Local run

From the repository root, with the required Go toolchain and a running local Kafka broker/topic:

```powershell
$env:OBSERVA_CONFIG_FILE = "configs/ingestion.local.json"
go run ./services/ingestion/cmd/ingestion
```

See [`docs/runbooks/ingestion-local-development.md`](../../docs/runbooks/ingestion-local-development.md). Unauthenticated mode is permitted only for loopback-bound local development. All non-development startup is blocked until producer authentication and trusted tenant/source resolution are implemented.
