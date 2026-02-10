#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
cd "${REPO_ROOT}"

ABS_ACCEPTANCE="${REPO_ROOT}/docs/acceptance.md"
ABS_ROUTER="${REPO_ROOT}/internal/access/http/router.go"
ABS_OPENAPI="${REPO_ROOT}/docs/api/openapi.yaml"

UPSTREAM="$(git rev-parse --abbrev-ref --symbolic-full-name @{upstream} 2>/dev/null || true)"
if [[ -n "${UPSTREAM}" ]]; then
  AHEAD_BEHIND="$(git rev-list --left-right --count "${UPSTREAM}"...HEAD 2>/dev/null || echo "unknown unknown")"
else
  AHEAD_BEHIND="unknown unknown"
fi

AHEAD_COUNT="$(awk '{print $2}' <<<"${AHEAD_BEHIND}")"
BEHIND_COUNT="$(awk '{print $1}' <<<"${AHEAD_BEHIND}")"

TOTAL_CHANGES="$(git status --short | wc -l | tr -d ' ')"
DIRTY_STATE="clean"
if [[ "${TOTAL_CHANGES}" != "0" ]]; then
  DIRTY_STATE="dirty"
fi

printf '## Baseline Snapshot\n'
printf -- '- repo_root: %s\n' "${REPO_ROOT}"
printf -- '- branch: %s\n' "$(git rev-parse --abbrev-ref HEAD)"
printf -- '- head: %s\n' "$(git rev-parse --short HEAD)"
printf -- '- upstream: %s\n' "${UPSTREAM:-none}"
printf -- '- ahead: %s\n' "${AHEAD_COUNT:-unknown}"
printf -- '- behind: %s\n' "${BEHIND_COUNT:-unknown}"
printf -- '- worktree_state: %s (%s changes)\n' "${DIRTY_STATE}" "${TOTAL_CHANGES}"
printf -- '- key_paths:\n'
printf -- '  - %s\n' "${ABS_ACCEPTANCE}"
printf -- '  - %s\n' "${ABS_ROUTER}"
printf -- '  - %s\n' "${ABS_OPENAPI}"
printf -- '- commands:\n'
printf -- '  - git status --short --branch\n'
printf -- '  - git worktree list\n'
printf -- '  - git log --oneline -n 10\n'

printf '\n## Worktree List\n'
git worktree list

printf '\n## Status (--short --branch)\n'
git status --short --branch

printf '\n## Recent Commits\n'
git log --oneline -n 10

printf '\n## Key Directory Presence\n'
for path in docs internal migrations web .agents/skills; do
  abs="${REPO_ROOT}/${path}"
  if [[ -e "${abs}" ]]; then
    printf -- '- confirmed: %s\n' "${abs}"
  else
    printf -- '- missing: %s\n' "${abs}"
  fi
done
