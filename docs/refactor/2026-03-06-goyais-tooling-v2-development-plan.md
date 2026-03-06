<!--
Copyright (c) 2026 Ysmjjsy
Author: Goya
SPDX-License-Identifier: MIT
-->

# Goyais Internal Tooling V2 Development Plan

## Mandatory Constraint

Before implementing any Tooling V2 capability:

1. Search the official Claude Code documentation for the capability.
2. Search [ClaudeHiddenToolkit.md](/Users/goya/Repo/Git/Goyais/docs/refactor/ClaudeHiddenToolkit.md) for the capability.
3. Prefer the official Claude Code recommendation when the two differ.
4. Record any intentional divergence in the implementation notes or commit message.

## Delivery Strategy

This plan assumes a one-shot cutover. New code replaces the old active path. Compatibility shims are not retained after the new path is verified.

## Phase 1: Capability Core

- Add unified capability types to `agent/core`.
- Add a capability resolver package that can build descriptors from builtin tools and MCP tools first.
- Introduce always-loaded vs searchable capability partitioning.
- Add tests for capability descriptor generation, scope precedence, conflict handling, and searchable threshold selection.

## Phase 2: Strong Runtime Config

- Extend runtime submit flow so `core.UserInput` carries a strong typed runtime config.
- Move model and tooling resolution away from metadata JSON bags.
- Keep metadata only for generic tracing data that is not used to reconstruct runtime behavior.
- Add tests proving the runtime executor prefers typed runtime config over metadata fallback.

## Phase 3: ToolSearch and Deferred MCP Exposure

- Add a first-class `ToolSearch` built-in tool.
- Feed searchable capability descriptors into the runner.
- Expose only always-loaded tools to the model prompt when the MCP budget threshold is exceeded.
- Return searchable capability details from `ToolSearch` with stable names and schemas.
- Add tests for threshold-triggered deferral and `ToolSearch` lookup behavior.

## Phase 4: Runtime Event Metadata

- Enrich `tool_call`, `tool_result`, and approval-related payloads with capability metadata.
- Preserve current event mapping semantics while adding the new fields.
- Add projection tests confirming the new fields survive runtime-to-execution event conversion.

## Phase 5: Workspace Agent Config and Snapshots

- Expand `WorkspaceAgentConfig` with Tooling V2 fields.
- Normalize defaults so existing callers sending the old minimal payload still receive a complete V2-shaped config.
- Expand execution snapshots to include resolved tooling data required by runtime and UI trace consumers.
- Update OpenAPI and shared TypeScript contracts.

## Phase 6: Extension Convergence

- Bring skills, slash commands, output styles, subagents, and plugins into the same capability catalog.
- Keep the first code pass focused on descriptor discovery and runtime identity, not on forcing every extension through the tool runner.
- Defer deeper UI editing surfaces until the shared contract is stable.

## Required Test Scenarios

- capability discovery precedence
- duplicate name conflict handling
- always-loaded vs searchable MCP partitioning
- `ToolSearch` result filtering
- hook -> sandbox -> permission -> approval ordering remains unchanged
- event payloads include capability metadata
- workspace agent config defaulting and persistence
- execution snapshots reflect the resolved V2 tooling state
- OpenAPI and shared-core type alignment

## Verification Commands

Minimum verification before claiming completion:

- `go test ./services/hub/internal/agent/...`
- `go test ./services/hub/internal/httpapi/...`

Repository verification required before final completion claim:

- `cd services/hub && go test ./... && go vet ./...`
- `pnpm lint`
- `pnpm test`
- `pnpm test:strict`
- `pnpm e2e:smoke`
- `pnpm lint:mobile`
- `pnpm test:mobile`
- `make health`

If any command is skipped due to time, environment, or unrelated failures, that must be stated explicitly in the completion report.
