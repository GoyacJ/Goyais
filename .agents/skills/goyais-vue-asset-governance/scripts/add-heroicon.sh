#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
DEST_DIR="${REPO_ROOT}/vue_web/src/assets/icons/heroicons/24/outline"
VERSION_TAG="v2.2.0"
BASE_URL="https://raw.githubusercontent.com/tailwindlabs/heroicons/${VERSION_TAG}/src/24/outline"

readonly WHITELIST=(
  home
  squares-2x2
  command-line
  cube
  puzzle-piece
  signal
  cog-6-tooth
  arrow-path
  cloud-arrow-up
  shield-exclamation
  exclamation-triangle
  magnifying-glass
  inbox-stack
)

contains() {
  local target="$1"
  shift
  for item in "$@"; do
    if [[ "$item" == "$target" ]]; then
      return 0
    fi
  done
  return 1
}

mkdir -p "${DEST_DIR}"

if [[ "$#" -eq 0 ]]; then
  set -- "${WHITELIST[@]}"
fi

for icon in "$@"; do
  if ! contains "$icon" "${WHITELIST[@]}"; then
    echo "[add-heroicon] blocked: '${icon}' is not in whitelist" >&2
    exit 1
  fi

  src="${BASE_URL}/${icon}.svg"
  dst="${DEST_DIR}/${icon}.svg"

  echo "[add-heroicon] download ${icon}"
  curl -fsSL "$src" \
    | sed -E 's/stroke="#[0-9A-Fa-f]{3,8}"/stroke="currentColor"/g' \
    | sed -E 's/fill="#[0-9A-Fa-f]{3,8}"/fill="currentColor"/g' \
    | sed -E 's/stroke-width="1\.5"/stroke-width="1.75"/g' \
    > "$dst"

done

echo "[add-heroicon] done: ${DEST_DIR}"
