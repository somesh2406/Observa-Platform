# Observa Schema Identifiers and Hosting Record

The checked-in schema `$id` values and the runtime `dataschema` constant use the
`https://somesh2406.github.io/observa-event-contracts/schemas/` URL family. This repository
contains local schema files under `schemas/`; the code does not fetch schemas from the network
during event validation.

- `schemas/producer-event-v1.schema.json`
- `schemas/cloudevent-v1.schema.json`
- `schemas/event-data-v1.schema.json`

The live availability of the public URLs is not established by the source tree. The runtime
validator compares `dataschema` to the literal URL constant in
`libs/eventcontract/event.go:18-26`; it does not retrieve the remote schema.

Run `make schema-check` to validate the checked-in schemas and fixture corpus locally.
