# syntax=docker/dockerfile:1

# Multi-stage API image — distroless nonroot runtime (doc 07 supply-chain).
FROM golang:1.26-bookworm@sha256:1ecb7edf62a0408027bd5729dfd6b1b8766e578e8df93995b225dfd0944eb651 AS base
WORKDIR /src
ENV CGO_ENABLED=0

FROM base AS dev
WORKDIR /src/services/api
CMD ["go", "run", "./cmd/api"]

FROM base AS builder
ENV GOWORK=off
COPY services/dbmigrate/go.mod services/dbmigrate/go.sum ./services/dbmigrate/
COPY services/outboundintegrations/go.mod services/outboundintegrations/go.sum ./services/outboundintegrations/
COPY services/integrations/go.mod services/integrations/go.sum ./services/integrations/
COPY services/engine/go.mod services/engine/go.sum ./services/engine/
COPY services/api/go.mod services/api/go.sum ./services/api/
RUN cd services/api && go mod download
COPY services/dbmigrate ./services/dbmigrate
COPY services/outboundintegrations ./services/outboundintegrations
COPY services/integrations ./services/integrations
COPY services/engine ./services/engine
COPY services/api ./services/api
RUN cd services/api && go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# UID 65532 (nonroot) — no shell, minimal attack surface.
FROM busybox:1.36.1-musl@sha256:3c6ae8008e2c2eedd141725c30b20d9c36b026eb796688f88205845ef17aa213 AS busybox
FROM gcr.io/distroless/static-debian12:nonroot@sha256:f5b485ea962d9bd1186b2f6b3a061191539b905b82ec395de78cbfae51f20e35 AS prod
COPY --from=busybox /bin/wget /wget
COPY --from=builder /out/api /app/api
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --retries=12 --start-period=30s \
  CMD ["/wget", "-q", "-O", "/dev/null", "http://127.0.0.1:8080/healthz"]
ENTRYPOINT ["/app/api"]
