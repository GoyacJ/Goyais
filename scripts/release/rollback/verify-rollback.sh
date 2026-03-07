#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"

if [[ ! -d "$ROOT_DIR/services/worker" ]]; then
  echo "[rollback-verify] services/worker missing; run restore-pre-t007.sh first" >&2
  exit 1
fi

echo "[rollback-verify] hub contract checks: health/auth/conversation/message"
(
  cd "$ROOT_DIR/services/hub"
  go test -count=1 ./internal/httpapi -run 'TestHealth|TestLoginValidationAndAuthErrors|TestProjectConversationFlowWithCursorPagination'
)

if command -v uv >/dev/null 2>&1; then
  echo "[rollback-verify] worker health/internal token checks"
  (
    cd "$ROOT_DIR/services/worker"
    uv sync
    uv run pytest -q tests/test_health.py tests/test_internal_tokens.py
  )
else
  echo "[rollback-verify] skip worker checks: uv not found in PATH"
fi

echo "[rollback-verify] rollback verification passed"
