# syntax=docker/dockerfile:1

# Minimal Dockerfile for local compose (#36). Production hardening lands in #37.
FROM golang:1.23-bookworm AS base
WORKDIR /src
ENV CGO_ENABLED=0

FROM base AS dev
WORKDIR /src/services/engine
CMD ["go", "run", "./cmd/engine"]

FROM base AS builder
ENV GOWORK=off
COPY services/engine/go.mod services/engine/go.sum ./services/engine/
RUN cd services/engine && go mod download
COPY services/engine ./services/engine
RUN cd services/engine && go build -trimpath -ldflags="-s -w" -o /out/engine ./cmd/engine

FROM debian:bookworm-slim AS prod
RUN groupadd --system --gid 10001 escalite \
  && useradd --system --uid 10001 --gid escalite --home-dir /app --no-create-home escalite
COPY --from=builder /out/engine /app/engine
USER 10001:10001
EXPOSE 8081
ENTRYPOINT ["/app/engine"]
