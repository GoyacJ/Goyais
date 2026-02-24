#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TARGET_TRIPLE="${1:-}"

if [[ -z "$TARGET_TRIPLE" ]]; then
  echo "usage: $0 <target-triple>" >&2
  exit 1
fi

EXT=""
case "$TARGET_TRIPLE" in
  aarch64-apple-darwin|x86_64-apple-darwin|x86_64-unknown-linux-gnu)
    EXT=""
    ;;
  x86_64-pc-windows-msvc)
    EXT=".exe"
    ;;
  *)
    echo "unsupported target triple: $TARGET_TRIPLE" >&2
    exit 1
    ;;
esac

WORKER_DIR="$ROOT_DIR/services/worker"
OUTPUT_DIR="$ROOT_DIR/apps/desktop/src-tauri/binaries"
OUTPUT_PATH="$OUTPUT_DIR/goyais-worker-$TARGET_TRIPLE$EXT"

echo "[worker-sidecar] building for $TARGET_TRIPLE -> $OUTPUT_PATH"
(
  cd "$WORKER_DIR"
  rm -rf dist build goyais-worker.spec
  uv sync
  uv run --with pyinstaller pyinstaller --noconfirm --clean --onefile --name goyais-worker --paths . app/main.py
)

SOURCE_PATH="$WORKER_DIR/dist/goyais-worker$EXT"
if [[ ! -f "$SOURCE_PATH" ]]; then
  echo "[worker-sidecar] build output missing: $SOURCE_PATH" >&2
  exit 1
fi

mkdir -p "$OUTPUT_DIR"
cp "$SOURCE_PATH" "$OUTPUT_PATH"

if [[ "$EXT" != ".exe" ]]; then
  chmod +x "$OUTPUT_PATH"
fi

echo "[worker-sidecar] done"
