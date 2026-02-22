package httpapi

import (
	"encoding/json"
	"net/http"
)

type StandardError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
	TraceID string         `json:"trace_id"`
}

type ListEnvelope struct {
	Items      []any   `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

func WriteStandardError(w http.ResponseWriter, r *http.Request, status int, code string, message string, details map[string]any) {
	traceID := TraceIDFromContext(r.Context())
	if traceID == "" {
		traceID = GenerateTraceID()
	}

	if details == nil {
		details = map[string]any{}
	}

	w.Header().Set(TraceHeader, traceID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(StandardError{
		Code:    code,
		Message: message,
		Details: details,
		TraceID: traceID,
	})
}
