# Engineering Baseline

**Status:** Go 1.27.3 target is encoded in go.mod; local verification is blocked until that toolchain is available.

## 1. Repository rules

- The uploaded `observa-backend-workspace.zip` is a read-only reference; never edit it as part of the clean implementation.
- This directory is the new source of truth for code and current documentation.
- Use a monorepo initially to coordinate versioned contracts and integration tests. Do not create a runtime service solely to mirror a diagram.
- A service boundary is justified by independent scaling, fault isolation, security boundary, or ownership—not by a desire to maximize microservice count.
- No service imports another service's private/internal packages. Cross-service communication uses versioned contracts.
- Every architecture decision with meaningful cost or risk gets one ADR. Do not create ADRs for trivial implementation details.

## 2. Toolchain

The current execution environment has Go 1.23.2, while the project baseline requires Go 1.27.3 (or a newer supported patch release). The module and toolchain directives are aligned to Go 1.27.3. Do not downgrade the project to fit an old execution environment. The target toolchain could not be installed or verified here because external network/DNS access is unavailable. Source: https://go.dev/doc/devel/release

Before the implementation can be considered verified:

- Install and verify Go 1.27.3 (or a newer supported patch release when the team starts).
- Pin the Go language/toolchain requirement consistently in module/workspace files and CI.
- Record `go version`, OS/architecture, and dependency download configuration in the baseline check.
- Use `go test`, `go vet`, formatting checks, race tests for concurrency-sensitive packages, and integration tests against real dependencies where semantics matter.
- Docker is not installed in the inspected execution environment. Verify Docker/Compose availability on the developer machine; do not make integration tests silently pass by skipping required dependencies.
- External Go module downloads are unavailable in this environment; the selected franz-go dependency has not been fetched, compiled, or security-reviewed here.

## 3. Minimum repository checks

The first CI pipeline should eventually enforce:

1. Formatting (`gofmt` and any agreed formatter).
2. Unit tests.
3. Static checks (`go vet` and selected linters with documented configuration).
4. Dependency vulnerability review and lock/sum-file integrity.
5. Build of each deployable binary.
6. Integration tests for infrastructure boundaries.
7. Container image build and vulnerability scanning before deployment is introduced.

CI should report each check independently and fail on a required check's failure. No production secret belongs in the repository or CI logs.

## 4. Design and implementation rules

- Every increment starts with a short problem statement, acceptance criteria, and relevant design only.
- Prefer standard-library capabilities until a dependency provides clear value.
- Use `context.Context` for cancellable I/O, explicit timeouts, bounded request/body sizes, and graceful shutdown.
- Validate input at trust boundaries; keep domain logic out of transport handlers.
- Define error categories and retryability; never retry indefinitely without a budget or backoff policy.
- Avoid claims of exactly-once delivery across Kafka and external stores unless the complete end-to-end design and tests prove the relevant semantics.
- Include observability fields and metrics when adding a new asynchronous boundary.
- Tests must cover expected behavior and at least the most important failure/timeout path for the increment.
- Do not introduce abstractions, frameworks, or deployment complexity without a concrete requirement.

## 5. Security baseline

- Threat-model each external or cross-trust boundary.
- Separate producer authentication from query-user authorization where appropriate.
- Authorize tenant access server-side; never trust tenant IDs supplied solely in payloads or query parameters.
- Keep secrets out of source control, images, logs, and error responses.
- Set payload, concurrency, timeout, retry, and query limits.
- Avoid logging event bodies by default; redact sensitive identifiers and tokens.
- Define TLS and at-rest encryption for each production boundary, plus key rotation and access audit requirements.
- Include negative tests for unauthorized access and cross-tenant access before exposing query APIs.

## 6. Operational baseline

Each service added must define:

- Readiness and liveness behavior with distinct purposes.
- Structured logs with correlation/trace identifiers but no sensitive payload by default.
- Metrics for request rate, errors, latency, saturation, queue/consumer lag, retries, and dropped/rejected work as applicable.
- Graceful shutdown and in-flight work semantics.
- Resource requests/limits and configuration validation before serving traffic.
- A runbook for dependency outage, backlog growth, and recovery.

## 7. Increment completion rule

An increment is complete only when:

- Its acceptance criteria are automated where practical.
- Unit tests and relevant integration/failure tests pass in a reproducible environment.
- Security and operational implications have been reviewed.
- The implementation diff is small enough to review and has no unrelated refactoring.
- Documentation and ADRs are updated only if the implementation changed a contract or decision.
- Known limitations are recorded, with no claim that untested behavior is production-ready.
