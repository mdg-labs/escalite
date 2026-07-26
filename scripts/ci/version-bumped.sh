#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 [--no-bump-check] [--prerelease-only|--stable-only] <file-path>" >&2
  exit 2
}

mode="any"
no_bump_check=0
while [[ "${1:-}" == --* ]]; do
  case "${1}" in
    --prerelease-only)
      mode="prerelease"
      shift
      ;;
    --stable-only)
      mode="stable"
      shift
      ;;
    --no-bump-check)
      no_bump_check=1
      shift
      ;;
    *)
      usage
      ;;
  esac
done

file_path="${1:-}"
if [[ -z "${file_path}" ]]; then
  usage
fi

if [[ ! -f "${file_path}" ]]; then
  echo "version file not found: ${file_path}" >&2
  exit 1
fi

file_path="$(cd "$(dirname "${file_path}")" && pwd)/$(basename "${file_path}")"
repo_root="$(git rev-parse --show-toplevel)"
rel_path="${file_path#${repo_root}/}"

current_version="$(tr -d '[:space:]' < "${file_path}")"
if [[ "${rel_path}" == *package.json ]]; then
  current_version="$(node -e "const fs=require('fs'); const pkg=JSON.parse(fs.readFileSync(process.argv[1],'utf8')); process.stdout.write(String(pkg.version ?? ''));" "${file_path}")"
fi
if [[ -z "${current_version}" ]]; then
  echo "version file is empty: ${file_path}" >&2
  exit 1
fi

previous_version=""
if git rev-parse HEAD~1 >/dev/null 2>&1; then
  if git cat-file -e "HEAD~1:${rel_path}" 2>/dev/null; then
    if [[ "${rel_path}" == *package.json ]]; then
      previous_version="$(git show "HEAD~1:${rel_path}" | node -e "let data='';process.stdin.on('data',c=>data+=c);process.stdin.on('end',()=>{const pkg=JSON.parse(data); process.stdout.write(String(pkg.version ?? ''))})")"
    else
      previous_version="$(git show "HEAD~1:${rel_path}" | tr -d '[:space:]')"
    fi
  fi
fi

node --input-type=module - "${current_version}" "${previous_version}" "${mode}" "${no_bump_check}" <<'EOF'
const [current, previous, mode, noBumpCheckArg] = process.argv.slice(2)
const noBumpCheck = noBumpCheckArg === '1'

function parseVersion(version) {
  const match = /^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$/.exec(version)
  if (!match) {
    throw new Error(`invalid semver: ${version}`)
  }
  return {
    major: Number(match[1]),
    minor: Number(match[2]),
    patch: Number(match[3]),
    prerelease: match[4] ?? '',
  }
}

function compareVersions(left, right) {
  for (const key of ['major', 'minor', 'patch']) {
    if (left[key] !== right[key]) {
      return left[key] - right[key]
    }
  }

  if (!left.prerelease && !right.prerelease) {
    return 0
  }
  if (!left.prerelease) {
    return 1
  }
  if (!right.prerelease) {
    return -1
  }

  const leftParts = left.prerelease.split('.')
  const rightParts = right.prerelease.split('.')
  const length = Math.max(leftParts.length, rightParts.length)
  for (let index = 0; index < length; index += 1) {
    const leftPart = leftParts[index]
    const rightPart = rightParts[index]
    if (leftPart === undefined) {
      return -1
    }
    if (rightPart === undefined) {
      return 1
    }
    const leftNumber = /^\d+$/.test(leftPart) ? Number(leftPart) : leftPart
    const rightNumber = /^\d+$/.test(rightPart) ? Number(rightPart) : rightPart
    if (leftNumber === rightNumber) {
      continue
    }
    return leftNumber > rightNumber ? 1 : -1
  }

  return 0
}

try {
  const currentVersion = parseVersion(current)
  const previousVersion = previous ? parseVersion(previous) : null
  const increased = previousVersion ? compareVersions(currentVersion, previousVersion) > 0 : true

  if (!noBumpCheck && !increased) {
    process.exit(1)
  }

  const isPrerelease = currentVersion.prerelease.length > 0
  if (mode === 'prerelease' && !isPrerelease) {
    process.exit(1)
  }
  if (mode === 'stable' && isPrerelease) {
    process.exit(1)
  }

  process.stdout.write(current)
} catch (error) {
  console.error(error instanceof Error ? error.message : String(error))
  process.exit(1)
}
EOF
