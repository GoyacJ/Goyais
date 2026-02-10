#!/usr/bin/env bash

set -u
set -o pipefail

BASE_URL="${GOYAIS_VERIFY_BASE_URL:-http://127.0.0.1:8080}"
BINARY_OVERRIDE="${GOYAIS_BINARY_PATH:-}"
START_CMD="${GOYAIS_START_CMD:-}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
cd "$REPO_ROOT"

LOG_FILE="$(mktemp -t verify_single_binary.XXXXXX.log)"
INDEX_FILE="$(mktemp -t verify_single_binary.XXXXXX.html)"
BEFORE_FILE="$(mktemp -t verify_single_binary.XXXXXX.before)"
AFTER_FILE="$(mktemp -t verify_single_binary.XXXXXX.after)"
NEW_FILE="$(mktemp -t verify_single_binary.XXXXXX.new)"

DIST_MOVED="0"
DIST_BACKUP=""
SERVER_PID=""
CHECK_FAILED="0"

log() {
  printf '[verify_single_binary] %s\n' "$*"
}

cleanup() {
  if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi

  if [ "$DIST_MOVED" = "1" ] && [ -n "$DIST_BACKUP" ] && [ -d "$DIST_BACKUP" ]; then
    rm -rf web/dist 2>/dev/null || true
    mv "$DIST_BACKUP" web/dist
  fi

  rm -f "$LOG_FILE" "$INDEX_FILE" "$BEFORE_FILE" "$AFTER_FILE" "$NEW_FILE"
}

trap cleanup EXIT INT TERM

get_mtime() {
  local target="$1"
  if stat -f '%m' "$target" >/dev/null 2>&1; then
    stat -f '%m' "$target"
  else
    stat -c '%Y' "$target"
  fi
}

is_excluded_path() {
  case "$1" in
    ./.git/*|./.agents/*|./docs/*|./node_modules/*|./vendor/*|./testdata/*|./tmp/*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

is_excluded_name() {
  local base
  base="$(basename "$1")"
  case "$base" in
    *test*|*.sh|*.py|*.md|*.txt)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

list_candidates() {
  local dir
  for dir in . ./bin ./build ./dist ./out ./release; do
    [ -d "$dir" ] || continue
    find "$dir" -type f -perm -111 2>/dev/null
  done | while IFS= read -r path; do
    is_excluded_path "$path" && continue
    is_excluded_name "$path" && continue
    printf '%s\n' "$path"
  done | sort -u
}

write_snapshot() {
  local output="$1"
  : > "$output"
  while IFS= read -r path; do
    local mtime
    mtime="$(get_mtime "$path" 2>/dev/null)" || continue
    printf '%s\t%s\n' "$path" "$mtime" >> "$output"
  done < <(list_candidates)
  sort -t $'\t' -k1,1 "$output" -o "$output"
}

choose_latest() {
  local snapshot="$1"
  [ -s "$snapshot" ] || return 1
  sort -t $'\t' -k2,2n -k1,1 "$snapshot" | tail -n 1 | cut -f1
}

http_code() {
  local path="$1"
  curl -sS -o /dev/null -w '%{http_code}' "${BASE_URL}${path}"
}

header_value() {
  local path="$1"
  local header="$2"
  curl -sS -D - -o /dev/null "${BASE_URL}${path}" | tr -d '\r' | awk -F': ' -v key="$header" 'tolower($1)==tolower(key){print $2; exit}'
}

check_equals() {
  local actual="$1"
  local expected="$2"
  local message="$3"
  if [ "$actual" != "$expected" ]; then
    log "FAIL: ${message} (expected=${expected}, actual=${actual})"
    CHECK_FAILED="1"
  fi
}

check_contains() {
  local actual="$1"
  local expected_substr="$2"
  local message="$3"
  if ! printf '%s' "$actual" | grep -Fqi "$expected_substr"; then
    log "FAIL: ${message} (expected contains=${expected_substr}, actual=${actual})"
    CHECK_FAILED="1"
  fi
}

check_js_content_type() {
  local ctype_lc
  ctype_lc="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')"

  case "$ctype_lc" in
    *application/javascript*|*text/javascript*|*application/x-javascript*|*application/ecmascript*|*text/ecmascript*)
      ;;
    *)
      return 1
      ;;
  esac

  case "$ctype_lc" in
    *application/octet-stream*)
      return 1
      ;;
    *)
      return 0
      ;;
  esac
}

write_snapshot "$BEFORE_FILE"

if ! make build; then
  log "make build failed"
  exit 3
fi

if [ -n "$BINARY_OVERRIDE" ]; then
  if [ -x "$BINARY_OVERRIDE" ]; then
    BINARY_PATH="$BINARY_OVERRIDE"
  elif [ -x "$REPO_ROOT/$BINARY_OVERRIDE" ]; then
    BINARY_PATH="$REPO_ROOT/$BINARY_OVERRIDE"
  else
    log "no executable artifact found; set GOYAIS_BINARY_PATH or check make build output"
    exit 2
  fi
else
  write_snapshot "$AFTER_FILE"
  awk -F '\t' 'NR==FNR {before[$1]=1; next} !($1 in before) {print $0}' "$BEFORE_FILE" "$AFTER_FILE" > "$NEW_FILE"

  if [ -s "$NEW_FILE" ]; then
    BINARY_PATH="$(choose_latest "$NEW_FILE" || true)"
  else
    BINARY_PATH="$(choose_latest "$AFTER_FILE" || true)"
  fi

  if [ -z "${BINARY_PATH:-}" ]; then
    log "no executable artifact found; set GOYAIS_BINARY_PATH or check make build output"
    exit 2
  fi
fi

log "selected binary: ${BINARY_PATH}"

if [ -d web/dist ]; then
  DIST_BACKUP="web/dist.__verify_backup_$$"
  mv web/dist "$DIST_BACKUP"
  DIST_MOVED="1"
fi

if [ -n "$START_CMD" ]; then
  sh -c "$START_CMD" >"$LOG_FILE" 2>&1 &
else
  "$BINARY_PATH" >"$LOG_FILE" 2>&1 &
fi
SERVER_PID="$!"

sleep 1
if ! kill -0 "$SERVER_PID" 2>/dev/null; then
  log "service exited immediately"
  log "startup log:"
  sed -n '1,120p' "$LOG_FILE"
  exit 4
fi

READY="0"
for _ in $(seq 1 40); do
  if curl -fsS "${BASE_URL}/api/v1/healthz" >/dev/null 2>&1; then
    READY="1"
    break
  fi
  sleep 1
done

if [ "$READY" != "1" ]; then
  log "service startup timeout waiting for /api/v1/healthz"
  log "startup log:"
  sed -n '1,120p' "$LOG_FILE"
  exit 4
fi

check_equals "$(http_code /)" "200" "GET / status"
check_equals "$(http_code /canvas)" "200" "GET /canvas status"
check_equals "$(http_code /api/v1/healthz)" "200" "GET /api/v1/healthz status"
check_equals "$(http_code /favicon.ico)" "404" "GET /favicon.ico status"
check_equals "$(http_code /robots.txt)" "404" "GET /robots.txt status"

ROOT_CONTENT_TYPE="$(header_value / Content-Type)"
CANVAS_CONTENT_TYPE="$(header_value /canvas Content-Type)"
ROOT_CACHE_CONTROL="$(header_value / Cache-Control)"
CANVAS_CACHE_CONTROL="$(header_value /canvas Cache-Control)"

check_contains "$ROOT_CONTENT_TYPE" "text/html" "GET / Content-Type"
check_contains "$CANVAS_CONTENT_TYPE" "text/html" "GET /canvas Content-Type"
check_equals "$ROOT_CACHE_CONTROL" "no-store" "GET / Cache-Control"
check_equals "$CANVAS_CACHE_CONTROL" "no-store" "GET /canvas Cache-Control"

if ! curl -fsS "${BASE_URL}/" > "$INDEX_FILE"; then
  log "FAIL: unable to fetch index html"
  CHECK_FAILED="1"
fi

ASSET_JS_LIST="$(grep -Eo "/assets/[^\"'[:space:]]+\\.js" "$INDEX_FILE" | sort -u || true)"
if [ -z "$ASSET_JS_LIST" ]; then
  log "FAIL: no /assets/*.js references found in index html"
  CHECK_FAILED="1"
else
  while IFS= read -r asset_js; do
    [ -n "$asset_js" ] || continue
    check_equals "$(http_code "$asset_js")" "200" "GET ${asset_js} status"
    asset_ctype="$(header_value "$asset_js" Content-Type)"
    if ! check_js_content_type "$asset_ctype"; then
      log "FAIL: ${asset_js} Content-Type invalid (${asset_ctype})"
      CHECK_FAILED="1"
    fi
  done <<EOF_ASSETS
$ASSET_JS_LIST
EOF_ASSETS
fi

if [ "$CHECK_FAILED" = "1" ]; then
  exit 1
fi

log "all checks passed"
exit 0
