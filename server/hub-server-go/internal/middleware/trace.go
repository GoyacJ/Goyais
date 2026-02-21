package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type traceKey string

const ctxTraceID traceKey = "trace_id"

// Trace injects a unique trace_id into every request context and response header.
func Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-Id")
		if traceID == "" {
			traceID = uuid.NewString()
		}
		ctx := context.WithValue(r.Context(), ctxTraceID, traceID)
		w.Header().Set("X-Trace-Id", traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TraceIDFromCtx extracts the trace_id from context.
func TraceIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxTraceID).(string)
	return v
}
