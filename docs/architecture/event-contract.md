# Observa Event Contract v1

## Profile

The shared Go validator implements a strict CloudEvents 1.0 structured JSON profile
(`libs/eventcontract/event.go:1-3`, `18-31`). The HTTP endpoint requires the media type
`application/cloudevents+json` (`services/ingestion/internal/ingestion/handler.go:21`,
`69-74`). The event's `datacontenttype` is `application/json`.

## Envelope

Required fields are `specversion`, `id`, `source`, `type`, `time`, `datacontenttype`,
`dataschema`, and `data`. Optional `subject` is omitted rather than set to null.
Unknown fields are rejected. One JSON value is allowed per request
(`libs/eventcontract/event.go:73-95`, `97-145`).

| Field | Accepted value / constraint |
|---|---|
| `specversion` | Exactly `"1.0"`. |
| `id` | 1–128 Unicode characters; non-whitespace content required. |
| `source` | 1–512 characters; URI-reference parseable; no whitespace, CR, LF, or tab. |
| `type` | `com.observa.user_action`, `com.observa.business_state`, or `com.observa.technical_signal`. |
| `time` | Parses as RFC3339Nano. |
| `subject` | Optional; if present, 1–256 characters. |
| `datacontenttype` | Exactly `"application/json"`. |
| `dataschema` | Exactly `https://somesh2406.github.io/observa-event-contracts/schemas/event-data-v1.schema.json`. |
| `data` | Required Observa event data object. |

The supported `type` values are category values. The validator does not implement a separate
registry of `event_name` namespaces.

## Event data

`data.event_name` is required, 1–128 characters, and must match
`^[a-z][a-z0-9]*(?:[._:-][a-z0-9]+)*$`. Examples include `loan.submit` and
`loan:decision:approved`. This is a syntax rule, not a registry or namespace authorization list
(`libs/eventcontract/event.go:28-31`, `196-199`).

Optional fields:

- `outcome`: one of `succeeded`, `failed`, `denied`, `cancelled`, `unknown`.
- `context.user_id`, `business_application_id`, `session_id`, `request_id`: 1–128 characters.
- `context.trace_id`: 32 lowercase hexadecimal characters and not all zero.
- `context.span_id`: 16 lowercase hexadecimal characters and not all zero.
- `attributes`: at most 64 properties; each key must be 1–64 characters and nonblank.
  Values are arbitrary JSON values decoded into Go `any`.
- `error`: at least one of `code`, `category`, or `message`; code 1–128 characters,
  category 1–64 characters, message at most 512 characters.
- `producer_version`: 1–64 characters.

Unknown nested fields are rejected. Optional typed fields explicitly set to JSON `null` are
rejected; omission is allowed (`libs/eventcontract/event.go:97-145`,
`196-237`, `240-257`).

## Validation boundary

The HTTP handler first checks media type and body size, then calls `eventcontract.Decode`.
Contract failures are returned by the HTTP layer as `400 invalid_event` with the generic message
`Event does not conform to the Observa event contract`
(`services/ingestion/internal/ingestion/handler.go:69-100`). The detailed validator error is
available to direct callers of `eventcontract.Decode`, not exposed in the HTTP response.

The code does not enforce source authorization against a registered producer, privacy allowlists,
JSON nesting depth, an attribute encoded-byte limit, clock-skew limits, or W3C `traceparent`
propagation. Do not infer these policies from the JSON Schemas' descriptive text.

## Valid example

```json
{
  "specversion": "1.0",
  "id": "evt-001",
  "source": "https://bank.example/mobile",
  "type": "com.observa.user_action",
  "time": "2026-10-30T10:00:00Z",
  "datacontenttype": "application/json",
  "dataschema": "https://somesh2406.github.io/observa-event-contracts/schemas/event-data-v1.schema.json",
  "data": {
    "event_name": "loan.submit",
    "outcome": "succeeded",
    "context": {"session_id": "session-123"}
  }
}
```

## Invalid example and exact validator error

```json
{
  "specversion": "0.3",
  "id": "evt-001",
  "source": "https://bank.example/mobile",
  "type": "com.observa.user_action",
  "time": "2026-10-30T10:00:00Z",
  "datacontenttype": "application/json",
  "dataschema": "https://somesh2406.github.io/observa-event-contracts/schemas/event-data-v1.schema.json",
  "data": {"event_name": "loan.submit"}
}
```

`eventcontract.Decode` rejects this with the exact error:

```text
specversion must be "1.0"
```

The HTTP endpoint returns `400` with code `invalid_event` and message
`Event does not conform to the Observa event contract`.

## Delivery semantics

Ingestion returns `202` only after Kafka delivery confirmation. The producer uses all-ISR
acknowledgements and retries; a timeout can leave the broker outcome ambiguous. The implemented
boundary is at-least-once, not exactly-once. Downstream consumers must deduplicate using the
CloudEvents `id`; no consumer or deduplication store is present
(`services/ingestion/internal/publisher/publisher.go:40-50`;
`services/ingestion/internal/ingestion/handler.go:154-159`).
