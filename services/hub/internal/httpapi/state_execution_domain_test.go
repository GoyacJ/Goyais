package httpapi

import "testing"

func TestHydrateExecutionDomainFromStoreKeepsRunState(t *testing.T) {
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
				State:          RunState("confirming"),
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
	if restoredExecution.State != RunState("confirming") {
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
				State:          RunStateCompleted,
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
				Type:           RunEventTypeDiffGenerated,
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
				Type:           RunEventTypeDiffGenerated,
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

func TestHydrateExecutionDomainFromStoreReplaysConversationEventSequence(t *testing.T) {
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
	snapshot := executionDomainSnapshot{
		Conversations: []Conversation{
			{
				ID:            "conv_seq_1",
				WorkspaceID:   localWorkspaceID,
				ProjectID:     "proj_seq_1",
				Name:          "Seq One",
				QueueState:    QueueStateIdle,
				DefaultMode:   PermissionModeDefault,
				ModelConfigID: "rc_model_legacy",
				BaseRevision:  0,
				CreatedAt:     "2026-02-24T00:00:00Z",
				UpdatedAt:     "2026-02-24T00:00:00Z",
			},
			{
				ID:            "conv_seq_2",
				WorkspaceID:   localWorkspaceID,
				ProjectID:     "proj_seq_2",
				Name:          "Seq Two",
				QueueState:    QueueStateIdle,
				DefaultMode:   PermissionModeDefault,
				ModelConfigID: "rc_model_legacy",
				BaseRevision:  0,
				CreatedAt:     "2026-02-24T00:00:00Z",
				UpdatedAt:     "2026-02-24T00:00:00Z",
			},
		},
		ExecutionEvents: []ExecutionEvent{
			{
				EventID:        "evt_seq_1",
				ConversationID: "conv_seq_1",
				ExecutionID:    "exec_seq_1",
				Sequence:       1,
				Type:           RunEventTypeExecutionStarted,
				Payload:        map[string]any{},
			},
			{
				EventID:        "evt_seq_3",
				ConversationID: "conv_seq_1",
				ExecutionID:    "exec_seq_1",
				Sequence:       3,
				Type:           RunEventTypeExecutionDone,
				Payload:        map[string]any{},
			},
			{
				EventID:        "evt_seq_2",
				ConversationID: "conv_seq_1",
				ExecutionID:    "exec_seq_1",
				Sequence:       2,
				Type:           RunEventTypeExecutionStarted,
				Payload:        map[string]any{},
			},
			{
				EventID:        "evt_seq_7",
				ConversationID: "conv_seq_2",
				ExecutionID:    "exec_seq_2",
				Sequence:       7,
				Type:           RunEventTypeExecutionDone,
				Payload:        map[string]any{},
			},
		},
	}

	if err := store.replaceExecutionDomainSnapshot(snapshot); err != nil {
		t.Fatalf("seed execution domain snapshot failed: %v", err)
	}

	state.hydrateExecutionDomainFromStore()

	if state.conversationEventSeq["conv_seq_1"] != 3 {
		t.Fatalf("expected conv_seq_1 max sequence 3, got %d", state.conversationEventSeq["conv_seq_1"])
	}
	if state.conversationEventSeq["conv_seq_2"] != 7 {
		t.Fatalf("expected conv_seq_2 max sequence 7, got %d", state.conversationEventSeq["conv_seq_2"])
	}
}

func TestHydrateExecutionDomainFromStoreRestoresHooks(t *testing.T) {
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
	snapshot := executionDomainSnapshot{
		HookPolicies: []HookPolicy{
			{
				ID:          "policy_1",
				Scope:       HookScopeGlobal,
				Event:       HookEventTypePreToolUse,
				HandlerType: HookHandlerTypeAgent,
				ToolName:    "Write",
				Enabled:     true,
				Decision: HookDecision{
					Action: HookDecisionActionDeny,
					Reason: "blocked",
				},
				UpdatedAt: "2026-03-01T00:00:00Z",
			},
		},
		HookExecutionRecords: []HookExecutionRecord{
			{
				ID:        "hook_exec_1",
				RunID:     "run_1",
				TaskID:    "task_1",
				SessionID: "conv_1",
				Event:     HookEventTypePreToolUse,
				ToolName:  "Write",
				PolicyID:  "policy_1",
				Decision: HookDecision{
					Action: HookDecisionActionDeny,
					Reason: "blocked",
				},
				Timestamp: "2026-03-01T00:00:01Z",
			},
		},
	}
	if err := store.replaceExecutionDomainSnapshot(snapshot); err != nil {
		t.Fatalf("seed execution domain snapshot failed: %v", err)
	}

	state.hydrateExecutionDomainFromStore()

	state.mu.RLock()
	defer state.mu.RUnlock()
	if len(state.hookPolicies) != 1 {
		t.Fatalf("expected 1 hook policy after hydrate, got %#v", state.hookPolicies)
	}
	policy := state.hookPolicies["policy_1"]
	if policy.ID != "policy_1" || policy.Decision.Action != HookDecisionActionDeny {
		t.Fatalf("unexpected hydrated hook policy: %#v", policy)
	}
	if len(state.hookExecutionRecords["conv_1"]) != 1 {
		t.Fatalf("expected 1 hook execution record for conv_1, got %#v", state.hookExecutionRecords["conv_1"])
	}
	record := state.hookExecutionRecords["conv_1"][0]
	if record.ID != "hook_exec_1" || record.RunID != "run_1" {
		t.Fatalf("unexpected hydrated hook execution record: %#v", record)
	}
}

func TestCaptureExecutionDomainSnapshotIncludesHooks(t *testing.T) {
	state := NewAppState(nil)
	state.mu.Lock()
	state.hookPolicies["policy_1"] = HookPolicy{
		ID:          "policy_1",
		Scope:       HookScopeGlobal,
		Event:       HookEventTypePreToolUse,
		HandlerType: HookHandlerTypeAgent,
		ToolName:    "Write",
		Enabled:     true,
		Decision: HookDecision{
			Action: HookDecisionActionDeny,
			Reason: "blocked",
			UpdatedInput: map[string]any{
				"path": "README.md",
			},
		},
		UpdatedAt: "2026-03-01T00:00:00Z",
	}
	state.hookExecutionRecords["conv_1"] = []HookExecutionRecord{
		{
			ID:        "hook_exec_1",
			RunID:     "run_1",
			SessionID: "conv_1",
			Event:     HookEventTypePreToolUse,
			ToolName:  "Write",
			PolicyID:  "policy_1",
			Decision: HookDecision{
				Action: HookDecisionActionDeny,
				Reason: "blocked",
			},
			Timestamp: "2026-03-01T00:00:01Z",
		},
	}
	state.mu.Unlock()

	snapshot := captureExecutionDomainSnapshot(state)
	if len(snapshot.HookPolicies) != 1 {
		t.Fatalf("expected 1 hook policy in snapshot, got %#v", snapshot.HookPolicies)
	}
	if snapshot.HookPolicies[0].ID != "policy_1" || snapshot.HookPolicies[0].Decision.Action != HookDecisionActionDeny {
		t.Fatalf("unexpected hook policy in snapshot: %#v", snapshot.HookPolicies[0])
	}
	if len(snapshot.HookExecutionRecords) != 1 {
		t.Fatalf("expected 1 hook execution record in snapshot, got %#v", snapshot.HookExecutionRecords)
	}
	if snapshot.HookExecutionRecords[0].ID != "hook_exec_1" || snapshot.HookExecutionRecords[0].RunID != "run_1" {
		t.Fatalf("unexpected hook execution record in snapshot: %#v", snapshot.HookExecutionRecords[0])
	}
}
