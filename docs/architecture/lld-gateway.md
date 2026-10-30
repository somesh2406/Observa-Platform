# Envoy Gateway Low-Level Design

## Deployment forms

- `deploy/gateway/envoy.yaml` is the plain-Envoy configuration used by local Compose.
- `deploy/gateway/gatewayclass.yaml`, `gateway.yaml.tmpl`, `http-route.yaml.tmpl`,
  `health-route.yaml.tmpl`, `security-policy.yaml.tmpl`, and
  `backend-traffic-policy.yaml.tmpl` define Gateway API resources for an Envoy Gateway
  controller. The templates require environment substitution before application.
- `deploy/gateway/Dockerfile` builds the local Envoy image and renders `envoy.yaml` with
  `envsubst` at container startup.

## Gateway configuration layout

- `deploy/gateway/envoy.yaml` — complete plain-Envoy listener, routes, filters, and clusters.
- `deploy/gateway/gatewayclass.yaml` — Envoy Gateway controller class.
- `deploy/gateway/gateway.yaml.tmpl` — HTTPS Gateway listener and TLS Secret reference.
- `deploy/gateway/http-route.yaml.tmpl` — event POST route to the ingestion Service.
- `deploy/gateway/health-route.yaml.tmpl` — health path route.
- `deploy/gateway/security-policy.yaml.tmpl` — JWT provider and claim-to-header mapping.
- `deploy/gateway/backend-traffic-policy.yaml.tmpl` — shared global rate-limit policy.
- `deploy/gateway/Dockerfile` — local plain-Envoy image and template rendering entrypoint.

## Local plain-Envoy request path

`envoy.yaml` listens on `0.0.0.0:8443` and terminates TLS using mounted certificate/key files
(lines 5-17). It routes `POST /v1/events` to the `ingestion:8080` cluster and `/health/` to the
same backend; other paths receive `404` (lines 27-42).

The JWT filter protects `/v1/events`, validates issuer, audience, and remote JWKS, and maps
`tenant_id` and `source_id` claims to `x-observa-tenant` and `x-observa-source` headers
(lines 43-67). The configuration sets `forward: false`, so the bearer token is not forwarded
upstream. Health paths allow missing JWTs.

The external rate-limit filter uses domain `observa-ingestion`, gRPC cluster `ratelimit:8081`,
and `failure_mode_deny: true` (lines 68-77, 93-104). The rate-limit service reads its policy
from the rendered runtime volume and uses `redis-edge`
(`deploy/local/docker-compose.yml:70-94`).

## Gateway API resources

The `HTTPRoute` for `/v1/events` matches POST and targets service `observa-ingestion` port
`8080` (`http-route.yaml.tmpl:1-21`). The separate health route matches `/health/`
(`health-route.yaml.tmpl:1-18`). `SecurityPolicy` maps verified JWT claims to the same identity
headers (`security-policy.yaml.tmpl:10-22`). `BackendTrafficPolicy` configures a shared global
rate limit (`backend-traffic-policy.yaml.tmpl:10-16`).

## Configuration

Local Compose passes `OBSERVA_JWT_ISSUER`, `OBSERVA_JWT_AUDIENCE`, `OBSERVA_JWKS_URL`,
`OBSERVA_JWKS_HOST`, and `OBSERVA_JWKS_PORT` to Envoy
(`deploy/local/docker-compose.yml:181-191`). The edge rate limit is rendered from
`OBSERVA_EDGE_RATE_LIMIT_REQUESTS` and `OBSERVA_EDGE_RATE_LIMIT_UNIT`
(`deploy/local/docker-compose.yml:58-68`; `deploy/local/rate-limit/config.yaml.tmpl:1-8`).
TLS file paths are mounted from `OBSERVA_GATEWAY_TLS_CERT_FILE` and
`OBSERVA_GATEWAY_TLS_KEY_FILE` (`deploy/local/docker-compose.yml:192-198`).

## Error behavior

The plain-Envoy rate-limit filter denies when its rate-limit service fails because
`failure_mode_deny` is true (`deploy/gateway/envoy.yaml:68-77`). JWT validation failures are
handled at the gateway; the ingestion handler does not receive those requests. The ingestion
handler independently rejects missing trusted identity headers in authenticated mode.

## Testing evidence

The repository's integration smoke test checks that liveness and readiness are reachable through
Envoy and that a POST without a JWT returns `401` (`scripts/test-integration.sh:29-34`).
Gateway API template application is documented in `deploy/gateway/README.md`; no Kubernetes
cluster integration test is present in the repository.
