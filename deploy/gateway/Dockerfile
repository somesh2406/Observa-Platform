# Local plain-Envoy fallback. The template is rendered from environment variables
# before Envoy starts; the Gateway API resources remain the production control plane.
FROM envoyproxy/envoy:v1.35.0
USER root
RUN mkdir -p /etc/envoy/certs && apt-get update && apt-get install -y --no-install-recommends gettext-base ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*
USER envoy
ENTRYPOINT ["/bin/sh", "-ec", "envsubst < /etc/envoy/envoy.yaml > /tmp/envoy.yaml; exec envoy -c /tmp/envoy.yaml"]
