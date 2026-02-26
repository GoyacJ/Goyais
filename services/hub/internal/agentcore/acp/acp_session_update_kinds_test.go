package acp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"goyais/services/hub/internal/agentcore/protocol"
	"goyais/services/hub/internal/agentcore/runtime"
	"goyais/services/hub/internal/agentcore/state"
)

type scriptedEngine struct {
	events []protocol.RunEvent
}

func (s scriptedEngine) StartSession(_ context.Context, _ runtime.StartSessionRequest) (runtime.SessionHandle, error) {
	return runtime.SessionHandle{SessionID: "sess_engine"}, nil
}

func (s scriptedEngine) Submit(_ context.Context, _ string, _ runtime.UserInput) (string, error) {
	return "run_scripted", nil
}

func (s scriptedEngine) Control(_ context.Context, _ string, _ state.ControlAction) error {
	return nil
}

func (s scriptedEngine) Subscribe(_ context.Context, _ string, _ string) (<-chan protocol.RunEvent, error) {
	out := make(chan protocol.RunEvent, len(s.events))
	for _, event := range s.events {
		out <- event
	}
	close(out)
	return out, nil
}

func TestACPSessionUpdateKindsExtendedFromRunOutputPayload(t *testing.T) {
	baseDir := t.TempDir()
	cwd := filepath.Join(baseDir, "workspace")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	now := time.Now().UTC()
	engine := scriptedEngine{
		events: []protocol.RunEvent{
			{
				Type:      protocol.RunEventTypeRunOutputDelta,
				SessionID: "sess_engine",
				RunID:     "run_scripted",
				Sequence:  0,
				Timestamp: now,
				Payload: map[string]any{
					"thought": "thinking...",
					"tool_call": map[string]any{
						"toolCallId": "tool-1",
						"title":      "Read file",
						"kind":       "read",
						"status":     "pending",
					},
					"tool_call_update": map[string]any{
						"toolCallId": "tool-1",
						"status":     "completed",
					},
					"plan": []any{
						map[string]any{
							"content":  "step one",
							"priority": "high",
							"status":   "in_progress",
						},
					},
					"delta": "final answer",
				},
			},
			{
				Type:      protocol.RunEventTypeRunCompleted,
				SessionID: "sess_engine",
				RunID:     "run_scripted",
				Sequence:  1,
				Timestamp: now.Add(10 * time.Millisecond),
				Payload:   map[string]any{},
			},
		},
	}

	h := newHarnessWithEngine(t, baseDir, engine)
	_, _ = h.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": 1,
		},
	})
	newResp, _ := h.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "session/new",
		"params": map[string]any{
			"cwd":        cwd,
			"mcpServers": []any{},
		},
	})
	sessionID := asString(asMap(newResp["result"])["sessionId"])
	if sessionID == "" {
		t.Fatalf("expected session id in session/new response")
	}

	_, promptMsgs := h.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "session/prompt",
		"params": map[string]any{
			"sessionId": sessionID,
			"prompt": []any{
				map[string]any{
					"type": "text",
					"text": "run scripted events",
				},
			},
		},
	})

	assertHasUpdateKind(t, promptMsgs, "agent_thought_chunk")
	assertHasUpdateKind(t, promptMsgs, "tool_call")
	assertHasUpdateKind(t, promptMsgs, "tool_call_update")
	assertHasUpdateKind(t, promptMsgs, "plan")
	assertHasUpdateKind(t, promptMsgs, "agent_message_chunk")
}
