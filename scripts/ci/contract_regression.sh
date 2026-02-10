#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${REPO_ROOT}"

echo "[contract_regression] start"

echo "[contract_regression] worktree audit"
bash scripts/git/worktree_audit.sh

echo "[contract_regression] precommit guard"
bash scripts/git/precommit_guard.sh

echo "[contract_regression] go test"
go test ./...

echo "[contract_regression] web typecheck"
pnpm -C web typecheck

echo "[contract_regression] web tests"
pnpm -C web test:run

echo "[contract_regression] build"
make build

echo "[contract_regression] single binary verify"
GOYAIS_VERIFY_BASE_URL="${GOYAIS_VERIFY_BASE_URL:-http://127.0.0.1:18080}" \
GOYAIS_START_CMD="${GOYAIS_START_CMD:-GOYAIS_SERVER_ADDR=:18080 ./build/goyais}" \
  bash .agents/skills/goyais-single-binary-acceptance/scripts/verify_single_binary.sh

echo "[contract_regression] passed"
