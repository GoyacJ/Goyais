#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"

TARGETS=(
  ".github"
  "docs"
  "go_server/docs"
  "go_server/scripts"
  "vue_web/docs"
  "README.md"
  "go_server/README.md"
  "AGENTS.md"
)

PATTERN="pnpm -C web|/Users/goya/Repo/Git/Goyais/web\\b|(^|/)web/|(^|[[:space:]\\\"'(])scripts/ci/contract_regression\\.sh\\b|(^|[[:space:]\\\"'(])scripts/git/precommit_guard\\.sh\\b|(^|[[:space:]\\\"'(])scripts/git/worktree_audit\\.sh\\b|(^|[[:space:]\\\"'(])docs/api/openapi\\.yaml\\b|(^|[[:space:]\\\"'(])docs/arch/(overview|data-model|state-machines)\\.md\\b|(^|[[:space:]\\\"'(])docs/acceptance\\.md\\b|(^|[[:space:]\\\"'(])docs/spec/v0\\.1\\.md\\b"

echo "[path_migration_audit] scanning legacy path references..."
if rg -n "${PATTERN}" "${TARGETS[@]}" --glob '!go_server/scripts/ci/path_migration_audit.sh'; then
  echo "[path_migration_audit] legacy references detected" >&2
  exit 1
fi

echo "[path_migration_audit] no legacy path references found"
