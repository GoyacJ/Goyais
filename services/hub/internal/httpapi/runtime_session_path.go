package httpapi

import (
	"net/http"
	"strings"
)

func runtimeSessionIDFromPath(r *http.Request) string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.PathValue("session_id"))
}

func runtimeSessionIDFromQuery(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return strings.TrimSpace(r.URL.Query().Get("session_id"))
}
