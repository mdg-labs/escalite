# syntax=docker/dockerfile:1

# Minimal Dockerfile for local compose (#36). Full Vite/nginx build lands in #37.
FROM node:20-bookworm-slim AS base
WORKDIR /app
ENV PNPM_HOME=/pnpm
ENV PATH="${PNPM_HOME}:${PATH}"
RUN corepack enable

FROM base AS deps
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml turbo.json ./
COPY apps/web/package.json apps/web/
COPY packages/config/package.json packages/config/
RUN pnpm install --frozen-lockfile

FROM deps AS dev
COPY . .
WORKDIR /app/apps/web
EXPOSE 5173
CMD ["pnpm", "dev"]

FROM deps AS builder
COPY . .
WORKDIR /app/apps/web
RUN pnpm build

FROM nginx:1.27-alpine AS prod
COPY deploy/docker-compose/nginx/web.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/apps/web/dist /usr/share/nginx/html
EXPOSE 5173
