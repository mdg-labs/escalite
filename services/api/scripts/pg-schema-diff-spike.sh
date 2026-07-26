#!/usr/bin/env bash
# Spike: prove pg-schema-diff plan + apply on Escalite schema (task #142).
# Self-contained: starts its own ephemeral Postgres and removes it on exit.
# Day-to-day migration generation uses `task schema:diff` against compose Postgres instead.
# Usage: ./services/api/scripts/pg-schema-diff-spike.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SCHEMA_DIR="${ROOT}/services/api/schema/sql"
PG_IMAGE="postgres:16-alpine"
CONTAINER_NAME="escalite-pg-schema-diff-spike-$$"
PG_USER="escalite"
PG_PASS="escalite"
PG_DB="escalite"
DSN="postgres://${PG_USER}:${PG_PASS}@127.0.0.1:5433/${PG_DB}?sslmode=disable"
PLAN_FILE="$(mktemp)"
trap 'docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true; rm -f "${PLAN_FILE}"' EXIT

if ! command -v pg-schema-diff >/dev/null; then
  echo "pg-schema-diff not found; install: go install github.com/stripe/pg-schema-diff/cmd/pg-schema-diff@v1.0.7"
  exit 1
fi

echo "=== pg-schema-diff spike (Escalite schema.sql + realtime_notify.sql) ==="
echo "pg-schema-diff version: $(pg-schema-diff version)"
echo "schema dir: ${SCHEMA_DIR}"
echo

echo "--- Starting Postgres ${PG_IMAGE} on :5433 ---"
docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
docker run -d --name "${CONTAINER_NAME}" \
  -e POSTGRES_USER="${PG_USER}" \
  -e POSTGRES_PASSWORD="${PG_PASS}" \
  -e POSTGRES_DB="${PG_DB}" \
  -p 5433:5432 \
  "${PG_IMAGE}" >/dev/null

for _ in $(seq 1 30); do
  if docker exec "${CONTAINER_NAME}" pg_isready -U "${PG_USER}" -d "${PG_DB}" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker exec "${CONTAINER_NAME}" pg_isready -U "${PG_USER}" -d "${PG_DB}"

echo
echo "--- pg-schema-diff plan (empty DB -> schema dir) ---"
# schema.sql must load before realtime_notify.sql (triggers reference tables).
# pg-schema-diff applies files in lexical order within a dir; use two --to-dir
# flags so tables precede NOTIFY triggers without renaming source files.
TMP_SCHEMA=$(mktemp -d)
TMP_NOTIFY=$(mktemp -d)
cp "${SCHEMA_DIR}/schema.sql" "${TMP_SCHEMA}/"
cp "${SCHEMA_DIR}/realtime_notify.sql" "${TMP_NOTIFY}/"
trap 'docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true; rm -f "${PLAN_FILE}"; rm -rf "${TMP_SCHEMA}" "${TMP_NOTIFY}"' EXIT

pg-schema-diff plan \
  --from-dsn "${DSN}" \
  --to-dir "${TMP_SCHEMA}" \
  --to-dir "${TMP_NOTIFY}" \
  --output-format sql \
  | tee "${PLAN_FILE}"

echo
echo "--- Applying generated plan via psql ---"
# pg-schema-diff plan output includes SET SESSION lines; psql handles them.
docker exec -i "${CONTAINER_NAME}" psql -U "${PG_USER}" -d "${PG_DB}" -v ON_ERROR_STOP=1 < "${PLAN_FILE}"

echo
echo "--- Verify: tables, functions, triggers ---"
docker exec "${CONTAINER_NAME}" psql -U "${PG_USER}" -d "${PG_DB}" -c \
  "SELECT count(*) AS table_count FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE';"
docker exec "${CONTAINER_NAME}" psql -U "${PG_USER}" -d "${PG_DB}" -c \
  "SELECT proname FROM pg_proc WHERE pronamespace = 'public'::regnamespace AND proname LIKE 'notify_%' ORDER BY 1;"
docker exec "${CONTAINER_NAME}" psql -U "${PG_USER}" -d "${PG_DB}" -c \
  "SELECT tgname FROM pg_trigger t JOIN pg_class c ON t.tgrelid = c.oid WHERE NOT t.tgisinternal AND c.relnamespace = 'public'::regnamespace ORDER BY 1;"

echo
echo "=== Spike PASS ==="
