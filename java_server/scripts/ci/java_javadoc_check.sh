#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"

collect_files() {
  rg --files java_server \
    -g '*.java' \
    -g '!**/target/**' \
    -g '!**/build/**'
}

line_trimmed() {
  local file="$1"
  local lineno="$2"
  sed -n "${lineno}p" "${file}" | sed -E 's/^[[:space:]]+//;s/[[:space:]]+$//'
}

has_javadoc_before() {
  local file="$1"
  local line="$2"
  local cursor=$(( line - 1 ))
  local text

  # Skip decorators and empty lines immediately above declaration.
  while (( cursor > 0 )); do
    text="$(line_trimmed "${file}" "${cursor}")"
    if [[ -z "${text}" || "${text}" =~ ^@ ]]; then
      cursor=$(( cursor - 1 ))
      continue
    fi
    break
  done

  if (( cursor <= 0 )); then
    return 1
  fi

  text="$(line_trimmed "${file}" "${cursor}")"
  if [[ "${text}" != *"*/"* ]]; then
    return 1
  fi

  while (( cursor > 0 )); do
    text="$(line_trimmed "${file}" "${cursor}")"
    if [[ "${text}" == "/**"* ]]; then
      return 0
    fi
    if [[ "${text}" == "/*"* && "${text}" != "/**"* ]]; then
      return 1
    fi
    cursor=$(( cursor - 1 ))
  done

  return 1
}

main() {
  local failed=0
  local total=0
  local line
  local file
  local lineno
  local text

  while IFS= read -r file; do
    [[ -n "${file}" ]] || continue
    total=$((total + 1))

    while IFS=: read -r lineno text; do
      [[ -n "${lineno}" ]] || continue
      if ! has_javadoc_before "${file}" "${lineno}"; then
        echo "[java_javadoc_check] missing_type_javadoc: ${file}:${lineno}"
        failed=$((failed + 1))
      fi
    done < <(rg -n '^[[:space:]]*public[[:space:]]+(class|interface|enum|record)\b' "${file}" || true)

    while IFS=: read -r lineno text; do
      [[ -n "${lineno}" ]] || continue
      if [[ "${text}" =~ \b(class|interface|enum|record)\b ]]; then
        continue
      fi
      if ! has_javadoc_before "${file}" "${lineno}"; then
        echo "[java_javadoc_check] missing_method_javadoc: ${file}:${lineno}"
        failed=$((failed + 1))
      fi
    done < <(rg -n '^[[:space:]]*public[[:space:]]+[^=;]*\(' "${file}" || true)
  done < <(collect_files)

  if (( failed > 0 )); then
    echo "[java_javadoc_check] failed=${failed} files=${total}"
    exit 1
  fi

  echo "[java_javadoc_check] passed files=${total}"
}

main "$@"
