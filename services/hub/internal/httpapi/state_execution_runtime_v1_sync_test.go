// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Team
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"testing"
)

func TestSyncExecutionDomainBestEffortPersistsRuntimeV1Snapshot(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	state := NewAppState(store)
	now := "2026-03-04T00:00:00Z"
	conversationID := "conv_runtime_v1_sync"
	executionID := "run_runtime_v1_sync"

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_runtime_v1_sync",
		Name:              "Runtime V1 Sync",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "mcfg_runtime_v1_sync",
		RuleIDs:           []string{"rule_1"},
		SkillIDs:          []string{"skill_1"},
		MCPIDs:            []string{"mcp_1"},
		ActiveExecutionID: stringPtrOrNil(executionID),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_runtime_v1_sync",
		State:          RunStateExecuting,
		Mode:           PermissionModePlan,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModePlan,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3", ConfigID: "mcfg_runtime_v1_sync"},
		TokensIn:       11,
		TokensOut:      22,
		TraceID:        "trace_runtime_v1_sync",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executionEvents[conversationID] = []ExecutionEvent{
		{
			EventID:        "evt_runtime_v1_sync_1",
			ExecutionID:    executionID,
			ConversationID: conversationID,
			TraceID:        "trace_runtime_v1_sync",
			Sequence:       1,
			Type:           RunEventTypeExecutionStarted,
			Timestamp:      now,
			Payload:        map[string]any{"status": "started"},
		},
	}
	state.hookExecutionRecords[conversationID] = []HookExecutionRecord{
		{
			ID:             "hook_runtime_v1_sync_1",
			RunID:          executionID,
			TaskID:         "task_runtime_v1_sync_1",
			ConversationID: conversationID,
			Event:          HookEventTypePreToolUse,
			ToolName:       "bash",
			PolicyID:       "policy_runtime_v1_sync",
			Decision: HookDecision{
				Action: HookDecisionActionAllow,
				Reason: "ok",
			},
			Timestamp: now,
		},
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	repositories := NewSQLiteRuntimeV1RepositorySet(store.db)
	ctx := context.Background()

	sessions, err := repositories.Sessions.ListByWorkspace(ctx, localWorkspaceID, RepositoryPage{Limit: 100, Offset: 0})
	if err != nil {
		t.Fatalf("list runtime v1 sessions failed: %v", err)
	}
	foundSession := false
	for _, item := range sessions {
		if item.ID == conversationID {
			foundSession = true
			break
		}
	}
	if !foundSession {
		t.Fatalf("expected runtime v1 session %q to be persisted", conversationID)
	}

	runs, err := repositories.Runs.ListBySession(ctx, conversationID, RepositoryPage{Limit: 100, Offset: 0})
	if err != nil {
		t.Fatalf("list runtime v1 runs failed: %v", err)
	}
	if len(runs) != 1 || runs[0].ID != executionID {
		t.Fatalf("expected one runtime v1 run %q, got %#v", executionID, runs)
	}
	if runs[0].State != string(RunStateExecuting) {
		t.Fatalf("expected runtime v1 run state %q, got %q", RunStateExecuting, runs[0].State)
	}
	if runs[0].ModelConfigID != "mcfg_runtime_v1_sync" {
		t.Fatalf("expected runtime v1 run model_config_id mcfg_runtime_v1_sync, got %q", runs[0].ModelConfigID)
	}

	events, err := repositories.RunEvents.ListBySession(ctx, conversationID, 0, 100)
	if err != nil {
		t.Fatalf("list runtime v1 run events failed: %v", err)
	}
	if len(events) != 1 || events[0].EventID != "evt_runtime_v1_sync_1" {
		t.Fatalf("expected one runtime v1 event evt_runtime_v1_sync_1, got %#v", events)
	}

	hookRecords, err := repositories.HookRecords.ListByRun(ctx, executionID, RepositoryPage{Limit: 100, Offset: 0})
	if err != nil {
		t.Fatalf("list runtime v1 hook records failed: %v", err)
	}
	if len(hookRecords) != 1 || hookRecords[0].ID != "hook_runtime_v1_sync_1" {
		t.Fatalf("expected one runtime v1 hook record hook_runtime_v1_sync_1, got %#v", hookRecords)
	}
	if hookRecords[0].TaskID == nil || *hookRecords[0].TaskID != "task_runtime_v1_sync_1" {
		t.Fatalf("expected runtime v1 hook record task_id task_runtime_v1_sync_1, got %#v", hookRecords[0].TaskID)
	}
}
