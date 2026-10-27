# Go Workspace Migration Record

## Goal

Establish a multi-module Go workspace without changing the canonical CloudEvents 1.0 profile or losing the previous repository artifacts.

## Active layout

- `go.work`: workspace declaration, targeting Go 1.27.3.
- `libs/eventcontract`: shared Go event decoder/validator and its tests.
- `services/ingestion`: independently buildable ingestion module, HTTP handler, configuration, Kafka publisher, and Vault adapter.
- `schemas/`: canonical JSON Schema files retained at their original paths.
- `tests/fixtures/`: canonical shared contract corpus retained at its original path.
- `docs/`, `examples/`, `scripts/`, `configs/`, and `deploy/`: retained and extended.
- `migration-preservation/source-archives/`: original baseline and prior implementation ZIP archives plus SHA-256 manifest.

## Migration mapping

| Previous path                          | Active path                                | Treatment                                                                                        |
| -------------------------------------- | ------------------------------------------ | ------------------------------------------------------------------------------------------------ |
| `internal/eventcontract/event.go`      | `libs/eventcontract/event.go`              | Copied into shared contract module; imports remain transport-independent.                        |
| `internal/eventcontract/event_test.go` | `libs/eventcontract/event_test.go`         | Copied; reads canonical corpus from `tests/fixtures/`.                                           |
| `cmd/ingestion/main.go`                | `services/ingestion/cmd/ingestion/main.go` | Copied; module import paths updated.                                                             |
| `internal/config/`                     | `services/ingestion/internal/config/`      | Copied as service-owned configuration.                                                           |
| `internal/ingestion/`                  | `services/ingestion/internal/ingestion/`   | Copied; imports shared contract module.                                                          |
| `internal/publisher/`                  | `services/ingestion/internal/publisher/`   | Copied as ingestion-owned Kafka adapter.                                                         |
| `internal/secrets/`                    | `services/ingestion/internal/secrets/`     | Copied as ingestion-owned Vault adapter.                                                         |
| root `go.mod`                          | module-specific `go.mod` files + `go.work` | Replaced as the active build model; original file preserved in the prior implementation archive. |

## Preservation and authority

The two source archives in `migration-preservation/source-archives/` are byte-for-byte copies of the uploaded original repository and the prior ingestion implementation. Their SHA-256 hashes are recorded in `SHA256SUMS.json`. Active code is maintained only in the new module locations; the archived ZIPs are historical snapshots, not build inputs.

The schema files and shared fixture remain at their existing root paths to avoid silently changing contract artifact locations. Do not create a second maintained schema copy under `libs/eventcontract`.

## Verification limits

The recorded baseline requires Go 1.27.3, while the generation environment has Go 1.23.2. The target modules could not be compiled with the required toolchain here. External Go module downloads and Docker were unavailable, so Kafka/Vault integration has not been verified. Run the full acceptance checks on a machine with the required toolchain and dependencies before treating the workspace as production-ready.
