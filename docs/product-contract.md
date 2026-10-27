# Observa Product Contract

**Status:** Draft v0.1 — assumptions must be validated before they become release commitments.

## 1. Product purpose

Observa is a user-action observability and application-traceability platform for banking and fintech applications. It collects events describing application activity, preserves evidence needed for investigation, processes events into a consistent representation, and makes the resulting history queryable.

## 2. Primary users and jobs

Initial user groups to validate:

- Application support / operations: investigate what happened during a customer journey.
- Engineering: diagnose failed or unexpected application behavior across services.
- Risk, audit, or compliance teams: retrieve appropriately authorized event evidence for a defined investigation.

The platform must not assume that every event is an audit record or that all captured payloads are safe to retain. Event classification, access rules, retention, and redaction need explicit policy.

## 3. Core product capabilities

All four views must be derived from a common, versioned event model and shared correlation semantics. They are different query and presentation models over related events, not four independent ingestion pipelines. Technical incidents require an explicit distinction between raw technical signals emitted by sources and incident records derived by detection/correlation rules.

### 3.1 User action timeline

**Investigation question:** What actions are associated with a particular user, and when did they occur?

The timeline presents authorized events associated with a user identity in chronological order across the applications and sources connected to Observa. The product must define identity resolution explicitly; it must not assume that a user identifier is globally unique across tenants or systems. Where events cannot be reliably attributed to a user, they must remain distinguishable as unattributed rather than being guessed onto a timeline.

### 3.2 Application journey

**Investigation question:** What happened to a particular business application as it progressed through its lifecycle?

The journey groups relevant events around a business entity such as a loan application and presents lifecycle stages, actions, outcomes, failures, and gaps where the available data supports them. A journey may span multiple user identities, application sources, and sessions. Observa records and reconstructs observed activity; it must not invent business state transitions that were never emitted or otherwise authoritatively supplied.

### 3.3 Session action trace

**Investigation question:** What actions and technical events occurred during one particular session?

The session trace presents events associated with a session in event-time order, with enough timing, source, outcome, and correlation context to investigate a sequence. API calls, errors, screen transitions, and other technical events are only available when instrumented and emitted. Session boundaries and session identifiers must be defined by the producing application or an explicit platform contract, not inferred from inactivity without a documented rule.

### 3.4 Technical incidents

**Investigation question:** What technical failures or abnormal conditions affected the application, a user session, or a business application, when did they occur, and what related activity may explain their impact?

The incident view surfaces technical signals such as unhandled exceptions, failed API calls, timeouts, dependency failures, application errors, and other explicitly instrumented failures. It should correlate these signals with user, session, business application, source, request, and trace identifiers when those relationships are available.

The product must distinguish: (a) a technical event emitted by a source, (b) a detected incident created by a defined rule or correlation process, and (c) a confirmed/root-cause conclusion made by an investigator. Observa must not label every error event as an incident, infer root cause without evidence, or imply that it detects failures that the source does not instrument. Incident grouping, severity, deduplication, lifecycle/status, and alerting rules are explicit design decisions to define when the incident workflow is scoped.

## 4. Core domain concepts

- **Tenant:** an organization or logically isolated customer boundary.
- **Source:** an instrumented application or producer.
- **User identity:** a tenant-scoped identity reference; mapping identities across systems requires an explicit trusted identity-resolution rule.
- **Business application:** a domain entity being processed, such as a loan application. It is distinct from the software application that emitted an event.
- **Session:** a bounded interaction context defined by a source or an agreed sessionization contract.
- **Event:** a uniquely identifiable observation emitted by a source, with event time and ingestion time kept distinct.
- **Technical signal:** an emitted observation of an error, exception, timeout, failed request, or other technical condition.
- **Incident:** a derived, correlated or grouped technical problem under explicit detection rules; it is not synonymous with a single error event.
- **Correlation identifiers:** identifiers linking events to a user, business application, session, request, or trace where applicable. These relationships are optional when not known and must not be fabricated.
- **Raw evidence:** the accepted original event representation, subject to integrity, privacy, retention, and access policies.
- **Normalized event:** a validated event transformed into the canonical query-oriented representation.
- **Query result:** data returned only to an authorized caller within the caller's permitted tenant and data scope.

## 5. Product flow

Instrumented application → authenticated ingestion → durable event acceptance → event processing and normalization → evidence and analytical storage → query API → authorized investigation UI or client.

This is a logical flow, not a commitment to one service per box. Runtime boundaries must be justified by independent scaling, failure isolation, security, or ownership requirements.

## 6. Proposed product guarantees

These are target guarantees to turn into measurable requirements, not claims about the current or future implementation until tested.

1. **Explicit acceptance:** an API success response means the event has reached the durability boundary specified by the ingestion contract.
2. **Traceability:** each event has a stable identity and enough metadata to trace its processing state across system boundaries.
3. **Recoverability:** failures after partial progress can be retried or reconciled without silently losing logical events.
4. **Duplicate tolerance:** duplicate delivery is expected in a distributed system; the platform defines how duplicates are detected and how query semantics avoid misleading results.
5. **Tenant isolation:** a caller cannot ingest as, read from, or administer another tenant without explicit authorization.
6. **Data protection:** sensitive fields are minimized, classified, protected in transit and at rest, and exposed only under policy.
7. **Operational visibility:** ingestion rate, rejection rate, processing lag, retries, dead-letter volume, destination failures, and query latency can be measured.
8. **Bounded resource use:** payload size, concurrency, queueing, retries, and query cost have explicit limits.

## 7. Initial scope

### In scope

- Event ingestion and validation.
- Versioned event contract.
- Durable event transport.
- Processing, normalization, and replay/recovery behavior.
- Original-evidence handling and analytical storage decisions.
- Query API and tenant-scoped access.
- Security, observability, deployment, testing, and operational readiness.

### Not yet in scope

- A full visual journey-builder or workflow engine.
- Automated business decisions or loan eligibility decisions.
- A promise of regulatory certification or compliance without a mapped control framework and evidence.
- Multi-region active-active operation before availability, recovery, residency, and cost requirements justify it.
- Choosing Iceberg and ClickHouse merely because both exist in the reference implementation.

## 8. Quality attributes to quantify

The following values are deliberately open; they must be supplied from product needs or workload evidence rather than invented.

| Attribute | Decision needed |
|---|---|
| Ingestion throughput | Average and peak events/second, burst duration |
| Payload | Typical, p95, maximum size; permitted content types |
| Acceptance latency | p95/p99 target and precise acknowledgement meaning |
| Data freshness | Maximum delay from accepted event to query visibility |
| Availability | Service SLO and maintenance expectations |
| Retention | Raw evidence and normalized event retention by data class |
| Recovery | RPO/RTO for each durable boundary and destination |
| Query workload | Main filters, joins/groupings, concurrency, result-size limits |
| Tenant scale | Expected tenant count and distribution/skew |
| Data residency | Regions and legal/contractual constraints |
| Cost | Storage and compute budget per event volume or tenant |

## 9. Security and privacy constraints

- Tenant identity must come from a trusted authentication/authorization boundary, not from an untrusted event field alone.
- Do not log raw event payloads, credentials, tokens, or sensitive customer data by default.
- Apply data minimization and field-level classification before defining retention and query access.
- Define authentication for producer clients separately from human/API consumer access where their trust models differ.
- Establish encryption, key/secret management, audit logging, deletion/retention behavior, and incident response requirements before production.
- Threat-model spoofed producers, cross-tenant access, replay/abuse, oversized payloads, injection, credential compromise, and storage/query exfiltration.

## 10. Open decisions (must not be silently assumed)

1. What are the canonical definitions and identifiers for a user, business application, session, source, and event?
2. How are user identities resolved across sources while preserving tenant isolation?
3. Which events and authoritative business-state changes are available to construct a journey?
4. What exact user-action event schema and producer SDKs are required first?
5. Is the product intended to retain all raw payloads, or only selected/redacted evidence?
6. What does a successful ingestion response promise: Kafka acknowledgement, durable evidence write, or another explicit boundary?
7. What are the target throughput, freshness, availability, retention, RPO, and RTO values?
8. Which query patterns must be fast in the first product release?
9. Are events ordered per source, session, journey, or not at all?
10. Which authentication/identity provider and tenant model are available?
11. Which regulatory, residency, and customer contractual constraints apply?

## 11. Definition of product-contract acceptance

The contract is ready to drive architecture when:

- Primary users and all four investigation views—user timeline, application journey, session trace, and technical incidents—are named.
- The difference between emitted technical signals, derived incidents, and investigator-confirmed root causes is explicit.
- Event identity, timestamp, tenant, source, and correlation semantics are defined.
- The distinction between a business application and a software application is explicit.
- The system does not fabricate user attribution, session boundaries, or business-state transitions.
- Ingestion acknowledgement and durability semantics are unambiguous.
- Initial throughput, freshness, retention, availability, and recovery targets are agreed or explicitly marked as prototype-only assumptions.
- Sensitive-data policy and trust boundaries are defined.
- Each guarantee has a proposed verification method.
