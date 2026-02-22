package httpapi

import (
	"encoding/json"
	"net/http"
)

func ListOrNotImplementedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListEnvelope{Items: []any{}, NextCursor: nil})
		return
	}

	WriteStandardError(
		w,
		r,
		http.StatusNotImplemented,
		"INTERNAL_NOT_IMPLEMENTED",
		"Route is not implemented yet",
		map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
		},
	)
}
