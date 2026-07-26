#!/usr/bin/env bash
# Block shell commands that truncate live output (tail/head pipes).
# See .cursor/rules/16-terminal-output.mdc
set -euo pipefail

input="$(cat)"
command="$(printf '%s' "$input" | jq -r '.command // empty')"

if [[ -z "$command" ]]; then
  printf '%s\n' '{ "permission": "allow" }'
  exit 0
fi

# Piping to tail/head hides streaming progress from long-running commands.
if printf '%s' "$command" | grep -Eq '\|[[:space:]]*([0-9]*[[:space:]]+)?(tail|head)([[:space:]]|$|-)'; then
  cat <<'EOF'
{
  "permission": "deny",
  "user_message": "Blocked: piping shell output to tail/head hides live progress and can look like a hang. Run the full command, then search completed output with rg/grep or Read the terminal file with offset/limit.",
  "agent_message": "Do not pipe running commands to tail or head (.cursor/rules/16-terminal-output.mdc). Run the command without truncation; use Await with a pattern for long jobs, or rg/grep on finished output."
}
EOF
  exit 0
fi

# Standalone tail/head on log files — use Read with offset/limit instead.
if printf '%s' "$command" | grep -Eq '(^|[;&|][[:space:]]*)(tail|head)([[:space:]]|$|-)'; then
  cat <<'EOF'
{
  "permission": "deny",
  "user_message": "Blocked: use the Read tool with offset/limit or rg/grep on completed output instead of tail/head.",
  "agent_message": "Do not use tail/head in shell commands (.cursor/rules/16-terminal-output.mdc). Read terminal output files directly or filter with rg/grep after the command finishes."
}
EOF
  exit 0
fi

printf '%s\n' '{ "permission": "allow" }'
exit 0
