# Goyais PRD Strict Refactor Plan (Truth-Based Progress, 2026-02-11)

## 1. Scope and Baseline

This document tracks strict PRD alignment by **current code truth** instead of historical assumptions.

- PRD baseline: `docs/prd.md`
- Strict contracts: `go_server/docs/api/openapi.yaml`, `go_server/docs/arch/*`, `go_server/docs/acceptance.md`
- Unified regression command:
  - `bash go_server/scripts/ci/contract_regression.sh`

Latest verification baseline (2026-02-11): regression pipeline passed (Go tests, Vue typecheck/tests, single-binary verification).

## 2. Current Truth Matrix

### 2.1 Completed

| Domain | Status | Evidence |
|---|---|---|
| Workflow engine backbone (DAG scheduling, retry, run-from-here, test-node) | Completed (backbone) | `go_server/internal/workflow/engine.go:69`, `go_server/internal/workflow/sqlite_engine.go:429` |
| Canvas orchestration core (minimap, undo/redo, run-from-here, test-node) | Completed (core interaction) | `vue_web/src/views/CanvasView.vue:103`, `vue_web/src/views/CanvasView.vue:744` |
| Stream control backend (`update-auth` / `delete`) | Completed (backend) | `go_server/internal/app/command_executors.go:1320`, `go_server/internal/access/http/streams.go:67` |
| AI workbench command pipeline and session/event chain | Completed (feature gated) | `go_server/internal/access/http/ai_context.go:25`, `go_server/internal/app/command_executors.go:518` |

### 2.2 Partially Completed

| Domain | Status | Evidence | Remaining Gap |
|---|---|---|---|
| AI planning semantics | Partial | `go_server/internal/app/command_executors.go:518`, `vue_web/src/views/AIWorkbenchView.vue:498` | Planner still rule-based; needs richer intent strategy and stronger explainability |
| Canvas strict semantics | Partial | `vue_web/src/views/CanvasView.vue:322` | Node taxonomy, AI patch governance, and replay depth need further hardening |
| ContextBundle rebuild | Partial | `go_server/internal/contextbundle/service.go:115` | Needs richer domain synthesis quality for large workspace scopes |

### 2.3 Gaps

| Domain | Status | Evidence | Target |
|---|---|---|---|
| PRD page coverage (independent Run Center / Algorithm Library / Permission Management / ContextBundle pages) | Gap | `vue_web/src/router/index.ts:24` | Add dedicated routes/views and keep Commands as audit view |
| Stream frontend parity for `update-auth` / `delete` | Gap | `vue_web/src/api/streams.ts:21`, `vue_web/src/views/StreamsView.vue:24` | Add API wrappers + UI actions + pre-check and errors |

## 3. Next Work Plan

## 3.1 P0 - Truth Document and Strict Semantic Gaps

### P0-A: Keep this document as source-of-truth progress ledger
- DoD:
  - Use only relative repository paths
  - Every gap line must include evidence, DoD, acceptance commands, risk, rollback
- Acceptance commands:
  - `rg -n '(/home/|[A-Za-z]:\\\\)' go_server/docs/prd-strict-refactor-plan.md` returns empty
  - `bash go_server/scripts/ci/contract_regression.sh`
- Risk:
  - Drift against `go_server/docs/acceptance.md`
- Rollback:
  - Revert only this doc commit; no runtime impact

### P0-B: AI strict semantics uplift (while keeping Command-first)
- Scope:
  - Extend planner from fixed token parser to extensible intent planner chain
  - Add controlled `workflow.patch` plan generation and explainable reject messages
  - Keep `ai.intent.plan` and `ai.command.execute` API paths stable
- DoD:
  - Same input yields deterministic, explainable plan output
  - Unsupported input returns reject reason + alternatives
  - Execution still goes through command/tool gates and audit
- Acceptance commands:
  - `go test ./internal/app ./internal/access/http`
  - `bash go_server/scripts/ci/contract_regression.sh`
- Risk:
  - Over-permissive planner may route unsafe command payload
- Rollback:
  - Disable AI workbench via feature gate (`GOYAIS_FEATURE_AI_WORKBENCH=false`)

### P0-C: Canvas strict semantics uplift
- Scope:
  - Enforce 7-node taxonomy: Input/Tool/Model/Algorithm/Transform/Control/Output
  - Add AI patch source marker + one-click apply + validation-failure explanation
  - Add run events + step details dual replay view
- DoD:
  - All seven node families can be created and connected under type constraints
  - AI patch UX surfaces source and validation reason
  - Replay includes event stream and step runtime details
- Acceptance commands:
  - `pnpm -C vue_web typecheck`
  - `pnpm -C vue_web test:run`
  - `bash go_server/scripts/ci/contract_regression.sh`
- Risk:
  - More state transitions may increase UI complexity and regression surface
- Rollback:
  - Keep previous panel actions and disable AI patch apply entry via guarded UI flag

## 3.2 P1 - Capability Alignment and Product Surface

### P1-A: Stream frontend parity
- Scope:
  - Add `updateStreamAuth` and `deleteStream` client API wrappers
  - Add status pre-check and localized error feedback in stream page
- DoD:
  - Frontend can execute both actions and command trail matches backend
- Acceptance commands:
  - `pnpm -C vue_web test:run`
  - `go test ./internal/access/http`
- Risk:
  - Deletion pre-check mismatch with backend policy
- Rollback:
  - Keep buttons hidden behind guard while preserving backend endpoints

### P1-B: ContextBundle practical synthesis quality
- Scope:
  - Rebuild aggregates real run/session/workspace references (not placeholder text)
  - Include useful facts/summaries/refs/timeline for auditing and replay context
- DoD:
  - Rebuilt bundle contains real IDs and status references from workflow/ai/command/asset domains
- Acceptance commands:
  - `go test ./internal/contextbundle ./internal/access/http`
  - `bash go_server/scripts/ci/contract_regression.sh`
- Risk:
  - Multi-service reads may fail partially under provider degradation
- Rollback:
  - Fallback to minimal payload mode by removing optional readers from wiring

### P1-C: Product page coverage closure
- Scope:
  - Add independent pages: Run Center / Algorithm Library / Permission Management / ContextBundle
  - Keep `/commands` as audit-oriented runtime history page
- DoD:
  - New routes are accessible and support window-panel interaction (drag/resize/fullscreen)
- Acceptance commands:
  - `pnpm -C vue_web typecheck`
  - `pnpm -C vue_web test:run`
- Risk:
  - i18n or nav drift across layouts
- Rollback:
  - Route-level rollback without touching backend APIs

## 4. API and Contract Stability

- Keep `/api/v1` prefix unchanged.
- Keep command-first semantics unchanged.
- Keep existing AI endpoints (`ai.intent.plan`, `ai.command.execute`) unchanged.
- Expected frontend expansions:
  - Stream actions: `POST /streams/{id}:update-auth`, `DELETE /streams/{id}`
  - Workflow replay consumption: `GET /workflow-runs/{runId}/events`

## 5. Acceptance Command Set (Fixed)

- `go test ./...` (under `go_server/`)
- `pnpm -C vue_web typecheck`
- `pnpm -C vue_web test:run`
- `bash go_server/scripts/ci/contract_regression.sh`

## 6. Change Discipline

For any API/entity/state-machine/ACL/static-routing change, update in the same PR:

- `go_server/docs/api/openapi.yaml`
- `go_server/docs/arch/data-model.md`
- `go_server/docs/arch/state-machines.md`
- `go_server/docs/arch/overview.md`
- `go_server/docs/acceptance.md`
