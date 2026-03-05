// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package core defines the contract interfaces, shared types, and error
// semantics for the Agent v4 runtime. All sub-packages and adapters depend
// on these definitions; core itself has zero internal dependencies beyond
// the standard library.
//
// This file is the immovable anchor for the entire Agent v4 refactor.
// Every implementation produced by later phases (A2–F) MUST satisfy
// the interfaces defined here. Changes to these signatures require
// explicit justification and version bump in the architecture governance docs.
//
// Ref: docs/site/guide/overview.md §10.2
package core

import "context"

// ──────────────────────────────────────────────────────────────────────
// Engine — unified execution runtime (§3, §10.2)
// ──────────────────────────────────────────────────────────────────────

// Engine is the unified execution runtime.
// CLI, ACP, and HTTP all funnel through a single Engine instance.
// Replaces the split between agentcore.Engine (stub) and
// legacy HTTP execution paths.
type Engine interface {
	// StartSession creates a new agent session and returns a handle.
	StartSession(ctx context.Context, req StartSessionRequest) (SessionHandle, error)

	// Submit sends user input to the session, starting a new run.
	// Returns the assigned run ID.
	Submit(ctx context.Context, sessionID string, input UserInput) (runID string, err error)

	// Control applies an external control action (stop, approve, deny,
	// resume, answer) to an active run.
	Control(ctx context.Context, runID string, action ControlAction) error

	// Subscribe returns an event subscription for the given session.
	// Events are delivered from the given cursor position onward.
	// The caller MUST call EventSubscription.Close when done.
	Subscribe(ctx context.Context, sessionID string, cursor string) (EventSubscription, error)
}

// ──────────────────────────────────────────────────────────────────────
// CommandBus — slash command dispatch (§9.1)
// ──────────────────────────────────────────────────────────────────────

// CommandBus dispatches slash commands without creating a fake run.
// Replaces the buildSlashEvents() hack that forged
// run_queued → run_completed event sequences.
type CommandBus interface {
	// Execute runs a slash command and returns the result through a
	// dedicated CommandResponse channel, not via fake RunEvents.
	Execute(ctx context.Context, sessionID string, cmd SlashCommand) (CommandResponse, error)
}

// ──────────────────────────────────────────────────────────────────────
// ToolExecutor — tool call pipeline (§10.2, EO group G)
// ──────────────────────────────────────────────────────────────────────

// ToolExecutor runs a single tool call through the
// pre-hook → approval → execute → retry → post-hook pipeline.
type ToolExecutor interface {
	Execute(ctx context.Context, call ToolCall) (ToolResult, error)
}

// ──────────────────────────────────────────────────────────────────────
// HookDispatcher — hook evaluation & handler execution (§7.1)
// ──────────────────────────────────────────────────────────────────────

// HookDispatcher evaluates hook rules and runs matched handlers.
// Supports 17 event types × 4 handler types (command, http, prompt,
// agent) with per-event return schemas.
type HookDispatcher interface {
	Dispatch(ctx context.Context, event HookEvent) (HookDecision, error)
}

// ──────────────────────────────────────────────────────────────────────
// SkillLoader — skill discovery & resolution (§7.2)
// ──────────────────────────────────────────────────────────────────────

// SkillLoader discovers and resolves skill definitions across
// enterprise, personal, and project scopes.
type SkillLoader interface {
	// Discover returns all skills visible in the given scope.
	Discover(ctx context.Context, scope SkillScope) ([]SkillMeta, error)

	// Resolve loads the full skill definition for a given reference.
	Resolve(ctx context.Context, ref SkillRef) (SkillDefinition, error)
}

// ──────────────────────────────────────────────────────────────────────
// SubagentRunner — child agent execution (§7.4)
// ──────────────────────────────────────────────────────────────────────

// SubagentRunner launches isolated child agents.
// Nesting depth is fixed at 1 (no nested subagents).
type SubagentRunner interface {
	Run(ctx context.Context, req SubagentRequest) (SubagentResult, error)
}

// ──────────────────────────────────────────────────────────────────────
// TeamCoordinator — agent team collaboration (§7.5)
// ──────────────────────────────────────────────────────────────────────

// TeamCoordinator manages shared task lists, direct messaging, and
// plan approval flows between teammates.
type TeamCoordinator interface {
	// Assign adds or updates a task on the shared task list.
	Assign(ctx context.Context, task TeamTask) error

	// Inbox retrieves pending messages for the given agent.
	Inbox(ctx context.Context, agentID string) ([]TeamMessage, error)
}

// ──────────────────────────────────────────────────────────────────────
// ContextBuilder — prompt assembly (§5)
// ──────────────────────────────────────────────────────────────────────

// ContextBuilder assembles the system prompt by merging instructions,
// settings, rules, memory, skills, and MCP tool definitions in the
// strict loading order defined in §5.2.
type ContextBuilder interface {
	Build(ctx context.Context, req BuildContextRequest) (PromptContext, error)
}

// ──────────────────────────────────────────────────────────────────────
// PermissionGate — permission evaluation (§8.1, §8.2)
// ──────────────────────────────────────────────────────────────────────

// PermissionGate evaluates tool calls against the three-layer rule
// chain (deny → ask → allow) and the active permission mode matrix.
type PermissionGate interface {
	Evaluate(ctx context.Context, req PermissionRequest) (PermissionDecision, error)
}

// ──────────────────────────────────────────────────────────────────────
// CheckpointStore — file-level snapshot & restore (§9.3)
// ──────────────────────────────────────────────────────────────────────

// CheckpointStore provides file-level snapshotting and restore,
// independent of git, for edit-step rollback and session rewind.
type CheckpointStore interface {
	// Snapshot creates a point-in-time snapshot of the specified files.
	Snapshot(ctx context.Context, req SnapshotRequest) (CheckpointID, error)

	// Restore rolls back files to the state captured by the given checkpoint.
	Restore(ctx context.Context, id CheckpointID) error
}
