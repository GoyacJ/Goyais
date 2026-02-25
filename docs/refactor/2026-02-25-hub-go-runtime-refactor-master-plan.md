# Hub+Go Runtime Refactor Master Plan (Frozen)

- Date: 2026-02-25
- Owner: Goyais Refactor Program
- Status: Approved Baseline (Frozen)
- Scope: Documentation governance only in this commit. Technical route unchanged.

## Background and Goals

This plan freezes the already-approved refactor strategy:

- Remove Python Worker and move runtime capability into Hub (Go, single process embedded runtime).
- Use original Kode-cli behavior as the authoritative baseline.
- Core first, UI later: business/protocol/tools/config/state machine in core; UI only presentation and input.
- One mergeable small change at a time with code, tests, runbook, change list.
- Desktop + Mobile full coverage.
- Keep `/v1` path but redefine semantics; one-shot full cutover; historical execution data reset (no migration).

This document is the source-of-truth baseline for execution. Except the "Change Record" section, content must not be edited without explicit approval record.

## Hard Constraints

The following constraints are mandatory and unchanged:

1. Any implementation must use original project behavior as baseline: CLI args, interaction flow, output format (including error output), exit codes, config semantics, defaults, env vars, plugin/protocol interactions must align.
2. Abstract core first (business/protocol/tool execution/config/state machine), then implement UI (TUI/CLI). UI only handles presentation and input; no business logic.
3. Deliver exactly one mergeable small change each time: includes code, tests (unit/integration), run instructions, and change list.
4. For uncertain behavior: infer by reading original repo source/README/scripts/tests and record evidence; do not guess. If still uncertain, output "pending confirmation list" and minimal probe experiments.
5. Every output must include: implementation approach, file tree/module boundaries, key interfaces, test strategy, and regression risks.
6. Coding requirements: maintainable code, complete error handling, explicit logging/debug switches; no unnecessary dependencies; no secrets in code.
7. Default to cross-platform distributable strategy (Go single binary / Python packaging where needed), and guarantee startup responsiveness and non-blocking interaction (concurrency/async must not block UI thread).

## Refactor Roadmap

### Phase 0: Baseline Contract Freeze

- Build parity framework against original Kode-cli behavior.
- Capture golden artifacts for stdout/stderr/exit code/interaction transcripts.
- Produce pending confirmation list and probe recipes.

### Phase 1: Core Abstraction (No UI coupling)

- Implement core modules for config/protocol/state machine/tool runtime/safety/model adapters/events.
- Keep UI thin and disconnected from business logic.

### Phase 2: Hub Integration

- Integrate core into Hub routes and runtime pipeline.
- Replace internal execution flow with run-centric model.
- Preserve path prefix `/v1` while upgrading semantics.

### Phase 3: UI Integration (CLI/TUI as shell)

- Build Go CLI/TUI adapters on top of core.
- Match behavior contracts from baseline matrix.

### Phase 4: Cutover and Cleanup

- One-shot full cutover after all gates pass.
- Remove Worker and Kode-cli artifacts from this repo.
- Keep rollback package and verified restore procedure.

## Contract / Parity Checklist

### CLI Contract

- Command names, aliases, arguments, defaults, help behavior.
- Non-interactive output structure.
- stderr placement and exit code mapping.

### Interaction Contract

- Prompt loop and step transitions.
- Plan mode enter/exit behavior.
- Approval request and resolution behavior.

### Output Contract

- stdout text layout.
- stderr error shape and wording class.
- Deterministic sections and ordering where applicable.

### Config and Defaults Contract

- Config file locations, precedence, default fallback values.
- Invalid config failure behavior.

### Environment Variable Contract

- Model/network/debug/path variables and precedence.
- Missing/conflicting env handling behavior.

### Protocol / Plugin Contract

- MCP discovery/invocation/timeout/error mapping.
- ACP frame-level behavior.
- Plugin interaction behavior and failure propagation.

### Tool Runtime Contract

- Tool input/output schema alignment.
- Risk gating and approval semantics.
- Diff/patch artifact generation semantics.

## Core Module Boundaries and File Tree

```text
services/hub/internal/agentcore/
  config/
  protocol/
  state/
  model/
  tools/
  safety/
  runtime/
  io/

services/hub/internal/httpapi/
  # HTTP adapter layer only; delegates to agentcore

cmd/goyais-cli/
  cli/        # args + command routing only
  tui/        # rendering/input only
  adapters/   # UI <-> core adapters
```

Boundary rules:

- `agentcore` owns all business logic.
- `httpapi` and `cmd/goyais-cli` are adapter layers only.
- No business logic in UI or command parsing layer.

## Key Interface Definitions

```go
type Engine interface {
  StartSession(ctx context.Context, req StartSessionRequest) (SessionHandle, error)
  Submit(ctx context.Context, sessionID string, input UserInput) (RunID, error)
  Control(ctx context.Context, runID string, action ControlAction) error
  Subscribe(ctx context.Context, sessionID string, cursor string) (<-chan RunEvent, error)
}

type Tool interface {
  Spec() ToolSpec
  Execute(ctx ToolContext, call ToolCall) (ToolResult, error)
}

type ConfigProvider interface {
  Load(globalPath, projectPath string, env map[string]string) (ResolvedConfig, error)
}
```

## Public API / Type Changes

The following public changes are part of the approved technical route:

- `POST /v1/conversations/{conversation_id}/messages`: execution semantics move to run semantics.
- `GET /v1/conversations/{conversation_id}/events`: unified `RunEvent` event family.
- `POST /v1/runs/{run_id}/control`: unified control actions (`stop`, `approve`, `deny`, `resume`).
- `GET /v1/runs/{run_id}/diff` and `GET /v1/runs/{run_id}/patch`: run-bound artifacts.
- `packages/shared-core`: migrate from `Execution*` to `Run*` types for Desktop/Mobile sync.

## Test Strategy and Acceptance Gates

- Unit: config/state/safety/tools/model.
- Integration: Hub API + SSE + control + approval flow.
- Parity: golden comparisons versus original Kode-cli (stdout/stderr/exit code/interaction transcript).
- Interaction: PTY replay tests.
- Performance: cold start, first response latency, long-session stability.
- Cross-platform: macOS/Linux/Windows distributable verification.

All gates must pass before cutover.

## Regression Risks and Rollback

Primary regression risks:

1. `/v1` semantic rewrite breaks Desktop/Mobile event handling.
2. Minor output differences break scripts and automation.
3. Approval/state machine edge conditions cause deadlock or duplicate runs.
4. MCP/ACP timeout or error mapping mismatches.
5. Full one-shot cutover amplifies any missed defect.

Rollback requirements:

- Preserve last stable Hub/Desktop artifacts.
- Provide one-command rollback procedure.
- Validate `/health`, auth, conversation create, and message submit after rollback.

## Assumptions and Defaults

- "Do not change plan" means technical route and constraints remain fixed; only documentation governance and execution tracking are added.
- `docs/refactor` did not exist and is created by this work.
- Any unverified behavior must be recorded in pending confirmation list; no guess-based implementation.
- Historical execution data will be reset at migration cutover (no backward data migration).

## Change Record

| Date | Change | Author | Approval |
| --- | --- | --- | --- |
| 2026-02-25 | Initial frozen baseline document created under `docs/refactor`. | Codex | User directive in-thread |

