#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
ROLLBACK_REF="${1:-${GOYAIS_ROLLBACK_REF:-v0.4.0}}"

if ! git -C "$ROOT_DIR" rev-parse --verify "${ROLLBACK_REF}^{commit}" >/dev/null 2>&1; then
  echo "[rollback] unknown rollback ref: ${ROLLBACK_REF}" >&2
  echo "[rollback] pass an explicit ref, e.g.:" >&2
  echo "  scripts/release/rollback/restore-pre-t007.sh <stable-tag-or-sha>" >&2
  exit 1
fi

if ! git -C "$ROOT_DIR" diff --quiet || ! git -C "$ROOT_DIR" diff --cached --quiet; then
  echo "[rollback] working tree is dirty; commit or stash before rollback restore" >&2
  exit 1
fi

RESTORE_PATHS=(
  "services/worker"
  ".github/workflows/ci.yml"
  ".github/workflows/release.yml"
  "Makefile"
  "README.md"
  "README.zh-CN.md"
  "apps/desktop/README.md"
  "apps/desktop/src-tauri/src/sidecar.rs"
  "apps/desktop/src-tauri/tauri.conf.json"
  "docs/release-checklist.md"
  "package.json"
  "scripts/dev/print_commands.sh"
  "scripts/release/build-worker-sidecar.sh"
  "scripts/release/ensure-local-sidecars.sh"
  "scripts/smoke/health_check.sh"
)

RESTORE_CANDIDATES=()
for path in "${RESTORE_PATHS[@]}"; do
  if git -C "$ROOT_DIR" cat-file -e "${ROLLBACK_REF}:${path}" >/dev/null 2>&1; then
    RESTORE_CANDIDATES+=("$path")
  else
    echo "[rollback] skip missing path in ${ROLLBACK_REF}: ${path}"
  fi
done

if [[ ${#RESTORE_CANDIDATES[@]} -eq 0 ]]; then
  echo "[rollback] no restorable paths found in ${ROLLBACK_REF}" >&2
  exit 1
fi

echo "[rollback] restoring pre-T007 assets from ${ROLLBACK_REF}"
git -C "$ROOT_DIR" checkout "$ROLLBACK_REF" -- "${RESTORE_CANDIDATES[@]}"
echo "[rollback] restore complete"
echo "[rollback] next: scripts/release/rollback/verify-rollback.sh"
