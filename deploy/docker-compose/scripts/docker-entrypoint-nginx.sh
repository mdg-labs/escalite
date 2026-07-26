#!/bin/sh
set -eu

NGINX_TEMPLATE="${NGINX_TEMPLATE:-/etc/escalite/nginx/default.conf.template}"
NGINX_OUTPUT="${NGINX_OUTPUT:-/etc/nginx/conf.d/default.conf}"

if [ -n "${ESCALITE_API_PUBLIC_URL:-}" ] && [ "${ESCALITE_API_UPSTREAM:-}" = "" ]; then
  export ESCALITE_API_UPSTREAM="${ESCALITE_API_PUBLIC_URL}"
elif [ -z "${ESCALITE_API_UPSTREAM:-}" ]; then
  export ESCALITE_API_UPSTREAM="http://api:8080"
fi

echo "escalite: ESCALITE_API_UPSTREAM=${ESCALITE_API_UPSTREAM}" >&2

GENERATE_RUNTIME_CONFIG="${GENERATE_RUNTIME_CONFIG:-/docker-entrypoint.d/generate-runtime-config.sh}"
"${GENERATE_RUNTIME_CONFIG}" "${RUNTIME_CONFIG_OUTPUT:-/usr/share/nginx/html/runtime-config.js}"

# Prevent the stock nginx entrypoint from writing a stale compose-only config.
rm -f /etc/nginx/templates/default.conf.template

envsubst '${ESCALITE_API_UPSTREAM}' < "${NGINX_TEMPLATE}" > "${NGINX_OUTPUT}"

exec "$@"
