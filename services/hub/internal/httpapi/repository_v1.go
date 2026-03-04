// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Team
// SPDX-License-Identifier: MIT

// Package httpapi defines Hub runtime v1 repository contracts used by
// handler/service code so runtime persistence can move away from in-memory
// AppState maps to sqlite-backed repositories.
package httpapi

import "context"

const (
	hubRuntimeV1SchemaComponent = "hub_runtime_v1"
	hubRuntimeV1SchemaVersion   = "1"
)

// RepositoryPage defines deterministic offset pagination for repository queries.
type RepositoryPage struct {
	Limit  int
	Offset int
}

func (p RepositoryPage) normalize(defaultLimit int, maxLimit int) RepositoryPage {
	limit := p.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if maxLimit > 0 && limit > maxLimit {
		limit = maxLimit
	}

	offset := p.Offset
	if offset < 0 {
		offset = 0
	}

	return RepositoryPage{Limit: limit, Offset: offset}
}

// RuntimeSessionRecord is the repository shape for runtime session entities.
type RuntimeSessionRecord struct {
	ID            string
	WorkspaceID   string
	ProjectID     string
	Name          string
	DefaultMode   string
	ModelConfigID string
	RuleIDs       []string
	SkillIDs      []string
	MCPIDs        []string
	ActiveRunID   *string
	CreatedAt     string
	UpdatedAt     string
}

// RuntimeRunRecord is the repository shape for runtime run entities.
type RuntimeRunRecord struct {
	ID            string
	SessionID     string
	WorkspaceID   string
	MessageID     string
	State         string
	Mode          string
	ModelID       string
	ModelConfigID string
	TokensIn      int
	TokensOut     int
	TraceID       string
	CreatedAt     string
	UpdatedAt     string
}

// RuntimeRunEventRecord is the repository shape for runtime run event entities.
type RuntimeRunEventRecord struct {
	EventID    string
	RunID      string
	SessionID  string
	Sequence   int64
	Type       string
	Timestamp  string
	Payload    map[string]any
	OccurredAt string
}

// RuntimeRunTaskRecord is the repository shape for runtime run task entities.
type RuntimeRunTaskRecord struct {
	TaskID       string
	RunID        string
	ParentTaskID *string
	Title        string
	State        string
	Metadata     map[string]any
	CreatedAt    string
	UpdatedAt    string
	FinishedAt   *string
}

// RuntimeChangeSetRecord is the repository shape for runtime changeset entities.
type RuntimeChangeSetRecord struct {
	ChangeSetID string
	SessionID   string
	RunID       *string
	Payload     map[string]any
	CreatedAt   string
	UpdatedAt   string
}

// RuntimeHookRecord is the repository shape for runtime hook execution entities.
type RuntimeHookRecord struct {
	ID        string
	RunID     string
	SessionID string
	TaskID    *string
	Event     string
	ToolName  *string
	PolicyID  *string
	Decision  HookDecision
	Timestamp string
}

// RuntimeSessionRepository defines persistence operations for runtime sessions.
type RuntimeSessionRepository interface {
	ReplaceAll(ctx context.Context, items []RuntimeSessionRecord) error
	GetByID(ctx context.Context, sessionID string) (RuntimeSessionRecord, bool, error)
	ListByWorkspace(ctx context.Context, workspaceID string, page RepositoryPage) ([]RuntimeSessionRecord, error)
}

// RuntimeRunRepository defines persistence operations for runtime runs.
type RuntimeRunRepository interface {
	ReplaceAll(ctx context.Context, items []RuntimeRunRecord) error
	GetByID(ctx context.Context, runID string) (RuntimeRunRecord, bool, error)
	ListByWorkspace(ctx context.Context, workspaceID string, page RepositoryPage) ([]RuntimeRunRecord, error)
	ListBySession(ctx context.Context, sessionID string, page RepositoryPage) ([]RuntimeRunRecord, error)
}

// RuntimeRunEventRepository defines persistence operations for runtime run events.
type RuntimeRunEventRepository interface {
	ReplaceAll(ctx context.Context, items []RuntimeRunEventRecord) error
	ListBySession(ctx context.Context, sessionID string, afterSequence int64, limit int) ([]RuntimeRunEventRecord, error)
}

// RuntimeRunTaskRepository defines persistence operations for runtime run tasks.
type RuntimeRunTaskRepository interface {
	ReplaceAll(ctx context.Context, items []RuntimeRunTaskRecord) error
	ListByRun(ctx context.Context, runID string, page RepositoryPage) ([]RuntimeRunTaskRecord, error)
}

// RuntimeChangeSetRepository defines persistence operations for runtime changesets.
type RuntimeChangeSetRepository interface {
	ReplaceAll(ctx context.Context, items []RuntimeChangeSetRecord) error
	ListBySession(ctx context.Context, sessionID string, page RepositoryPage) ([]RuntimeChangeSetRecord, error)
}

// RuntimeHookRecordRepository defines persistence operations for hook execution records.
type RuntimeHookRecordRepository interface {
	ReplaceAll(ctx context.Context, items []RuntimeHookRecord) error
	ListByRun(ctx context.Context, runID string, page RepositoryPage) ([]RuntimeHookRecord, error)
}

// RuntimeV1RepositorySet groups all runtime v1 repositories for service wiring.
type RuntimeV1RepositorySet struct {
	Sessions    RuntimeSessionRepository
	Runs        RuntimeRunRepository
	RunEvents   RuntimeRunEventRepository
	RunTasks    RuntimeRunTaskRepository
	ChangeSets  RuntimeChangeSetRepository
	HookRecords RuntimeHookRecordRepository
}
