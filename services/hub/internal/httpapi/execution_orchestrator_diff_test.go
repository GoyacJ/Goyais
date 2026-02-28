package httpapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"goyais/services/hub/internal/agentcore/safety"
	coretools "goyais/services/hub/internal/agentcore/tools"
)

func TestExecuteSingleOpenAIToolCallEmitsDiffGeneratedForFileTools(t *testing.T) {
	testCases := []struct {
		name           string
		toolName       string
		input          map[string]any
		setup          func(t *testing.T, workdir string)
		expectedPath   string
		expectedChange string
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
			for _, event := range events {
				if event.Type == ExecutionEventTypeDiffGenerated {
					diffEvents = append(diffEvents, event)
				}
			}
			if len(diffEvents) != 1 {
				t.Fatalf("expected one diff_generated event, got %#v", events)
			}
			eventDiff := parseDiffItemsFromPayload(diffEvents[0].Payload)
			if len(eventDiff) != 1 {
				t.Fatalf("expected one diff item in event payload, got %#v", eventDiff)
			}
			if eventDiff[0].Path != testCase.expectedPath || eventDiff[0].ChangeType != testCase.expectedChange {
				t.Fatalf("unexpected event diff item: %#v", eventDiff[0])
			}
			if len(diffState) != 1 {
				t.Fatalf("expected one accumulated diff item in state, got %#v", diffState)
			}
			if diffState[0].Path != testCase.expectedPath || diffState[0].ChangeType != testCase.expectedChange {
				t.Fatalf("unexpected accumulated diff item: %#v", diffState[0])
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
}

func prepareExecutionToolLoopTestContext(
	t *testing.T,
	workdir string,
) (*AppState, *ExecutionOrchestrator, Execution, *coretools.Executor, map[string]coretools.ToolSpec, coretools.ToolContext) {
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

	registry := coretools.NewRegistry()
	if err := coretools.RegisterCoreTools(registry); err != nil {
		t.Fatalf("register core tools failed: %v", err)
	}
	specs := map[string]coretools.ToolSpec{}
	for _, tool := range registry.ListOrdered() {
		spec := tool.Spec()
		specs[spec.Name] = spec
	}
	executor := coretools.NewExecutor(registry, safety.NewGate(safety.DefaultPolicy()))
	orchestrator := NewExecutionOrchestrator(state)
	toolCtx := coretools.ToolContext{
		Context:    context.Background(),
		WorkingDir: workdir,
		Env:        map[string]string{},
	}
	return state, orchestrator, execution, executor, specs, toolCtx
}
