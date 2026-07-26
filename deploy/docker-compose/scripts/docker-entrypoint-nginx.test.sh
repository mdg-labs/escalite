#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

export ESCALITE_API_UPSTREAM='http://nb469w5exa3hcyljj9zqfp0r:8080'
export NGINX_TEMPLATE="${REPO_ROOT}/deploy/docker-compose/nginx/web.conf.template"
export NGINX_OUTPUT="${TMP_DIR}/default.conf"
export GENERATE_RUNTIME_CONFIG="${REPO_ROOT}/deploy/docker-compose/scripts/generate-runtime-config.sh"
export RUNTIME_CONFIG_TEMPLATE="${REPO_ROOT}/deploy/docker-compose/nginx/runtime-config.js.template"
export RUNTIME_CONFIG_OUTPUT="${TMP_DIR}/runtime-config.js"

bash "${REPO_ROOT}/deploy/docker-compose/scripts/docker-entrypoint-nginx.sh" true

if ! grep -q 'set $escalite_api_upstream "http://nb469w5exa3hcyljj9zqfp0r:8080"' "${NGINX_OUTPUT}"; then
  echo "expected envsubst to inject ESCALITE_API_UPSTREAM into nginx config" >&2
  cat "${NGINX_OUTPUT}" >&2
  exit 1
fi

if grep -q 'upstream "api"' "${NGINX_OUTPUT}"; then
  echo "nginx config must not contain compose-only api upstream" >&2
  exit 1
fi

echo "docker-entrypoint-nginx.sh: ok"
