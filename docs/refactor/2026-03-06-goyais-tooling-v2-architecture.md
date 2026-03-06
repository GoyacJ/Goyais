<!--
Copyright (c) 2026 Ysmjjsy
Author: Goya
SPDX-License-Identifier: MIT
-->

# Goyais Internal Tooling V2 Architecture

## Mandatory Constraint

Before implementing any Tooling V2 feature:

1. Search the official Claude Code documentation for the target capability and preferred implementation pattern.
2. Search [ClaudeHiddenToolkit.md](/Users/goya/Repo/Git/Goyais/docs/refactor/ClaudeHiddenToolkit.md) for the same capability and compare the documented behavior.
3. Use the official Claude Code documentation as the primary source of truth.
4. Use `ClaudeHiddenToolkit.md` only as a supplementary reference for product behavior that is absent or under-specified in the official documentation.
5. Do not design or implement a capability without completing both searches first.

## Problem Statement

The current Agent v4 runtime unified the tool execution pipeline, but it did not unify the broader capability system around that pipeline.

Current structural problems:

- Tool execution is centralized, but capability discovery, loading, exposure, configuration, and audit are still split across unrelated modules.
- Runtime tooling is injected through metadata string bags such as `builtin_tools_json`, `mcp_servers_json`, and `rules_dsl`, which prevents strong typing and makes the execution path opaque.
- MCP tools are still modeled as static server configuration plus weak runtime schemas. There is no first-class support for searchable/deferred exposure or prompt-budget-aware loading.
- Skills, slash commands, output styles, subagents, and plugins already exist, but they are parallel extension points instead of one capability system.
- `WorkspaceAgentConfig` only configures turn limits and trace display. It does not govern the actual tooling surface.

## Design Goals

- Build one capability graph for all runtime-visible agent extensions.
- Separate capability declaration, discovery/loading, and execution.
- Replace metadata string bag handoff with strong runtime snapshots.
- Add searchable/deferred capability exposure for MCP tools based on prompt budget.
- Keep the existing execution order for side-effecting tools: hook -> sandbox -> permission -> approval -> execute -> interaction.
- Make run events auditable with capability identity, source, scope, and risk metadata.

## Non-Goals

- Reproducing Claude.ai consumer-only hidden tools verbatim.
- Requiring the Desktop UI to expose every new V2 field in one pass.
- Keeping backward compatibility with the old runtime metadata bag.

## Target Model

Tooling V2 uses three layers.

### 1. Capability Declaration Layer

Introduce a unified `CapabilityDescriptor` with these required fields:

- `id`
- `kind`
- `name`
- `description`
- `source`
- `scope`
- `version`
- `input_schema`
- `risk_level`
- `read_only`
- `concurrency_safe`
- `requires_permissions`
- `visibility_policy`
- `prompt_budget_cost`

Supported kinds:

- `builtin_tool`
- `mcp_tool`
- `mcp_prompt`
- `skill`
- `slash_command`
- `subagent`
- `output_style`

`tools/spec.ToolSpec` remains as an execution-facing tool contract for tool-capable descriptors only. It is no longer the top-level capability model.

### 2. Capability Discovery and Loading Layer

Add a unified resolver that merges:

- built-in tools
- workspace MCP tools
- local/project/user skills
- slash commands
- output styles
- subagents
- plugins

Resolver responsibilities:

- apply scope precedence and conflict resolution
- compute prompt budget cost
- partition runtime-visible capabilities into `always_loaded` and `searchable`
- enable searchable/deferred MCP exposure when the total MCP description cost exceeds the configured budget threshold
- preserve stable name lookup so runtime events and approval payloads can reference a single resolved descriptor

### 3. Runtime Execution Layer

Execution remains ordered:

- pre-hook
- sandbox
- permission gate
- approval
- execute
- interaction

But dispatch now resolves through capability metadata instead of ad hoc name-prefix checks.

Current execution mapping:

- `builtin_tool` -> builtin runner
- `mcp_tool` -> MCP runner
- `mcp_prompt` -> prompt resolver path
- `subagent` -> subagent runner
- interaction payloads -> interaction bridge
- plugin-backed capabilities -> future runner path

The model loop consumes a capability provider instead of concatenating builtin and MCP tool lists manually.

## Runtime Snapshot Strategy

The runtime handoff must use strong typed snapshots carried by `core.UserInput.RuntimeConfig`.

Required runtime snapshot content:

- model configuration
- permission mode
- merged rules DSL
- selected MCP server definitions
- always-loaded capabilities
- searchable capabilities
- tool-search enablement state
- budget values that affected the resolution

The old runtime metadata JSON bag must be removed from the active execution path.

## Public Contract Changes

`WorkspaceAgentConfig` expands to include at least:

- `default_mode`
- `builtin_tools`
- `capability_budgets`
- `mcp_search`
- `output_style`
- `subagent_defaults`
- `feature_flags`

`ExecutionAgentConfigSnapshot` and `ExecutionResourceProfile` must capture the resolved Tooling V2 state instead of relying on transport metadata reconstruction.

Run events for `tool_call` and `tool_result` must include:

- `capability_kind`
- `capability_source`
- `capability_scope`
- `resolved_name`
- `risk_level`

## Deleted Paths

Tooling V2 removes these design paths from the active runtime:

- runtime metadata JSON bags for builtin tools, MCP servers, and rules DSL
- split builtin-vs-MCP registration entrypoints as the source of truth
- MCP prompt discovery that only exists in prompt assembly and does not participate in capability governance
- simplified workspace agent config limited to max turns and trace display

## Validation Gates

Required validation commands for this refactor:

- `cd services/hub && go test ./... && go vet ./...`
- `pnpm lint`
- `pnpm test`
- `pnpm test:strict`
- `pnpm e2e:smoke`
- `pnpm lint:mobile`
- `pnpm test:mobile`
- `make health`

The implementation may stage these checks incrementally, but Tooling V2 is not complete until the full gate is green.
