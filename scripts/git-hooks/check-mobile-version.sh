#!/usr/bin/env bash
# Pre-commit gate: staged apps/mobile changes require a beta version bump.
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
VERSION_FILE="apps/mobile/package.json"
WATCH_PREFIX="apps/mobile"

mapfile -t staged_files < <(git diff --cached --name-only -- "${WATCH_PREFIX}/" 2>/dev/null || true)

relevant=()
for file in "${staged_files[@]}"; do
  [[ -z "${file}" ]] && continue
  if [[ "${file}" == "${WATCH_PREFIX}/.gitignore" ]]; then
    continue
  fi
  relevant+=("${file}")
done

if [[ ${#relevant[@]} -eq 0 ]]; then
  exit 0
fi

head_version=""
if git cat-file -e "HEAD:${VERSION_FILE}" 2>/dev/null; then
  head_version="$(
    git show "HEAD:${VERSION_FILE}" | node -e "
      let data = '';
      process.stdin.on('data', (chunk) => { data += chunk; });
      process.stdin.on('end', () => {
        const pkg = JSON.parse(data);
        process.stdout.write(String(pkg.version ?? ''));
      });
    "
  )"
fi

if git diff --cached --name-only -- "${VERSION_FILE}" | grep -q .; then
  staged_version="$(
    git show ":${VERSION_FILE}" | node -e "
      let data = '';
      process.stdin.on('data', (chunk) => { data += chunk; });
      process.stdin.on('end', () => {
        const pkg = JSON.parse(data);
        process.stdout.write(String(pkg.version ?? ''));
      });
    "
  )"
else
  staged_version="${head_version}"
fi

if node --input-type=module - "${staged_version}" "${head_version}" <<'EOF'
const [current, previous] = process.argv.slice(2);

function parseVersion(version) {
  const match = /^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$/.exec(version);
  if (!match) {
    throw new Error(`invalid semver: ${version}`);
  }
  return {
    major: Number(match[1]),
    minor: Number(match[2]),
    patch: Number(match[3]),
    prerelease: match[4] ?? '',
  };
}

function compareVersions(left, right) {
  for (const key of ['major', 'minor', 'patch']) {
    if (left[key] !== right[key]) {
      return left[key] - right[key];
    }
  }

  if (!left.prerelease && !right.prerelease) {
    return 0;
  }
  if (!left.prerelease) {
    return 1;
  }
  if (!right.prerelease) {
    return -1;
  }

  const leftParts = left.prerelease.split('.');
  const rightParts = right.prerelease.split('.');
  const length = Math.max(leftParts.length, rightParts.length);
  for (let index = 0; index < length; index += 1) {
    const leftPart = leftParts[index];
    const rightPart = rightParts[index];
    if (leftPart === undefined) {
      return -1;
    }
    if (rightPart === undefined) {
      return 1;
    }
    const leftNumber = /^\d+$/.test(leftPart) ? Number(leftPart) : leftPart;
    const rightNumber = /^\d+$/.test(rightPart) ? Number(rightPart) : rightPart;
    if (leftNumber === rightNumber) {
      continue;
    }
    return leftNumber > rightNumber ? 1 : -1;
  }

  return 0;
}

try {
  const currentVersion = parseVersion(current);
  const previousVersion = previous ? parseVersion(previous) : null;
  const increased = previousVersion ? compareVersions(currentVersion, previousVersion) > 0 : true;

  if (!increased) {
    process.exit(1);
  }

  if (!currentVersion.prerelease) {
    process.exit(1);
  }

  process.exit(0);
} catch (error) {
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(1);
}
EOF
then
  exit 0
fi

cat >&2 <<EOF
pre-commit: staged changes under ${WATCH_PREFIX}/ require a prerelease bump in ${VERSION_FILE}.

Staged version: ${staged_version:-<missing>}
HEAD version:   ${head_version:-<none>}

Bump version (e.g. 0.1.0-beta.N -> 0.1.0-beta.N+1) and stage ${VERSION_FILE}.
EOF
exit 1
