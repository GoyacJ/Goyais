# ADR-0009: Remote Runtime Gateway (Phase 3)

- Status: Accepted
- Date: 2026-02-20

## Context

Phase 3 introduces remote run execution for multi-user workspaces. Desktop already supports workspace switching and remote control-plane/data-plane (Phase 1/2), but run/event/confirmation flow must be remotely usable with strict server-side enforcement and without breaking existing python-agent protocol semantics.

## Decision

1. Hub acts as runtime gateway for all remote run traffic:
   - Desktop -> Hub `/v1/runtime/*`
   - Hub -> Runtime `/v1/*` with injected headers:
     - `X-Hub-Auth`
     - `X-User-Id`
     - `X-Trace-Id`
2. `workspace_id` is passed externally only via query (`?workspace_id=...`), consistent with Phase 2.
3. P0 runtime topology is manual registry:
   - one workspace -> one runtime instance
   - admin registers `runtime_base_url` by workspace
4. Hub validates runtime health binding:
   - runtime `/v1/health` must self-report matching `workspace_id`
   - mismatch returns `E_RUNTIME_MISCONFIGURED` and gateway rejects traffic
5. Authorization is strictly separated:
   - `run:create` only for run creation
   - `run:read` only for runs/events/replay
   - `confirm:write` only for tool confirmations
6. Secret handling:
   - Hub keeps AES-256-GCM `enc:v1` at rest (`GOYAIS_HUB_SECRET_KEY`)
   - runtime resolves `secret:*` via Hub internal endpoint, one-shot use, no persistence
7. Protocol version source of truth:
   - loaded from `packages/protocol/schemas/**/protocol-version.json`
   - runtime emits this value in events; hub passthroughs runtime events

## Consequences

- Positive:
  - Centralized RBAC + audit ingress at Hub
  - Remote runtime no longer directly reachable by Desktop
  - Workspace isolation is explicit and verifiable
  - Existing event envelope/semantics remain intact
- Tradeoffs:
  - Additional hop (Hub proxy) introduces latency and operational complexity
  - P0 requires manual runtime registration and lifecycle management
- Follow-up (P1):
  - Hub-managed runtime spawn/heartbeat orchestration
  - mTLS/workload identity for Hub<->Runtime
  - extended audit ingestion and query APIs
