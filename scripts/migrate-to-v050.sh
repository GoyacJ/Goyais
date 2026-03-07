#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DB_PATH="${HUB_DB_PATH:-}"

if [[ -z "${DB_PATH}" ]]; then
  DB_PATH="${HOME}/.config/goyais/hub.sqlite3"
fi

echo "Running v0.5.0 stage migrations against ${DB_PATH}"
(
  cd "${ROOT_DIR}/services/hub"
  HUB_DB_PATH="${DB_PATH}" go run ./cmd/hub migrate
)
