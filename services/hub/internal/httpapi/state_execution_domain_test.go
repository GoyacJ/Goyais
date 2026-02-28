package httpapi

import "testing"

func TestHydrateExecutionDomainFromStoreKeepsExecutionState(t *testing.T) {
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
	conversationID := "conv_legacy"
	executionID := "exec_confirming"
	activeExecutionID := executionID

	snapshot := executionDomainSnapshot{
		Conversations: []Conversation{
			{
				ID:                conversationID,
				WorkspaceID:       localWorkspaceID,
				ProjectID:         "proj_legacy",
				Name:              "Legacy Runtime",
				QueueState:        QueueStateRunning,
				DefaultMode:       PermissionModeDefault,
				ModelConfigID:     "rc_model_legacy",
				RuleIDs:           []string{"rule_alpha"},
				SkillIDs:          []string{"skill_alpha"},
				MCPIDs:            []string{"mcp_alpha"},
				BaseRevision:      0,
				ActiveExecutionID: &activeExecutionID,
				CreatedAt:         "2026-02-24T00:00:00Z",
				UpdatedAt:         "2026-02-24T00:00:00Z",
			},
		},
		Executions: []Execution{
			{
				ID:             executionID,
				WorkspaceID:    localWorkspaceID,
				ConversationID: conversationID,
				MessageID:      "msg_legacy",
				State:          ExecutionState("confirming"),
				Mode:           PermissionModeDefault,
				ModelID:        "gpt-5.3",
				ModeSnapshot:   PermissionModeDefault,
				ModelSnapshot: ModelSnapshot{
					ModelID: "gpt-5.3",
				},
				ResourceProfileSnapshot: &ExecutionResourceProfile{
					ModelID:  "gpt-5.3",
					RuleIDs:  []string{"rule_alpha"},
					SkillIDs: []string{"skill_alpha"},
					MCPIDs:   []string{"mcp_alpha"},
				},
				QueueIndex: 0,
				TraceID:    "tr_legacy",
				CreatedAt:  "2026-02-24T00:00:00Z",
				UpdatedAt:  "2026-02-24T00:00:00Z",
			},
		},
	}

	if err := store.replaceExecutionDomainSnapshot(snapshot); err != nil {
		t.Fatalf("seed execution domain snapshot failed: %v", err)
	}

	state.hydrateExecutionDomainFromStore()

	restoredExecution, exists := state.executions[executionID]
	if !exists {
		t.Fatalf("expected execution %s to exist after hydrate", executionID)
	}
	if restoredExecution.State != ExecutionState("confirming") {
		t.Fatalf("expected execution state confirming to remain unchanged, got %s", restoredExecution.State)
	}

	restoredConversation, conversationExists := state.conversations[conversationID]
	if !conversationExists {
		t.Fatalf("expected conversation %s to exist after hydrate", conversationID)
	}
	if len(restoredConversation.RuleIDs) != 1 || restoredConversation.RuleIDs[0] != "rule_alpha" {
		t.Fatalf("expected conversation rule_ids preserved, got %#v", restoredConversation.RuleIDs)
	}
	if len(restoredConversation.SkillIDs) != 1 || restoredConversation.SkillIDs[0] != "skill_alpha" {
		t.Fatalf("expected conversation skill_ids preserved, got %#v", restoredConversation.SkillIDs)
	}
	if len(restoredConversation.MCPIDs) != 1 || restoredConversation.MCPIDs[0] != "mcp_alpha" {
		t.Fatalf("expected conversation mcp_ids preserved, got %#v", restoredConversation.MCPIDs)
	}
	if restoredConversation.ActiveExecutionID == nil || *restoredConversation.ActiveExecutionID != executionID {
		t.Fatalf("expected active execution id %s preserved, got %#v", executionID, restoredConversation.ActiveExecutionID)
	}
}

func TestHydrateExecutionDomainFromStoreAccumulatesExecutionDiffEvents(t *testing.T) {
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
	conversationID := "conv_diff_hydrate"
	executionID := "exec_diff_hydrate"

	snapshot := executionDomainSnapshot{
		Conversations: []Conversation{
			{
				ID:            conversationID,
				WorkspaceID:   localWorkspaceID,
				ProjectID:     "proj_hydrate",
				Name:          "Hydrate Diff",
				QueueState:    QueueStateIdle,
				DefaultMode:   PermissionModeDefault,
				ModelConfigID: "rc_model_legacy",
				BaseRevision:  0,
				CreatedAt:     "2026-02-24T00:00:00Z",
				UpdatedAt:     "2026-02-24T00:00:00Z",
			},
		},
		Executions: []Execution{
			{
				ID:             executionID,
				WorkspaceID:    localWorkspaceID,
				ConversationID: conversationID,
				MessageID:      "msg_hydrate",
				State:          ExecutionStateCompleted,
				Mode:           PermissionModeDefault,
				ModelID:        "gpt-5.3",
				ModeSnapshot:   PermissionModeDefault,
				ModelSnapshot: ModelSnapshot{
					ModelID: "gpt-5.3",
				},
				QueueIndex: 0,
				TraceID:    "tr_hydrate",
				CreatedAt:  "2026-02-24T00:00:00Z",
				UpdatedAt:  "2026-02-24T00:00:00Z",
			},
		},
		ExecutionEvents: []ExecutionEvent{
			{
				EventID:        "evt_diff_1",
				ExecutionID:    executionID,
				ConversationID: conversationID,
				TraceID:        "tr_hydrate",
				Sequence:       1,
				QueueIndex:     0,
				Type:           ExecutionEventTypeDiffGenerated,
				Timestamp:      "2026-02-24T00:00:01Z",
				Payload: map[string]any{
					"diff": []any{
						map[string]any{
							"path":        "a.txt",
							"change_type": "added",
							"summary":     "created",
						},
					},
				},
			},
			{
				EventID:        "evt_diff_2",
				ExecutionID:    executionID,
				ConversationID: conversationID,
				TraceID:        "tr_hydrate",
				Sequence:       2,
				QueueIndex:     0,
				Type:           ExecutionEventTypeDiffGenerated,
				Timestamp:      "2026-02-24T00:00:02Z",
				Payload: map[string]any{
					"diff": []any{
						map[string]any{
							"path":        "a.txt",
							"change_type": "modified",
							"summary":     "updated",
						},
						map[string]any{
							"path":        "b.txt",
							"change_type": "added",
							"summary":     "created",
						},
					},
				},
			},
		},
	}

	if err := store.replaceExecutionDomainSnapshot(snapshot); err != nil {
		t.Fatalf("seed execution domain snapshot failed: %v", err)
	}

	state.hydrateExecutionDomainFromStore()

	diff := state.executionDiffs[executionID]
	if len(diff) != 2 {
		t.Fatalf("expected 2 merged diff entries after hydrate, got %#v", diff)
	}
	if diff[0].Path != "a.txt" || diff[0].ChangeType != "modified" || diff[0].Summary != "updated" {
		t.Fatalf("expected first path a.txt updated by latest event, got %#v", diff[0])
	}
	if diff[1].Path != "b.txt" || diff[1].ChangeType != "added" {
		t.Fatalf("expected second path b.txt preserved, got %#v", diff[1])
	}
}
