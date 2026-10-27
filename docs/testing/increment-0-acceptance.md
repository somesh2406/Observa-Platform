# Increment 0 — Product Contract and Engineering Baseline

## Acceptance criteria

- [x] A clean workspace exists separately from the read-only reference archive.
- [x] Product purpose, initial users, domain concepts, scope, and proposed guarantees are documented.
- [x] Unresolved workload, retention, recovery, and security decisions are explicit rather than invented.
- [x] Initial logical data flow and deferred architectural decisions are documented.
- [x] Repository/service-boundary decision is recorded with alternatives and consequences.
- [ ] Developer machine has the pinned supported Go toolchain installed and verified.
- [ ] Docker/Compose availability is verified if needed for local integration dependencies.
- [ ] Product owner confirms the initial use cases and target workload/retention assumptions.

## Evidence gathered in this environment

- The uploaded archive was listed and selected documentation/event-contract files were inspected without editing the archive.
- Available Go in this execution environment: `go1.23.2 linux/amd64`.
- Docker CLI is not installed in this execution environment.
- These environment checks do not establish the state of the user's laptop/WSL environment.

## Exit condition

Do not implement the ingestion service until the open product-contract decisions that affect its acceptance semantics, identity, tenant trust boundary, and payload policy are resolved or explicitly recorded as prototype assumptions.


## Product-model acceptance additions

- [x] The four primary product views are explicitly defined in the product contract.
- [x] Business application (for example, a loan application) is distinguished from source application (the software emitting events).
- [x] The contract states that all views derive from a shared event model.
- [x] Technical signals are distinguished from derived incidents and confirmed root-cause conclusions.
- [x] The contract prohibits fabricated user attribution, session boundaries, or business-state transitions.
- [ ] Canonical identifiers and event-time/ingestion-time semantics are agreed.
- [ ] Incident grouping, severity, deduplication, and lifecycle semantics are scoped.
- [ ] The first event schema is specified with required, optional, and sensitive fields.
- [ ] Acceptance behavior and security tests are defined for the first ingestion vertical slice.
