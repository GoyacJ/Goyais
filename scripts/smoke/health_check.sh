#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HUB_PORT="${HUB_PORT:-8787}"
WORKER_PORT="${WORKER_PORT:-8788}"
DESKTOP_PORT="${DESKTOP_PORT:-5173}"
EXPECTED_VERSION="${GOYAIS_VERSION:-0.0.0-dev}"

ARTIFACT_DIR="$ROOT_DIR/artifacts"
SMOKE_JSON="$ARTIFACT_DIR/smoke.json"
LOG_DIR="$(mktemp -d -t goyais-smoke-XXXXXX)"
CHECKS_FILE="$LOG_DIR/checks.tsv"

mkdir -p "$ARTIFACT_DIR"
: >"$CHECKS_FILE"

declare -a PIDS=()
SIGNAL_REASON=""

record_check() {
  local name="$1"
  local status="$2"
  local detail="${3:-}"
  printf '%s\t%s\t%s\n' "$name" "$status" "$detail" >>"$CHECKS_FILE"
}

port_in_use() {
  local port="$1"
  python3 - "$port" <<'PY'
import socket
import sys

port = int(sys.argv[1])
sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
try:
    sock.bind(("127.0.0.1", port))
except OSError:
    sys.exit(0)
else:
    sock.close()
    sys.exit(1)
PY
}

find_available_port() {
  local port="$1"
  local exclude1="${2:-}"
  local exclude2="${3:-}"

  while port_in_use "$port" || [[ "$port" == "$exclude1" ]] || [[ "$port" == "$exclude2" ]]; do
    port=$((port + 1))
  done

  echo "$port"
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

cleanup_processes() {
  local pid
  for pid in "${PIDS[@]:-}"; do
    if [[ -n "$pid" ]]; then
      kill_process_tree "$pid"
    fi
  done
}

write_smoke_json() {
  local overall_status="$1"
  local summary="$2"

  python3 - "$SMOKE_JSON" "$overall_status" "$summary" "$HUB_PORT" "$WORKER_PORT" "$DESKTOP_PORT" "$CHECKS_FILE" <<'PY'
import datetime as dt
import json
import pathlib
import sys

smoke_path = pathlib.Path(sys.argv[1])
overall_status = sys.argv[2]
summary = sys.argv[3]
hub_port = int(sys.argv[4])
worker_port = int(sys.argv[5])
desktop_port = int(sys.argv[6])
checks_file = pathlib.Path(sys.argv[7])

checks = []
if checks_file.exists():
    for line in checks_file.read_text(encoding="utf-8").splitlines():
        if not line.strip():
            continue
        parts = line.split("\t", 2)
        if len(parts) < 3:
            parts += [""] * (3 - len(parts))
        checks.append({"name": parts[0], "status": parts[1], "detail": parts[2]})

payload = {
    "status": overall_status,
    "summary": summary,
    "ports": {
        "hub": hub_port,
        "worker": worker_port,
        "desktop": desktop_port,
    },
    "checks": checks,
    "generated_at": dt.datetime.now(dt.timezone.utc).isoformat(),
}

smoke_path.parent.mkdir(parents=True, exist_ok=True)
smoke_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
}

on_signal() {
  SIGNAL_REASON="$1"
  exit 130
}

cleanup_and_report() {
  local status=$?
  cleanup_processes

  local overall_status="pass"
  local summary="all checks passed"
  if [[ $status -ne 0 ]]; then
    overall_status="fail"
    summary="smoke checks failed"
  fi
  if [[ -n "$SIGNAL_REASON" ]]; then
    overall_status="fail"
    summary="interrupted by signal $SIGNAL_REASON"
  fi

  write_smoke_json "$overall_status" "$summary"

  if [[ $status -ne 0 ]]; then
    echo "[smoke] failed"
    echo "[smoke] ports: hub=${HUB_PORT} worker=${WORKER_PORT} desktop=${DESKTOP_PORT}"
    echo "[smoke] artifact: ${SMOKE_JSON}"
    echo "[smoke] logs: ${LOG_DIR}"
    for file in "$LOG_DIR"/*.log; do
      if [[ -f "$file" ]]; then
        echo "----- ${file} -----"
        tail -n 120 "$file" || true
      fi
    done
  else
    echo "[smoke] artifact: ${SMOKE_JSON}"
    rm -rf "$LOG_DIR"
  fi

  return "$status"
}

trap 'on_signal INT' INT
trap 'on_signal TERM' TERM
trap cleanup_and_report EXIT

wait_for_http() {
  local check_name="$1"
  local url="$2"
  local attempts="${3:-80}"
  local delay="${4:-0.25}"

  for _ in $(seq 1 "$attempts"); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      record_check "$check_name" "pass" "url=${url}"
      return 0
    fi
    sleep "$delay"
  done

  record_check "$check_name" "fail" "url=${url}; timeout"
  return 1
}

check_health_json() {
  local check_name="$1"
  local url="$2"
  local output_file="$3"

  if ! curl -fsS "$url" >"$output_file"; then
    record_check "$check_name" "fail" "request failed; url=${url}"
    return 1
  fi

  if python3 - "$output_file" "$EXPECTED_VERSION" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)

if data.get("ok") is not True:
    raise SystemExit(1)
if data.get("version") != sys.argv[2]:
    raise SystemExit(1)
PY
  then
    record_check "$check_name" "pass" "ok=true version=${EXPECTED_VERSION}"
    return 0
  fi

  record_check "$check_name" "fail" "unexpected body in ${output_file}"
  return 1
}

check_list_envelope() {
  local check_name="$1"
  local url="$2"
  local output_file="$3"

  if ! curl -fsS "$url" >"$output_file"; then
    record_check "$check_name" "fail" "request failed; url=${url}"
    return 1
  fi

  if python3 - "$output_file" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)

items = data.get("items")
if not isinstance(items, list):
    raise SystemExit(1)
if "next_cursor" not in data:
    raise SystemExit(1)
next_cursor = data.get("next_cursor")
if next_cursor is not None and not isinstance(next_cursor, str):
    raise SystemExit(1)
PY
  then
    local item_count
    item_count="$(python3 - "$output_file" <<'PY'
import json
import sys
with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)
print(len(data.get("items", [])))
PY
)"
    record_check "$check_name" "pass" "items=${item_count} next_cursor=present"
    return 0
  fi

  record_check "$check_name" "fail" "unexpected body in ${output_file}"
  return 1
}

check_workspace_list_envelope() {
  local check_name="$1"
  local url="$2"
  local output_file="$3"

  if ! curl -fsS "$url" >"$output_file"; then
    record_check "$check_name" "fail" "request failed; url=${url}"
    return 1
  fi

  if python3 - "$output_file" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)

items = data.get("items")
if not isinstance(items, list):
    raise SystemExit(1)
if data.get("next_cursor", "missing") is not None:
    raise SystemExit(1)

found_local = any(item.get("id") == "ws_local" and item.get("mode") == "local" for item in items if isinstance(item, dict))
if not found_local:
    raise SystemExit(1)
PY
  then
    record_check "$check_name" "pass" "contains ws_local/local and next_cursor=null"
    return 0
  fi

  record_check "$check_name" "fail" "unexpected body in ${output_file}"
  return 1
}

check_error_trace_consistency() {
  local check_name="$1"
  local method="$2"
  local url="$3"
  local expected_trace="$4"
  local headers_file="$5"
  local body_file="$6"

  curl -sS -D "$headers_file" -o "$body_file" -X "$method" -H "X-Trace-Id: ${expected_trace}" "$url"

  local status_code
  status_code="$(awk 'NR==1{print $2}' "$headers_file")"
  local header_trace
  header_trace="$(grep -i '^X-Trace-Id:' "$headers_file" | head -n 1 | cut -d ':' -f 2- | tr -d '\r' | xargs || true)"
  local body_trace
  body_trace="$(python3 - "$body_file" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)

print(data.get("trace_id", ""))
PY
)"

  if [[ "$status_code" =~ ^[45][0-9][0-9]$ ]] \
    && [[ "$header_trace" == "$expected_trace" ]] \
    && ([[ -z "$body_trace" ]] || [[ "$body_trace" == "$expected_trace" ]]); then
    record_check "$check_name" "pass" "status=${status_code} trace_id_header=${expected_trace}"
    return 0
  fi

  record_check "$check_name" "fail" "status=${status_code} header_trace=${header_trace} body_trace=${body_trace}"
  return 1
}

check_desktop_index() {
  local check_name="$1"
  local url="$2"

  if curl -fsS "$url" | grep -q '<div id="app"></div>'; then
    record_check "$check_name" "pass" "found app mount node"
    return 0
  fi

  record_check "$check_name" "fail" "missing app mount node"
  return 1
}

HUB_PORT="$(find_available_port "$HUB_PORT")"
WORKER_PORT="$(find_available_port "$WORKER_PORT" "$HUB_PORT")"
DESKTOP_PORT="$(find_available_port "$DESKTOP_PORT" "$HUB_PORT" "$WORKER_PORT")"

echo "[smoke] selected ports: hub=${HUB_PORT} worker=${WORKER_PORT} desktop=${DESKTOP_PORT}"
record_check "ports.selected" "pass" "hub=${HUB_PORT} worker=${WORKER_PORT} desktop=${DESKTOP_PORT}"

echo "[smoke] starting hub"
(
  cd "$ROOT_DIR/services/hub"
  PORT="$HUB_PORT" go run ./cmd/hub >"$LOG_DIR/hub.log" 2>&1
) &
PIDS+=("$!")
wait_for_http "hub.readiness" "http://127.0.0.1:${HUB_PORT}/health"
check_health_json "hub.health" "http://127.0.0.1:${HUB_PORT}/health" "$LOG_DIR/hub_health.json"
check_workspace_list_envelope "hub.list.workspaces" "http://127.0.0.1:${HUB_PORT}/v1/workspaces" "$LOG_DIR/_v1_workspaces_list.json"

for path in /v1/projects /v1/conversations /v1/executions; do
  check_name="hub.list.${path#/v1/}"
  output_file="$LOG_DIR/${path//\//_}_list.json"
  check_list_envelope "$check_name" "http://127.0.0.1:${HUB_PORT}${path}" "$output_file"
done

check_error_trace_consistency \
  "hub.trace.error_consistency" \
  "POST" \
  "http://127.0.0.1:${HUB_PORT}/v1/projects" \
  "tr_smoke_hub" \
  "$LOG_DIR/hub_error_headers.txt" \
  "$LOG_DIR/hub_error_body.json"

echo "[smoke] starting worker"
(
  cd "$ROOT_DIR/services/worker"
  PORT="$WORKER_PORT" uv run python -m app.main >"$LOG_DIR/worker.log" 2>&1
) &
PIDS+=("$!")
wait_for_http "worker.readiness" "http://127.0.0.1:${WORKER_PORT}/health"
check_health_json "worker.health" "http://127.0.0.1:${WORKER_PORT}/health" "$LOG_DIR/worker_health.json"

check_error_trace_consistency \
  "worker.trace.error_consistency" \
  "POST" \
  "http://127.0.0.1:${WORKER_PORT}/internal/executions" \
  "tr_smoke_worker" \
  "$LOG_DIR/worker_error_headers.txt" \
  "$LOG_DIR/worker_error_body.json"

echo "[smoke] starting desktop web dev server"
(
  cd "$ROOT_DIR"
  pnpm --filter @goyais/desktop dev --port "$DESKTOP_PORT" --host 127.0.0.1 --strictPort >"$LOG_DIR/desktop.log" 2>&1
) &
PIDS+=("$!")
wait_for_http "desktop.readiness" "http://127.0.0.1:${DESKTOP_PORT}"
check_desktop_index "desktop.index" "http://127.0.0.1:${DESKTOP_PORT}"

echo "[smoke] all checks passed"
