# ADR-004: Vault-backed runtime secrets

- **Status:** Accepted
- **Decision scope:** Runtime secret retrieval for the ingestion service

## Context

Ingestion needs ordinary runtime settings and may need Kafka SASL credentials. Committing credentials in JSON configuration or environment files risks disclosure. Putting every non-secret setting in Vault would also make startup depend on Vault unnecessarily.

## Decision

- Viper loads the service configuration from a JSON file and permits explicit `OBSERVA_*` environment overrides. Unknown keys and invalid effective values fail startup.
- Non-secret settings (listener address, Kafka brokers/topic, timeouts, log settings, and feature flags) remain in the configuration file or deployment environment.
- When Kafka SASL is configured, the service retrieves the username and password from Vault KV v2 at startup. The configured path is relative to the `secret` mount, e.g. `observa/ingestion`; the client reads `secret/data/observa/ingestion` through the KV v2 API.
- The Vault token is supplied through an environment variable and is not stored in the configuration file.
- Vault failure is fatal when Vault is enabled. The service does not silently fall back to credentials in source-controlled configuration.
- The local Docker Compose configuration uses Vault development mode only, bound to loopback with an explicitly unsafe development root token.
- `vault.skip_tls_verify` is permitted only in development. Production must use verified TLS.

## Consequences

- Local plaintext Kafka does not require Vault for ingestion to work. The local Vault setup can be tested separately; placeholder credentials do not authenticate to Kafka.
- Secret rotation currently requires a service restart; dynamic reload is not implemented.
- Vault dev mode is in-memory and loses secrets on restart.
- The local root token and placeholder values must never be reused outside local development.
- Production Vault authentication, policy scope, audit configuration, TLS trust, secret rotation, and high availability must be configured for the deployment environment.
