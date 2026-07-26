#!/usr/bin/env bash
# Run govulncheck per Go module (go.work monorepo has no root go.mod).
set -euo pipefail

readonly MODULES=(
	services/api
	services/engine
	services/integrations
)

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

go install golang.org/x/vuln/cmd/govulncheck@latest

for mod in "${MODULES[@]}"; do
	echo "govulncheck: ${mod}"
	(cd "${mod}" && govulncheck ./...)
done
