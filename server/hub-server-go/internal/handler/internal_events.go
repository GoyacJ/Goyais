package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/goyais/hub/internal/middleware"
	"github.com/goyais/hub/internal/model"
	"github.com/goyais/hub/internal/service"
)

// InternalEventsHandler receives batch event POST from workers.
type InternalEventsHandler struct {
	db        *sql.DB
	sseMan    *service.SSEManager
	scheduler *service.ExecutionScheduler
}

func NewInternalEventsHandler(db *sql.DB, sseMan *service.SSEManager, scheduler *service.ExecutionScheduler) *InternalEventsHandler {
	return &InternalEventsHandler{db: db, sseMan: sseMan, scheduler: scheduler}
}

type workerEventPayload struct {
	Seq     int             `json:"seq"`
	Ts      string          `json:"ts"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type workerEventsRequest struct {
	Events []workerEventPayload `json:"events"`
}

// POST /internal/executions/{execution_id}/events
func (h *InternalEventsHandler) ReceiveEvents(w http.ResponseWriter, r *http.Request) {
	executionID := chi.URLParam(r, "execution_id")

	var req workerEventsRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid body")
		return
	}

	ctx := r.Context()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Fetch trace_id once for all events in this batch (used for SSE propagation).
	var traceID string
	_ = h.db.QueryRowContext(ctx,
		`SELECT COALESCE(trace_id, '') FROM executions WHERE execution_id = ?`, executionID).Scan(&traceID)

	for _, ev := range req.Events {
		payloadStr := string(ev.Payload)
		if payloadStr == "" {
			payloadStr = "{}"
		}
		ts := ev.Ts
		if ts == "" {
			ts = now
		}

		// Persist to execution_events
		if _, err := h.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO execution_events (execution_id, seq, ts, type, payload_json)
			VALUES (?, ?, ?, ?, ?)`,
			executionID, ev.Seq, ts, ev.Type, payloadStr,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
			return
		}

		// Update last_event_ts
		_, _ = h.db.ExecContext(ctx,
			`UPDATE executions SET last_event_ts = ? WHERE execution_id = ?`, ts, executionID)

		// Fan-out via SSE
		h.sseMan.Publish(executionID, &model.ExecutionEvent{
			ExecutionID: executionID,
			TraceID:     traceID,
			Seq:         ev.Seq,
			Ts:          ts,
			Type:        ev.Type,
			PayloadJSON: payloadStr,
		})

		// Handle terminal events
		switch ev.Type {
		case "done":
			var state string
			// Determine state from payload
			var donePayload struct {
				Status string `json:"status"`
			}
			_ = json.Unmarshal(ev.Payload, &donePayload)
			state = donePayload.Status
			if state == "" {
				state = "completed"
			}
			_ = h.scheduler.CompleteExecution(ctx, executionID, state)

		case "confirmation_request":
			// Create tool_confirmations record + update session status
			var confPayload struct {
				CallID        string `json:"call_id"`
				ToolName      string `json:"tool_name"`
				RiskLevel     string `json:"risk_level"`
				ParamsSummary string `json:"parameters_summary"`
			}
			_ = json.Unmarshal(ev.Payload, &confPayload)
			if confPayload.CallID != "" {
				_, _ = h.db.ExecContext(ctx, `
					INSERT OR IGNORE INTO tool_confirmations
						(confirmation_id, execution_id, call_id, tool_name, risk_level, parameters_summary, status, created_at)
					VALUES (?, ?, ?, ?, ?, ?, 'pending', ?)`,
					randomID(), executionID,
					confPayload.CallID, confPayload.ToolName,
					coalesceStr(confPayload.RiskLevel, "medium"),
					confPayload.ParamsSummary, now)
				_, _ = h.db.ExecContext(ctx,
					`UPDATE sessions SET status = 'waiting_confirmation'
					 WHERE active_execution_id = ?`, executionID)
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /v1/confirmations?workspace_id=...
type ConfirmationHandler struct {
	db           *sql.DB
	sseMan       *service.SSEManager
	workerURL    string
	sharedSecret string
}

func NewConfirmationHandler(
	db *sql.DB,
	sseMan *service.SSEManager,
	workerURL string,
	sharedSecret string,
) *ConfirmationHandler {
	return &ConfirmationHandler{
		db:           db,
		sseMan:       sseMan,
		workerURL:    workerURL,
		sharedSecret: sharedSecret,
	}
}

func (h *ConfirmationHandler) Decide(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ExecutionID string `json:"execution_id"`
		CallID      string `json:"call_id"`
		Decision    string `json:"decision"` // "approved" | "denied"
	}
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid body")
		return
	}
	if body.Decision != "approved" && body.Decision != "denied" {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "decision must be approved or denied")
		return
	}

	ctx := r.Context()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	userModel := middleware.UserFromCtx(ctx)
	user := ""
	if userModel != nil {
		user = userModel.UserID
	}
	if _, err := h.db.ExecContext(ctx, `
		UPDATE tool_confirmations SET status = ?, decided_by = ?, decided_at = ?
		WHERE execution_id = ? AND call_id = ? AND status = 'pending'`,
		body.Decision, user, now, body.ExecutionID, body.CallID); err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}

	// Update session status back to executing
	_, _ = h.db.ExecContext(ctx,
		`UPDATE sessions SET status = 'executing' WHERE active_execution_id = ? AND status = 'waiting_confirmation'`,
		body.ExecutionID)

	// Forward decision to worker
	if h.workerURL != "" {
		_ = postJSONWithHeaders(h.workerURL+"/internal/confirmations", map[string]string{
			"execution_id": body.ExecutionID,
			"call_id":      body.CallID,
			"decision":     body.Decision,
		}, map[string]string{
			"X-Hub-Auth": h.sharedSecret,
			"X-User-Id":  user,
			"X-Trace-Id": middleware.TraceIDFromCtx(ctx),
		})
	}

	// SSE push
	h.sseMan.Publish(body.ExecutionID, &model.ExecutionEvent{
		ExecutionID: body.ExecutionID,
		Seq:         -1,
		Ts:          now,
		Type:        "confirmation_decision",
		PayloadJSON: `{"call_id":"` + body.CallID + `","decision":"` + body.Decision + `"}`,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// helpers
func randomID() string {
	return time.Now().Format("20060102150405.999999999")
}

func coalesceStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
