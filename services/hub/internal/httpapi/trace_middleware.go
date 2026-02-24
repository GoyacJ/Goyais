package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const TraceHeader = "X-Trace-Id"

type traceIDKeyType string

const traceIDKey traceIDKeyType = "trace_id"

func WithTrace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := strings.TrimSpace(r.Header.Get(TraceHeader))
		if traceID == "" {
			traceID = GenerateTraceID()
		}

		ctx := context.WithValue(r.Context(), traceIDKey, traceID)
		r = r.WithContext(ctx)

		w.Header().Set(TraceHeader, traceID)
		next.ServeHTTP(w, r)
	})
}

func TraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}

func GenerateTraceID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("tr_%d", time.Now().UnixNano())
	}
	return "tr_" + hex.EncodeToString(buf)
}
