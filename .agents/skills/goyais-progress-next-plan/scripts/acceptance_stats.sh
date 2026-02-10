#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
ACCEPTANCE_FILE="${REPO_ROOT}/docs/acceptance.md"

if [[ ! -f "${ACCEPTANCE_FILE}" ]]; then
  echo "error: missing acceptance file: ${ACCEPTANCE_FILE}" >&2
  exit 1
fi

TOTAL_ITEMS="$( (rg -n '^- \[(x| )\]' "${ACCEPTANCE_FILE}" || true) | wc -l | tr -d ' ' )"
DONE_ITEMS="$( (rg -n '^- \[x\]' "${ACCEPTANCE_FILE}" || true) | wc -l | tr -d ' ' )"
TODO_ITEMS="$( (rg -n '^- \[ \]' "${ACCEPTANCE_FILE}" || true) | wc -l | tr -d ' ' )"

if [[ "${TOTAL_ITEMS}" == "0" ]]; then
  RATIO="0.0"
else
  RATIO="$(awk -v d="${DONE_ITEMS}" -v t="${TOTAL_ITEMS}" 'BEGIN { printf "%.1f", (d*100)/t }')"
fi

printf '## Acceptance Progress\n'
printf -- '- file: %s\n' "${ACCEPTANCE_FILE}"
printf -- '- total: %s\n' "${TOTAL_ITEMS}"
printf -- '- done: %s\n' "${DONE_ITEMS}"
printf -- '- todo: %s\n' "${TODO_ITEMS}"
printf -- '- ratio: %s%%\n' "${RATIO}"
printf -- '- command: rg -n "^- \\[(x| )\\]" %s\n' "${ACCEPTANCE_FILE}"

printf '\n## Remaining Items\n'
if [[ "${TODO_ITEMS}" == "0" ]]; then
  echo "- none"
else
  nl -ba "${ACCEPTANCE_FILE}" | rg '\- \[ \]' | sed "s#^#- ${ACCEPTANCE_FILE}:#"
fi
