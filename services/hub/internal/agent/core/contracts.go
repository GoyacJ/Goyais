// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package core

import (
	"errors"
	"strings"
	"time"
)

// SlashCommand is the normalized representation of a slash invocation.
type SlashCommand struct {
	Name      string
	Arguments []string
	Raw       string
}

// Validate ensures command dispatch has a concrete command target.
func (c SlashCommand) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("slash command name is required")
	}
	return nil
}

// CommandResponse is the structured result returned by CommandBus.
type CommandResponse struct {
	Output   string
	Metadata map[string]any
}

// ToolCall represents one tool execution request inside a run.
type ToolCall struct {
	RunID     RunID
	SessionID SessionID
	ToolName  string
	Input     map[string]any
}

// Validate guarantees call identity and target tool name are present.
func (c ToolCall) Validate() error {
	if strings.TrimSpace(string(c.RunID)) == "" {
		return errors.New("run_id is required")
	}
	if strings.TrimSpace(string(c.SessionID)) == "" {
		return errors.New("session_id is required")
	}
	if strings.TrimSpace(c.ToolName) == "" {
		return errors.New("tool_name is required")
	}
	return nil
}

// ToolResult is the normalized output of a tool call pipeline.
type ToolResult struct {
	ToolName string
	Output   map[string]any
	Diff     []DiffItem
	Error    *RunError
}

// HookEvent is the unified hook input envelope.
type HookEvent struct {
	Type      string
	SessionID SessionID
	RunID     RunID
	Payload   map[string]any
}

// HookDecision expresses the policy result of hook evaluation.
type HookDecision struct {
	Decision        string
	MatchedPolicyID string
	Metadata        map[string]any
}

// SkillScope identifies where a skill was discovered.
type SkillScope string

const (
	SkillScopeEnterprise SkillScope = "enterprise"
	SkillScopePersonal   SkillScope = "personal"
	SkillScopeProject    SkillScope = "project"
	SkillScopeLocal      SkillScope = "local"
	SkillScopeManaged    SkillScope = "managed"
)

// SkillMeta contains catalog-level metadata for a skill.
type SkillMeta struct {
	Name        string
	Description string
	Source      string
}

// SkillRef identifies a concrete skill in one scope.
type SkillRef struct {
	Scope SkillScope
	Name  string
}

// SkillDefinition is the fully resolved skill document plus parsed metadata.
type SkillDefinition struct {
	Meta        SkillMeta
	Frontmatter map[string]any
	Body        string
}

// SubagentRequest defines one child-agent execution request.
type SubagentRequest struct {
	AgentName    string
	Prompt       string
	AllowedTools []string
	MaxTurns     int
}

// SubagentResult is the summarized outcome returned to the parent agent.
type SubagentResult struct {
	Summary        string
	TranscriptPath string
}

// TeamTask is a shared task-list entry used by TeamCoordinator.
type TeamTask struct {
	ID        string
	Title     string
	Status    string
	DependsOn []string
}

// TeamMessage is one directed mailbox message between teammates.
type TeamMessage struct {
	ID        string
	FromAgent string
	ToAgent   string
	Body      string
	SentAt    time.Time
}

// BuildContextRequest defines the minimum inputs for prompt assembly.
type BuildContextRequest struct {
	SessionID  SessionID
	WorkingDir string
	UserInput  string
}

// PromptSection is one attributable segment of the final prompt context.
type PromptSection struct {
	Source  string
	Content string
}

// PromptContext is the assembled prompt output consumed by runtime/model.
type PromptContext struct {
	SystemPrompt string
	Sections     []PromptSection
}

// PermissionMode selects the execution permission policy profile.
type PermissionMode string

const (
	PermissionModeDefault           PermissionMode = "default"
	PermissionModeAcceptEdits       PermissionMode = "acceptEdits"
	PermissionModePlan              PermissionMode = "plan"
	PermissionModeDontAsk           PermissionMode = "dontAsk"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// PermissionRequest is evaluated by PermissionGate.
type PermissionRequest struct {
	Mode       PermissionMode
	ToolName   string
	Arguments  string
	WorkingDir string
}

// PermissionDecisionKind is the tri-state policy result.
type PermissionDecisionKind string

const (
	PermissionDecisionAllow PermissionDecisionKind = "allow"
	PermissionDecisionAsk   PermissionDecisionKind = "ask"
	PermissionDecisionDeny  PermissionDecisionKind = "deny"
)

// PermissionDecision contains the verdict and traceability details.
type PermissionDecision struct {
	Kind        PermissionDecisionKind
	Reason      string
	MatchedRule string
}

// CheckpointID is the stable handle for a file snapshot.
type CheckpointID string

// SnapshotRequest describes files to snapshot before mutable operations.
type SnapshotRequest struct {
	SessionID SessionID
	Paths     []string
	Reason    string
}
