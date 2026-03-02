package httpapi

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteSingleOpenAIToolCallHonorsHookDenyDecision(t *testing.T) {
	workdir := t.TempDir()
	state, orchestrator, execution, executor, specs, toolCtx := prepareExecutionToolLoopTestContext(t, workdir)

	state.mu.Lock()
	state.hookPolicies["policy_deny_write"] = HookPolicy{
		ID:          "policy_deny_write",
		Scope:       HookScopeGlobal,
		Event:       HookEventTypePreToolUse,
		HandlerType: HookHandlerTypePlugin,
		ToolName:    "Write",
		Enabled:     true,
		Decision: HookDecision{
			Action: HookDecisionActionDeny,
			Reason: "blocked by hook policy",
		},
		UpdatedAt: "2026-03-02T00:00:00Z",
	}
	state.mu.Unlock()

	result, err := orchestrator.executeSingleOpenAIToolCall(
		context.Background(),
		execution,
		executor,
		specs["Write"],
		openAIToolCall{
			CallID: "call_hook_deny",
			Name:   "Write",
			Input: map[string]any{
				"path":    "notes/blocked.txt",
				"content": "blocked",
			},
		},
		toolCtx,
	)
	if err != nil {
		t.Fatalf("expected no hard failure, got %v", err)
	}
	if !strings.Contains(strings.ToLower(result.Text), "hook policy") {
		t.Fatalf("expected hook deny message in result, got %q", result.Text)
	}

	state.mu.RLock()
	events := append([]ExecutionEvent{}, state.executionEvents[execution.ConversationID]...)
	records := append([]HookExecutionRecord{}, state.hookExecutionRecords[execution.ConversationID]...)
	state.mu.RUnlock()

	hasPreToolUse := false
	hasPermissionRequest := false
	hasPostToolUseFailure := false
	hasDiffGenerated := false
	for _, event := range events {
		switch event.Type {
		case ExecutionEventTypePreToolUse:
			hasPreToolUse = true
		case ExecutionEventTypePermissionRequest:
			hasPermissionRequest = true
		case ExecutionEventTypePostToolUseFailure:
			hasPostToolUseFailure = true
		case ExecutionEventTypeDiffGenerated:
			hasDiffGenerated = true
		}
	}
	if !hasPreToolUse || !hasPermissionRequest || !hasPostToolUseFailure {
		t.Fatalf("expected pre/permission/post failure hook events, got %#v", events)
	}
	if hasDiffGenerated {
		t.Fatalf("did not expect diff_generated when hook denies tool, got %#v", events)
	}
	if len(records) == 0 {
		t.Fatalf("expected hook execution record, got %#v", records)
	}
	if records[0].Decision.Action != HookDecisionActionDeny {
		t.Fatalf("expected deny decision record, got %#v", records[0])
	}
}

func TestExecuteSingleOpenAIToolCallAppliesLocalScopeOnlyInLocalWorkspace(t *testing.T) {
	workdir := t.TempDir()
	state, orchestrator, execution, executor, specs, toolCtx := prepareExecutionToolLoopTestContext(t, workdir)

	state.mu.Lock()
	state.hookPolicies["policy_local_deny_write"] = HookPolicy{
		ID:             "policy_local_deny_write",
		Scope:          HookScopeLocal,
		Event:          HookEventTypePreToolUse,
		HandlerType:    HookHandlerTypePlugin,
		ToolName:       "Write",
		ConversationID: execution.ConversationID,
		Enabled:        true,
		Decision: HookDecision{
			Action: HookDecisionActionDeny,
			Reason: "local policy deny",
		},
		UpdatedAt: "2026-03-02T00:00:00Z",
	}
	state.mu.Unlock()

	localResult, err := orchestrator.executeSingleOpenAIToolCall(
		context.Background(),
		execution,
		executor,
		specs["Write"],
		openAIToolCall{
			CallID: "call_local_scope_deny",
			Name:   "Write",
			Input: map[string]any{
				"path":    "notes/local_scope.txt",
				"content": "blocked",
			},
		},
		toolCtx,
	)
	if err != nil {
		t.Fatalf("local execution failed: %v", err)
	}
	if !strings.Contains(strings.ToLower(localResult.Text), "local policy deny") {
		t.Fatalf("expected local policy deny result, got %q", localResult.Text)
	}

	state.mu.Lock()
	remoteExecution := state.executions[execution.ID]
	remoteExecution.WorkspaceID = "ws_remote_scope"
	state.executions[execution.ID] = remoteExecution
	remoteConversation := state.conversations[execution.ConversationID]
	remoteConversation.WorkspaceID = "ws_remote_scope"
	state.conversations[execution.ConversationID] = remoteConversation
	state.executionEvents[execution.ConversationID] = []ExecutionEvent{}
	state.executionDiffs[execution.ID] = []DiffItem{}
	state.hookExecutionRecords[execution.ConversationID] = []HookExecutionRecord{}
	state.mu.Unlock()

	remotePath := filepath.Join(workdir, "notes", "remote_scope.txt")
	_ = os.Remove(remotePath)
	remoteResult, err := orchestrator.executeSingleOpenAIToolCall(
		context.Background(),
		remoteExecution,
		executor,
		specs["Write"],
		openAIToolCall{
			CallID: "call_local_scope_skip",
			Name:   "Write",
			Input: map[string]any{
				"path":    "notes/remote_scope.txt",
				"content": "allowed",
			},
		},
		toolCtx,
	)
	if err != nil {
		t.Fatalf("remote execution failed: %v", err)
	}
	if strings.Contains(strings.ToLower(remoteResult.Text), "denied") {
		t.Fatalf("expected remote execution not denied by local scope, got %q", remoteResult.Text)
	}
	if _, statErr := os.Stat(remotePath); statErr != nil {
		t.Fatalf("expected write to succeed in remote workspace context, stat err=%v", statErr)
	}
}

func TestExecuteSingleOpenAIToolCallAppliesProjectScopeBinding(t *testing.T) {
	workdir := t.TempDir()
	state, orchestrator, execution, executor, specs, toolCtx := prepareExecutionToolLoopTestContext(t, workdir)

	state.mu.Lock()
	state.hookPolicies["policy_project_bound"] = HookPolicy{
		ID:          "policy_project_bound",
		Scope:       HookScopeProject,
		Event:       HookEventTypePreToolUse,
		HandlerType: HookHandlerTypePlugin,
		ToolName:    "Write",
		ProjectID:   "proj_target",
		Enabled:     true,
		Decision: HookDecision{
			Action: HookDecisionActionDeny,
			Reason: "project scoped deny",
		},
		UpdatedAt: "2026-03-02T00:00:00Z",
	}
	conversation := state.conversations[execution.ConversationID]
	conversation.ProjectID = "proj_other"
	state.conversations[execution.ConversationID] = conversation
	state.mu.Unlock()

	firstPath := filepath.Join(workdir, "notes", "project_scope_skip.txt")
	_ = os.Remove(firstPath)
	firstResult, err := orchestrator.executeSingleOpenAIToolCall(
		context.Background(),
		execution,
		executor,
		specs["Write"],
		openAIToolCall{
			CallID: "call_project_scope_skip",
			Name:   "Write",
			Input: map[string]any{
				"path":    "notes/project_scope_skip.txt",
				"content": "allowed",
			},
		},
		toolCtx,
	)
	if err != nil {
		t.Fatalf("execution with non-matching project failed: %v", err)
	}
	if strings.Contains(strings.ToLower(firstResult.Text), "project scoped deny") {
		t.Fatalf("expected non-matching project to skip policy, got %q", firstResult.Text)
	}
	if _, statErr := os.Stat(firstPath); statErr != nil {
		t.Fatalf("expected write to succeed when project binding mismatches, stat err=%v", statErr)
	}

	state.mu.Lock()
	conversation = state.conversations[execution.ConversationID]
	conversation.ProjectID = "proj_target"
	state.conversations[execution.ConversationID] = conversation
	state.executionEvents[execution.ConversationID] = []ExecutionEvent{}
	state.executionDiffs[execution.ID] = []DiffItem{}
	state.hookExecutionRecords[execution.ConversationID] = []HookExecutionRecord{}
	state.mu.Unlock()

	secondResult, err := orchestrator.executeSingleOpenAIToolCall(
		context.Background(),
		execution,
		executor,
		specs["Write"],
		openAIToolCall{
			CallID: "call_project_scope_deny",
			Name:   "Write",
			Input: map[string]any{
				"path":    "notes/project_scope_deny.txt",
				"content": "blocked",
			},
		},
		toolCtx,
	)
	if err != nil {
		t.Fatalf("execution with matching project failed: %v", err)
	}
	if !strings.Contains(strings.ToLower(secondResult.Text), "project scoped deny") {
		t.Fatalf("expected matching project to enforce deny, got %q", secondResult.Text)
	}
}
