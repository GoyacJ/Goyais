#!/usr/bin/env bash
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

THREAD_BRANCH=""
REPO_ROOT="${DEFAULT_REPO_ROOT}"
TEST_CMD="go test ./..."
PUSH=0
CLEANUP=1
DRY_RUN=0

usage() {
  cat <<'EOF'
Usage:
  merge_thread.sh --thread-branch <branch> [options]

Options:
  --thread-branch <value>  Thread branch name, e.g. goya/thread39-ui-fix (required)
  --repo-root <path>       Repo root path; default auto-detected
  --test-cmd <value>       Command run after merge; default "go test ./..."
  --push                   Push master to origin after merge
  --push=<bool>            Explicit push flag: true/false
  --cleanup                Enable cleanup (default)
  --cleanup=<bool>         Explicit cleanup flag: true/false
  --no-cleanup             Disable cleanup
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

parse_bool() {
  local raw="${1:-}"
  case "$(printf '%s' "${raw}" | tr '[:upper:]' '[:lower:]')" in
    1|true|yes|on)
      printf '1'
      ;;
    0|false|no|off)
      printf '0'
      ;;
    *)
      die "invalid boolean value: ${raw}"
      ;;
  esac
}

run_cmd() {
  if [[ "${DRY_RUN}" -eq 1 ]]; then
    printf '[dry-run] %s\n' "$*"
    return 0
  fi
  "$@"
}

run_test_cmd() {
  if [[ "${DRY_RUN}" -eq 1 ]]; then
    printf '[dry-run] (cd %s && %s)\n' "${REPO_ROOT}" "${TEST_CMD}"
    return 0
  fi
  (cd "${REPO_ROOT}" && sh -c "${TEST_CMD}")
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --thread-branch)
      [[ $# -ge 2 ]] || die "missing value for --thread-branch"
      THREAD_BRANCH="$2"
      shift 2
      ;;
    --repo-root)
      [[ $# -ge 2 ]] || die "missing value for --repo-root"
      REPO_ROOT="$2"
      shift 2
      ;;
    --test-cmd)
      [[ $# -ge 2 ]] || die "missing value for --test-cmd"
      TEST_CMD="$2"
      shift 2
      ;;
    --push)
      PUSH=1
      shift
      ;;
    --push=*)
      PUSH="$(parse_bool "${1#*=}")"
      shift
      ;;
    --cleanup)
      CLEANUP=1
      shift
      ;;
    --cleanup=*)
      CLEANUP="$(parse_bool "${1#*=}")"
      shift
      ;;
    --no-cleanup)
      CLEANUP=0
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

[[ -n "${THREAD_BRANCH}" ]] || die "--thread-branch is required"
[[ -d "${REPO_ROOT}" ]] || die "repo root does not exist: ${REPO_ROOT}"
git -C "${REPO_ROOT}" rev-parse --is-inside-work-tree >/dev/null 2>&1 || die "not a git repo: ${REPO_ROOT}"

if [[ -n "$(git -C "${REPO_ROOT}" status --porcelain)" ]]; then
  die "repo root must be clean before merge: ${REPO_ROOT}"
fi

git -C "${REPO_ROOT}" show-ref --verify --quiet refs/heads/master || die "master branch not found"
git -C "${REPO_ROOT}" show-ref --verify --quiet "refs/heads/${THREAD_BRANCH}" || die "thread branch not found: ${THREAD_BRANCH}"
[[ "${THREAD_BRANCH}" != "master" ]] || die "--thread-branch cannot be master"

run_cmd git -C "${REPO_ROOT}" switch master
run_cmd git -C "${REPO_ROOT}" fetch origin --prune
run_cmd git -C "${REPO_ROOT}" pull --ff-only
run_cmd git -C "${REPO_ROOT}" merge --no-ff "${THREAD_BRANCH}"
run_test_cmd

if [[ "${PUSH}" -eq 1 ]]; then
  run_cmd git -C "${REPO_ROOT}" push origin master
else
  log "skip push: use --push to publish master"
fi

WORKTREE_PATH=""
if [[ "${CLEANUP}" -eq 1 ]]; then
  WORKTREE_PATH="$(git -C "${REPO_ROOT}" worktree list --porcelain | awk -v target="refs/heads/${THREAD_BRANCH}" '
    $1=="worktree" {path=$2}
    $1=="branch" && $2==target {print path}
  ' | head -n 1)"

  if [[ -n "${WORKTREE_PATH}" && "${WORKTREE_PATH}" != "${REPO_ROOT}" ]]; then
    run_cmd git -C "${REPO_ROOT}" worktree remove "${WORKTREE_PATH}"
  else
    log "cleanup: no removable worktree found for ${THREAD_BRANCH}"
  fi

  run_cmd git -C "${REPO_ROOT}" branch -d "${THREAD_BRANCH}"
  run_cmd git -C "${REPO_ROOT}" worktree prune
else
  log "skip cleanup: disabled by flag"
fi

MERGE_COMMIT="dry-run"
if [[ "${DRY_RUN}" -eq 0 ]]; then
  MERGE_COMMIT="$(git -C "${REPO_ROOT}" rev-parse --short HEAD)"
fi

printf '\n'
log "merge flow complete"
printf 'thread_branch=%s\n' "${THREAD_BRANCH}"
printf 'merge_commit=%s\n' "${MERGE_COMMIT}"
printf 'push=%s\n' "${PUSH}"
printf 'cleanup=%s\n' "${CLEANUP}"
if [[ -n "${WORKTREE_PATH}" ]]; then
  printf 'cleanup_worktree=%s\n' "${WORKTREE_PATH}"
fi
