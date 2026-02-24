package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultWorkerInternalToken = "goyais-internal-token"

type workerClient struct {
	baseURL       string
	internalToken string
	httpClient    *http.Client
}

func newWorkerClientFromEnv() *workerClient {
	baseURL := strings.TrimSpace(os.Getenv("WORKER_BASE_URL"))
	internalToken := strings.TrimSpace(os.Getenv("WORKER_INTERNAL_TOKEN"))
	return newWorkerClient(baseURL, internalToken)
}

func newWorkerClient(baseURL string, internalToken string) *workerClient {
	normalizedURL := strings.TrimSpace(baseURL)
	if normalizedURL == "" {
		return nil
	}
	token := strings.TrimSpace(internalToken)
	if token == "" {
		token = defaultWorkerInternalToken
	}
	return &workerClient{
		baseURL:       strings.TrimRight(normalizedURL, "/"),
		internalToken: token,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *workerClient) submitExecution(ctx context.Context, execution Execution, content string) error {
	if c == nil {
		return nil
	}
	payload := map[string]any{
		"execution_id":              execution.ID,
		"workspace_id":              execution.WorkspaceID,
		"conversation_id":           execution.ConversationID,
		"message_id":                execution.MessageID,
		"mode":                      execution.Mode,
		"model_id":                  execution.ModelID,
		"mode_snapshot":             execution.ModeSnapshot,
		"model_snapshot":            execution.ModelSnapshot,
		"project_revision_snapshot": execution.ProjectRevisionSnapshot,
		"queue_index":               execution.QueueIndex,
		"trace_id":                  firstNonEmpty(execution.TraceID, TraceIDFromContext(ctx)),
		"content":                   strings.TrimSpace(content),
	}
	return c.postJSON(ctx, "/internal/executions", payload)
}

func (c *workerClient) submitExecutionEvent(ctx context.Context, execution Execution, eventType string, sequence int) error {
	if c == nil {
		return nil
	}
	payload := map[string]any{
		"event_id":        "evt_" + randomHex(8),
		"execution_id":    execution.ID,
		"conversation_id": execution.ConversationID,
		"type":            strings.TrimSpace(eventType),
		"sequence":        sequence,
		"queue_index":     execution.QueueIndex,
		"trace_id":        firstNonEmpty(execution.TraceID, TraceIDFromContext(ctx)),
		"payload": map[string]any{
			"state": execution.State,
		},
	}
	return c.postJSON(ctx, "/internal/events", payload)
}

func (c *workerClient) stopExecution(ctx context.Context, executionID string) error {
	if c == nil {
		return nil
	}
	return c.postJSON(ctx, "/internal/executions/"+strings.TrimSpace(executionID)+"/stop", map[string]any{})
}

func (c *workerClient) submitExecutionConfirmation(ctx context.Context, executionID string, decision string) error {
	if c == nil {
		return nil
	}
	payload := map[string]any{"decision": strings.TrimSpace(decision)}
	return c.postJSON(ctx, "/internal/executions/"+strings.TrimSpace(executionID)+"/confirm", payload)
}

func (c *workerClient) postJSON(ctx context.Context, path string, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal worker payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build worker request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", c.internalToken)
	if traceID := strings.TrimSpace(TraceIDFromContext(ctx)); traceID != "" {
		req.Header.Set(TraceHeader, traceID)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("worker request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= http.StatusMultipleChoices {
		rawBody, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("worker status=%d body=%s", res.StatusCode, strings.TrimSpace(string(rawBody)))
	}
	return nil
}
