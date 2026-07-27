#!/usr/bin/env bash
# commit-msg gate: require Signed-off-by matching git user.name / user.email.
set -euo pipefail

commit_msg_file="${1:?commit message file required}"

# Merge commits reuse upstream history; DCO is enforced per parent commit.
if [[ -f "$(git rev-parse --git-path MERGE_HEAD)" ]]; then
  exit 0
fi

author_name="$(git config --get user.name 2>/dev/null || true)"
author_email="$(git config --get user.email 2>/dev/null || true)"

if [[ -z "${author_name}" || -z "${author_email}" ]]; then
  cat >&2 <<'EOF'
commit-msg: git user.name and user.email must be set before committing.

Example:
  git config user.name "Jane Doe"
  git config user.email "jane@example.com"
EOF
  exit 1
fi

author_email_lower="$(printf '%s' "${author_email}" | tr '[:upper:]' '[:lower:]')"

matched=0
while IFS= read -r line; do
  [[ "${line}" =~ ^Signed-off-by:\ (.+)\ \<([^>]+)\>$ ]] || continue

  sign_name="${BASH_REMATCH[1]}"
  sign_email="${BASH_REMATCH[2]}"
  sign_email_lower="$(printf '%s' "${sign_email}" | tr '[:upper:]' '[:lower:]')"

  if [[ "${sign_name}" == "${author_name}" && "${sign_email_lower}" == "${author_email_lower}" ]]; then
    matched=1
    break
  fi
done < <(grep -E '^Signed-off-by: ' "${commit_msg_file}" || true)

if [[ "${matched}" -eq 1 ]]; then
  exit 0
fi

cat >&2 <<EOF
commit-msg: missing DCO sign-off for ${author_name} <${author_email}>.

Add a Signed-off-by trailer that matches your commit author, for example:
  git commit -s -m "feat(web): your message"

Or amend the last commit:
  git commit --amend -s --no-edit
EOF
exit 1
