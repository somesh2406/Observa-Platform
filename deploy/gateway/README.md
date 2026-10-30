# Envoy Edge Gateway

The local Compose stack uses plain Envoy from `envoy.yaml`. The repository also contains
Gateway API templates for an Envoy Gateway controller.

## Current edge behavior

- TLS terminates at Envoy.
- `POST /v1/events` requires a JWT validated against configured issuer, audience, and remote
  JWKS.
- JWT claims `tenant_id` and `source_id` are mapped to `X-Observa-Tenant` and
  `X-Observa-Source`.
- The plain Envoy configuration sets `forward: false` for the JWT provider.
- `/health/` is routed separately and does not require JWT.
- A shared edge limit is applied through the external rate-limit service.
- The plain Envoy filter sets `failure_mode_deny: true`, so edge rate-limit check failures are
  denied rather than passed through.

Implementation: `deploy/gateway/envoy.yaml:4-80`,
`deploy/local/rate-limit/config.yaml.tmpl:1-8`, and
`deploy/local/docker-compose.yml:58-94`.

## Gateway API templates

Apply the Gateway API controller and TLS Secret before rendering and applying templates. The
templates target a Kubernetes Service named `observa-ingestion` on port `8080`; this repository
does not contain that service's Kubernetes Deployment or Service manifest.

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

The templates describe gateway resources; they are not a complete Kubernetes deployment of the
ingestion service.

## Out of Scope (Deferred)

OpenTelemetry instrumentation, Prometheus metrics, consumer processing, evidence storage,
ClickHouse, Iceberg, and object-storage integration are not implemented by this gateway setup.
