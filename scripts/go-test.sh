#!/usr/bin/env bash
# Run Go tests via gotestsum (readable summaries locally; github-actions format in CI).
set -euo pipefail

readonly GOTESTSUM_VERSION="v1.13.0"
readonly PACKAGES=(
	./services/api/...
	./services/engine/...
	./services/integrations/...
	./services/outboundintegrations/...
	./tools/goalert-importer/...
	./tools/pagerduty-importer/...
	./tools/schema-diff/...
)

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

export ALLURE_RESULTS_DIR="${ALLURE_RESULTS_DIR:-${ROOT}/allure-results/go}"
mkdir -p "${ALLURE_RESULTS_DIR}"

CI_MODE=false
if [[ "${CI:-}" == "true" || "${1:-}" == "--ci" ]]; then
	CI_MODE=true
fi

FORMAT="${GOTESTSUM_FORMAT:-testdox}"
JUNITFILE=""
GO_TEST_ARGS=()

if [[ "${CI_MODE}" == "true" ]]; then
	FORMAT="${GOTESTSUM_FORMAT:-github-actions}"
	JUNITFILE="/tmp/go-test-results.xml"
	GO_TEST_ARGS=(-count=1 -timeout 15m)
fi

args=(--format "${FORMAT}")
if [[ -n "${JUNITFILE}" ]]; then
	args+=(--junitfile "${JUNITFILE}")
fi

exec go run "gotest.tools/gotestsum@${GOTESTSUM_VERSION}" \
	"${args[@]}" \
	-- "${GO_TEST_ARGS[@]}" "${PACKAGES[@]}"
