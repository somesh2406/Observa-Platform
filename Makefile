SHELL := /bin/sh
GO_PACKAGES := ./libs/eventcontract/... ./services/ingestion/...
COMPOSE := docker compose -f deploy/local/docker-compose.yml

.PHONY: fmt test test-race vet schema-check check run-ingestion vault-local-up vault-local-down vault-local-setup test-kafka-integration local-up local-down local-logs test-integration

fmt:
	gofmt -w $$(find libs services -name '*.go' -type f)

test:
	go test $(GO_PACKAGES)

test-race:
	go test -race $(GO_PACKAGES)

vet:
	go vet $(GO_PACKAGES)

schema-check:
	python scripts/validate_event_schemas.py

check: test vet schema-check

run-ingestion:
	OBSERVA_CONFIG_FILE=configs/ingestion.local.json go run ./services/ingestion/cmd/ingestion

vault-local-up:
	docker compose -f deploy/local/vault/docker-compose.yml up -d

vault-local-down:
	docker compose -f deploy/local/vault/docker-compose.yml down

vault-local-setup:
	./scripts/vault-local-setup.sh

test-kafka-integration:
	@test -n "$$OBSERVA_KAFKA_INTEGRATION_BROKERS" || (echo "ERROR: set OBSERVA_KAFKA_INTEGRATION_BROKERS" >&2; exit 2)
	@test -n "$$OBSERVA_KAFKA_INTEGRATION_TOPIC" || (echo "ERROR: set OBSERVA_KAFKA_INTEGRATION_TOPIC" >&2; exit 2)
	go test ./services/ingestion/internal/publisher -run TestKafkaPublishIntegration -count=1 -v

# The integrated local stack exposes ingestion only through Envoy on loopback:8443.
local-up:
	$(COMPOSE) up -d --build

local-down:
	$(COMPOSE) down --remove-orphans

local-logs:
	$(COMPOSE) logs -f --tail=100

test-integration: local-up
	./scripts/test-integration.sh
