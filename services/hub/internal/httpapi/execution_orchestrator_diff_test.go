package httpapi

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentcoretools "goyais/services/hub/internal/legacybridge/agentcoretools"
)

func TestExecuteSingleOpenAIToolCallEmitsDiffGeneratedForFileTools(t *testing.T) {
	testCases := []struct {
		name           string
		toolName       string
		input          map[string]any
		setup          func(t *testing.T, workdir string)
		expectedPath   string
		expectedChange string
		expectAdded    *int
		expectDeleted  *int
	}{
		{
			name:     "write",
			toolName: "Write",
			input: map[string]any{
				"path":    "notes/write.txt",
				"content": "hello",
			},
			expectedPath:   "notes/write.txt",
			expectedChange: "added",
			expectAdded:    diffIntPtr(1),
			expectDeleted:  diffIntPtr(0),
		},
		{
			name:     "edit",
			toolName: "Edit",
			input: map[string]any{
				"path":       "notes/edit.txt",
				"old_string": "before",
				"new_string": "after",
			},
			setup: func(t *testing.T, workdir string) {
				t.Helper()
				target := filepath.Join(workdir, "notes", "edit.txt")
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					t.Fatalf("prepare edit dir failed: %v", err)
				}
				if err := os.WriteFile(target, []byte("before"), 0o644); err != nil {
					t.Fatalf("prepare edit file failed: %v", err)
				}
			},
			expectedPath:   "notes/edit.txt",
			expectedChange: "modified",
			expectAdded:    diffIntPtr(1),
			expectDeleted:  diffIntPtr(1),
		},
		{
			name:     "notebook-edit",
			toolName: "NotebookEdit",
			input: map[string]any{
				"path":       "notes/notebook.ipynb",
				"cell_index": 0,
				"new_source": "print('after')",
			},
			setup: func(t *testing.T, workdir string) {
				t.Helper()
				target := filepath.Join(workdir, "notes", "notebook.ipynb")
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					t.Fatalf("prepare notebook dir failed: %v", err)
				}
				if err := os.WriteFile(target, []byte(`{"cells":[{"cell_type":"code","source":["print('before')"]}]}`), 0o644); err != nil {
					t.Fatalf("prepare notebook failed: %v", err)
				}
			},
			expectedPath:   "notes/notebook.ipynb",
			expectedChange: "modified",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			workdir := t.TempDir()
			if testCase.setup != nil {
				testCase.setup(t, workdir)
			}
			state, orchestrator, execution, executor, specs, toolCtx := prepareExecutionToolLoopTestContext(t, workdir)

			_, err := orchestrator.executeSingleOpenAIToolCall(
				context.Background(),
				execution,
				executor,
				specs[testCase.toolName],
				openAIToolCall{
					CallID: "call_" + testCase.name,
					Name:   testCase.toolName,
					Input:  testCase.input,
				},
				toolCtx,
			)
			if err != nil {
				t.Fatalf("execute tool %s failed: %v", testCase.toolName, err)
			}

			state.mu.RLock()
			events := append([]ExecutionEvent{}, state.executionEvents[execution.ConversationID]...)
			diffState := append([]DiffItem{}, state.executionDiffs[execution.ID]...)
			state.mu.RUnlock()

			diffEvents := make([]ExecutionEvent, 0, len(events))
			taskArtifactEvents := make([]ExecutionEvent, 0, len(events))
			for _, event := range events {
				if event.Type == ExecutionEventTypeDiffGenerated {
					diffEvents = append(diffEvents, event)
				}
				if event.Type == ExecutionEventTypeTaskArtifactEmitted {
					taskArtifactEvents = append(taskArtifactEvents, event)
				}
			}
			if len(diffEvents) != 1 {
				t.Fatalf("expected one diff_generated event, got %#v", events)
			}
			if len(taskArtifactEvents) != 1 {
				t.Fatalf("expected one task_artifact_emitted event, got %#v", events)
			}
			eventDiff := parseDiffItemsFromPayload(diffEvents[0].Payload)
			if len(eventDiff) != 1 {
				t.Fatalf("expected one diff item in event payload, got %#v", eventDiff)
			}
			if eventDiff[0].Path != testCase.expectedPath || eventDiff[0].ChangeType != testCase.expectedChange {
				t.Fatalf("unexpected event diff item: %#v", eventDiff[0])
			}
			if testCase.expectAdded != nil {
				if eventDiff[0].AddedLines == nil || *eventDiff[0].AddedLines != *testCase.expectAdded {
					t.Fatalf("unexpected event added_lines, got %#v", eventDiff[0].AddedLines)
				}
			} else if eventDiff[0].AddedLines == nil {
				t.Fatalf("expected event added_lines to be present, got %#v", eventDiff[0].AddedLines)
			}
			if testCase.expectDeleted != nil {
				if eventDiff[0].DeletedLines == nil || *eventDiff[0].DeletedLines != *testCase.expectDeleted {
					t.Fatalf("unexpected event deleted_lines, got %#v", eventDiff[0].DeletedLines)
				}
			} else if eventDiff[0].DeletedLines == nil {
				t.Fatalf("expected event deleted_lines to be present, got %#v", eventDiff[0].DeletedLines)
			}
			if len(diffState) != 1 {
				t.Fatalf("expected one accumulated diff item in state, got %#v", diffState)
			}
			if diffState[0].Path != testCase.expectedPath || diffState[0].ChangeType != testCase.expectedChange {
				t.Fatalf("unexpected accumulated diff item: %#v", diffState[0])
			}
			if eventDiff[0].AddedLines == nil || diffState[0].AddedLines == nil || *eventDiff[0].AddedLines != *diffState[0].AddedLines {
				t.Fatalf("expected accumulated added_lines to match event, event=%#v state=%#v", eventDiff[0].AddedLines, diffState[0].AddedLines)
			}
			if eventDiff[0].DeletedLines == nil || diffState[0].DeletedLines == nil || *eventDiff[0].DeletedLines != *diffState[0].DeletedLines {
				t.Fatalf("expected accumulated deleted_lines to match event, event=%#v state=%#v", eventDiff[0].DeletedLines, diffState[0].DeletedLines)
			}
			artifactPayload, ok := taskArtifactEvents[0].Payload["artifact"].(map[string]any)
			if !ok {
				t.Fatalf("expected artifact payload map, got %#v", taskArtifactEvents[0].Payload)
			}
			if gotKind := asStringValue(artifactPayload["kind"]); gotKind != "diff" {
				t.Fatalf("expected artifact kind diff, got %q", gotKind)
			}
		})
	}
}

func TestExecuteSingleOpenAIToolCallWriteUpdatesChangeTypeToModifiedOnOverwrite(t *testing.T) {
	workdir := t.TempDir()
	state, orchestrator, execution, executor, specs, toolCtx := prepareExecutionToolLoopTestContext(t, workdir)

	_, err := orchestrator.executeSingleOpenAIToolCall(
		context.Background(),
		execution,
		executor,
		specs["Write"],
		openAIToolCall{
			CallID: "call_write_1",
			Name:   "Write",
			Input: map[string]any{
				"path":    "notes/same.txt",
				"content": "first",
			},
		},
		toolCtx,
	)
	if err != nil {
		t.Fatalf("first write failed: %v", err)
	}

	_, err = orchestrator.executeSingleOpenAIToolCall(
		context.Background(),
		execution,
		executor,
		specs["Write"],
		openAIToolCall{
			CallID: "call_write_2",
			Name:   "Write",
			Input: map[string]any{
				"path":    "notes/same.txt",
				"content": "second",
			},
		},
		toolCtx,
	)
	if err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	state.mu.RLock()
	events := append([]ExecutionEvent{}, state.executionEvents[execution.ConversationID]...)
	diffState := append([]DiffItem{}, state.executionDiffs[execution.ID]...)
	state.mu.RUnlock()

	diffEvents := make([]ExecutionEvent, 0, len(events))
	for _, event := range events {
		if event.Type == ExecutionEventTypeDiffGenerated {
			diffEvents = append(diffEvents, event)
		}
	}
	if len(diffEvents) != 2 {
		t.Fatalf("expected two diff_generated events for two writes, got %#v", events)
	}
	secondDiff := parseDiffItemsFromPayload(diffEvents[1].Payload)
	if len(secondDiff) != 1 {
		t.Fatalf("expected merged second diff payload to contain one path, got %#v", secondDiff)
	}
	if secondDiff[0].Path != "notes/same.txt" || secondDiff[0].ChangeType != "modified" {
		t.Fatalf("expected second write to mark modified, got %#v", secondDiff[0])
	}
	if len(diffState) != 1 || diffState[0].ChangeType != "modified" {
		t.Fatalf("expected accumulated state diff to keep modified status, got %#v", diffState)
	}
	if secondDiff[0].AddedLines == nil || secondDiff[0].DeletedLines == nil || *secondDiff[0].AddedLines != 1 || *secondDiff[0].DeletedLines != 1 {
		t.Fatalf("expected second write to include internal line counts, got %#v", secondDiff[0])
	}
}

func prepareExecutionToolLoopTestContext(
	t *testing.T,
	workdir string,
) (*AppState, *ExecutionOrchestrator, Execution, *agentcoretools.Executor, map[string]agentcoretools.ToolSpec, agentcoretools.ToolContext) {
	t.Helper()

	state := NewAppState(nil)
	conversation := Conversation{
		ID:            "conv_tool_diff",
		WorkspaceID:   localWorkspaceID,
		ProjectID:     "proj_tool_diff",
		Name:          "Tool Diff",
		QueueState:    QueueStateRunning,
		DefaultMode:   PermissionModeAcceptEdits,
		ModelConfigID: "rc_model_1",
		BaseRevision:  0,
		CreatedAt:     "2026-02-28T00:00:00Z",
		UpdatedAt:     "2026-02-28T00:00:00Z",
	}
	execution := Execution{
		ID:             "exec_tool_diff",
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversation.ID,
		MessageID:      "msg_tool_diff",
		State:          ExecutionStateExecuting,
		Mode:           PermissionModeAcceptEdits,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeAcceptEdits,
		ModelSnapshot: ModelSnapshot{
			ModelID: "gpt-5.3",
		},
		QueueIndex: 0,
		TraceID:    "tr_tool_diff",
		CreatedAt:  "2026-02-28T00:00:00Z",
		UpdatedAt:  "2026-02-28T00:00:00Z",
	}
	state.mu.Lock()
	state.conversations[conversation.ID] = conversation
	state.executions[execution.ID] = execution
	state.mu.Unlock()

	registry := agentcoretools.NewRegistry()
	if err := agentcoretools.RegisterCoreTools(registry); err != nil {
		t.Fatalf("register core tools failed: %v", err)
	}
	specs := map[string]agentcoretools.ToolSpec{}
	for _, tool := range registry.ListOrdered() {
		spec := tool.Spec()
		specs[spec.Name] = spec
	}
	executor := agentcoretools.NewExecutor(registry)
	orchestrator := NewExecutionOrchestrator(state)
	toolCtx := agentcoretools.ToolContext{
		Context:    context.Background(),
		WorkingDir: workdir,
		Env:        map[string]string{},
	}
	return state, orchestrator, execution, executor, specs, toolCtx
}

func diffIntPtr(value int) *int {
	result := value
	return &result
}

func TestTransitionExecutionToFailedEmitsTaskFailedEvent(t *testing.T) {
	state := NewAppState(nil)
	orchestrator := NewExecutionOrchestrator(state)
	now := "2026-03-02T00:00:00Z"

	conversation := Conversation{
		ID:                "conv_failed_task",
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_failed_task",
		Name:              "Failed Task",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_1",
		ActiveExecutionID: ptrString("exec_failed_task"),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	execution := Execution{
		ID:             "exec_failed_task",
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversation.ID,
		MessageID:      "msg_failed_task",
		State:          ExecutionStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_failed_task",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	state.mu.Lock()
	state.conversations[conversation.ID] = conversation
	state.executions[execution.ID] = execution
	state.conversationExecutionOrder[conversation.ID] = []string{execution.ID}
	state.mu.Unlock()

	_ = orchestrator.transitionExecutionToFailed(execution.ID, errors.New("runner crashed"))

	state.mu.RLock()
	events := append([]ExecutionEvent{}, state.executionEvents[conversation.ID]...)
	state.mu.RUnlock()

	foundTaskFailed := false
	for _, event := range events {
		if event.Type != ExecutionEventTypeTaskFailed {
			continue
		}
		if gotTaskID := strings.TrimSpace(asStringValue(event.Payload["task_id"])); gotTaskID != execution.ID {
			continue
		}
		if gotMessage := strings.TrimSpace(asStringValue(event.Payload["error_message"])); gotMessage != "runner crashed" {
			t.Fatalf("expected error_message runner crashed, got %#v", event.Payload)
		}
		foundTaskFailed = true
	}
	if !foundTaskFailed {
		t.Fatalf("expected task_failed event, got %#v", events)
	}
}

func TestTransitionExecutionToFailedEmitsSubagentStopHookRecord(t *testing.T) {
	state := NewAppState(nil)
	orchestrator := NewExecutionOrchestrator(state)
	now := "2026-03-02T00:00:00Z"

	conversation := Conversation{
		ID:                "conv_failed_subagent_hook",
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_failed_subagent",
		Name:              "Failed SubagentStop Hook",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_1",
		ActiveExecutionID: ptrString("exec_failed_subagent_hook"),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	execution := Execution{
		ID:             "exec_failed_subagent_hook",
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversation.ID,
		MessageID:      "msg_failed_subagent_hook",
		State:          ExecutionStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_failed_subagent_hook",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	state.mu.Lock()
	state.conversations[conversation.ID] = conversation
	state.executions[execution.ID] = execution
	state.conversationExecutionOrder[conversation.ID] = []string{execution.ID}
	state.hookPolicies["policy_subagent_stop_failed"] = HookPolicy{
		ID:          "policy_subagent_stop_failed",
		Scope:       HookScopeGlobal,
		Event:       HookEventTypeSubagentStop,
		HandlerType: HookHandlerTypePlugin,
		Enabled:     true,
		Decision: HookDecision{
			Action: HookDecisionActionAllow,
			Reason: "subagent failed stop logged",
		},
		UpdatedAt: now,
	}
	state.mu.Unlock()

	_ = orchestrator.transitionExecutionToFailed(execution.ID, errors.New("runner crashed"))

	state.mu.RLock()
	records := append([]HookExecutionRecord{}, state.hookExecutionRecords[conversation.ID]...)
	state.mu.RUnlock()

	foundSubagentStopRecord := false
	for _, record := range records {
		if record.RunID != execution.ID || record.Event != HookEventTypeSubagentStop {
			continue
		}
		if record.PolicyID != "policy_subagent_stop_failed" {
			t.Fatalf("unexpected subagent_stop hook record policy: %#v", record)
		}
		foundSubagentStopRecord = true
	}
	if !foundSubagentStopRecord {
		t.Fatalf("expected subagent_stop hook record for failed run %s, got %#v", execution.ID, records)
	}
}

func TestBeginExecutionEmitsTaskStartedEvent(t *testing.T) {
	state := NewAppState(nil)
	orchestrator := NewExecutionOrchestrator(state)
	now := "2026-03-02T00:00:00Z"

	conversation := Conversation{
		ID:            "conv_started_task",
		WorkspaceID:   localWorkspaceID,
		ProjectID:     "proj_started_task",
		Name:          "Started Task",
		QueueState:    QueueStateQueued,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	execution := Execution{
		ID:             "exec_started_task",
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversation.ID,
		MessageID:      "msg_started_task",
		State:          ExecutionStatePending,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_started_task",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	state.mu.Lock()
	state.conversations[conversation.ID] = conversation
	state.executions[execution.ID] = execution
	state.mu.Unlock()

	_, _, ok := orchestrator.beginExecution(execution.ID)
	if !ok {
		t.Fatalf("expected beginExecution to succeed")
	}

	state.mu.RLock()
	events := append([]ExecutionEvent{}, state.executionEvents[conversation.ID]...)
	state.mu.RUnlock()

	foundTaskStarted := false
	for _, event := range events {
		if event.Type != ExecutionEventTypeTaskStarted {
			continue
		}
		if gotTaskID := strings.TrimSpace(asStringValue(event.Payload["task_id"])); gotTaskID != execution.ID {
			continue
		}
		foundTaskStarted = true
	}
	if !foundTaskStarted {
		t.Fatalf("expected task_started event, got %#v", events)
	}
}

func TestBeginExecutionEmitsSessionStartHookRecord(t *testing.T) {
	state := NewAppState(nil)
	orchestrator := NewExecutionOrchestrator(state)
	now := "2026-03-02T00:00:00Z"

	conversation := Conversation{
		ID:            "conv_session_start",
		WorkspaceID:   localWorkspaceID,
		ProjectID:     "proj_session_start",
		Name:          "Session Start",
		QueueState:    QueueStateQueued,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	execution := Execution{
		ID:             "exec_session_start",
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversation.ID,
		MessageID:      "msg_session_start",
		State:          ExecutionStatePending,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_session_start",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	state.mu.Lock()
	state.conversations[conversation.ID] = conversation
	state.executions[execution.ID] = execution
	state.hookPolicies["policy_session_start_deny"] = HookPolicy{
		ID:          "policy_session_start_deny",
		Scope:       HookScopeGlobal,
		Event:       HookEventTypeSessionStart,
		HandlerType: HookHandlerTypePlugin,
		Enabled:     true,
		Decision: HookDecision{
			Action: HookDecisionActionDeny,
			Reason: "test session start hook deny",
		},
		UpdatedAt: now,
	}
	state.mu.Unlock()

	_, _, ok := orchestrator.beginExecution(execution.ID)
	if !ok {
		t.Fatalf("expected beginExecution to succeed")
	}

	state.mu.RLock()
	records := append([]HookExecutionRecord{}, state.hookExecutionRecords[conversation.ID]...)
	state.mu.RUnlock()

	foundSessionStartRecord := false
	for _, record := range records {
		if record.RunID != execution.ID || record.Event != HookEventTypeSessionStart {
			continue
		}
		if record.PolicyID != "policy_session_start_deny" || record.Decision.Action != HookDecisionActionDeny {
			t.Fatalf("unexpected session_start hook record: %#v", record)
		}
		foundSessionStartRecord = true
	}
	if !foundSessionStartRecord {
		t.Fatalf("expected session_start hook record for run %s, got %#v", execution.ID, records)
	}
}

func TestTransitionExecutionToCompletedEmitsTaskCompletedEvent(t *testing.T) {
	state := NewAppState(nil)
	orchestrator := NewExecutionOrchestrator(state)
	now := "2026-03-02T00:00:00Z"

	conversation := Conversation{
		ID:                "conv_completed_task",
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_completed_task",
		Name:              "Completed Task",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_1",
		ActiveExecutionID: ptrString("exec_completed_task"),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	execution := Execution{
		ID:             "exec_completed_task",
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversation.ID,
		MessageID:      "msg_completed_task",
		State:          ExecutionStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_completed_task",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	state.mu.Lock()
	state.conversations[conversation.ID] = conversation
	state.executions[execution.ID] = execution
	state.conversationExecutionOrder[conversation.ID] = []string{execution.ID}
	state.mu.Unlock()

	_ = orchestrator.transitionExecutionToCompleted(execution.ID, "ok", map[string]any{})

	state.mu.RLock()
	events := append([]ExecutionEvent{}, state.executionEvents[conversation.ID]...)
	state.mu.RUnlock()

	foundTaskCompleted := false
	for _, event := range events {
		if event.Type != ExecutionEventTypeTaskCompleted {
			continue
		}
		if gotTaskID := strings.TrimSpace(asStringValue(event.Payload["task_id"])); gotTaskID != execution.ID {
			continue
		}
		foundTaskCompleted = true
	}
	if !foundTaskCompleted {
		t.Fatalf("expected task_completed event, got %#v", events)
	}
}

func TestTransitionExecutionToCancelledEmitsTaskCancelledEvent(t *testing.T) {
	state := NewAppState(nil)
	orchestrator := NewExecutionOrchestrator(state)
	now := "2026-03-02T00:00:00Z"

	conversation := Conversation{
		ID:                "conv_cancelled_task",
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_cancelled_task",
		Name:              "Cancelled Task",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_1",
		ActiveExecutionID: ptrString("exec_cancelled_task"),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	execution := Execution{
		ID:             "exec_cancelled_task",
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversation.ID,
		MessageID:      "msg_cancelled_task",
		State:          ExecutionStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_cancelled_task",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	state.mu.Lock()
	state.conversations[conversation.ID] = conversation
	state.executions[execution.ID] = execution
	state.conversationExecutionOrder[conversation.ID] = []string{execution.ID}
	state.mu.Unlock()

	_ = orchestrator.transitionExecutionToCancelled(execution.ID, "run_cancelled")

	state.mu.RLock()
	events := append([]ExecutionEvent{}, state.executionEvents[conversation.ID]...)
	state.mu.RUnlock()

	foundTaskCancelled := false
	for _, event := range events {
		if event.Type != ExecutionEventTypeTaskCancelled {
			continue
		}
		if gotTaskID := strings.TrimSpace(asStringValue(event.Payload["task_id"])); gotTaskID != execution.ID {
			continue
		}
		if gotReason := strings.TrimSpace(asStringValue(event.Payload["reason"])); gotReason != "run_cancelled" {
			t.Fatalf("expected reason run_cancelled, got %#v", event.Payload)
		}
		foundTaskCancelled = true
	}
	if !foundTaskCancelled {
		t.Fatalf("expected task_cancelled event, got %#v", events)
	}
}

func TestTransitionExecutionToCancelledEmitsSubagentStopHookRecord(t *testing.T) {
	state := NewAppState(nil)
	orchestrator := NewExecutionOrchestrator(state)
	now := "2026-03-02T00:00:00Z"

	conversation := Conversation{
		ID:                "conv_subagent_stop_hook",
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_subagent_stop",
		Name:              "SubagentStop Hook",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_1",
		ActiveExecutionID: ptrString("exec_subagent_stop_hook"),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	execution := Execution{
		ID:             "exec_subagent_stop_hook",
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversation.ID,
		MessageID:      "msg_subagent_stop_hook",
		State:          ExecutionStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_subagent_stop_hook",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	state.mu.Lock()
	state.conversations[conversation.ID] = conversation
	state.executions[execution.ID] = execution
	state.conversationExecutionOrder[conversation.ID] = []string{execution.ID}
	state.hookPolicies["policy_subagent_stop"] = HookPolicy{
		ID:          "policy_subagent_stop",
		Scope:       HookScopeGlobal,
		Event:       HookEventTypeSubagentStop,
		HandlerType: HookHandlerTypePlugin,
		Enabled:     true,
		Decision: HookDecision{
			Action: HookDecisionActionAllow,
			Reason: "subagent stop logged",
		},
		UpdatedAt: now,
	}
	state.mu.Unlock()

	_ = orchestrator.transitionExecutionToCancelled(execution.ID, "run_cancelled")

	state.mu.RLock()
	records := append([]HookExecutionRecord{}, state.hookExecutionRecords[conversation.ID]...)
	state.mu.RUnlock()

	foundSubagentStopRecord := false
	for _, record := range records {
		if record.RunID != execution.ID || record.Event != HookEventTypeSubagentStop {
			continue
		}
		if record.PolicyID != "policy_subagent_stop" {
			t.Fatalf("unexpected subagent_stop hook record policy: %#v", record)
		}
		foundSubagentStopRecord = true
	}
	if !foundSubagentStopRecord {
		t.Fatalf("expected subagent_stop hook record for run %s, got %#v", execution.ID, records)
	}
}

func TestTransitionExecutionToAwaitingInputEmitsNotificationHookRecord(t *testing.T) {
	state := NewAppState(nil)
	orchestrator := NewExecutionOrchestrator(state)
	now := "2026-03-02T00:00:00Z"

	conversation := Conversation{
		ID:                "conv_notification_hook",
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_notification_hook",
		Name:              "Notification Hook",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_1",
		ActiveExecutionID: ptrString("exec_notification_hook"),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	execution := Execution{
		ID:             "exec_notification_hook",
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversation.ID,
		MessageID:      "msg_notification_hook",
		State:          ExecutionStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_notification_hook",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	state.mu.Lock()
	state.conversations[conversation.ID] = conversation
	state.executions[execution.ID] = execution
	state.hookPolicies["policy_notification_deny"] = HookPolicy{
		ID:          "policy_notification_deny",
		Scope:       HookScopeGlobal,
		Event:       HookEventTypeNotification,
		HandlerType: HookHandlerTypePlugin,
		Enabled:     true,
		Decision: HookDecision{
			Action: HookDecisionActionDeny,
			Reason: "test notification hook deny",
		},
		UpdatedAt: now,
	}
	state.mu.Unlock()

	orchestrator.transitionExecutionToAwaitingInput(execution.ID, pendingUserQuestion{
		QuestionID:          "q_notification_1",
		Question:            "Need your confirmation",
		Options:             []map[string]any{{"id": "yes", "label": "Yes"}},
		RecommendedOptionID: "yes",
		AllowText:           false,
		Required:            true,
		CallID:              "call_notification_1",
		ToolName:            "Edit",
	})

	state.mu.RLock()
	records := append([]HookExecutionRecord{}, state.hookExecutionRecords[conversation.ID]...)
	state.mu.RUnlock()

	foundNotificationRecord := false
	for _, record := range records {
		if record.RunID != execution.ID || record.Event != HookEventTypeNotification {
			continue
		}
		if record.ToolName != "Edit" {
			t.Fatalf("expected notification hook tool name Edit, got %#v", record)
		}
		if record.PolicyID != "policy_notification_deny" || record.Decision.Action != HookDecisionActionDeny {
			t.Fatalf("unexpected notification hook record: %#v", record)
		}
		foundNotificationRecord = true
	}
	if !foundNotificationRecord {
		t.Fatalf("expected notification hook record for run %s, got %#v", execution.ID, records)
	}
}

func ptrString(value string) *string {
	v := value
	return &v
}
