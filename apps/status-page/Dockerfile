# syntax=docker/dockerfile:1

FROM node:20-bookworm-slim@sha256:2cf067cfed83d5ea958367df9f966191a942351a2df77d6f0193e162b5febfc0 AS base
WORKDIR /app
ENV PNPM_HOME=/pnpm
ENV PATH="${PNPM_HOME}:${PATH}"
RUN corepack enable

FROM base AS deps
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml turbo.json ./
COPY patches patches
COPY apps/status-page/package.json apps/status-page/
COPY packages/config/package.json packages/config/
COPY packages/runtime-config/package.json packages/runtime-config/
COPY packages/ui/package.json packages/ui/
RUN pnpm install --frozen-lockfile

FROM deps AS dev
COPY . .
WORKDIR /app/apps/status-page
EXPOSE 5174
CMD ["pnpm", "dev"]

FROM deps AS builder
COPY . .
WORKDIR /app/apps/status-page
RUN pnpm build

FROM busybox:1.36.1-musl@sha256:3c6ae8008e2c2eedd141725c30b20d9c36b026eb796688f88205845ef17aa213 AS busybox
FROM nginxinc/nginx-unprivileged:1.27-alpine@sha256:65e3e85dbaed8ba248841d9d58a899b6197106c23cb0ff1a132b7bfe0547e4c0 AS prod
USER root
RUN apk add --no-cache gettext
COPY --from=busybox /bin/wget /usr/local/bin/wget
COPY deploy/docker-compose/nginx/status-page.conf.template /etc/escalite/nginx/default.conf.template
COPY deploy/docker-compose/nginx/runtime-config.js.template /etc/nginx/runtime-config.js.template
COPY deploy/docker-compose/scripts/generate-runtime-config.sh /docker-entrypoint.d/generate-runtime-config.sh
COPY deploy/docker-compose/scripts/docker-entrypoint-nginx.sh /docker-entrypoint-nginx.sh
RUN chmod +x /docker-entrypoint.d/generate-runtime-config.sh /docker-entrypoint-nginx.sh \
  && mkdir -p /etc/escalite/nginx \
  && rm -f /etc/nginx/templates/default.conf.template \
  && chown -R nginx:nginx /etc/nginx/conf.d /etc/escalite/nginx /usr/share/nginx/html
COPY --from=builder /app/apps/status-page/dist /usr/share/nginx/html
USER nginx
EXPOSE 5174
ENTRYPOINT ["/docker-entrypoint-nginx.sh"]
CMD ["nginx", "-g", "daemon off;"]
HEALTHCHECK --interval=5s --timeout=3s --retries=12 --start-period=15s \
  CMD ["/usr/local/bin/wget", "-q", "-O", "/dev/null", "http://127.0.0.1:5174/healthz"]
