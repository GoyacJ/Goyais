// Copyright (c) 2026 Ysmjjsy
// Author: Goya
// SPDX-License-Identifier: MIT

package httpapi

import (
	"strings"
	"testing"
	"time"

	agentcore "goyais/services/hub/internal/agent/core"
)

func TestProjectRuntimeEvent_ToolCallCarriesCapabilityMetadata(t *testing.T) {
	state := NewAppState(nil)
	conversationID := "conv_projector_capability"
	executionID := "exec_projector_capability"
	runID := "run_projector_capability"
	now := "2026-03-06T13:00:00Z"

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_projector_capability",
		Name:              "Projection Capability",
		QueueState:        QueueStateRunning,
		ActiveExecutionID: stringPtrOrNil(executionID),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_projector_capability",
		State:          RunStateExecuting,
		QueueIndex:     0,
		TraceID:        "trace_projector_capability",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executionRunIDs[executionID] = runID
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	_, mappedType, projected := state.projectRuntimeEvent(conversationID, agentcore.EventEnvelope{
		Type:      agentcore.RunEventTypeRunOutputDelta,
		SessionID: "sess_projector_capability",
		RunID:     agentcore.RunID(runID),
		Sequence:  7,
		Timestamp: time.Date(2026, 3, 6, 13, 0, 1, 0, time.UTC),
		Payload: agentcore.OutputDeltaPayload{
			Stage:            "tool_call",
			CallID:           "call_projector_capability",
			Name:             "mcp__local__search_docs",
			ResolvedName:     "search_docs",
			CapabilityKind:   "mcp_tool",
			CapabilitySource: "local-search",
			CapabilityScope:  "workspace",
			RiskLevel:        "high",
			Input:            map[string]any{"query": "tooling"},
		},
	})
	if !projected {
		t.Fatal("expected event to project")
	}
	if mappedType != RunEventTypeToolCall {
		t.Fatalf("expected tool_call event, got %s", mappedType)
	}

	state.mu.RLock()
	events := append([]ExecutionEvent{}, state.executionEvents[conversationID]...)
	state.mu.RUnlock()
	if len(events) == 0 {
		t.Fatal("expected projected execution event")
	}
	payload := events[len(events)-1].Payload
	if got := strings.TrimSpace(asString(payload["capability_kind"])); got != "mcp_tool" {
		t.Fatalf("expected capability_kind mcp_tool, got %q", got)
	}
	if got := strings.TrimSpace(asString(payload["capability_source"])); got != "local-search" {
		t.Fatalf("expected capability_source local-search, got %q", got)
	}
	if got := strings.TrimSpace(asString(payload["capability_scope"])); got != "workspace" {
		t.Fatalf("expected capability_scope workspace, got %q", got)
	}
	if got := strings.TrimSpace(asString(payload["resolved_name"])); got != "search_docs" {
		t.Fatalf("expected resolved_name search_docs, got %q", got)
	}
}
