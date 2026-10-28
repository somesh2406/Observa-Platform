# Build context is the repository root so the Go workspace and shared contract module are available.
FROM golang:1.27.3-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.work go.work.sum ./
COPY libs/eventcontract ./libs/eventcontract
COPY services/ingestion ./services/ingestion
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ingestion ./services/ingestion/cmd/ingestion

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && addgroup -S observa && adduser -S -G observa observa
WORKDIR /app
COPY --from=build /out/ingestion /usr/local/bin/ingestion
COPY configs/ingestion.local.json /app/configs/ingestion.local.json
USER observa
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/ingestion"]
