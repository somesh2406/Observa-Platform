FROM alpine:3.22
RUN apk add --no-cache gettext
ENTRYPOINT ["/bin/sh", "-ec", "export OBSERVA_EDGE_RATE_LIMIT_UNIT_LOWER=$(printf '%s' \"$OBSERVA_EDGE_RATE_LIMIT_UNIT\" | tr '[:upper:]' '[:lower:]'); envsubst < /template/config.yaml.tmpl > /output/config.yaml"]
