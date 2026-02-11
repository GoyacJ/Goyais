#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"

status=0
seen_branches=""
current_path=""
current_branch=""
current_detached="false"

flush_entry() {
  if [[ -z "${current_path}" ]]; then
    return
  fi

  if [[ "${current_detached}" == "true" ]]; then
    echo "[worktree_audit] detached worktree: ${current_path}" >&2
    status=1
    return
  fi

  if [[ -z "${current_branch}" ]]; then
    echo "[worktree_audit] missing branch binding: ${current_path}" >&2
    status=1
    return
  fi

  if printf '%s\n' "${seen_branches}" | rg -Fx -- "${current_branch}" >/dev/null 2>&1; then
    echo "[worktree_audit] duplicate branch in multiple worktrees: ${current_branch}" >&2
    status=1
    return
  fi
  seen_branches="${seen_branches}
${current_branch}"

  echo "[worktree_audit] ok ${current_path} -> ${current_branch}"
}

while IFS= read -r line; do
  if [[ -z "${line}" ]]; then
    flush_entry
    current_path=""
    current_branch=""
    current_detached="false"
    continue
  fi

  if [[ "${line}" == worktree\ * ]]; then
    current_path="${line#worktree }"
  elif [[ "${line}" == branch\ refs/heads/* ]]; then
    current_branch="${line#branch refs/heads/}"
  elif [[ "${line}" == detached ]]; then
    current_detached="true"
  fi
done < <(git worktree list --porcelain; echo)

if [[ ${status} -ne 0 ]]; then
  echo "[worktree_audit] failed" >&2
  exit ${status}
fi

echo "[worktree_audit] passed"
