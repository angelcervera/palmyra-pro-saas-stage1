# syntax=docker/dockerfile:1.7

FROM golang:1.25 AS builder

WORKDIR /src

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy the remaining source
COPY . .

# Build the API binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./apps/api

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/api /app/api
COPY contracts /app/contracts

ENV PORT=3000
EXPOSE 3000

CMD ["./api"]

