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

echo "[1/6] targeted tests (httpapi + core)"
(
  cd "$hub_dir"
  go test ./internal/httpapi/... ./internal/agent/core/...
)
echo

echo "[2/6] runstate function coverage gate"
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

echo "[3/6] runtime-mode default gate"
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

echo "[4/6] runtime hardening regression gates"
legacy_orchestrator_file="$hub_dir/internal/httpapi/execution_orchestrator.go"
legacy_orchestrator_test_files="$(
  cd "$hub_dir"
  (rg --files internal/httpapi | rg 'execution_orchestrator.*_test\\.go$' || true) \
    | wc -l | tr -d ' '
)"
state_legacy_wiring_hits="$(
  cd "$hub_dir"
  (rg -n 'Legacy\\s*:' internal/httpapi/state.go || true) \
    | wc -l | tr -d ' '
)"
legacy_alias_hits="$(
  cd "$hub_dir"
  (rg -n 'case "legacy", string\(executionRuntimeModeHybrid\):' internal/httpapi/execution_runtime_router.go || true) \
    | wc -l | tr -d ' '
)"
legacy_mode_symbol_hits="$(
  cd "$hub_dir"
  (rg -n 'executionRuntimeModeLegacy' internal/httpapi/execution_runtime_router.go || true) \
    | wc -l | tr -d ' '
)"
legacy_fake_run_builder_defs="$(
  cd "$hub_dir"
  (rg -n --glob '!**/*_test.go' 'func[[:space:]]+buildSlashEvents\(' internal cmd || true) \
    | wc -l | tr -d ' '
)"
legacy_stdout_guard_refs="$(
  cd "$hub_dir"
  (rg -n --glob '!**/*_test.go' '\bStdoutGuard\b' internal cmd || true) \
    | wc -l | tr -d ' '
)"
echo "legacy orchestrator file exists: $([[ -f "$legacy_orchestrator_file" ]] && echo yes || echo no)"
echo "legacy orchestrator test files count: $legacy_orchestrator_test_files"
echo "AppState legacy backend wiring hits: $state_legacy_wiring_hits"
echo "runtime mode alias (legacy->hybrid) hits: $legacy_alias_hits"
echo "legacy runtime mode symbol hits: $legacy_mode_symbol_hits"
echo "legacy fake-run builder definitions: $legacy_fake_run_builder_defs"
echo "legacy StdoutGuard refs: $legacy_stdout_guard_refs"
if [[ -f "$legacy_orchestrator_file" ]]; then
  echo "FAIL: legacy orchestrator file must stay deleted"
  exit 1
fi
if [[ "$legacy_orchestrator_test_files" -ne 0 ]]; then
  echo "FAIL: legacy orchestrator test files must stay deleted"
  exit 1
fi
if [[ "$state_legacy_wiring_hits" -ne 0 ]]; then
  echo "FAIL: AppState must not wire legacy backend in production path"
  exit 1
fi
if [[ "$legacy_alias_hits" -eq 0 ]]; then
  echo "FAIL: runtime mode parser must keep legacy compatibility alias to hybrid"
  exit 1
fi
if [[ "$legacy_mode_symbol_hits" -ne 0 ]]; then
  echo "FAIL: independent legacy runtime mode symbol must not be reintroduced"
  exit 1
fi
if [[ "$legacy_fake_run_builder_defs" -ne 0 ]]; then
  echo "FAIL: buildSlashEvents fake-run builder must not be reintroduced"
  exit 1
fi
if [[ "$legacy_stdout_guard_refs" -ne 0 ]]; then
  echo "FAIL: StdoutGuard refs must stay removed"
  exit 1
fi
echo

echo "[5/6] legacy reference counters"
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
direct_orchestrator_refs="$(
  cd "$hub_dir"
  (rg -n --glob '!**/*_test.go' 'state\\.orchestrator|\\.orchestrator\\b' internal/httpapi || true) \
    | wc -l | tr -d ' '
)"
echo "internal/agentcore external prod refs (excluding legacybridge): $agentcore_external_refs"
echo "ExecutionState/EventType non-httpapi prod refs: $execution_enum_non_httpapi_refs"
echo "direct state.orchestrator prod refs: $direct_orchestrator_refs"
if [[ "$strict" -eq 1 ]]; then
  if [[ "$agentcore_external_refs" -ne 0 ]]; then
    echo "FAIL: strict mode requires zero external agentcore refs"
    exit 1
  fi
  if [[ "$execution_enum_non_httpapi_refs" -ne 0 ]]; then
    echo "FAIL: strict mode requires zero non-httpapi ExecutionState/EventType refs"
    exit 1
  fi
  if [[ "$direct_orchestrator_refs" -ne 0 ]]; then
    echo "FAIL: strict mode requires zero direct state.orchestrator prod refs"
    exit 1
  fi
fi
echo

echo "[6/6] three-surface anchor checks"
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
