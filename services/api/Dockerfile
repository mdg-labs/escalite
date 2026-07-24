# syntax=docker/dockerfile:1

# Multi-stage API image — distroless nonroot runtime (doc 07 supply-chain).
FROM golang:1.23-bookworm@sha256:167053a2bb901972bf2c1611f8f52c44d5fe7e762e5cab213708d82c421614db AS base
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

# UID 65532 (nonroot) — no shell, minimal attack surface.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:f5b485ea962d9bd1186b2f6b3a061191539b905b82ec395de78cbfae51f20e35 AS prod
COPY --from=builder /out/api /app/api
EXPOSE 8080
ENTRYPOINT ["/app/api"]
