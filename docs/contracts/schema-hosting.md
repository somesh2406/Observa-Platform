# Observa public schema hosting

The public schema endpoint is hosted using GitHub Pages in `somesh2406/observa-event-contracts`.

## Published URLs

- Producer event (canonical entry point): https://somesh2406.github.io/observa-event-contracts/schemas/producer-event-v1.schema.json
- CloudEvents profile: https://somesh2406.github.io/observa-event-contracts/schemas/cloudevent-v1.schema.json
- Observa payload: https://somesh2406.github.io/observa-event-contracts/schemas/event-data-v1.schema.json

The producer schema references the CloudEvents profile; the profile references the Observa payload schema. `dataschema` identifies the payload schema, not the composed producer schema. The `$id` values are canonical public schema identifiers.

## Release rules

1. Keep published v1 schema URLs immutable after external producers adopt them.
2. For breaking changes, publish a new versioned set of schema files and update producer SDKs deliberately.
3. Do not publish credentials, real customer data, or sensitive examples in this repository.
4. After each Pages deployment, open all three URLs and verify they return the expected JSON documents rather than a 404 or an HTML page.
5. Run `make check` locally and in CI. The automated schema validator resolves references from local checked-out files, so local checks do not depend on network access.

## Publication verification status

The URLs above were supplied for this workspace update. The current execution environment could not independently fetch GitHub Pages (network DNS access was unavailable), so live HTTP availability has not been confirmed here. Verify the deployed endpoints in a browser before treating them as publicly reachable.
