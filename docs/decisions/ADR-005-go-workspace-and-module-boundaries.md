# ADR-005: Go workspace and module boundaries

- **Status:** Accepted for the current implementation increment
- **Date:** 2026-10-21

## Context

Observa needs to preserve the existing CloudEvents 1.0 contract and initial ingestion implementation while preparing for independently deployable backend services. A single root Go module made the first increment simple, but did not establish explicit module boundaries for shared contracts and services.

## Decision

Use one Git monorepo and a root `go.work` workspace. Give each independently deployable Go service its own module. Establish `libs/eventcontract` as a shared module containing the Go validator and tests. Keep the existing JSON Schema files in root `schemas/` as the canonical schema artifacts, and keep the shared contract acceptance corpus in `tests/fixtures/`. The ingestion service imports the contract module and does not import another service's private packages.

Kafka remains the runtime boundary between ingestion and the future processing service. The workspace is a development/build arrangement; it does not merge service processes or imply synchronous in-process calls.

## Alternatives considered

1. One Go module for every service: rejected for now because it weakens module ownership and dependency boundaries as services grow.
2. Separate Git repositories: deferred until team ownership, access control, or independent repository governance justifies the coordination overhead.
3. A service for every package: rejected because packages without independent runtime/operational responsibilities should remain libraries.

## Consequences

- `go.work` includes `libs/eventcontract` and `services/ingestion`.
- Each module has its own `go.mod` and toolchain target.
- The root Makefile runs tests/vet across all workspace modules.
- CI must eventually test each module both in the workspace and with workspace overrides disabled, with the shared module version/replacement policy made explicit.
- Local module replacement is retained for in-repository builds; it is not a published dependency release strategy.
- The original and prior implementation archives are retained under `migration-preservation/source-archives/` with SHA-256 checksums.
