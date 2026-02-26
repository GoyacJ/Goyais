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
				DefaultMode:       ConversationModeAgent,
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
				Mode:           ConversationModeAgent,
				ModelID:        "gpt-5.3",
				ModeSnapshot:   ConversationModeAgent,
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
