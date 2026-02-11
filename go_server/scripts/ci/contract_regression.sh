#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
GO_SERVER_DIR="${REPO_ROOT}/go_server"
VUE_WEB_DIR="${REPO_ROOT}/vue_web"
cd "${REPO_ROOT}"

echo "[contract_regression] start"

echo "[contract_regression] worktree audit"
bash go_server/scripts/git/worktree_audit.sh

echo "[contract_regression] precommit guard"
bash go_server/scripts/git/precommit_guard.sh

echo "[contract_regression] path migration audit"
bash go_server/scripts/ci/path_migration_audit.sh

echo "[contract_regression] go test"
(cd "${GO_SERVER_DIR}" && go test ./...)

echo "[contract_regression] web typecheck"
pnpm -C "${VUE_WEB_DIR}" typecheck

echo "[contract_regression] web tests"
pnpm -C "${VUE_WEB_DIR}" test:run

echo "[contract_regression] build"
make -C "${GO_SERVER_DIR}" build

echo "[contract_regression] single binary verify"
GOYAIS_VERIFY_BASE_URL="${GOYAIS_VERIFY_BASE_URL:-http://127.0.0.1:18080}" \
GOYAIS_START_CMD="${GOYAIS_START_CMD:-GOYAIS_SERVER_ADDR=:18080 ./go_server/build/goyais}" \
  bash .agents/skills/goyais-single-binary-acceptance/scripts/verify_single_binary.sh

echo "[contract_regression] passed"
