#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"

echo "[precommit_guard] staged files"
git diff --cached --name-only

BLOCKED='^(data/objects/|.*\.db$|go_server/build/|go_server/internal/access/webstatic/dist/assets/|go_server/internal/access/webstatic/dist/index\.html$|vue_web/dist/|vue_web/node_modules/)'
if git diff --cached --name-only | rg "${BLOCKED}"; then
  echo "[precommit_guard] blocked paths detected in staged files" >&2
  exit 1
fi

echo "[precommit_guard] passed"
