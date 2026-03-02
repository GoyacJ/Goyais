package httpapi

import (
	"encoding/json"
	runtimeinfra "goyais/services/hub/internal/runtime/infra/sqlite"
	"testing"
)

func TestReplaceExecutionDomainSnapshotNormalizesConversationMessageRole(t *testing.T) {
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
				ID:            "conv_msg_norm",
				WorkspaceID:   localWorkspaceID,
				ProjectID:     "proj_msg_norm",
				Name:          "Message Normalize",
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
				ID:             "msg_norm_1",
				ConversationID: "conv_msg_norm",
				Role:           MessageRole(" assistant "),
				Content:        "hello",
				CreatedAt:      "2026-03-01T00:00:01Z",
			},
		},
	}

	if err := store.replaceExecutionDomainSnapshot(snapshot); err != nil {
		t.Fatalf("replace execution domain snapshot failed: %v", err)
	}

	rows, err := runtimeinfra.NewConversationMessageStore(store.db).LoadAll()
	if err != nil {
		t.Fatalf("load conversation message rows failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 message row, got %#v", rows)
	}
	if rows[0].Role != "assistant" {
		t.Fatalf("expected normalized role assistant, got %q", rows[0].Role)
	}
}

func TestReplaceExecutionDomainSnapshotNormalizesConversationSnapshotFields(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	queueIndex := 4
	canRollback := true
	worktree := " wt_1 "
	snapshot := executionDomainSnapshot{
		Conversations: []Conversation{
			{
				ID:            "conv_snap_norm",
				WorkspaceID:   localWorkspaceID,
				ProjectID:     "proj_snap_norm",
				Name:          "Snapshot Normalize",
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
				ID:                     "snap_norm_1",
				ConversationID:         "conv_snap_norm",
				RollbackPointMessageID: "msg_1",
				QueueState:             QueueState(" running "),
				WorktreeRef:            &worktree,
				InspectorState:         ConversationInspector{Tab: "console"},
				Messages: []ConversationMessage{
					{
						ID:             "msg_1",
						ConversationID: "conv_snap_norm",
						Role:           MessageRole(" assistant "),
						Content:        "hello",
						QueueIndex:     &queueIndex,
						CanRollback:    &canRollback,
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

	rows, err := runtimeinfra.NewConversationSnapshotStore(store.db).LoadAll()
	if err != nil {
		t.Fatalf("load conversation snapshot rows failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 snapshot row, got %#v", rows)
	}
	row := rows[0]
	if row.QueueState != "running" {
		t.Fatalf("expected normalized queue_state running, got %q", row.QueueState)
	}
	if row.WorktreeRef == nil || *row.WorktreeRef != "wt_1" {
		t.Fatalf("expected normalized worktree_ref wt_1, got %#v", row.WorktreeRef)
	}
	decodedMessages := []ConversationMessage{}
	if err := json.Unmarshal([]byte(row.MessagesJSON), &decodedMessages); err != nil {
		t.Fatalf("unmarshal messages_json failed: %v", err)
	}
	if len(decodedMessages) != 1 || decodedMessages[0].Role != MessageRoleAssistant {
		t.Fatalf("expected normalized snapshot message role assistant, got %#v", decodedMessages)
	}
}

func TestReplaceExecutionDomainSnapshotNormalizesConversationFields(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	activeExecutionID := " exec_1 "
	snapshot := executionDomainSnapshot{
		Conversations: []Conversation{
			{
				ID:                "conv_norm_1",
				WorkspaceID:       localWorkspaceID,
				ProjectID:         "proj_norm_1",
				Name:              "Conversation Normalize",
				QueueState:        QueueState(" running "),
				DefaultMode:       PermissionMode(" default "),
				ModelConfigID:     "rc_model_legacy",
				RuleIDs:           []string{"rule_1", " rule_1 ", " ", ""},
				SkillIDs:          []string{"skill_1", "skill_1"},
				MCPIDs:            []string{"mcp_1", " mcp_1 "},
				BaseRevision:      0,
				ActiveExecutionID: &activeExecutionID,
				CreatedAt:         "2026-03-01T00:00:00Z",
				UpdatedAt:         "2026-03-01T00:00:00Z",
			},
		},
	}

	if err := store.replaceExecutionDomainSnapshot(snapshot); err != nil {
		t.Fatalf("replace execution domain snapshot failed: %v", err)
	}

	rows, err := runtimeinfra.NewConversationStore(store.db).LoadAll()
	if err != nil {
		t.Fatalf("load conversation rows failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 conversation row, got %#v", rows)
	}
	row := rows[0]
	if row.QueueState != "running" {
		t.Fatalf("expected normalized queue_state running, got %q", row.QueueState)
	}
	if row.DefaultMode != "default" {
		t.Fatalf("expected normalized default_mode default, got %q", row.DefaultMode)
	}
	if row.ActiveExecutionID == nil || *row.ActiveExecutionID != "exec_1" {
		t.Fatalf("expected normalized active_execution_id exec_1, got %#v", row.ActiveExecutionID)
	}
	if row.RuleIDsJSON != "[\"rule_1\"]" || row.SkillIDsJSON != "[\"skill_1\"]" || row.MCPIDsJSON != "[\"mcp_1\"]" {
		t.Fatalf("expected normalized id json arrays, got rule=%s skill=%s mcp=%s", row.RuleIDsJSON, row.SkillIDsJSON, row.MCPIDsJSON)
	}
}

func TestReplaceExecutionDomainSnapshotNormalizesExecutionFields(t *testing.T) {
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
				ID:            "conv_exec_norm",
				WorkspaceID:   localWorkspaceID,
				ProjectID:     "proj_exec_norm",
				Name:          "Execution Normalize",
				QueueState:    QueueStateIdle,
				DefaultMode:   PermissionModeDefault,
				ModelConfigID: "rc_model_legacy",
				BaseRevision:  0,
				CreatedAt:     "2026-03-01T00:00:00Z",
				UpdatedAt:     "2026-03-01T00:00:00Z",
			},
		},
		Executions: []Execution{
			{
				ID:             "exec_norm_1",
				WorkspaceID:    localWorkspaceID,
				ConversationID: "conv_exec_norm",
				MessageID:      "msg_exec_norm_1",
				State:          ExecutionState(" running "),
				Mode:           ConversationMode(" default "),
				ModelID:        "gpt-5.3",
				ModeSnapshot:   ConversationMode(" default "),
				ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
				QueueIndex:     0,
				TraceID:        "tr_exec_norm_1",
				CreatedAt:      "2026-03-01T00:00:01Z",
				UpdatedAt:      "2026-03-01T00:00:01Z",
			},
		},
	}

	if err := store.replaceExecutionDomainSnapshot(snapshot); err != nil {
		t.Fatalf("replace execution domain snapshot failed: %v", err)
	}

	rows, err := runtimeinfra.NewExecutionStore(store.db).LoadAll()
	if err != nil {
		t.Fatalf("load execution rows failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 execution row, got %#v", rows)
	}
	row := rows[0]
	if row.State != "running" {
		t.Fatalf("expected normalized state running, got %q", row.State)
	}
	if row.Mode != "default" {
		t.Fatalf("expected normalized mode default, got %q", row.Mode)
	}
	if row.ModeSnapshot != "default" {
		t.Fatalf("expected normalized mode_snapshot default, got %q", row.ModeSnapshot)
	}
}

func TestReplaceExecutionDomainSnapshotNormalizesHookFields(t *testing.T) {
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
				ID:             "policy_1",
				Scope:          HookScope(" local "),
				Event:          HookEventType(" pre_tool_use "),
				HandlerType:    HookHandlerType(" plugin "),
				ToolName:       " Write ",
				WorkspaceID:    " ws_local ",
				ProjectID:      "",
				ConversationID: " conv_1 ",
				Enabled:        true,
				Decision: HookDecision{
					Action: HookDecisionAction(" deny "),
					Reason: " blocked by test ",
					UpdatedInput: map[string]any{
						"path": "README.md",
					},
				},
				UpdatedAt: "2026-03-01T00:00:00Z",
			},
		},
		HookExecutionRecords: []HookExecutionRecord{
			{
				ID:             "hook_exec_1",
				RunID:          " run_1 ",
				TaskID:         " task_1 ",
				ConversationID: " conv_1 ",
				Event:          HookEventType(" pre_tool_use "),
				ToolName:       " Write ",
				PolicyID:       " policy_1 ",
				Decision: HookDecision{
					Action: HookDecisionAction(" deny "),
					Reason: " blocked ",
				},
				Timestamp: "2026-03-01T00:00:01Z",
			},
		},
	}

	if err := store.replaceExecutionDomainSnapshot(snapshot); err != nil {
		t.Fatalf("replace execution domain snapshot failed: %v", err)
	}

	policyRows, err := runtimeinfra.NewHookPolicyStore(store.db).LoadAll()
	if err != nil {
		t.Fatalf("load hook policy rows failed: %v", err)
	}
	if len(policyRows) != 1 {
		t.Fatalf("expected 1 hook policy row, got %#v", policyRows)
	}
	if policyRows[0].Scope != "local" || policyRows[0].Event != "pre_tool_use" || policyRows[0].HandlerType != "plugin" {
		t.Fatalf("expected normalized hook policy enums, got %#v", policyRows[0])
	}
	if policyRows[0].ToolName != "Write" || !policyRows[0].Enabled {
		t.Fatalf("expected normalized hook policy tool/enabled, got %#v", policyRows[0])
	}
	if policyRows[0].WorkspaceID == nil || *policyRows[0].WorkspaceID != "ws_local" {
		t.Fatalf("expected normalized workspace_id ws_local, got %#v", policyRows[0].WorkspaceID)
	}
	if policyRows[0].ProjectID != nil {
		t.Fatalf("expected normalized project_id nil for local scope, got %#v", policyRows[0].ProjectID)
	}
	if policyRows[0].ConversationID == nil || *policyRows[0].ConversationID != "conv_1" {
		t.Fatalf("expected normalized conversation_id conv_1, got %#v", policyRows[0].ConversationID)
	}

	execRows, err := runtimeinfra.NewHookExecutionRecordStore(store.db).LoadAll()
	if err != nil {
		t.Fatalf("load hook execution rows failed: %v", err)
	}
	if len(execRows) != 1 {
		t.Fatalf("expected 1 hook execution row, got %#v", execRows)
	}
	row := execRows[0]
	if row.RunID != "run_1" || row.ConversationID != "conv_1" || row.Event != "pre_tool_use" {
		t.Fatalf("expected normalized hook execution identity fields, got %#v", row)
	}
	if row.TaskID == nil || *row.TaskID != "task_1" {
		t.Fatalf("expected normalized task_id task_1, got %#v", row.TaskID)
	}
	if row.ToolName == nil || *row.ToolName != "Write" {
		t.Fatalf("expected normalized tool_name Write, got %#v", row.ToolName)
	}
	if row.PolicyID == nil || *row.PolicyID != "policy_1" {
		t.Fatalf("expected normalized policy_id policy_1, got %#v", row.PolicyID)
	}
}
