#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

export ESCALITE_GRAPHQL_URL='https://api.example.com/graphql'
export ESCALITE_API_PUBLIC_URL='https://api.example.com'
export ESCALITE_STATUS_POLL_INTERVAL_MS='45000'
export RUNTIME_CONFIG_TEMPLATE="${REPO_ROOT}/deploy/docker-compose/nginx/runtime-config.js.template"

bash "${REPO_ROOT}/deploy/docker-compose/scripts/generate-runtime-config.sh" "${TMP_DIR}/runtime-config.js"

OUTPUT="$(cat "${TMP_DIR}/runtime-config.js")"

if ! grep -q 'graphqlUrl: "https://api.example.com/graphql"' <<<"${OUTPUT}"; then
  echo "expected runtime graphqlUrl in generated config" >&2
  exit 1
fi

if ! grep -q 'apiPublicUrl: "https://api.example.com"' <<<"${OUTPUT}"; then
  echo "expected runtime apiPublicUrl in generated config" >&2
  exit 1
fi

if ! grep -q 'statusPollIntervalMs: "45000"' <<<"${OUTPUT}"; then
  echo "expected runtime statusPollIntervalMs in generated config" >&2
  exit 1
fi

echo "generate-runtime-config.sh: ok"
