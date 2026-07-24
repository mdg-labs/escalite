# syntax=docker/dockerfile:1

# Multi-stage engine image — distroless nonroot runtime (doc 07 supply-chain).
FROM golang:1.26-bookworm@sha256:1ecb7edf62a0408027bd5729dfd6b1b8766e578e8df93995b225dfd0944eb651 AS base
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

# UID 65532 (nonroot) — no shell, minimal attack surface.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:f5b485ea962d9bd1186b2f6b3a061191539b905b82ec395de78cbfae51f20e35 AS prod
COPY --from=builder /out/engine /app/engine
EXPOSE 8081
ENTRYPOINT ["/app/engine"]
