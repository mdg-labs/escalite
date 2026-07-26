# syntax=docker/dockerfile:1

# Multi-stage web image — nginx unprivileged (non-root) runtime (doc 07 supply-chain).
FROM node:20-bookworm-slim@sha256:2cf067cfed83d5ea958367df9f966191a942351a2df77d6f0193e162b5febfc0 AS base
WORKDIR /app
ENV PNPM_HOME=/pnpm
ENV PATH="${PNPM_HOME}:${PATH}"
RUN corepack enable

FROM base AS deps
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml turbo.json ./
COPY patches patches
COPY apps/web/package.json apps/web/
COPY packages/config/package.json packages/config/
COPY packages/ui/package.json packages/ui/
COPY packages/ts-types/package.json packages/ts-types/
COPY packages/schema/package.json packages/schema/
RUN pnpm install --frozen-lockfile

FROM deps AS dev
COPY . .
WORKDIR /app/apps/web
EXPOSE 5173
CMD ["pnpm", "dev"]

FROM deps AS builder
COPY . .
WORKDIR /app
RUN pnpm turbo run build --filter=@escalite/web...

# UID 101 (nginx) — listens on unprivileged ports only.
FROM busybox:1.36.1-musl@sha256:3c6ae8008e2c2eedd141725c30b20d9c36b026eb796688f88205845ef17aa213 AS busybox
FROM nginxinc/nginx-unprivileged:1.27-alpine@sha256:65e3e85dbaed8ba248841d9d58a899b6197106c23cb0ff1a132b7bfe0547e4c0 AS prod
COPY --from=busybox /bin/wget /usr/local/bin/wget
COPY deploy/docker-compose/nginx/web.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/apps/web/dist /usr/share/nginx/html
EXPOSE 5173
HEALTHCHECK --interval=5s --timeout=3s --retries=12 --start-period=15s \
  CMD ["/usr/local/bin/wget", "-q", "-O", "/dev/null", "http://127.0.0.1:5173/"]
