#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TARGET_TRIPLE="${1:-}"

if [[ -z "$TARGET_TRIPLE" ]]; then
  echo "usage: $0 <target-triple>" >&2
  exit 1
fi

GOOS=""
GOARCH=""
EXT=""

case "$TARGET_TRIPLE" in
  aarch64-apple-darwin)
    GOOS="darwin"
    GOARCH="arm64"
    ;;
  x86_64-apple-darwin)
    GOOS="darwin"
    GOARCH="amd64"
    ;;
  x86_64-unknown-linux-gnu)
    GOOS="linux"
    GOARCH="amd64"
    ;;
  x86_64-pc-windows-msvc)
    GOOS="windows"
    GOARCH="amd64"
    EXT=".exe"
    ;;
  *)
    echo "unsupported target triple: $TARGET_TRIPLE" >&2
    exit 1
    ;;
esac

OUTPUT_DIR="$ROOT_DIR/apps/desktop/src-tauri/binaries"
OUTPUT_PATH="$OUTPUT_DIR/goyais-hub-$TARGET_TRIPLE$EXT"

mkdir -p "$OUTPUT_DIR"

echo "[hub-sidecar] building for $TARGET_TRIPLE -> $OUTPUT_PATH"
(
  cd "$ROOT_DIR/services/hub"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -o "$OUTPUT_PATH" ./cmd/hub
)

if [[ ! -f "$OUTPUT_PATH" ]]; then
  echo "[hub-sidecar] build output missing: $OUTPUT_PATH" >&2
  exit 1
fi

if [[ "$EXT" != ".exe" ]]; then
  chmod +x "$OUTPUT_PATH"
fi

echo "[hub-sidecar] done"
