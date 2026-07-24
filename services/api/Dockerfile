# syntax=docker/dockerfile:1

# Minimal Dockerfile for local compose (#36). Production hardening lands in #37.
FROM golang:1.23-bookworm AS base
WORKDIR /src
ENV CGO_ENABLED=0

FROM base AS dev
WORKDIR /src/services/api
CMD ["go", "run", "./cmd/api"]

FROM base AS builder
ENV GOWORK=off
COPY services/api/go.mod services/api/go.sum ./services/api/
RUN cd services/api && go mod download
COPY services/api ./services/api
RUN cd services/api && go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM debian:bookworm-slim AS prod
RUN groupadd --system --gid 10001 escalite \
  && useradd --system --uid 10001 --gid escalite --home-dir /app --no-create-home escalite
COPY --from=builder /out/api /app/api
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/app/api"]
