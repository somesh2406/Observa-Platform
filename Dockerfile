
# Shared build environment
FROM golang:1.27.3-alpine AS build-base

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.work go.work.sum ./
COPY libs/eventcontract ./libs/eventcontract
COPY services/ingestion ./services/ingestion

# Build the ingestion executable
FROM build-base AS ingestion-build

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/ingestion \
    ./services/ingestion/cmd/ingestion

# Build the one-shot Kafka topic provisioner
FROM build-base AS topic-provisioner-build

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/kafka-topic-provisioner \
    ./services/ingestion/cmd/kafka-topic-provisioner

# Shared minimal runtime base
FROM alpine:3.22 AS runtime-base

RUN apk add --no-cache ca-certificates \
    && addgroup -S observa \
    && adduser -S -G observa observa

WORKDIR /app

USER observa

# Ingestion runtime image
FROM runtime-base AS ingestion

COPY --from=ingestion-build /out/ingestion /usr/local/bin/ingestion
COPY --chown=observa:observa configs/ingestion.local.json /app/configs/ingestion.local.json

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/ingestion"]

# One-shot provisioning runtime image
FROM runtime-base AS topic-provisioner

COPY --from=topic-provisioner-build \
    /out/kafka-topic-provisioner \
    /usr/local/bin/kafka-topic-provisioner

ENTRYPOINT ["/usr/local/bin/kafka-topic-provisioner"]
