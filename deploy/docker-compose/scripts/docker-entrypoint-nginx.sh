#!/bin/sh
set -eu

NGINX_TEMPLATE="${NGINX_TEMPLATE:-/etc/nginx/templates/default.conf.template}"
NGINX_OUTPUT="${NGINX_OUTPUT:-/etc/nginx/conf.d/default.conf}"

export ESCALITE_API_UPSTREAM="${ESCALITE_API_UPSTREAM:-http://api:8080}"

/docker-entrypoint.d/generate-runtime-config.sh

envsubst '${ESCALITE_API_UPSTREAM}' < "${NGINX_TEMPLATE}" > "${NGINX_OUTPUT}"

exec "$@"
