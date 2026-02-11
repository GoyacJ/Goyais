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
readonly JAVA_PARAGRAPH_PATTERN='<p>.*</p>'
readonly JAVA_AUTHOR_PATTERN='@author[[:space:]]+Goya'
readonly JAVA_SINCE_PATTERN='@since[[:space:]]+[0-9]{4}-[0-9]{2}-[0-9]{2}[[:space:]]+[0-9]{2}:[0-9]{2}:[0-9]{2}'

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

first_java_header_bounds() {
  local file="$1"
  awk '
    NR <= 120 {
      if (start == 0 && $0 ~ /^[[:space:]]*\/\*\*/) {
        start = NR
      }
      if (start > 0 && $0 ~ /\*\//) {
        printf "%d:%d\n", start, NR
        exit
      }
    }
  ' "${file}"
}

relative_line() {
  local pattern="$1"
  local text="$2"
  local line
  line="$(printf '%s\n' "${text}" | rg -n -m1 "${pattern}" | head -n1 | cut -d: -f1 || true)"
  echo "${line}"
}

check_non_java_header() {
  local file="$1"

  local spdx_line copyright_line author_line created_line version_line description_line
  spdx_line="$(first_line "${SPDX_PATTERN}" "${file}")"
  copyright_line="$(first_line "${COPYRIGHT_PATTERN}" "${file}")"
  author_line="$(first_line "${AUTHOR_PATTERN}" "${file}")"
  created_line="$(first_line "${CREATED_PATTERN}" "${file}")"
  version_line="$(first_line "${VERSION_PATTERN}" "${file}")"
  description_line="$(first_line "${DESCRIPTION_PATTERN}" "${file}")"

  if [[ -z "${spdx_line}" || -z "${copyright_line}" || -z "${author_line}" || -z "${created_line}" || -z "${version_line}" || -z "${description_line}" ]]; then
    echo "[source_header_check] missing_fields: ${file}"
    return 1
  fi

  if ! (( spdx_line < copyright_line && copyright_line < author_line && author_line < created_line && created_line < version_line && version_line < description_line )); then
    echo "[source_header_check] bad_order: ${file}"
    return 1
  fi

  return 0
}

check_java_header() {
  local file="$1"

  local bounds start_line end_line
  bounds="$(first_java_header_bounds "${file}")"
  if [[ -z "${bounds}" ]]; then
    echo "[source_header_check] missing_java_header_block: ${file}"
    return 1
  fi
  start_line="${bounds%%:*}"
  end_line="${bounds##*:}"

  local header_block
  header_block="$(sed -n "${start_line},${end_line}p" "${file}")"

  local spdx_line paragraph_line author_line since_line
  spdx_line="$(relative_line "${SPDX_PATTERN}" "${header_block}")"
  paragraph_line="$(relative_line "${JAVA_PARAGRAPH_PATTERN}" "${header_block}")"
  author_line="$(relative_line "${JAVA_AUTHOR_PATTERN}" "${header_block}")"
  since_line="$(relative_line "${JAVA_SINCE_PATTERN}" "${header_block}")"

  if [[ -z "${spdx_line}" || -z "${paragraph_line}" || -z "${author_line}" || -z "${since_line}" ]]; then
    echo "[source_header_check] missing_java_fields: ${file}"
    return 1
  fi

  if ! (( spdx_line < paragraph_line && paragraph_line < author_line && author_line < since_line )); then
    echo "[source_header_check] bad_java_order: ${file}"
    return 1
  fi

  return 0
}

main() {
  local total=0
  local failed=0

  while IFS= read -r file; do
    [[ -n "${file}" ]] || continue
    total=$((total + 1))

    if [[ "${file}" == *.java ]]; then
      if ! check_java_header "${file}"; then
        failed=$((failed + 1))
      fi
      continue
    fi
    if ! check_non_java_header "${file}"; then
      failed=$((failed + 1))
    fi
  done < <(collect_files)

  if (( failed > 0 )); then
    echo "[source_header_check] failed=${failed} total=${total}"
    exit 1
  fi

  echo "[source_header_check] passed total=${total}"
}

main "$@"
