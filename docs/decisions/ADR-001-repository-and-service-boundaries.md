# ADR-001: Repository and service boundaries

- **Status:** Superseded in part by ADR-005 for Go module/workspace layout; runtime boundary principles remain accepted.
- **Original date:** 2026-10-16
- **Updated:** 2026-10-21

## Context

Observa is an early-stage distributed backend with a shared event contract and infrastructure integrations. The initial single-module implementation was useful for the first increment, but the project now needs explicit Go module boundaries while preserving all prior implementation artifacts.

## Decision

Use a Git monorepo and Go workspace. Independently deployable services own separate Go modules; shared, stable cross-service contracts live in explicit libraries. No service imports another service's private/internal packages. Runtime communication uses versioned contracts over Kafka or explicitly defined APIs.

See [ADR-005: Go workspace and module boundaries](ADR-005-go-workspace-and-module-boundaries.md) for the module layout and migration details.

## Alternatives considered

1. Keep all implementation in one Go module: simplest initially, but weaker module boundaries as services grow.
2. Separate repositories: deferred until independent access control, release governance, or team ownership warrants it.
3. Split every package into a runtime service: rejected because runtime boundaries should be based on scaling, fault isolation, security, or ownership rather than folder count.

## Consequences

- The workspace contains the shared event contract and ingestion service modules.
- The prior implementation archive is retained byte-for-byte for traceability; active source is migrated into the new module layout.
- The number of runtime services remains driven by actual operational requirements.

## Implementation notes

The current Go workspace includes `libs/eventcontract` and `services/ingestion` (`go.work:5-8`).
The ingestion module contains separate `cmd/ingestion` and `cmd/kafka-topic-provisioner`
entrypoints (`services/ingestion/cmd/ingestion/main.go:11-29`;
`services/ingestion/cmd/kafka-topic-provisioner/main.go:18-23`).

