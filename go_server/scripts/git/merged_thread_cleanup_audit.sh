#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 Goya
# Author: Goya
# Created: 2026-02-11
# Version: v1.0.0
# Description: Warn on local goya/* branches already merged into master but not cleaned.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"

git show-ref --verify --quiet refs/heads/master || {
  echo "[merged_thread_cleanup_audit] skip: master branch not found" >&2
  exit 0
}

merged_count=0
while IFS= read -r branch; do
  [[ -n "${branch}" ]] || continue
  echo "[merged_thread_cleanup_audit] WARN merged thread branch not cleaned: ${branch}" >&2
  echo "[merged_thread_cleanup_audit] hint: git branch -d ${branch}" >&2
  merged_count=$((merged_count + 1))
done < <(git for-each-ref --merged=master refs/heads/goya --format='%(refname:lstrip=2)')

if [[ "${merged_count}" -eq 0 ]]; then
  echo "[merged_thread_cleanup_audit] ok no merged goya/* local branches pending cleanup"
else
  echo "[merged_thread_cleanup_audit] warnings=${merged_count} (warn-only, non-blocking)"
fi

exit 0
