# Ingestion Local-Development Runbook

The current local workflow is documented in the [Local Setup guide](../getting-started/local-setup.md)
and [Local Stack Runbook](../operations/runbook-local.md).

Important implementation correction: the ingestion bootstrap always constructs a Redis-backed
tenant limiter (`services/ingestion/internal/platform/bootstrap/build.go:81-106`). The sample
configuration uses tenant Redis at `127.0.0.1:6380`; no in-memory fallback is implemented.
