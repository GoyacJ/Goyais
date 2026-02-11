#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"

TARGETS=(
  ".github"
  ".agents"
  "docs"
  "go_server/docs"
  "go_server/scripts"
  "vue_web/docs"
  "README.md"
  "go_server/README.md"
  "go_server/AGENTS.md"
  "vue_web/AGENTS.md"
  "java_server/AGENTS.md"
  "python_server/AGENTS.md"
  "flutter_mobile/AGENTS.md"
  "AGENTS.md"
)

PATTERN="pnpm -C web|/Users/goya/Repo/Git/Goyais/web\\b|(^|/)web/|(^|[[:space:]\\\"'(])scripts/ci/contract_regression\\.sh\\b|(^|[[:space:]\\\"'(])scripts/git/precommit_guard\\.sh\\b|(^|[[:space:]\\\"'(])scripts/git/worktree_audit\\.sh\\b|(^|[[:space:]\\\"'(])docs/api/openapi\\.yaml\\b|(^|[[:space:]\\\"'(])docs/arch/(overview|data-model|state-machines)\\.md\\b|(^|[[:space:]\\\"'(])docs/acceptance\\.md\\b|(^|[[:space:]\\\"'(])docs/spec/v0\\.1\\.md\\b|\\.agents/skills/goyais-single-binary-acceptance/|\\.agents/skills/goyais-web-asset-governance/|\\bcodex/"

echo "[path_migration_audit] scanning legacy path references..."
if rg -n "${PATTERN}" "${TARGETS[@]}" --glob '!go_server/scripts/ci/path_migration_audit.sh'; then
  echo "[path_migration_audit] legacy references detected" >&2
  exit 1
fi

echo "[path_migration_audit] scanning hardcoded absolute repo paths under .agents..."
if rg -n '/Users/goya/Repo/Git/Goyais' .agents --glob '!go_server/scripts/ci/path_migration_audit.sh'; then
  echo "[path_migration_audit] hardcoded absolute paths detected under .agents" >&2
  exit 1
fi

echo "[path_migration_audit] no legacy path references found"
