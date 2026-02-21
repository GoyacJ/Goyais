package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/goyais/hub/internal/model"
	"github.com/goyais/hub/internal/service"
)

// ExecutionHandler handles /v1/sessions/{id}/execute and related endpoints.
type ExecutionHandler struct {
	scheduler *service.ExecutionScheduler
	sseMan    *service.SSEManager
	db        *sql.DB
}

func NewExecutionHandler(scheduler *service.ExecutionScheduler, sseMan *service.SSEManager, db *sql.DB) *ExecutionHandler {
	return &ExecutionHandler{scheduler: scheduler, sseMan: sseMan, db: db}
}

// POST /v1/sessions/{session_id}/execute?workspace_id=...
func (h *ExecutionHandler) Execute(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	sessionID := chi.URLParam(r, "session_id")

	var body struct {
		Message string `json:"message"`
	}
	if err := decodeBody(r, &body); err != nil || body.Message == "" {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "message is required")
		return
	}

	info, err := h.scheduler.Execute(r.Context(), wsID, sessionID, body.Message)
	if err != nil {
		switch e := err.(type) {
		case *model.SessionBusyError:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":      "E_SESSION_BUSY",
					"message":   "Session has an active execution",
					"retryable": false,
					"meta": map[string]string{
						"active_execution_id": e.ActiveExecutionID,
						"session_id":          e.SessionID,
					},
				},
			})
		case *model.QuotaExceededError:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":      "E_QUOTA_EXCEEDED",
					"message":   "Workspace concurrent execution limit reached",
					"retryable": true,
					"meta": map[string]any{
						"workspace_id": e.WorkspaceID,
						"limit":        e.Limit,
						"current":      e.Current,
					},
				},
			})
		case *model.NotFoundError:
			writeError(w, http.StatusNotFound, "E_NOT_FOUND", err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusAccepted, info)
}

// DELETE /v1/executions/{execution_id}/cancel?workspace_id=...
func (h *ExecutionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	executionID := chi.URLParam(r, "execution_id")
	if err := h.scheduler.CancelExecution(r.Context(), wsID, executionID); err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /v1/sessions/{session_id}/events?workspace_id=...&since_seq=0
// SSE stream of execution events for the session's active execution.
// Supports standard SSE reconnect: browser sends Last-Event-ID header on reconnect,
// which takes precedence over the since_seq query param.
func (h *ExecutionHandler) StreamEvents(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")
	sinceSeqStr := r.URL.Query().Get("since_seq")
	sinceSeq, _ := strconv.Atoi(sinceSeqStr)

	// Last-Event-ID header (sent automatically by browsers on SSE reconnect) takes precedence.
	if lastEventID := r.Header.Get("Last-Event-ID"); lastEventID != "" {
		if v, err := strconv.Atoi(lastEventID); err == nil {
			sinceSeq = v
		}
	}

	// Lookup current active_execution_id
	var activeExecID sql.NullString
	_ = h.db.QueryRowContext(r.Context(),
		`SELECT active_execution_id FROM sessions WHERE session_id = ?`, sessionID).
		Scan(&activeExecID)

	if !activeExecID.Valid || activeExecID.String == "" {
		// No active execution: stream recent events from DB for the latest completed execution
		var lastExecID string
		_ = h.db.QueryRowContext(r.Context(),
			`SELECT execution_id FROM executions WHERE session_id = ? ORDER BY created_at DESC LIMIT 1`,
			sessionID).Scan(&lastExecID)
		if lastExecID == "" {
			writeError(w, http.StatusNotFound, "E_NOT_FOUND", "no executions for this session")
			return
		}
		activeExecID.String = lastExecID
		activeExecID.Valid = true
	}

	executionID := activeExecID.String

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", "streaming not supported")
		return
	}

	ctx := r.Context()
	ch, cancel := h.sseMan.Subscribe(ctx, executionID, sinceSeq)
	defer cancel()

	// Keepalive ticker
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			fmt.Fprintf(w, ":keepalive\n\n")
			flusher.Flush()

		case event, open := <-ch:
			if !open {
				return
			}
			// SSE format: id, event, data (+ optional trace comment for log correlation)
			fmt.Fprintf(w, "id: %d\n", event.Seq)
			fmt.Fprintf(w, "event: %s\n", event.Type)
			if event.TraceID != "" {
				fmt.Fprintf(w, ": trace=%s\n", event.TraceID)
			}
			fmt.Fprintf(w, "data: %s\n\n", event.PayloadJSON)
			flusher.Flush()
		}
	}
}
