#!/usr/bin/env sh
set -eu

# Local-only helper. Vault dev mode has no durable storage and a known root token.
export VAULT_ADDR="${VAULT_ADDR:-http://127.0.0.1:8200}"
export VAULT_TOKEN="${VAULT_TOKEN:-local-dev-only-root-token}"

command -v docker >/dev/null 2>&1 || { echo "docker is required" >&2; exit 1; }
# Wait for Vault to become ready.
i=0
until docker exec observa-vault-dev vault status >/dev/null 2>&1; do
  i=$((i + 1))
  if [ "$i" -ge 30 ]; then echo "Vault did not become ready" >&2; exit 1; fi
  sleep 1
done
# The dev server normally enables the KV v2 secret/ mount. Enable only if absent.
if ! docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN="$VAULT_TOKEN" observa-vault-dev vault secrets list -format=json | grep -q '"secret/"'; then
  docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN="$VAULT_TOKEN" observa-vault-dev vault secrets enable -path=secret kv-v2
fi
# Placeholder values are for demonstrating retrieval only. Do not enable SASL unless these
# are replaced with credentials that actually exist on the local Kafka broker.
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN="$VAULT_TOKEN" observa-vault-dev vault kv put secret/observa/ingestion kafka_username='local-placeholder' kafka_password='local-placeholder'
echo "Local Vault KV v2 secret written to secret/observa/ingestion."
echo "Export VAULT_TOKEN=local-dev-only-root-token only in this local shell."
