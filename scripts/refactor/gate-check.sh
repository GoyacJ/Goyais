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

echo "[1/7] targeted tests (httpapi + core)"
(
  cd "$hub_dir"
  go test ./internal/httpapi/... ./internal/agent/core/...
)
echo

echo "[2/7] runstate function coverage gate"
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

echo "[3/7] runtime-mode default gate"
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

echo "[4/7] runtime hardening regression gates"
legacy_orchestrator_file="$hub_dir/internal/httpapi/execution_orchestrator.go"
legacy_agentcore_dir="$hub_dir/internal/agentcore"
legacy_agentcoretools_dir="$hub_dir/internal/legacybridge/agentcoretools"
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
legacy_route_audit_hits="$(
  cd "$hub_dir"
  (rg -n --glob '!**/*_test.go' 'execution\\.runtime\\.route_legacy|route_legacy' internal/httpapi internal/agent cmd || true) \
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
echo "legacy agentcore dir exists: $([[ -d "$legacy_agentcore_dir" ]] && echo yes || echo no)"
echo "legacy agentcoretools dir exists: $([[ -d "$legacy_agentcoretools_dir" ]] && echo yes || echo no)"
echo "legacy orchestrator test files count: $legacy_orchestrator_test_files"
echo "AppState legacy backend wiring hits: $state_legacy_wiring_hits"
echo "runtime mode legacy alias hits: $legacy_alias_hits"
echo "legacy route audit hits: $legacy_route_audit_hits"
echo "legacy runtime mode symbol hits: $legacy_mode_symbol_hits"
echo "legacy fake-run builder definitions: $legacy_fake_run_builder_defs"
echo "legacy StdoutGuard refs: $legacy_stdout_guard_refs"
if [[ -f "$legacy_orchestrator_file" ]]; then
  echo "FAIL: legacy orchestrator file must stay deleted"
  exit 1
fi
if [[ -d "$legacy_agentcore_dir" ]]; then
  echo "FAIL: internal/agentcore directory must stay deleted"
  exit 1
fi
if [[ -d "$legacy_agentcoretools_dir" ]]; then
  echo "FAIL: internal/legacybridge/agentcoretools directory must stay deleted"
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
if [[ "$legacy_alias_hits" -ne 0 ]]; then
  echo "FAIL: runtime mode parser must not keep legacy compatibility alias"
  exit 1
fi
if [[ "$legacy_route_audit_hits" -ne 0 ]]; then
  echo "FAIL: legacy route audit markers must stay deleted"
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

echo "[5/7] legacy reference counters"
agentcore_external_refs="$(
  cd "$hub_dir"
  (rg -n --glob '!**/*_test.go' 'internal/agentcore' internal cmd || true) \
    | wc -l | tr -d ' '
)"
execution_enum_hub_refs="$(
  cd "$hub_dir"
  (rg -n --glob '!**/*_test.go' 'ExecutionState|ExecutionEventType' internal cmd || true) \
    | wc -l | tr -d ' '
)"
execution_enum_contract_refs="$(
  cd "$repo_root"
  (rg -n --glob '!**/*_test.go' 'ExecutionState|ExecutionEventType' \
    packages/contracts/openapi.yaml \
    packages/shared-core/src/api-common.ts \
    packages/shared-core/src/api-project.ts \
    packages/shared-core/src/generated/openapi.ts || true) \
    | wc -l | tr -d ' '
)"
direct_orchestrator_refs="$(
  cd "$hub_dir"
  (rg -n --glob '!**/*_test.go' 'state\\.orchestrator|\\.orchestrator\\b' internal/httpapi || true) \
    | wc -l | tr -d ' '
)"
echo "internal/agentcore prod refs: $agentcore_external_refs"
echo "ExecutionState/EventType hub prod refs: $execution_enum_hub_refs"
echo "ExecutionState/EventType contracts refs: $execution_enum_contract_refs"
echo "direct state.orchestrator prod refs: $direct_orchestrator_refs"
if [[ "$strict" -eq 1 ]]; then
  if [[ "$agentcore_external_refs" -ne 0 ]]; then
    echo "FAIL: strict mode requires zero external agentcore refs"
    exit 1
  fi
  if [[ "$execution_enum_hub_refs" -ne 0 ]]; then
    echo "FAIL: strict mode requires zero hub ExecutionState/EventType refs"
    exit 1
  fi
  if [[ "$execution_enum_contract_refs" -ne 0 ]]; then
    echo "FAIL: strict mode requires zero contracts ExecutionState/EventType refs"
    exit 1
  fi
  if [[ "$direct_orchestrator_refs" -ne 0 ]]; then
    echo "FAIL: strict mode requires zero direct state.orchestrator prod refs"
    exit 1
  fi
fi
echo

echo "[6/7] three-surface anchor checks"
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

echo "[7/7] incremental legacy-token gate (new additions only)"
diff_paths=(
  services/hub
  apps/desktop
  packages/shared-core
  .github/workflows
  scripts/refactor
)
if ! changed_files="$(
  cd "$repo_root"
  changed_working="$(git diff --name-only -- "${diff_paths[@]}" || true)"
  if [[ -n "$changed_working" ]]; then
    printf "%s\n" "$changed_working"
    exit 0
  fi
  if git rev-parse --verify HEAD~1 >/dev/null 2>&1; then
    git diff --name-only HEAD~1...HEAD -- "${diff_paths[@]}" || true
    exit 0
  fi
  printf ""
)"; then
  changed_files=""
fi

forbidden_addition_patterns=(
  'execution_runtime_'
  'v4Service'
  'V4Runner'
  'runtimebridge'
)
if [[ -n "$changed_files" ]]; then
  while IFS= read -r file; do
    if [[ -z "$file" ]]; then
      continue
    fi
    if [[ "$file" == *_test.go ]]; then
      continue
    fi
    if [[ "$file" != services/hub/* && "$file" != apps/desktop/* && "$file" != packages/shared-core/* ]]; then
      continue
    fi
    if ! file_added_lines="$(
      cd "$repo_root"
      file_working="$(git diff --unified=0 -- "$file" || true)"
      if [[ -n "$file_working" ]]; then
        printf "%s\n" "$file_working" | rg '^\+' | rg -v '^\+\+\+' || true
        exit 0
      fi
      if git rev-parse --verify HEAD~1 >/dev/null 2>&1; then
        git diff --unified=0 HEAD~1...HEAD -- "$file" | rg '^\+' | rg -v '^\+\+\+' || true
        exit 0
      fi
      printf ""
    )"; then
      file_added_lines=""
    fi
    for pattern in "${forbidden_addition_patterns[@]}"; do
      if printf "%s\n" "$file_added_lines" | rg -n "$pattern" >/tmp/goyais_gate_forbidden_additions.log; then
        echo "FAIL: detected forbidden addition pattern: $pattern in $file"
        cat /tmp/goyais_gate_forbidden_additions.log
        exit 1
      fi
    done
  done <<<"$changed_files"
fi

runtime_surface_regex='^services/hub/internal/agent/(runtime/loop|adapters/httpapi|adapters/acp|adapters/cli)/.*\\.go$'
if ! runtime_changed_files="$(
  cd "$repo_root"
  changed_working="$(git diff --name-only -- "${diff_paths[@]}" || true)"
  if [[ -n "$changed_working" ]]; then
    printf "%s\n" "$changed_working" | rg "$runtime_surface_regex" || true
    exit 0
  fi
  if git rev-parse --verify HEAD~1 >/dev/null 2>&1; then
    git diff --name-only HEAD~1...HEAD -- "${diff_paths[@]}" | rg "$runtime_surface_regex" || true
    exit 0
  fi
  printf ""
)"; then
  runtime_changed_files=""
fi

if [[ -n "$runtime_changed_files" ]]; then
  while IFS= read -r file; do
    if [[ -z "$file" ]]; then
      continue
    fi
    if ! file_added_lines="$(
      cd "$repo_root"
      file_working="$(git diff --unified=0 -- "$file" || true)"
      if [[ -n "$file_working" ]]; then
        printf "%s\n" "$file_working" | rg '^\+' | rg -v '^\+\+\+' || true
        exit 0
      fi
      if git rev-parse --verify HEAD~1 >/dev/null 2>&1; then
        git diff --unified=0 HEAD~1...HEAD -- "$file" | rg '^\+' | rg -v '^\+\+\+' || true
        exit 0
      fi
      printf ""
    )"; then
      file_added_lines=""
    fi
    if printf "%s\n" "$file_added_lines" | rg -n '\\bConversation\\b|\\bExecution\\b' >/tmp/goyais_gate_runtime_terms.log; then
      echo "FAIL: runtime surface added legacy Conversation/Execution term in $file"
      cat /tmp/goyais_gate_runtime_terms.log
      exit 1
    fi
  done <<<"$runtime_changed_files"
fi
echo

echo "PASS: Agent v4 gate check completed."
if [[ "$strict" -eq 0 ]]; then
  echo "Note: strict mode not enabled; migration-window counters are informational."
fi
