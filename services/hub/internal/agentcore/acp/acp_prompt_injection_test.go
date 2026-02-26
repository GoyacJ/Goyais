package acp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"goyais/services/hub/internal/agentcore/protocol"
	"goyais/services/hub/internal/agentcore/runtime"
	"goyais/services/hub/internal/agentcore/state"
)

type capturePromptEngine struct {
	mu        sync.Mutex
	lastInput runtime.UserInput
}

func (e *capturePromptEngine) StartSession(_ context.Context, _ runtime.StartSessionRequest) (runtime.SessionHandle, error) {
	return runtime.SessionHandle{SessionID: "sess_capture"}, nil
}

func (e *capturePromptEngine) Submit(_ context.Context, _ string, input runtime.UserInput) (string, error) {
	e.mu.Lock()
	e.lastInput = input
	e.mu.Unlock()
	return "run_capture", nil
}

func (e *capturePromptEngine) Control(_ context.Context, _ string, _ state.ControlAction) error {
	return nil
}

func (e *capturePromptEngine) Subscribe(_ context.Context, _ string, _ string) (<-chan protocol.RunEvent, error) {
	out := make(chan protocol.RunEvent, 2)
	now := time.Now().UTC()
	out <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunOutputDelta,
		SessionID: "sess_capture",
		RunID:     "run_capture",
		Sequence:  0,
		Timestamp: now,
		Payload: map[string]any{
			"delta": "ok",
		},
	}
	out <- protocol.RunEvent{
		Type:      protocol.RunEventTypeRunCompleted,
		SessionID: "sess_capture",
		RunID:     "run_capture",
		Sequence:  1,
		Timestamp: now.Add(5 * time.Millisecond),
		Payload:   map[string]any{},
	}
	close(out)
	return out, nil
}

func (e *capturePromptEngine) LastInput() runtime.UserInput {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.lastInput
}

func TestACPPromptInjectsProjectContextAndInstructions(t *testing.T) {
	baseDir := t.TempDir()
	root := filepath.Join(baseDir, "workspace")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("create git root failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("ACP_ROOT_RULE"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md failed: %v", err)
	}

	cwd := filepath.Join(root, "apps", "service")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("create cwd failed: %v", err)
	}

	engine := &capturePromptEngine{}
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
	sessionID := strings.TrimSpace(asString(asMap(newResp["result"])["sessionId"]))
	if sessionID == "" {
		t.Fatalf("expected session id in session/new response")
	}

	_, _ = h.callRequest(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "session/prompt",
		"params": map[string]any{
			"sessionId": sessionID,
			"prompt": []any{
				map[string]any{
					"type": "text",
					"text": "show current project",
				},
			},
		},
	})

	injected := strings.TrimSpace(engine.LastInput().Text)
	if !strings.Contains(injected, "# Project Context") {
		t.Fatalf("expected project context header in injected prompt, got %q", injected)
	}
	if !strings.Contains(injected, "- Root Path: "+cwd) {
		t.Fatalf("expected cwd path in injected prompt, got %q", injected)
	}
	if !strings.Contains(injected, "ACP_ROOT_RULE") {
		t.Fatalf("expected AGENTS instructions in injected prompt, got %q", injected)
	}
	if !strings.Contains(injected, "# User Prompt") || !strings.HasSuffix(injected, "show current project") {
		t.Fatalf("expected user prompt block in injected prompt, got %q", injected)
	}
}
