# Envoy edge gateway

Envoy is the single public HTTP ingress for ingestion in production and local development. The production resources use Envoy Gateway's Gateway API controller; `envoy.yaml` is the plain-Envoy fallback used by the local Compose stack when no Gateway API controller is installed.

## Edge policy

- **JWT authentication:** the `/v1/events` POST route requires a signed JWT with the configured issuer and audience. Envoy verifies it using the configured JWKS URL. `tenant_id` and `source_id` claims are overwritten into `X-Observa-Tenant` and `X-Observa-Source`; client-supplied copies are not trusted. The health route is separate and intentionally not JWT-protected.
- **Global/coarse rate limit:** the event route shares one edge-wide bucket (default 100 requests per second) across requests. The plain-Envoy fallback uses Envoy's external rate-limit filter with the local rate-limit service and Redis. The Gateway API `BackendTrafficPolicy` specifies a global limit; the deployed Envoy Gateway controller must have its global rate-limit service/store enabled and healthy.
- **TLS termination:** Gateway API terminates HTTPS using the `observa-gateway-tls` Secret. The plain-Envoy fallback terminates TLS on `8443` with the self-signed local certificate under `local/`.

## What remains in ingestion

Ingestion still validates the CloudEvent contract and body size, requires the gateway-forwarded tenant/source identity in authenticated mode, and enforces a separate per-tenant business quota. The tenant limiter can be `memory` for a single local process or `redis` for a shared production quota across ingestion replicas. The gateway's global abuse limit is not a substitute for tenant quotas. Keep the ingestion Service private and allow network access only from the gateway; headers alone do not cryptographically authenticate an upstream caller.

## Gateway API installation

Install a compatible Gateway API and Envoy Gateway controller first. Create the TLS Secret `observa-gateway-tls` in the namespace where the Gateway is deployed. Copy `.env.example` to `.env`, set the issuer, audience, JWKS URL/host/port, Gateway hostname, and edge request limit, then render and apply the templates:

```sh
set -a
. ./.env
set +a
kubectl apply -f deploy/gateway/gatewayclass.yaml
for template in \
  deploy/gateway/gateway.yaml.tmpl \
  deploy/gateway/http-route.yaml.tmpl \
  deploy/gateway/health-route.yaml.tmpl \
  deploy/gateway/security-policy.yaml.tmpl \
  deploy/gateway/backend-traffic-policy.yaml.tmpl; do
  envsubst < "$template" | kubectl apply -f -
done
```

The templates reference an existing Kubernetes Service named `observa-ingestion` on port `8080`; deploy that Service privately and apply network policy so only gateway data-plane pods can reach it. The root `.env.example` values are examples, not a live IdP. A production issuer must issue string `tenant_id` and `source_id` claims and the expected audience.

`OBSERVA_GATEWAY_RATE_LIMIT_POLICY_PATH` names the `BackendTrafficPolicy` template for deployment automation. The values in the `Gateway`/`HTTPRoute` templates are rendered through `envsubst`; they are templates, not directly applicable YAML until rendered.

## Local plain-Envoy fallback

`make local-up` builds the local Envoy image from `deploy/gateway/Dockerfile`, renders `envoy.yaml` using the same JWT issuer/audience/JWKS environment values, terminates TLS at `https://127.0.0.1:8443`, and renders `deploy/local/rate-limit/config.yaml.tmpl` into the rate-limit runtime volume and uses Redis for the shared coarse edge limit. The health endpoints are forwarded without JWT; the event endpoint rejects missing/invalid JWTs. Point `.env` at a real test issuer before testing a valid token. The included certificate and any default example URLs are local-development values only.

## Out of Scope (Deferred)

OpenTelemetry instrumentation, Prometheus metrics, consumer processing, evidence storage, ClickHouse, and Iceberg are not implemented by this gateway/ingestion change.
