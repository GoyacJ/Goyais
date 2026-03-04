// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Team
// SPDX-License-Identifier: MIT

package httpapi

import "testing"

func TestComputeTokenUsageAggregateUsesRepositoryWhenExecutionMapMissing(t *testing.T) {
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
	projectID := "proj_usage_repo_" + randomHex(4)
	conversationID := "conv_usage_repo_" + randomHex(4)
	runID := "run_usage_repo_" + randomHex(4)

	state.mu.Lock()
	state.projects[projectID] = Project{
		ID:                   projectID,
		WorkspaceID:          localWorkspaceID,
		Name:                 "Usage Repo",
		RepoPath:             "/tmp/usage-repo",
		IsGit:                true,
		DefaultModelConfigID: "mcfg_usage_repo",
		DefaultMode:          PermissionModeDefault,
		CurrentRevision:      0,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Usage Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "mcfg_usage_repo",
		BaseRevision:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationExecutionOrder[conversationID] = []string{runID}
	state.executions[runID] = Execution{
		ID:             runID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_usage_repo",
		State:          RunStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot: ModelSnapshot{
			ModelID:  "gpt-5.3",
			ConfigID: "mcfg_usage_repo",
		},
		TokensIn:  7,
		TokensOut: 9,
		TraceID:   "trace_usage_repo",
		CreatedAt: now,
		UpdatedAt: now,
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.executions = map[string]Execution{}
	state.conversationExecutionOrder = map[string][]string{}
	state.mu.Unlock()

	aggregate := computeTokenUsageAggregate(state, localWorkspaceID)
	projectTotals := aggregate.projectTotals[projectID]
	if projectTotals.Input != 7 || projectTotals.Output != 9 || projectTotals.Total != 16 {
		t.Fatalf("unexpected project usage totals: %#v", projectTotals)
	}
	projectModelTotals := aggregate.projectModelTotals[projectID]["mcfg_usage_repo"]
	if projectModelTotals.Total != 16 {
		t.Fatalf("unexpected project model usage totals: %#v", projectModelTotals)
	}
	workspaceModelTotals := aggregate.workspaceModelTotals[localWorkspaceID]["mcfg_usage_repo"]
	if workspaceModelTotals.Total != 16 {
		t.Fatalf("unexpected workspace model usage totals: %#v", workspaceModelTotals)
	}
}

func TestComputeTokenUsageAggregateFallsBackToInMemoryMap(t *testing.T) {
	state := NewAppState(nil)
	now := "2026-03-04T00:00:00Z"
	projectID := "proj_usage_mem_" + randomHex(4)
	conversationID := "conv_usage_mem_" + randomHex(4)
	runID := "run_usage_mem_" + randomHex(4)

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Usage Memory Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "mcfg_usage_mem",
		BaseRevision:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executions[runID] = Execution{
		ID:             runID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_usage_mem",
		State:          RunStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot: ModelSnapshot{
			ModelID:  "gpt-5.3",
			ConfigID: "mcfg_usage_mem",
		},
		TokensIn:  2,
		TokensOut: 3,
		TraceID:   "trace_usage_mem",
		CreatedAt: now,
		UpdatedAt: now,
	}
	state.mu.Unlock()

	aggregate := computeTokenUsageAggregate(state, localWorkspaceID)
	projectTotals := aggregate.projectTotals[projectID]
	if projectTotals.Input != 2 || projectTotals.Output != 3 || projectTotals.Total != 5 {
		t.Fatalf("unexpected in-memory project usage totals: %#v", projectTotals)
	}
}
