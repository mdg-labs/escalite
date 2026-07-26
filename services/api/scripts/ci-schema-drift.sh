#!/usr/bin/env bash
# CI drift gate: apply committed goose migrations and verify DB matches schema/sql.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
# Pinned version; must match tools/schema-diff/main.go and tools/schema-diff/go.mod tool directive.
readonly PG_SCHEMA_DIFF_VERSION="v1.0.7"
readonly PG_IMAGE="postgres:16-alpine"
readonly CONTAINER_NAME="escalite-ci-schema-drift-$$"
readonly PG_USER="escalite"
readonly PG_PASS="escalite"
readonly PG_DB="escalite"
readonly PG_PORT="${PG_PORT:-5434}"
readonly DSN="postgres://${PG_USER}:${PG_PASS}@127.0.0.1:${PG_PORT}/${PG_DB}?sslmode=disable"

cleanup() {
  docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "=== CI schema drift check (pg-schema-diff ${PG_SCHEMA_DIFF_VERSION}) ==="

docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
docker run -d --name "${CONTAINER_NAME}" \
  -e POSTGRES_USER="${PG_USER}" \
  -e POSTGRES_PASSWORD="${PG_PASS}" \
  -e POSTGRES_DB="${PG_DB}" \
  -p "${PG_PORT}:5432" \
  "${PG_IMAGE}" >/dev/null

for _ in $(seq 1 30); do
  if docker exec "${CONTAINER_NAME}" pg_isready -U "${PG_USER}" -d "${PG_DB}" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker exec "${CONTAINER_NAME}" pg_isready -U "${PG_USER}" -d "${PG_DB}"

echo "--- applying goose migrations ---"
cd "${ROOT}"
go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 \
  -dir services/api/migrations postgres "${DSN}" up

echo "--- checking drift (migrations vs schema/sql) ---"
go install "github.com/stripe/pg-schema-diff/cmd/pg-schema-diff@${PG_SCHEMA_DIFF_VERSION}"
go run ./tools/schema-diff --check-drift --dsn "${DSN}" --skip-validation

echo "=== schema drift check PASS ==="
