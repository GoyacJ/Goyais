#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"

readonly SPDX_LINE="SPDX-License-Identifier: Apache-2.0"
readonly COPYRIGHT_LINE="Copyright (c) 2026 Goya"
readonly AUTHOR_LINE="Author: Goya"
readonly CREATED_LINE="Created: 2026-02-11"
readonly VERSION_LINE="Version: v1.0.0"
readonly DESCRIPTION_LINE="Description: Goyais source file."
readonly JAVA_AUTHOR_LINE="@author Goya"

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

has_spdx() {
  local file="$1"
  rg -n -m1 "${SPDX_LINE}" "${file}" >/dev/null 2>&1
}

prepend_line_comment_header() {
  local file="$1"
  local tmp
  tmp="$(mktemp)"
  {
    echo "// ${SPDX_LINE}"
    echo "// ${COPYRIGHT_LINE}"
    echo "// ${AUTHOR_LINE}"
    echo "// ${CREATED_LINE}"
    echo "// ${VERSION_LINE}"
    echo "// ${DESCRIPTION_LINE}"
    echo
    cat "${file}"
  } >"${tmp}"
  mv "${tmp}" "${file}"
}

prepend_hash_comment_header() {
  local file="$1"
  local tmp
  tmp="$(mktemp)"
  {
    echo "# ${SPDX_LINE}"
    echo "# ${COPYRIGHT_LINE}"
    echo "# ${AUTHOR_LINE}"
    echo "# ${CREATED_LINE}"
    echo "# ${VERSION_LINE}"
    echo "# ${DESCRIPTION_LINE}"
    echo
    cat "${file}"
  } >"${tmp}"
  mv "${tmp}" "${file}"
}

prepend_block_comment_header() {
  local file="$1"
  local tmp
  tmp="$(mktemp)"
  {
    echo "/**"
    echo " * ${SPDX_LINE}"
    echo " * ${COPYRIGHT_LINE}"
    echo " * ${AUTHOR_LINE}"
    echo " * ${CREATED_LINE}"
    echo " * ${VERSION_LINE}"
    echo " * ${DESCRIPTION_LINE}"
    echo " */"
    echo
    cat "${file}"
  } >"${tmp}"
  mv "${tmp}" "${file}"
}

prepend_java_comment_header() {
  local file="$1"
  local since_now="$2"
  local tmp
  tmp="$(mktemp)"
  {
    echo "/**"
    echo " * ${SPDX_LINE}"
    echo " * <p>Goyais source file.</p>"
    echo " * ${JAVA_AUTHOR_LINE}"
    echo " * @since ${since_now}"
    echo " */"
    echo
    cat "${file}"
  } >"${tmp}"
  mv "${tmp}" "${file}"
}

prepend_vue_html_comment_header() {
  local file="$1"
  local tmp
  tmp="$(mktemp)"
  {
    echo "<!--"
    echo "${SPDX_LINE}"
    echo "${COPYRIGHT_LINE}"
    echo "${AUTHOR_LINE}"
    echo "${CREATED_LINE}"
    echo "${VERSION_LINE}"
    echo "${DESCRIPTION_LINE}"
    echo "-->"
    echo
    cat "${file}"
  } >"${tmp}"
  mv "${tmp}" "${file}"
}

insert_vue_script_header() {
  local file="$1"
  local tmp
  tmp="$(mktemp)"

  awk \
    -v spdx="${SPDX_LINE}" \
    -v copyright="${COPYRIGHT_LINE}" \
    -v author="${AUTHOR_LINE}" \
    -v created="${CREATED_LINE}" \
    -v version="${VERSION_LINE}" \
    -v description="${DESCRIPTION_LINE}" '
    BEGIN { inserted = 0 }
    {
      print $0
      if (!inserted && $0 ~ /^[[:space:]]*<script([[:space:]>]|$)/) {
        print "/**"
        print " * " spdx
        print " * " copyright
        print " * " author
        print " * " created
        print " * " version
        print " * " description
        print " */"
        inserted = 1
      }
    }
    END {
      if (!inserted) {
        exit 2
      }
    }
  ' "${file}" >"${tmp}"

  mv "${tmp}" "${file}"
}

main() {
  local total=0
  local updated=0
  local since_now
  since_now="$(date '+%Y-%m-%d %H:%M:%S')"

  while IFS= read -r file; do
    [[ -n "${file}" ]] || continue
    total=$((total + 1))

    if has_spdx "${file}"; then
      continue
    fi

    case "${file}" in
      *.go|*.dart)
        prepend_line_comment_header "${file}"
        ;;
      *.py)
        prepend_hash_comment_header "${file}"
        ;;
      *.ts|*.js)
        prepend_block_comment_header "${file}"
        ;;
      *.java)
        prepend_java_comment_header "${file}" "${since_now}"
        ;;
      *.vue)
        if rg -n -m1 '^[[:space:]]*<script([[:space:]>]|$)' "${file}" >/dev/null 2>&1; then
          insert_vue_script_header "${file}"
        else
          prepend_vue_html_comment_header "${file}"
        fi
        ;;
      *)
        continue
        ;;
    esac

    updated=$((updated + 1))
  done < <(collect_files)

  echo "[source_header_backfill] total=${total} updated=${updated}"
}

main "$@"
