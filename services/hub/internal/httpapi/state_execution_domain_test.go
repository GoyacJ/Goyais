package httpapi

import "testing"

func TestRepairLegacyExecutionDomainLockedConvertsConfirmingExecution(t *testing.T) {
	state := NewAppState(nil)
	conversationID := "conv_legacy"
	executionID := "exec_confirming"
	queuedID := "exec_queued"

	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_1",
		Name:              "Legacy Confirming",
		QueueState:        QueueStateRunning,
		DefaultMode:       ConversationModeAgent,
		ModelID:           "gpt-5.3",
		BaseRevision:      0,
		ActiveExecutionID: &executionID,
		CreatedAt:         "2026-02-24T00:00:00Z",
		UpdatedAt:         "2026-02-24T00:00:00Z",
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		State:          ExecutionState("confirming"),
		QueueIndex:     0,
		CreatedAt:      "2026-02-24T00:00:00Z",
		UpdatedAt:      "2026-02-24T00:00:00Z",
	}
	state.executions[queuedID] = Execution{
		ID:             queuedID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		State:          ExecutionStateQueued,
		QueueIndex:     1,
		CreatedAt:      "2026-02-24T00:00:01Z",
		UpdatedAt:      "2026-02-24T00:00:01Z",
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID, queuedID}
	state.executionLeases[executionID] = ExecutionLease{
		ExecutionID: executionID,
		WorkerID:    "worker-1",
	}
	state.executionControlQueues[executionID] = []ExecutionControlCommand{
		{
			ID:          "ctrl_confirm",
			ExecutionID: executionID,
			Type:        ExecutionControlCommandType("confirm"),
			Seq:         1,
		},
		{
			ID:          "ctrl_stop",
			ExecutionID: executionID,
			Type:        ExecutionControlCommandTypeStop,
			Seq:         2,
		},
	}
	state.executionControlSeq[executionID] = 2

	repairedExecutionIDs, removedControlCommands := repairLegacyExecutionDomainLocked(state)

	if len(repairedExecutionIDs) != 1 || repairedExecutionIDs[0] != executionID {
		t.Fatalf("expected repaired execution ids [%s], got %#v", executionID, repairedExecutionIDs)
	}
	if removedControlCommands != 1 {
		t.Fatalf("expected removed control commands=1, got %d", removedControlCommands)
	}
	if lease, exists := state.executionLeases[executionID]; exists {
		t.Fatalf("expected execution lease removed, got %#v", lease)
	}

	repaired := state.executions[executionID]
	if repaired.State != ExecutionStatePending {
		t.Fatalf("expected repaired execution state pending, got %s", repaired.State)
	}
	commands := state.executionControlQueues[executionID]
	if len(commands) != 1 || commands[0].Type != ExecutionControlCommandTypeStop {
		t.Fatalf("expected only stop control command, got %#v", commands)
	}

	conversation := state.conversations[conversationID]
	if conversation.QueueState != QueueStateRunning {
		t.Fatalf("expected conversation queue state running, got %s", conversation.QueueState)
	}
	if conversation.ActiveExecutionID == nil || *conversation.ActiveExecutionID != executionID {
		t.Fatalf("expected conversation active execution %s, got %#v", executionID, conversation.ActiveExecutionID)
	}
}
