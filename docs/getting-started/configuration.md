# Configuration

The ingestion service loads a JSON file through Viper. The path comes from `--config`, then
`OBSERVA_CONFIG_FILE`; if neither is set, it searches for `configs/ingestion.local.json`.
Environment variables use the `OBSERVA_` prefix and replace dots with underscores
(`services/ingestion/cmd/ingestion/main.go:11-18`;
`services/ingestion/internal/config/config.go:77-109`).

The defaults below are set in `setDefaults`; where no default is set, the value must come from
the configuration file or environment and may be required by validation or startup
(`services/ingestion/internal/config/config.go:112-138`).

## Fields

| `mapstructure` field | Go type | Default | Environment variable | Validation / behavior |
|---|---|---|---|---|
| `service` | `string` | `Observa Ingestion Service` | `OBSERVA_SERVICE` | No additional validation. |
| `environment` | `string` | none | `OBSERVA_ENVIRONMENT` | Required and nonblank. |
| `http.address` | `string` | `127.0.0.1:8080` | `OBSERVA_HTTP_ADDRESS` | In unauthenticated dev mode, must parse as host:port and host must be literal `127.0.0.1` or `::1`. |
| `http.read_header_timeout` | `time.Duration` | `5s` | `OBSERVA_HTTP_READ_HEADER_TIMEOUT` | Must be positive. |
| `http.read_timeout` | `time.Duration` | `15s` | `OBSERVA_HTTP_READ_TIMEOUT` | Must be positive. |
| `http.write_timeout` | `time.Duration` | `15s` | `OBSERVA_HTTP_WRITE_TIMEOUT` | Must be positive. |
| `http.idle_timeout` | `time.Duration` | `60s` | `OBSERVA_HTTP_IDLE_TIMEOUT` | Must be positive. |
| `http.shutdown_timeout` | `time.Duration` | `15s` | `OBSERVA_HTTP_SHUTDOWN_TIMEOUT` | Must be positive. |
| `http.max_body_bytes` | `int64` | `1048576` | `OBSERVA_HTTP_MAX_BODY_BYTES` | Range 1 byte through 16 MiB. |
| `http.publish_timeout` | `time.Duration` | `5s` | `OBSERVA_HTTP_PUBLISH_TIMEOUT` | Must be positive. |
| `development.allow_unauthenticated` | `bool` | `false` (zero value) | `OBSERVA_DEVELOPMENT_ALLOW_UNAUTHENTICATED` | If true, `environment` must equal `development`, tenant/source IDs must be nonblank, and bind address must use literal loopback IP. |
| `development.tenant_id` | `string` | empty | `OBSERVA_DEVELOPMENT_TENANT_ID` | Required when unauthenticated mode is enabled. |
| `development.source_id` | `string` | empty | `OBSERVA_DEVELOPMENT_SOURCE_ID` | Required when unauthenticated mode is enabled. |
| `kafka.brokers` | `[]string` | empty | `OBSERVA_KAFKA_BROKERS` | At least one host:port address; no blank entries. |
| `kafka.topic` | `string` | empty | `OBSERVA_KAFKA_TOPIC` | Required and nonblank. |
| `kafka.delivery_timeout` | `time.Duration` | `5s` | `OBSERVA_KAFKA_DELIVERY_TIMEOUT` | Must be positive. |
| `kafka.sasl_mechanism` | `string` | empty | `OBSERVA_KAFKA_SASL_MECHANISM` | Empty, `PLAIN`, `SCRAM-SHA-256`, or `SCRAM-SHA-512`; any configured SASL requires Vault enabled. `PLAIN` requires TLS. |
| `kafka.tls` | `bool` | `false` (zero value) | `OBSERVA_KAFKA_TLS` | `PLAIN` requires true. |
| `kafka.username_secret_key` | `string` | `kafka_username` | `OBSERVA_KAFKA_USERNAME_SECRET_KEY` | Nonblank when Vault is enabled. |
| `kafka.password_secret_key` | `string` | `kafka_password` | `OBSERVA_KAFKA_PASSWORD_SECRET_KEY` | Nonblank when Vault is enabled. |
| `vault.enabled` | `bool` | `false` (zero value) | `OBSERVA_VAULT_ENABLED` | Enables Vault credential retrieval. |
| `vault.address` | `string` | empty | `OBSERVA_VAULT_ADDRESS` | Required and nonblank when Vault is enabled. |
| `vault.kv_v2_path` | `string` | empty | `OBSERVA_VAULT_KV_V2_PATH` | Required when enabled; must be relative to the KV v2 mount and must not start `secret/` or contain `/data/`. |
| `vault.token_env` | `string` | `VAULT_TOKEN` | `OBSERVA_VAULT_TOKEN_ENV` | Names the environment variable holding the Vault token. |
| `vault.request_timeout` | `time.Duration` | `3s` | `OBSERVA_VAULT_REQUEST_TIMEOUT` | Must be positive. |
| `vault.skip_tls_verify` | `bool` | `false` (zero value) | `OBSERVA_VAULT_SKIP_TLS_VERIFY` | Permitted only when `environment=development`. |
| `tenant_rate_limit.requests` | `int` | `100` | `OBSERVA_TENANT_RATE_LIMIT_REQUESTS` | Must be positive. |
| `tenant_rate_limit.window` | `time.Duration` | `1m` | `OBSERVA_TENANT_RATE_LIMIT_WINDOW` | Must be positive and at least 1 ms. |
| `tenant_rate_limit.redis_address` | `string` | `127.0.0.1:6380` | `OBSERVA_TENANT_RATE_LIMIT_REDIS_ADDRESS` | Required and nonblank. |
| `tenant_rate_limit.redis_username_env` | `string` | `OBSERVA_TENANT_RATE_LIMIT_REDIS_USERNAME` | `OBSERVA_TENANT_RATE_LIMIT_REDIS_USERNAME_ENV` | Names the environment variable containing Redis username. |
| `tenant_rate_limit.redis_password_env` | `string` | `OBSERVA_TENANT_RATE_LIMIT_REDIS_PASSWORD` | `OBSERVA_TENANT_RATE_LIMIT_REDIS_PASSWORD_ENV` | Names the environment variable containing Redis password. |
| `tenant_rate_limit.redis_db` | `int` | `0` | `OBSERVA_TENANT_RATE_LIMIT_REDIS_DB` | Must not be negative. |
| `tenant_rate_limit.redis_tls` | `bool` | `false` (zero value) | `OBSERVA_TENANT_RATE_LIMIT_REDIS_TLS` | Enables TLS with minimum TLS 1.2. |
| `tenant_rate_limit.dial_timeout` | `time.Duration` | `2s` | `OBSERVA_TENANT_RATE_LIMIT_DIAL_TIMEOUT` | Must be positive. |
| `logger.level` | `string` | `info` | `OBSERVA_LOGGER_LEVEL` | Case-insensitive `debug`, `info`, `warn`, or `error`. |
| `logger.format` | `string` | `json` | `OBSERVA_LOGGER_FORMAT` | Case-insensitive `json` or `text`. |
| `logger.add_source` | `bool` | `false` (zero value) | `OBSERVA_LOGGER_ADD_SOURCE` | No additional validation. |

The Go struct definitions and mapstructure tags are in
`services/ingestion/internal/config/config.go:13-75`. Default values and validation are in
`services/ingestion/internal/config/config.go:112-138` and `140-238`.

## Runtime-specific environment

The Compose file sets `OBSERVA_CONFIG_FILE`, `OBSERVA_ENVIRONMENT=production`,
`OBSERVA_HTTP_ADDRESS=0.0.0.0:8080`, `OBSERVA_DEVELOPMENT_ALLOW_UNAUTHENTICATED=false`,
and Kafka/tenant Redis connection overrides
(`deploy/local/docker-compose.yml:146-179`). The local JSON file's development mode is therefore
overridden in the integrated Compose stack.

The edge gateway uses separate variables that are read by Compose or Envoy templates rather
than by the ingestion `Config` struct: `OBSERVA_JWT_ISSUER`, `OBSERVA_JWT_AUDIENCE`,
`OBSERVA_JWKS_URL`, `OBSERVA_JWKS_HOST`, `OBSERVA_JWKS_PORT`,
`OBSERVA_EDGE_RATE_LIMIT_REQUESTS`, `OBSERVA_EDGE_RATE_LIMIT_UNIT`,
`OBSERVA_GATEWAY_HOST`, `OBSERVA_GATEWAY_TLS_CERT_FILE`, and
`OBSERVA_GATEWAY_TLS_KEY_FILE`. See `.env.example` and `deploy/local/docker-compose.yml`.
