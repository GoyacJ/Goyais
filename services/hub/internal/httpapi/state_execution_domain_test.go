package httpapi

import "testing"

func TestHydrateExecutionDomainFromStoreKeepsLegacyExecutionStateAndCommands(t *testing.T) {
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
				ModelID:           "gpt-5.3",
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
				QueueIndex: 0,
				TraceID:    "tr_legacy",
				CreatedAt:  "2026-02-24T00:00:00Z",
				UpdatedAt:  "2026-02-24T00:00:00Z",
			},
		},
		ExecutionControlCommands: []ExecutionControlCommand{
			{
				ID:          "ctrl_confirm",
				ExecutionID: executionID,
				Type:        ExecutionControlCommandType("confirm"),
				Seq:         1,
				CreatedAt:   "2026-02-24T00:00:01Z",
			},
			{
				ID:          "ctrl_stop",
				ExecutionID: executionID,
				Type:        ExecutionControlCommandTypeStop,
				Seq:         2,
				CreatedAt:   "2026-02-24T00:00:02Z",
			},
		},
		ExecutionLeases: []ExecutionLease{
			{
				ExecutionID: executionID,
				WorkerID:    "worker-legacy",
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

	commands := state.executionControlQueues[executionID]
	if len(commands) != 2 {
		t.Fatalf("expected 2 execution control commands, got %d", len(commands))
	}
	if commands[0].Type != ExecutionControlCommandType("confirm") || commands[1].Type != ExecutionControlCommandTypeStop {
		t.Fatalf("expected confirm+stop commands preserved, got %#v", commands)
	}

	if _, leaseExists := state.executionLeases[executionID]; !leaseExists {
		t.Fatalf("expected execution lease to remain after hydrate")
	}

	restoredConversation, conversationExists := state.conversations[conversationID]
	if !conversationExists {
		t.Fatalf("expected conversation %s to exist after hydrate", conversationID)
	}
	if restoredConversation.ActiveExecutionID == nil || *restoredConversation.ActiveExecutionID != executionID {
		t.Fatalf("expected active execution id %s preserved, got %#v", executionID, restoredConversation.ActiveExecutionID)
	}
}
