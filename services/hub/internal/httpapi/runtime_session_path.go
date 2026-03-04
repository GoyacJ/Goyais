package httpapi

import (
	"net/http"
	"strings"
)

func runtimeSessionIDFromPath(r *http.Request) string {
	if r == nil {
		return ""
	}
	sessionID := strings.TrimSpace(r.PathValue("session_id"))
	if sessionID != "" {
		return sessionID
	}
	return strings.TrimSpace(r.PathValue("conversation_id"))
}

func runtimeSessionIDFromQuery(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if sessionID != "" {
		return sessionID
	}
	return strings.TrimSpace(r.URL.Query().Get("conversation_id"))
}
