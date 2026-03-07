// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

// Package session provides lifecycle operations for Agent v4 sessions.
//
// It centralizes resume/fork/rewind/clear semantics so adapters can reuse
// one consistent implementation while keeping persistence and transport as
// injected boundaries.
package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"goyais/services/hub/internal/agent/core"
)

// SessionStarter abstracts session creation so lifecycle logic does not depend
// on concrete runtime loop implementations.
type SessionStarter interface {
	StartSession(ctx context.Context, req core.StartSessionRequest) (core.SessionHandle, error)
}

// Dependencies contains optional bridge dependencies for lifecycle operations.
type Dependencies struct {
	Starter         SessionStarter
	CheckpointStore core.CheckpointStore
}

// RegisterRequest records one runtime session into lifecycle tracking.
type RegisterRequest struct {
	Handle                core.SessionHandle
	ParentSessionID       core.SessionID
	WorkingDir            string
	AdditionalDirectories []string
	PermissionMode        core.PermissionMode
	TemporaryPermissions  []string
	HistoryEntries        int
	Summary               string
	LastCheckpointID      core.CheckpointID
	NextCursor            int64
}

// ResumeRequest resumes one existing session.
type ResumeRequest struct {
	SessionID core.SessionID
}

// ForkRequest creates one child session inheriting parent context.
type ForkRequest struct {
	SessionID             core.SessionID
	WorkingDir            string
	AdditionalDirectories []string
}

// RewindRequest rewinds one session to the given checkpoint/cursor boundary.
type RewindRequest struct {
	SessionID     core.SessionID
	CheckpointID  core.CheckpointID
	TargetCursor  int64
	ClearTempPerm bool
}

// ClearRequest clears conversation context and transient session state.
type ClearRequest struct {
	SessionID core.SessionID
	Reason    string
}

// HandoffTarget identifies the destination client surface for session transfer.
type HandoffTarget string

const (
	// HandoffTargetDesktop transfers session control to desktop clients.
	HandoffTargetDesktop HandoffTarget = "desktop"
	// HandoffTargetMobile transfers session control to mobile clients.
	HandoffTargetMobile HandoffTarget = "mobile"
)

// HandoffRequest requests one cross-surface session transfer snapshot.
type HandoffRequest struct {
	SessionID          core.SessionID
	Target             HandoffTarget
	PendingTaskSummary string
}

// HandoffSnapshot is the stable payload sent to the destination surface.
//
// Temporary permissions are intentionally excluded: handoff keeps identity,
// permission mode, and unfinished-task context, but does not propagate
// ephemeral grants across surfaces.
type HandoffSnapshot struct {
	SessionID             core.SessionID
	Target                HandoffTarget
	WorkingDir            string
	AdditionalDirectories []string
	PermissionMode        core.PermissionMode
	HistoryEntries        int
	Summary               string
	PendingTaskSummary    string
	LastCheckpointID      core.CheckpointID
	NextCursor            int64
	IssuedAt              time.Time
}

// State is the copy-safe session lifecycle snapshot returned by Manager APIs.
type State struct {
	SessionID             core.SessionID
	ParentSessionID       core.SessionID
	WorkingDir            string
	AdditionalDirectories []string
	PermissionMode        core.PermissionMode
	TemporaryPermissions  []string
	HistoryEntries        int
	Summary               string
	LastCheckpointID      core.CheckpointID
	NextCursor            int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
	LastClearedReason     string
	LastHandoffTarget     HandoffTarget
	LastHandoffAt         time.Time
}

// Manager owns runtime-agnostic session lifecycle state.
type Manager struct {
	starter         SessionStarter
	checkpointStore core.CheckpointStore

	mu       sync.RWMutex
	sessions map[core.SessionID]*State
}

// NewManager creates a lifecycle manager with optional dependency bridges.
func NewManager(deps Dependencies) *Manager {
	return &Manager{
		starter:         deps.Starter,
		checkpointStore: deps.CheckpointStore,
		sessions:        map[core.SessionID]*State{},
	}
}

// Register stores one session snapshot for later lifecycle operations.
func (m *Manager) Register(req RegisterRequest) (State, error) {
	if m == nil {
		return State{}, errors.New("session manager is nil")
	}
	if err := req.Handle.Validate(); err != nil {
		return State{}, err
	}
	sessionID := normalizeSessionID(req.Handle.SessionID)
	if sessionID == "" {
		return State{}, core.ErrSessionNotFound
	}
	if req.HistoryEntries < 0 {
		return State{}, errors.New("history_entries must be >= 0")
	}
	if req.NextCursor < 0 {
		return State{}, errors.New("next_cursor must be >= 0")
	}

	mode := normalizePermissionMode(req.PermissionMode)
	now := time.Now().UTC()
	state := &State{
		SessionID:             sessionID,
		ParentSessionID:       normalizeSessionID(req.ParentSessionID),
		WorkingDir:            strings.TrimSpace(req.WorkingDir),
		AdditionalDirectories: sanitizeDirectories(req.AdditionalDirectories),
		PermissionMode:        mode,
		TemporaryPermissions:  sanitizePermissions(req.TemporaryPermissions),
		HistoryEntries:        req.HistoryEntries,
		Summary:               strings.TrimSpace(req.Summary),
		LastCheckpointID:      normalizeCheckpointID(req.LastCheckpointID),
		NextCursor:            req.NextCursor,
		CreatedAt:             req.Handle.CreatedAt.UTC(),
		UpdatedAt:             now,
	}

	m.mu.Lock()
	m.sessions[sessionID] = state
	m.mu.Unlock()
	return cloneState(state), nil
}

// Resume returns existing session state while dropping temporary permissions.
func (m *Manager) Resume(_ context.Context, req ResumeRequest) (State, error) {
	if m == nil {
		return State{}, errors.New("session manager is nil")
	}
	sessionID := normalizeSessionID(req.SessionID)
	if sessionID == "" {
		return State{}, core.ErrSessionNotFound
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.sessions[sessionID]
	if state == nil {
		return State{}, core.ErrSessionNotFound
	}
	state.TemporaryPermissions = nil
	state.UpdatedAt = time.Now().UTC()
	return cloneState(state), nil
}

// Fork creates a child session that inherits history but not temporary grants.
func (m *Manager) Fork(ctx context.Context, req ForkRequest) (State, error) {
	if m == nil {
		return State{}, errors.New("session manager is nil")
	}
	if m.starter == nil {
		return State{}, errors.New("session starter is not configured")
	}
	parentID := normalizeSessionID(req.SessionID)
	if parentID == "" {
		return State{}, core.ErrSessionNotFound
	}

	m.mu.RLock()
	parent := m.sessions[parentID]
	m.mu.RUnlock()
	if parent == nil {
		return State{}, core.ErrSessionNotFound
	}

	workingDir := strings.TrimSpace(req.WorkingDir)
	if workingDir == "" {
		workingDir = parent.WorkingDir
	}
	additionalDirs := sanitizeDirectories(req.AdditionalDirectories)
	if len(additionalDirs) == 0 {
		additionalDirs = sanitizeDirectories(parent.AdditionalDirectories)
	}

	handle, err := m.starter.StartSession(ctx, core.StartSessionRequest{
		WorkingDir:            workingDir,
		AdditionalDirectories: additionalDirs,
	})
	if err != nil {
		return State{}, fmt.Errorf("start forked session failed: %w", err)
	}

	now := time.Now().UTC()
	child := &State{
		SessionID:             normalizeSessionID(handle.SessionID),
		ParentSessionID:       parentID,
		WorkingDir:            workingDir,
		AdditionalDirectories: additionalDirs,
		PermissionMode:        normalizePermissionMode(parent.PermissionMode),
		TemporaryPermissions:  nil,
		HistoryEntries:        parent.HistoryEntries,
		Summary:               parent.Summary,
		LastCheckpointID:      parent.LastCheckpointID,
		NextCursor:            0,
		CreatedAt:             handle.CreatedAt.UTC(),
		UpdatedAt:             now,
	}
	if child.SessionID == "" {
		return State{}, core.ErrSessionNotFound
	}

	m.mu.Lock()
	m.sessions[child.SessionID] = child
	m.mu.Unlock()
	return cloneState(child), nil
}

// Rewind restores a checkpoint and rewinds session cursor metadata.
func (m *Manager) Rewind(ctx context.Context, req RewindRequest) (State, error) {
	if m == nil {
		return State{}, errors.New("session manager is nil")
	}
	if m.checkpointStore == nil {
		return State{}, errors.New("checkpoint store is not configured")
	}
	sessionID := normalizeSessionID(req.SessionID)
	if sessionID == "" {
		return State{}, core.ErrSessionNotFound
	}
	checkpointID := normalizeCheckpointID(req.CheckpointID)
	if checkpointID == "" {
		return State{}, errors.New("checkpoint_id is required")
	}
	if req.TargetCursor < 0 {
		return State{}, errors.New("target_cursor must be >= 0")
	}

	if err := m.checkpointStore.Restore(ctx, checkpointID); err != nil {
		return State{}, fmt.Errorf("restore checkpoint failed: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.sessions[sessionID]
	if state == nil {
		return State{}, core.ErrSessionNotFound
	}
	state.LastCheckpointID = checkpointID
	state.NextCursor = req.TargetCursor
	if req.ClearTempPerm {
		state.TemporaryPermissions = nil
	}
	state.UpdatedAt = time.Now().UTC()
	return cloneState(state), nil
}

// Clear removes accumulated context while preserving stable session identity.
func (m *Manager) Clear(_ context.Context, req ClearRequest) (State, error) {
	if m == nil {
		return State{}, errors.New("session manager is nil")
	}
	sessionID := normalizeSessionID(req.SessionID)
	if sessionID == "" {
		return State{}, core.ErrSessionNotFound
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.sessions[sessionID]
	if state == nil {
		return State{}, core.ErrSessionNotFound
	}
	state.HistoryEntries = 0
	state.Summary = ""
	state.NextCursor = 0
	state.LastCheckpointID = ""
	state.TemporaryPermissions = nil
	state.LastClearedReason = strings.TrimSpace(req.Reason)
	state.UpdatedAt = time.Now().UTC()
	return cloneState(state), nil
}

// Handoff builds a cross-surface snapshot for desktop/mobile transfer.
func (m *Manager) Handoff(_ context.Context, req HandoffRequest) (HandoffSnapshot, error) {
	if m == nil {
		return HandoffSnapshot{}, errors.New("session manager is nil")
	}
	sessionID := normalizeSessionID(req.SessionID)
	if sessionID == "" {
		return HandoffSnapshot{}, core.ErrSessionNotFound
	}
	target, ok := normalizeHandoffTarget(req.Target)
	if !ok {
		return HandoffSnapshot{}, errors.New("handoff target must be desktop or mobile")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.sessions[sessionID]
	if state == nil {
		return HandoffSnapshot{}, core.ErrSessionNotFound
	}

	pendingTask := strings.TrimSpace(req.PendingTaskSummary)
	if pendingTask == "" {
		pendingTask = state.Summary
	}
	now := time.Now().UTC()
	state.LastHandoffTarget = target
	state.LastHandoffAt = now
	state.UpdatedAt = now

	return HandoffSnapshot{
		SessionID:             state.SessionID,
		Target:                target,
		WorkingDir:            state.WorkingDir,
		AdditionalDirectories: sanitizeDirectories(state.AdditionalDirectories),
		PermissionMode:        state.PermissionMode,
		HistoryEntries:        state.HistoryEntries,
		Summary:               state.Summary,
		PendingTaskSummary:    pendingTask,
		LastCheckpointID:      state.LastCheckpointID,
		NextCursor:            state.NextCursor,
		IssuedAt:              now,
	}, nil
}

func normalizeSessionID(input core.SessionID) core.SessionID {
	return core.SessionID(strings.TrimSpace(string(input)))
}

func normalizeCheckpointID(input core.CheckpointID) core.CheckpointID {
	return core.CheckpointID(strings.TrimSpace(string(input)))
}

func normalizePermissionMode(mode core.PermissionMode) core.PermissionMode {
	switch mode {
	case core.PermissionModeDefault,
		core.PermissionModeAcceptEdits,
		core.PermissionModePlan,
		core.PermissionModeDontAsk,
		core.PermissionModeBypassPermissions:
		return mode
	default:
		return core.PermissionModeDefault
	}
}

func normalizeHandoffTarget(target HandoffTarget) (HandoffTarget, bool) {
	switch HandoffTarget(strings.ToLower(strings.TrimSpace(string(target)))) {
	case HandoffTargetDesktop:
		return HandoffTargetDesktop, true
	case HandoffTargetMobile:
		return HandoffTargetMobile, true
	default:
		return "", false
	}
}

func sanitizeDirectories(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func sanitizePermissions(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneState(input *State) State {
	if input == nil {
		return State{}
	}
	return State{
		SessionID:             input.SessionID,
		ParentSessionID:       input.ParentSessionID,
		WorkingDir:            input.WorkingDir,
		AdditionalDirectories: sanitizeDirectories(input.AdditionalDirectories),
		PermissionMode:        input.PermissionMode,
		TemporaryPermissions:  sanitizePermissions(input.TemporaryPermissions),
		HistoryEntries:        input.HistoryEntries,
		Summary:               input.Summary,
		LastCheckpointID:      input.LastCheckpointID,
		NextCursor:            input.NextCursor,
		CreatedAt:             input.CreatedAt,
		UpdatedAt:             input.UpdatedAt,
		LastClearedReason:     input.LastClearedReason,
		LastHandoffTarget:     input.LastHandoffTarget,
		LastHandoffAt:         input.LastHandoffAt,
	}
}
