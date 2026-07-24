#!/usr/bin/env bash
# CI compose gate: build prod profile, docker compose up, E2E smoke, tear down.
# Fails on non-zero docker compose exit or E2E failure.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_DIR="${REPO_ROOT}/deploy/docker-compose"
COMPOSE_FILES=(-f docker-compose.yml -f docker-compose.prod.yml)

export ESCALITE_ENCRYPTION_KEY="${ESCALITE_ENCRYPTION_KEY:-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef}"
export ESCALITE_API_PORT="${ESCALITE_API_PORT:-8080}"
export ESCALITE_WEB_PORT="${ESCALITE_WEB_PORT:-5173}"
export ESCALITE_APP_ORIGIN="http://localhost:${ESCALITE_WEB_PORT}"
export ESCALITE_API_URL="http://localhost:${ESCALITE_API_PORT}"
export ESCALITE_DOCKER_TARGET=prod
export CI="${CI:-true}"

cleanup() {
  (
    cd "${COMPOSE_DIR}"
    docker compose "${COMPOSE_FILES[@]}" --profile prod down -v --remove-orphans
  ) || true
}
trap cleanup EXIT

cat >"${COMPOSE_DIR}/.env" <<EOF
ESCALITE_DOCKER_TARGET=prod
POSTGRES_USER=escalite
POSTGRES_PASSWORD=escalite
POSTGRES_DB=escalite
ESCALITE_DATABASE_URL=postgres://escalite:escalite@postgres:5432/escalite?sslmode=disable
ESCALITE_ENCRYPTION_KEY=${ESCALITE_ENCRYPTION_KEY}
ESCALITE_APP_ORIGIN=${ESCALITE_APP_ORIGIN}
ESCALITE_API_PORT=${ESCALITE_API_PORT}
ESCALITE_WEB_PORT=${ESCALITE_WEB_PORT}
ESCALITE_LOG_LEVEL=info
EOF

cd "${COMPOSE_DIR}"

echo "Starting compose prod stack (build + up)..."
docker compose "${COMPOSE_FILES[@]}" --profile prod up -d --build --wait

echo "Running E2E smoke tests against compose stack..."
cd "${REPO_ROOT}"
pnpm --filter @escalite/web e2e
