#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
ROLLBACK_REF="${1:-${GOYAIS_ROLLBACK_REF:-v0.4.0}}"

"$ROOT_DIR/scripts/release/rollback/restore-pre-t007.sh" "$ROLLBACK_REF"
"$ROOT_DIR/scripts/release/rollback/verify-rollback.sh"

echo "[rollback] done: pre-T007 stack restored and verified from ${ROLLBACK_REF}"
