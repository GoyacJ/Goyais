package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWorkerClientSubmitExecutionIncludesInternalTokenAndTrace(t *testing.T) {
	t.Parallel()

	var gotToken string
	var gotTrace string
	var gotPath string
	var gotPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		gotToken = r.Header.Get("X-Internal-Token")
		gotTrace = r.Header.Get(TraceHeader)
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := newWorkerClient(server.URL, "internal-secret")
	ctx := context.WithValue(context.Background(), traceIDKey, "tr_worker_submit")
	execution := Execution{
		ID:             "exec_001",
		WorkspaceID:    "ws_remote_001",
		ConversationID: "conv_001",
		MessageID:      "msg_001",
		Mode:           ConversationModeAgent,
		ModelID:        "gpt-4.1",
		QueueIndex:     2,
		TraceID:        "tr_execution",
	}

	if err := client.submitExecution(ctx, execution); err != nil {
		t.Fatalf("submit execution failed: %v", err)
	}

	if gotPath != "/internal/executions" {
		t.Fatalf("expected path /internal/executions, got %s", gotPath)
	}
	if gotToken != "internal-secret" {
		t.Fatalf("expected internal token header, got %s", gotToken)
	}
	if gotTrace != "tr_worker_submit" {
		t.Fatalf("expected trace header tr_worker_submit, got %s", gotTrace)
	}
	if gotPayload["execution_id"] != "exec_001" {
		t.Fatalf("unexpected execution payload: %#v", gotPayload)
	}
	if gotPayload["trace_id"] != "tr_execution" {
		t.Fatalf("unexpected execution trace payload: %#v", gotPayload)
	}
}

func TestWorkerClientSubmitExecutionEventReturnsUpstreamError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":"AUTH_INVALID_INTERNAL_TOKEN"}`))
	}))
	defer server.Close()

	client := newWorkerClient(server.URL, "invalid-token")
	ctx := context.WithValue(context.Background(), traceIDKey, "tr_worker_event")
	err := client.submitExecutionEvent(
		ctx,
		Execution{
			ID:             "exec_002",
			ConversationID: "conv_002",
			QueueIndex:     0,
		},
		"execution_stopped",
		1,
	)
	if err == nil {
		t.Fatal("expected error for non-2xx worker response")
	}
}
