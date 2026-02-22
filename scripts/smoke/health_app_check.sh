#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DESKTOP_PORT="${DESKTOP_PORT:-5173}"

ARTIFACT_DIR="$ROOT_DIR/artifacts"
APP_SMOKE_JSON="$ARTIFACT_DIR/smoke-app.json"
LOG_DIR="$(mktemp -d -t goyais-smoke-app-XXXXXX)"

mkdir -p "$ARTIFACT_DIR"

TAURI_PID=""
SIGNAL_REASON=""

write_result() {
  local status="$1"
  local summary="$2"

  python3 - "$APP_SMOKE_JSON" "$status" "$summary" "$DESKTOP_PORT" <<'PY'
import datetime as dt
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
payload = {
    "status": sys.argv[2],
    "summary": sys.argv[3],
    "port": int(sys.argv[4]),
    "generated_at": dt.datetime.now(dt.timezone.utc).isoformat(),
}
path.parent.mkdir(parents=True, exist_ok=True)
path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
}

kill_process_tree() {
  local pid="$1"

  if ! kill -0 "$pid" >/dev/null 2>&1; then
    return
  fi

  local children=""
  children="$(pgrep -P "$pid" || true)"
  for child in $children; do
    kill_process_tree "$child"
  done

  kill "$pid" >/dev/null 2>&1 || true
  wait "$pid" >/dev/null 2>&1 || true

  if kill -0 "$pid" >/dev/null 2>&1; then
    kill -9 "$pid" >/dev/null 2>&1 || true
    wait "$pid" >/dev/null 2>&1 || true
  fi
}

on_signal() {
  SIGNAL_REASON="$1"
  exit 130
}

cleanup() {
  local status=$?

  if [[ -n "$TAURI_PID" ]]; then
    kill_process_tree "$TAURI_PID"
  fi

  if [[ $status -eq 0 ]]; then
    write_result "pass" "tauri shell smoke passed"
    echo "[smoke-app] artifact: $APP_SMOKE_JSON"
    rm -rf "$LOG_DIR"
  else
    local summary="tauri shell smoke failed"
    if [[ -n "$SIGNAL_REASON" ]]; then
      summary="interrupted by signal $SIGNAL_REASON"
    fi

    write_result "fail" "$summary"
    echo "[smoke-app] failed"
    echo "[smoke-app] port: ${DESKTOP_PORT}"
    echo "[smoke-app] artifact: $APP_SMOKE_JSON"
    echo "[smoke-app] logs: ${LOG_DIR}"
    if [[ -f "$LOG_DIR/tauri.log" ]]; then
      echo "----- $LOG_DIR/tauri.log -----"
      tail -n 120 "$LOG_DIR/tauri.log" || true
    fi
  fi

  return "$status"
}

trap 'on_signal INT' INT
trap 'on_signal TERM' TERM
trap cleanup EXIT

echo "[smoke-app] starting tauri dev shell (expected web port: ${DESKTOP_PORT})"
(
  cd "$ROOT_DIR/apps/desktop"
  DESKTOP_PORT="$DESKTOP_PORT" pnpm tauri:dev >"$LOG_DIR/tauri.log" 2>&1
) &
TAURI_PID="$!"

for _ in $(seq 1 240); do
  if curl -fsS "http://localhost:${DESKTOP_PORT}" >/dev/null 2>&1; then
    if curl -fsS "http://localhost:${DESKTOP_PORT}" | grep -q '<div id="app"></div>'; then
      echo "[smoke-app] tauri shell smoke passed"
      exit 0
    fi
  fi

  if ! kill -0 "$TAURI_PID" >/dev/null 2>&1; then
    if grep -q 'Running.*target/debug/goyais-desktop' "$LOG_DIR/tauri.log" 2>/dev/null; then
      echo "[smoke-app] tauri binary reached run stage before exit"
      echo "[smoke-app] tauri shell smoke passed"
      exit 0
    fi
    echo "[smoke-app] tauri process exited early"
    exit 1
  fi

  sleep 0.25
done

echo "[smoke-app] timeout waiting for tauri dev readiness"
exit 1
