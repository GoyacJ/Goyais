#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 Goya
# Author: Goya
# Created: 2026-02-11
# Version: v1.0.0
# Description: Create a thread branch and isolated worktree under the repository.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

resolve_default_repo_root() {
  local detected repo_name parent canonical
  detected="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
  repo_name="$(basename "${detected}")"
  if [[ "${repo_name}" == Goyais-wt-* ]]; then
    parent="$(dirname "${detected}")"
    canonical="${parent}/Goyais"
    if git -C "${canonical}" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
      printf '%s\n' "${canonical}"
      return 0
    fi
  fi
  printf '%s\n' "${detected}"
}

DEFAULT_REPO_ROOT="$(resolve_default_repo_root)"

TOPIC=""
THREAD_ID=""
REPO_ROOT="${DEFAULT_REPO_ROOT}"
WORKTREE_ROOT=""
WORKTREE_ROOT_SET=0
SKIP_SYNC=0
DRY_RUN=0

usage() {
  cat <<'EOF'
Usage:
  create_worktree.sh --topic <topic> [options]

Options:
  --topic <value>          Thread topic (required)
  --thread-id <value>      Thread id; default threadYYYYMMDD-HHMMSS
  --repo-root <path>       Repo root path; default auto-detected
  --worktree-root <path>   Worktree root dir; default <repo-root>/.worktrees
  --skip-sync              Skip master sync (fetch/pull)
  --dry-run                Print commands without executing
  -h, --help               Show help
EOF
}

log() {
  printf '[goyais-worktree] %s\n' "$*"
}

die() {
  printf '[goyais-worktree] ERROR: %s\n' "$*" >&2
  exit 1
}

sanitize_segment() {
  local raw="$1"
  local normalized
  normalized="$(printf '%s' "${raw}" \
    | tr '[:upper:]' '[:lower:]' \
    | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//; s/-+/-/g')"
  printf '%s' "${normalized}"
}

run_cmd() {
  if [[ "${DRY_RUN}" -eq 1 ]]; then
    printf '[dry-run] %s\n' "$*"
    return 0
  fi
  "$@"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --topic)
      [[ $# -ge 2 ]] || die "missing value for --topic"
      TOPIC="$2"
      shift 2
      ;;
    --thread-id)
      [[ $# -ge 2 ]] || die "missing value for --thread-id"
      THREAD_ID="$2"
      shift 2
      ;;
    --repo-root)
      [[ $# -ge 2 ]] || die "missing value for --repo-root"
      REPO_ROOT="$2"
      shift 2
      ;;
    --worktree-root)
      [[ $# -ge 2 ]] || die "missing value for --worktree-root"
      WORKTREE_ROOT="$2"
      WORKTREE_ROOT_SET=1
      shift 2
      ;;
    --skip-sync)
      SKIP_SYNC=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown option: $1"
      ;;
  esac
done

[[ -n "${TOPIC}" ]] || die "--topic is required"
TOPIC_SLUG="$(sanitize_segment "${TOPIC}")"
[[ -n "${TOPIC_SLUG}" ]] || die "invalid --topic after normalization"

if [[ -z "${THREAD_ID}" ]]; then
  THREAD_ID="thread$(date +%Y%m%d-%H%M%S)"
fi
THREAD_ID_SLUG="$(sanitize_segment "${THREAD_ID}")"
[[ -n "${THREAD_ID_SLUG}" ]] || die "invalid --thread-id after normalization"

[[ -d "${REPO_ROOT}" ]] || die "repo root does not exist: ${REPO_ROOT}"
git -C "${REPO_ROOT}" rev-parse --is-inside-work-tree >/dev/null 2>&1 || die "not a git repo: ${REPO_ROOT}"
git -C "${REPO_ROOT}" show-ref --verify --quiet refs/heads/master || die "master branch not found in ${REPO_ROOT}"

if [[ "${WORKTREE_ROOT_SET}" -eq 0 ]]; then
  WORKTREE_ROOT="${REPO_ROOT}/.worktrees"
fi
run_cmd mkdir -p "${WORKTREE_ROOT}"

REPO_NAME="$(basename "${REPO_ROOT}")"
BRANCH="goya/${THREAD_ID_SLUG}-${TOPIC_SLUG}"
WORKTREE_PATH="${WORKTREE_ROOT}/${REPO_NAME}-wt-${THREAD_ID_SLUG}"

if git -C "${REPO_ROOT}" show-ref --verify --quiet "refs/heads/${BRANCH}"; then
  die "branch already exists: ${BRANCH}"
fi

if [[ -e "${WORKTREE_PATH}" ]]; then
  die "worktree path already exists: ${WORKTREE_PATH}"
fi

if git -C "${REPO_ROOT}" worktree list --porcelain | awk '$1=="worktree" {print $2}' | grep -Fxq "${WORKTREE_PATH}"; then
  die "worktree already registered: ${WORKTREE_PATH}"
fi

if [[ "${SKIP_SYNC}" -eq 0 ]]; then
  run_cmd git -C "${REPO_ROOT}" switch master
  run_cmd git -C "${REPO_ROOT}" fetch origin --prune
  run_cmd git -C "${REPO_ROOT}" pull --ff-only
else
  log "skip sync: master fetch/pull skipped"
fi

run_cmd git -C "${REPO_ROOT}" worktree add "${WORKTREE_PATH}" -b "${BRANCH}" master

printf '\n'
log "worktree ready"
printf 'thread_id=%s\n' "${THREAD_ID_SLUG}"
printf 'topic=%s\n' "${TOPIC_SLUG}"
printf 'branch=%s\n' "${BRANCH}"
printf 'worktree_path=%s\n' "${WORKTREE_PATH}"
printf 'repo_root=%s\n' "${REPO_ROOT}"
if [[ "${DRY_RUN}" -eq 1 ]]; then
  printf 'dry_run=1\n'
fi
printf '\n'
printf 'next_steps:\n'
printf '  1) cd %s\n' "${WORKTREE_PATH}"
printf '  2) start implementation in isolated thread\n'
