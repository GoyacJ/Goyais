#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
SCAN_DEPTH="${1:-deep}"

if [[ "${SCAN_DEPTH}" != "standard" && "${SCAN_DEPTH}" != "deep" ]]; then
  echo "usage: $0 [standard|deep]" >&2
  exit 1
fi

cd "${REPO_ROOT}"

ROUTER_FILE="${REPO_ROOT}/internal/access/http/router.go"
OPENAPI_FILE="${REPO_ROOT}/docs/api/openapi.yaml"
ROUTER_TEST="${REPO_ROOT}/internal/access/http/router_integration_test.go"
POSTGRES_IT="${REPO_ROOT}/internal/integration/postgres_contract_test.go"

exists_file() {
  local file="$1"
  if [[ -f "${file}" ]]; then
    echo "yes"
  else
    echo "no"
  fi
}

route_mode() {
  local api_prefix="$1"
  local placeholder_key="$2"
  local implemented_marker="${3:-}"
  local mounted="no"
  local placeholder="no"
  local has_implemented_marker="no"

  if rg -n "${api_prefix}" "${ROUTER_FILE}" >/dev/null 2>&1; then
    mounted="yes"
  fi
  if rg -n "${placeholder_key}" "${ROUTER_FILE}" >/dev/null 2>&1; then
    placeholder="yes"
  fi
  if [[ -n "${implemented_marker}" ]] && rg -n "${implemented_marker}" "${ROUTER_FILE}" >/dev/null 2>&1; then
    has_implemented_marker="yes"
  fi

  if [[ "${mounted}" == "yes" && "${placeholder}" == "yes" && "${has_implemented_marker}" == "yes" ]]; then
    echo "implemented"
    return
  fi
  if [[ "${mounted}" == "yes" && "${placeholder}" == "yes" && "${has_implemented_marker}" == "no" ]]; then
    echo "placeholder"
    return
  fi
  if [[ "${mounted}" == "yes" && "${placeholder}" == "no" ]]; then
    echo "implemented"
    return
  fi
  if [[ "${mounted}" == "no" && "${placeholder}" == "yes" ]]; then
    echo "placeholder"
    return
  fi
  echo "unknown"
}

test_hit() {
  local pattern="$1"
  if rg -n "${pattern}" "${ROUTER_TEST}" "${POSTGRES_IT}" >/dev/null 2>&1; then
    echo "yes"
  else
    echo "no"
  fi
}

migration_hit() {
  local keyword="$1"
  local hit_sqlite
  local hit_postgres

  hit_sqlite="$(rg -l "${keyword}" ${REPO_ROOT}/migrations/sqlite/*.sql 2>/dev/null | head -n 1 || true)"
  hit_postgres="$(rg -l "${keyword}" ${REPO_ROOT}/migrations/postgres/*.sql 2>/dev/null | head -n 1 || true)"

  if [[ -n "${hit_sqlite}" && -n "${hit_postgres}" ]]; then
    echo "yes"
    return
  fi
  if [[ -n "${hit_sqlite}" || -n "${hit_postgres}" ]]; then
    echo "partial"
    return
  fi
  echo "no"
}

domain_row() {
  local name="$1"
  local api_prefix="$2"
  local placeholder_key="$3"
  local implemented_marker="$4"
  local handler_file="$5"
  local service_file="$6"
  local repo_path="$7"
  local migration_keyword="$8"
  local test_pattern="$9"

  local route_status
  route_status="$(route_mode "${api_prefix}" "${placeholder_key}" "${implemented_marker}")"

  local handler_status
  local service_status
  local repo_status
  local migration_status
  local test_status

  handler_status="$(exists_file "${handler_file}")"
  service_status="$(exists_file "${service_file}")"

  if [[ -e "${repo_path}" ]]; then
    repo_status="yes"
  else
    repo_status="no"
  fi

  migration_status="$(migration_hit "${migration_keyword}")"
  test_status="$(test_hit "${test_pattern}")"

  local final_status="partial"
  if [[ "${route_status}" == "placeholder" ]]; then
    final_status="placeholder"
  elif [[ "${route_status}" == "implemented" && "${handler_status}" == "yes" && "${service_status}" == "yes" && "${repo_status}" == "yes" ]]; then
    if [[ "${migration_status}" == "yes" && "${test_status}" == "yes" ]]; then
      final_status="implemented"
    else
      final_status="partial"
    fi
  elif [[ "${route_status}" == "unknown" ]]; then
    final_status="unknown"
  fi

  printf '| %s | %s | %s | %s | %s | %s | %s | %s |\n' \
    "${name}" "${final_status}" "${route_status}" "${handler_status}" "${service_status}" "${repo_status}" "${migration_status}" "${test_status}"
}

printf '## Implementation Scan Matrix\n'
printf -- '- repo_root: %s\n' "${REPO_ROOT}"
printf -- '- scan_depth: %s\n' "${SCAN_DEPTH}"
printf -- '- router: %s\n' "${ROUTER_FILE}"
printf -- '- openapi: %s\n' "${OPENAPI_FILE}"
printf -- '- command: %s\n\n' "${REPO_ROOT}/.agents/skills/goyais-progress-next-plan/scripts/implementation_scan.sh ${SCAN_DEPTH}"

printf '| Domain | Status | Route | Handler | Service | Repo | Migration | Tests |\n'
printf '|---|---|---|---|---|---|---|---|\n'

domain_row \
  "commands" \
  '/api/v1/commands' \
  'error.command.not_implemented' \
  'deps.CommandService != nil' \
  "${REPO_ROOT}/internal/access/http/commands.go" \
  "${REPO_ROOT}/internal/command/service.go" \
  "${REPO_ROOT}/internal/command" \
  'commands|idempotency' \
  '/api/v1/commands'

domain_row \
  "shares" \
  '/api/v1/shares' \
  'error.share.not_implemented' \
  'deps.CommandService != nil' \
  "${REPO_ROOT}/internal/access/http/shares.go" \
  "${REPO_ROOT}/internal/command/service.go" \
  "${REPO_ROOT}/internal/command" \
  'acl_entries|shares' \
  '/api/v1/shares'

domain_row \
  "assets" \
  '/api/v1/assets' \
  'error.asset.not_implemented' \
  'deps.AssetService != nil' \
  "${REPO_ROOT}/internal/access/http/assets.go" \
  "${REPO_ROOT}/internal/asset/service.go" \
  "${REPO_ROOT}/internal/asset" \
  'assets|asset_lineage' \
  '/api/v1/assets'

domain_row \
  "workflow" \
  '/api/v1/workflow-templates|/api/v1/workflow-runs' \
  'error.workflow.not_implemented' \
  'deps.WorkflowService != nil' \
  "${REPO_ROOT}/internal/access/http/workflows.go" \
  "${REPO_ROOT}/internal/workflow/service.go" \
  "${REPO_ROOT}/internal/workflow" \
  'workflow_templates|workflow_runs|step_runs' \
  '/api/v1/workflow-templates|/api/v1/workflow-runs'

domain_row \
  "registry" \
  '/api/v1/registry/capabilities|/api/v1/registry/algorithms|/api/v1/registry/providers' \
  'error.registry.not_implemented' \
  'deps.RegistryService != nil' \
  "${REPO_ROOT}/internal/access/http/registry.go" \
  "${REPO_ROOT}/internal/registry/service.go" \
  "${REPO_ROOT}/internal/registry" \
  'capabilities|capability_providers|algorithms' \
  '/api/v1/registry/capabilities|/api/v1/registry/algorithms|/api/v1/registry/providers'

domain_row \
  "plugin-market" \
  '/api/v1/plugin-market/packages|/api/v1/plugin-market/installs' \
  'error.plugin.not_implemented' \
  'deps.PluginService != nil' \
  "${REPO_ROOT}/internal/access/http/plugins.go" \
  "${REPO_ROOT}/internal/plugin/service.go" \
  "${REPO_ROOT}/internal/plugin" \
  'plugin' \
  '/api/v1/plugin-market'

domain_row \
  "streams" \
  '/api/v1/streams' \
  'error.stream.not_implemented' \
  'deps.StreamService != nil' \
  "${REPO_ROOT}/internal/access/http/streams.go" \
  "${REPO_ROOT}/internal/stream/service.go" \
  "${REPO_ROOT}/internal/stream" \
  'stream' \
  '/api/v1/streams'

domain_row \
  "algorithms-mvp" \
  '/api/v1/algorithms/' \
  'error.algorithm.not_implemented' \
  'deps.CommandService != nil' \
  "${REPO_ROOT}/internal/access/http/algorithms.go" \
  "${REPO_ROOT}/internal/algorithm/service.go" \
  "${REPO_ROOT}/internal/algorithm" \
  'algorithm_runs|algorithms' \
  '/api/v1/algorithms'

printf '\n## Contract Drift Findings\n'
printf -- '- source_router: %s\n' "${ROUTER_FILE}"
printf -- '- source_openapi: %s\n' "${OPENAPI_FILE}"
printf -- '- source_router_tests: %s\n' "${ROUTER_TEST}"
printf -- '- source_postgres_it: %s\n' "${POSTGRES_IT}"

check_drift() {
  local domain="$1"
  local openapi_pattern="$2"
  local router_pattern="$3"

  local openapi_has="no"
  local router_has="no"

  if rg -n "${openapi_pattern}" "${OPENAPI_FILE}" >/dev/null 2>&1; then
    openapi_has="yes"
  fi
  if rg -n "${router_pattern}" "${ROUTER_FILE}" >/dev/null 2>&1; then
    router_has="yes"
  fi

  if [[ "${openapi_has}" == "yes" && "${router_has}" == "yes" ]]; then
    printf -- '- confirmed: %s path exists in both openapi and router\n' "${domain}"
    return
  fi

  if [[ "${openapi_has}" == "yes" && "${router_has}" == "no" ]]; then
    printf -- '- confirmed drift: %s path in openapi but missing in router\n' "${domain}"
    return
  fi

  if [[ "${openapi_has}" == "no" && "${router_has}" == "yes" ]]; then
    printf -- '- confirmed drift: %s path in router but missing in openapi\n' "${domain}"
    return
  fi

  printf -- '- unknown: %s path not found in quick scan\n' "${domain}"
}

check_drift "commands" '^  /commands' '/api/v1/commands'
check_drift "shares" '^  /shares' '/api/v1/shares'
check_drift "assets" '^  /assets' '/api/v1/assets'
check_drift "workflow" '^  /workflow-' '/api/v1/workflow-templates|/api/v1/workflow-runs'
check_drift "registry" '^  /registry/' '/api/v1/registry/'
check_drift "plugin-market" '^  /plugin-market/' '/api/v1/plugin-market/'
check_drift "streams" '^  /streams' '/api/v1/streams'
check_drift "algorithms-mvp" '^  /algorithms/' '/api/v1/algorithms/'

if [[ "${SCAN_DEPTH}" == "deep" ]]; then
  printf '\n## Deep Evidence Commands\n'
  echo "- go test ./..."
  echo "- pnpm -C web typecheck"
  echo "- pnpm -C web test:run"
  echo "- make build"
  echo "- GOYAIS_VERIFY_BASE_URL=http://127.0.0.1:18080 GOYAIS_START_CMD='GOYAIS_SERVER_ADDR=:18080 ./build/goyais' bash ${REPO_ROOT}/.agents/skills/goyais-single-binary-acceptance/scripts/verify_single_binary.sh"
else
  printf '\n## Deep Evidence Commands\n'
  echo "- skipped: scan_depth=standard"
  echo "- risk: regression confidence is partial without execution evidence"
fi
