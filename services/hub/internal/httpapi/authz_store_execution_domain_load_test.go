package httpapi

import "testing"

func TestLoadExecutionDomainSnapshotIncludesExecutionEvents(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	snapshot := executionDomainSnapshot{
		Conversations: []Conversation{
			{
				ID:            "conv_evt",
				WorkspaceID:   localWorkspaceID,
				ProjectID:     "proj_evt",
				Name:          "Event Conversation",
				QueueState:    QueueStateIdle,
				DefaultMode:   PermissionModeDefault,
				ModelConfigID: "rc_model_legacy",
				BaseRevision:  0,
				CreatedAt:     "2026-03-01T00:00:00Z",
				UpdatedAt:     "2026-03-01T00:00:00Z",
			},
		},
		ExecutionEvents: []ExecutionEvent{
			{
				EventID:        "evt_1",
				ExecutionID:    "exec_evt_1",
				ConversationID: "conv_evt",
				TraceID:        "tr_evt_1",
				Sequence:       1,
				QueueIndex:     0,
				Type:           RunEventTypeExecutionStarted,
				Timestamp:      "2026-03-01T00:00:01Z",
				Payload: map[string]any{
					"step": "start",
				},
			},
		},
	}
	if err := store.replaceExecutionDomainSnapshot(snapshot); err != nil {
		t.Fatalf("replace execution domain snapshot failed: %v", err)
	}

	loaded, err := store.loadExecutionDomainSnapshot()
	if err != nil {
		t.Fatalf("load execution domain snapshot failed: %v", err)
	}
	if len(loaded.ExecutionEvents) != 1 {
		t.Fatalf("expected 1 execution event, got %#v", loaded.ExecutionEvents)
	}
	event := loaded.ExecutionEvents[0]
	if event.EventID != "evt_1" || event.ConversationID != "conv_evt" || event.ExecutionID != "exec_evt_1" {
		t.Fatalf("unexpected loaded event: %#v", event)
	}
	if event.Payload["step"] != "start" {
		t.Fatalf("expected payload step=start, got %#v", event.Payload)
	}
}

func TestLoadExecutionDomainSnapshotHydratesLegacyExecutionTimeout(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	_, err = store.db.Exec(`INSERT INTO conversations(
		id, workspace_id, project_id, name, queue_state, default_mode, model_config_id, rule_ids_json, skill_ids_json, mcp_ids_json, base_revision, active_execution_id, created_at, updated_at
	) VALUES(
		'conv_legacy', 'ws_local', 'proj_legacy', 'Legacy Conversation', 'idle', 'default', 'rc_model_legacy', '[]', '[]', '[]', 0, NULL, '2026-03-01T00:00:00Z', '2026-03-01T00:00:00Z'
	)`)
	if err != nil {
		t.Fatalf("seed conversation failed: %v", err)
	}
	_, err = store.db.Exec(`INSERT INTO executions(
		id, workspace_id, conversation_id, message_id, state, mode, model_id, mode_snapshot, model_snapshot_json, resource_profile_snapshot_json, agent_config_snapshot_json, tokens_in, tokens_out, project_revision_snapshot, queue_index, trace_id, created_at, updated_at
	) VALUES(
		'exec_legacy', 'ws_local', 'conv_legacy', 'msg_legacy', 'queued', 'default', 'gpt-5.3', 'default', '{"model_id":"gpt-5.3","timeout_ms":15000}', NULL, NULL, 0, 0, 0, 0, 'tr_legacy', '2026-03-01T00:00:01Z', '2026-03-01T00:00:01Z'
	)`)
	if err != nil {
		t.Fatalf("seed execution failed: %v", err)
	}

	loaded, err := store.loadExecutionDomainSnapshot()
	if err != nil {
		t.Fatalf("load execution domain snapshot failed: %v", err)
	}
	if len(loaded.Executions) != 1 {
		t.Fatalf("expected 1 execution, got %#v", loaded.Executions)
	}
	execution := loaded.Executions[0]
	if execution.ModelSnapshot.Runtime == nil || execution.ModelSnapshot.Runtime.RequestTimeoutMS == nil {
		t.Fatalf("expected runtime.request_timeout_ms loaded from legacy timeout_ms, got %#v", execution.ModelSnapshot)
	}
	if got := *execution.ModelSnapshot.Runtime.RequestTimeoutMS; got != 15000 {
		t.Fatalf("expected request_timeout_ms=15000, got %d", got)
	}
}

func TestLoadExecutionDomainSnapshotHydratesConversationSnapshots(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	snapshot := executionDomainSnapshot{
		Conversations: []Conversation{
			{
				ID:            "conv_snap",
				WorkspaceID:   localWorkspaceID,
				ProjectID:     "proj_snap",
				Name:          "Snapshot Conversation",
				QueueState:    QueueStateIdle,
				DefaultMode:   PermissionModeDefault,
				ModelConfigID: "rc_model_legacy",
				BaseRevision:  0,
				CreatedAt:     "2026-03-01T00:00:00Z",
				UpdatedAt:     "2026-03-01T00:00:00Z",
			},
		},
		ConversationSnapshots: []ConversationSnapshot{
			{
				ID:                     "snap_1",
				ConversationID:         "conv_snap",
				RollbackPointMessageID: "msg_1",
				QueueState:             QueueStateRunning,
				WorktreeRef:            localStringPtr("wt_1"),
				InspectorState:         ConversationInspector{Tab: "console"},
				Messages: []ConversationMessage{
					{
						ID:             "msg_1",
						ConversationID: "conv_snap",
						Role:           MessageRoleAssistant,
						Content:        "hello",
						CreatedAt:      "2026-03-01T00:00:01Z",
					},
				},
				ExecutionIDs: []string{"exec_1"},
				CreatedAt:    "2026-03-01T00:00:02Z",
			},
		},
	}
	if err := store.replaceExecutionDomainSnapshot(snapshot); err != nil {
		t.Fatalf("replace execution domain snapshot failed: %v", err)
	}

	loaded, err := store.loadExecutionDomainSnapshot()
	if err != nil {
		t.Fatalf("load execution domain snapshot failed: %v", err)
	}
	if len(loaded.ConversationSnapshots) != 1 {
		t.Fatalf("expected 1 conversation snapshot, got %#v", loaded.ConversationSnapshots)
	}
	record := loaded.ConversationSnapshots[0]
	if record.ID != "snap_1" || record.ConversationID != "conv_snap" || record.RollbackPointMessageID != "msg_1" {
		t.Fatalf("unexpected conversation snapshot identity fields: %#v", record)
	}
	if record.WorktreeRef == nil || *record.WorktreeRef != "wt_1" {
		t.Fatalf("expected worktree_ref wt_1, got %#v", record.WorktreeRef)
	}
	if record.InspectorState.Tab != "console" {
		t.Fatalf("expected inspector tab console, got %#v", record.InspectorState)
	}
	if len(record.Messages) != 1 || record.Messages[0].ID != "msg_1" {
		t.Fatalf("expected messages decoded, got %#v", record.Messages)
	}
	if len(record.ExecutionIDs) != 1 || record.ExecutionIDs[0] != "exec_1" {
		t.Fatalf("expected execution ids decoded, got %#v", record.ExecutionIDs)
	}
}

func TestLoadExecutionDomainSnapshotHydratesConversationMessages(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	queueIndex := 3
	canRollback := true
	snapshot := executionDomainSnapshot{
		Conversations: []Conversation{
			{
				ID:            "conv_msg",
				WorkspaceID:   localWorkspaceID,
				ProjectID:     "proj_msg",
				Name:          "Message Conversation",
				QueueState:    QueueStateIdle,
				DefaultMode:   PermissionModeDefault,
				ModelConfigID: "rc_model_legacy",
				BaseRevision:  0,
				CreatedAt:     "2026-03-01T00:00:00Z",
				UpdatedAt:     "2026-03-01T00:00:00Z",
			},
		},
		ConversationMessages: []ConversationMessage{
			{
				ID:             "msg_1",
				ConversationID: "conv_msg",
				Role:           MessageRoleAssistant,
				Content:        "hello",
				QueueIndex:     &queueIndex,
				CanRollback:    &canRollback,
				CreatedAt:      "2026-03-01T00:00:01Z",
			},
			{
				ID:             "msg_2",
				ConversationID: "conv_msg",
				Role:           MessageRoleUser,
				Content:        "hi",
				CreatedAt:      "2026-03-01T00:00:02Z",
			},
		},
	}
	if err := store.replaceExecutionDomainSnapshot(snapshot); err != nil {
		t.Fatalf("replace execution domain snapshot failed: %v", err)
	}

	loaded, err := store.loadExecutionDomainSnapshot()
	if err != nil {
		t.Fatalf("load execution domain snapshot failed: %v", err)
	}
	if len(loaded.ConversationMessages) != 2 {
		t.Fatalf("expected 2 conversation messages, got %#v", loaded.ConversationMessages)
	}
	if loaded.ConversationMessages[0].Role != MessageRoleAssistant || loaded.ConversationMessages[1].Role != MessageRoleUser {
		t.Fatalf("expected roles preserved, got %#v", loaded.ConversationMessages)
	}
	if loaded.ConversationMessages[0].QueueIndex == nil || *loaded.ConversationMessages[0].QueueIndex != 3 {
		t.Fatalf("expected first queue index 3, got %#v", loaded.ConversationMessages[0].QueueIndex)
	}
	if loaded.ConversationMessages[0].CanRollback == nil || !*loaded.ConversationMessages[0].CanRollback {
		t.Fatalf("expected first can_rollback=true, got %#v", loaded.ConversationMessages[0].CanRollback)
	}
	if loaded.ConversationMessages[1].QueueIndex != nil || loaded.ConversationMessages[1].CanRollback != nil {
		t.Fatalf("expected second nullable fields nil, got %#v", loaded.ConversationMessages[1])
	}
}

func TestLoadExecutionDomainSnapshotHydratesHooks(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	snapshot := executionDomainSnapshot{
		HookPolicies: []HookPolicy{
			{
				ID:          "policy_deny_write",
				Scope:       HookScopeLocal,
				Event:       HookEventTypePreToolUse,
				HandlerType: HookHandlerTypeAgent,
				ToolName:    "Write",
				WorkspaceID: "ws_local",
				ProjectID:   "",
				SessionID:   "conv_1",
				Enabled:     true,
				Decision: HookDecision{
					Action: HookDecisionActionDeny,
					Reason: "blocked by policy",
					UpdatedInput: map[string]any{
						"path": "README.md",
					},
					AdditionalContext: map[string]any{
						"source": "test",
					},
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
				PolicyID:  "policy_deny_write",
				Decision: HookDecision{
					Action: HookDecisionActionDeny,
					Reason: "blocked by policy",
				},
				Timestamp: "2026-03-01T00:00:01Z",
			},
		},
	}
	if err := store.replaceExecutionDomainSnapshot(snapshot); err != nil {
		t.Fatalf("replace execution domain snapshot failed: %v", err)
	}

	loaded, err := store.loadExecutionDomainSnapshot()
	if err != nil {
		t.Fatalf("load execution domain snapshot failed: %v", err)
	}
	if len(loaded.HookPolicies) != 1 {
		t.Fatalf("expected 1 hook policy, got %#v", loaded.HookPolicies)
	}
	if loaded.HookPolicies[0].ID != "policy_deny_write" || loaded.HookPolicies[0].Decision.Action != HookDecisionActionDeny {
		t.Fatalf("unexpected loaded hook policy: %#v", loaded.HookPolicies[0])
	}
	if loaded.HookPolicies[0].WorkspaceID != "ws_local" || loaded.HookPolicies[0].ProjectID != "" || loaded.HookPolicies[0].SessionID != "conv_1" {
		t.Fatalf("expected explicit scope bindings to load, got %#v", loaded.HookPolicies[0])
	}
	if loaded.HookPolicies[0].Decision.UpdatedInput["path"] != "README.md" {
		t.Fatalf("expected updated_input[path]=README.md, got %#v", loaded.HookPolicies[0].Decision.UpdatedInput)
	}

	if len(loaded.HookExecutionRecords) != 1 {
		t.Fatalf("expected 1 hook execution record, got %#v", loaded.HookExecutionRecords)
	}
	record := loaded.HookExecutionRecords[0]
	if record.ID != "hook_exec_1" || record.RunID != "run_1" || record.TaskID != "task_1" {
		t.Fatalf("unexpected loaded hook execution record identity fields: %#v", record)
	}
	if record.Decision.Action != HookDecisionActionDeny || record.ToolName != "Write" || record.PolicyID != "policy_deny_write" {
		t.Fatalf("unexpected loaded hook execution record fields: %#v", record)
	}
}

func localStringPtr(value string) *string {
	return &value
}
