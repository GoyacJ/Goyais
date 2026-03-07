#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TARGET_TRIPLE="${TARGET_TRIPLE:-}"
FORCE_REBUILD="${GOYAIS_FORCE_SIDECAR_REBUILD:-0}"

if [[ -z "$TARGET_TRIPLE" ]]; then
  TARGET_TRIPLE="$(rustc -vV | awk '/^host:/ {print $2}')"
fi

if [[ -z "$TARGET_TRIPLE" ]]; then
  echo "[sidecar-prepare] failed to resolve target triple" >&2
  exit 1
fi

EXT=""
case "$TARGET_TRIPLE" in
  x86_64-pc-windows-msvc)
    EXT=".exe"
    ;;
  aarch64-apple-darwin|x86_64-apple-darwin|x86_64-unknown-linux-gnu)
    EXT=""
    ;;
  *)
    echo "[sidecar-prepare] unsupported target triple: $TARGET_TRIPLE" >&2
    exit 1
    ;;
esac

BIN_DIR="$ROOT_DIR/apps/desktop/src-tauri/binaries"
HUB_BIN="$BIN_DIR/goyais-hub-$TARGET_TRIPLE$EXT"

if [[ "$FORCE_REBUILD" == "1" ]]; then
  echo "[sidecar-prepare] force rebuild enabled via GOYAIS_FORCE_SIDECAR_REBUILD=1"
  rm -f "$HUB_BIN"
fi

if [[ ! -f "$HUB_BIN" ]]; then
  echo "[sidecar-prepare] hub sidecar missing, building..."
  "$ROOT_DIR/scripts/release/build-hub-sidecar.sh" "$TARGET_TRIPLE"
else
  echo "[sidecar-prepare] hub sidecar exists: $HUB_BIN"
fi

if [[ ! -f "$HUB_BIN" ]]; then
  echo "[sidecar-prepare] sidecar build incomplete for $TARGET_TRIPLE" >&2
  exit 1
fi

echo "[sidecar-prepare] hub sidecar ready for $TARGET_TRIPLE"
