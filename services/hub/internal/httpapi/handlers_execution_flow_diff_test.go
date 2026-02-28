package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExecutionDiffHandlerReturnsAccumulatedDiffEntries(t *testing.T) {
	state := NewAppState(nil)
	state.mu.Lock()
	state.executions["exec_diff_1"] = Execution{
		ID:             "exec_diff_1",
		WorkspaceID:    localWorkspaceID,
		ConversationID: "conv_diff_1",
		MessageID:      "msg_diff_1",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot: ModelSnapshot{
			ModelID: "gpt-5.3",
		},
		QueueIndex: 0,
		TraceID:    "tr_diff_1",
		CreatedAt:  "2026-02-28T00:00:00Z",
		UpdatedAt:  "2026-02-28T00:00:00Z",
	}
	appendExecutionEventLocked(state, ExecutionEvent{
		ExecutionID:    "exec_diff_1",
		ConversationID: "conv_diff_1",
		TraceID:        "tr_diff_1",
		QueueIndex:     0,
		Type:           ExecutionEventTypeDiffGenerated,
		Payload: map[string]any{
			"diff": []any{
				map[string]any{
					"path":        "README.md",
					"change_type": "modified",
					"summary":     "updated",
				},
			},
		},
	})
	appendExecutionEventLocked(state, ExecutionEvent{
		ExecutionID:    "exec_diff_1",
		ConversationID: "conv_diff_1",
		TraceID:        "tr_diff_1",
		QueueIndex:     0,
		Type:           ExecutionEventTypeDiffGenerated,
		Payload: map[string]any{
			"diff": []any{
				map[string]any{
					"path":        "src/main.ts",
					"change_type": "added",
					"summary":     "created",
				},
			},
		},
	})
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/executions/exec_diff_1/diff", nil)
	req.SetPathValue("execution_id", "exec_diff_1")
	recorder := httptest.NewRecorder()
	ExecutionDiffHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	diff := []DiffItem{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &diff); err != nil {
		t.Fatalf("decode diff response failed: %v", err)
	}
	if len(diff) != 2 {
		t.Fatalf("expected accumulated diff entries, got %#v", diff)
	}
	if diff[0].Path != "README.md" || diff[1].Path != "src/main.ts" {
		t.Fatalf("expected ordered accumulated paths, got %#v", diff)
	}
}
