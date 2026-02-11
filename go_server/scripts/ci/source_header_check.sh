#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"

readonly SPDX_PATTERN='SPDX-License-Identifier: Apache-2.0'
readonly COPYRIGHT_PATTERN='Copyright \(c\) 2026 Goya'
readonly AUTHOR_PATTERN='Author: Goya'
readonly CREATED_PATTERN='Created: 2026-02-11'
readonly VERSION_PATTERN='Version: v1.0.0'
readonly DESCRIPTION_PATTERN='Description:'

collect_files() {
  rg --files \
    -g '*.go' \
    -g '*.ts' \
    -g '*.vue' \
    -g '*.js' \
    -g '*.py' \
    -g '*.java' \
    -g '*.dart' \
    -g '!**/dist/**' \
    -g '!**/build/**' \
    -g '!**/node_modules/**' \
    -g '!**/.git/**' \
    -g '!**/.idea/**' \
    -g '!**/.worktrees/**'
}

first_line() {
  local pattern="$1"
  local file="$2"
  local line
  line="$(rg -n -m1 "${pattern}" "${file}" 2>/dev/null | head -n1 | cut -d: -f1)"
  echo "${line}"
}

main() {
  local total=0
  local failed=0

  while IFS= read -r file; do
    [[ -n "${file}" ]] || continue
    total=$((total + 1))

    local spdx_line copyright_line author_line created_line version_line description_line
    spdx_line="$(first_line "${SPDX_PATTERN}" "${file}")"
    copyright_line="$(first_line "${COPYRIGHT_PATTERN}" "${file}")"
    author_line="$(first_line "${AUTHOR_PATTERN}" "${file}")"
    created_line="$(first_line "${CREATED_PATTERN}" "${file}")"
    version_line="$(first_line "${VERSION_PATTERN}" "${file}")"
    description_line="$(first_line "${DESCRIPTION_PATTERN}" "${file}")"

    if [[ -z "${spdx_line}" || -z "${copyright_line}" || -z "${author_line}" || -z "${created_line}" || -z "${version_line}" || -z "${description_line}" ]]; then
      echo "[source_header_check] missing_fields: ${file}"
      failed=$((failed + 1))
      continue
    fi

    if ! (( spdx_line < copyright_line && copyright_line < author_line && author_line < created_line && created_line < version_line && version_line < description_line )); then
      echo "[source_header_check] bad_order: ${file}"
      failed=$((failed + 1))
      continue
    fi
  done < <(collect_files)

  if (( failed > 0 )); then
    echo "[source_header_check] failed=${failed} total=${total}"
    exit 1
  fi

  echo "[source_header_check] passed total=${total}"
}

main "$@"
