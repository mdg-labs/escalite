#!/usr/bin/env bash
set -euo pipefail

# Mobile version bump gate disabled for now.
# To re-enable: exec bash "${REPO_ROOT}/scripts/git-hooks/check-mobile-version.sh"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

exit 0
