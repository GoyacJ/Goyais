package handler

import (
	"net/http"

	"github.com/goyais/hub/internal/service"
)

type HealthHandler struct {
	version string
}

func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{version: version}
}

// GET /v1/health
func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": h.version})
}

// GET /v1/version
func (h *HealthHandler) Version(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": h.version})
}

// GET /v1/diagnostics
func (h *HealthHandler) Diagnostics(w http.ResponseWriter, r *http.Request) {
	_ = service.RedactedDiagnostics() // placeholder
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
