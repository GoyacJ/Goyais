#!/usr/bin/env bash
# Copyright (c) 2026 Ysmjjsy
# Author: Goya
# SPDX-License-Identifier: MIT

set -euo pipefail

strict=0
if [[ "${1:-}" == "--strict" ]]; then
  strict=1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
hub_dir="$repo_root/services/hub"
cover_file="/tmp/goyais_agent_core.cover"

echo "=== Agent v4 Gate Check ==="
echo "repo: $repo_root"
echo "hub:  $hub_dir"
echo

echo "[1/5] targeted tests (httpapi + core)"
(
  cd "$hub_dir"
  go test ./internal/httpapi/... ./internal/agent/core/...
)
echo

echo "[2/5] runstate function coverage gate"
(
  cd "$hub_dir"
  go test ./internal/agent/core -coverprofile="$cover_file" >/tmp/goyais_agent_core.cover.log
)
runstate_lines="$(
  cd "$hub_dir"
  go tool cover -func="$cover_file" | rg 'runstate.go' || true
)"
echo "$runstate_lines"
if [[ -z "$runstate_lines" ]]; then
  echo "FAIL: missing runstate.go coverage lines"
  exit 1
fi
runstate_not_full="$(echo "$runstate_lines" | awk '{gsub("%","",$3); if(($3+0)<100) c++} END {print c+0}')"
if [[ "$runstate_not_full" -gt 0 ]]; then
  echo "FAIL: runstate.go has functions below 100% coverage"
  exit 1
fi
echo

echo "[3/5] runtime-mode default gate"
default_mode_hits="$(
  cd "$hub_dir"
  rg -n 'mode = executionRuntimeModeHybrid' internal/httpapi/execution_runtime_router.go | wc -l | tr -d ' '
)"
echo "default hybrid assignment hits: $default_mode_hits"
if [[ "$default_mode_hits" -eq 0 ]]; then
  echo "FAIL: execution runtime default mode is not hybrid"
  exit 1
fi
echo

echo "[4/5] legacy reference counters"
agentcore_external_refs="$(
  cd "$hub_dir"
  (rg -n --glob '!**/*_test.go' 'internal/agentcore' internal cmd || true) \
    | awk '!/^internal\/agentcore\//' \
    | awk '!/^internal\/legacybridge\/agentcoretools\//' \
    | wc -l | tr -d ' '
)"
execution_enum_non_httpapi_refs="$(
  cd "$hub_dir"
  (rg -n --glob '!**/*_test.go' 'ExecutionState|ExecutionEventType' internal cmd || true) \
    | awk '!/^internal\/httpapi\//' \
    | wc -l | tr -d ' '
)"
echo "internal/agentcore external prod refs (excluding legacybridge): $agentcore_external_refs"
echo "ExecutionState/EventType non-httpapi prod refs: $execution_enum_non_httpapi_refs"
if [[ "$strict" -eq 1 ]]; then
  if [[ "$agentcore_external_refs" -ne 0 ]]; then
    echo "FAIL: strict mode requires zero external agentcore refs"
    exit 1
  fi
  if [[ "$execution_enum_non_httpapi_refs" -ne 0 ]]; then
    echo "FAIL: strict mode requires zero non-httpapi ExecutionState/EventType refs"
    exit 1
  fi
fi
echo

echo "[5/5] three-surface anchor checks"
loop_engine_hits="$(
  cd "$hub_dir"
  (rg -n 'loop.NewEngine' cmd/goyais-cli/main.go cmd/goyais-cli/adapters/v4_runner.go cmd/goyais-acp/main.go internal/httpapi/state.go || true) \
    | wc -l | tr -d ' '
)"
echo "loop.NewEngine anchor hits (cli main+adapter, acp, httpapi): $loop_engine_hits"
if [[ "$loop_engine_hits" -lt 3 ]]; then
  echo "FAIL: missing loop.NewEngine anchor in one or more surfaces"
  exit 1
fi
echo

echo "PASS: Agent v4 gate check completed."
if [[ "$strict" -eq 0 ]]; then
  echo "Note: strict mode not enabled; migration-window counters are informational."
fi
