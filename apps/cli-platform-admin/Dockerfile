# syntax=docker/dockerfile:1.7

FROM golang:1.25 AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/cli-platform-admin ./apps/cli

RUN install -m 0755 apps/cli/wait-for-postgres.sh /out/wait-for-postgres.sh

FROM debian:bookworm-slim AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates netcat-openbsd bash \
    && rm -rf /var/lib/apt/lists/*

# Busybox sh in Debian image is fine; nc is available for readiness check.

COPY --from=builder /out/cli-platform-admin /usr/local/bin/cli-platform-admin
COPY --from=builder /out/wait-for-postgres.sh /usr/local/bin/wait-for-postgres.sh

ENTRYPOINT ["/usr/local/bin/cli-platform-admin"]
