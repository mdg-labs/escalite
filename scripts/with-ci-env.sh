#!/usr/bin/env bash
# Local CI gate wrapper — sets CI-like env vars before running checks.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

export CI=true
export NODE_ENV=test

exec "$@"
