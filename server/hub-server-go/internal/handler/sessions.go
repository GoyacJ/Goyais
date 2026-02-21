package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/goyais/hub/internal/service"
)

type WorkspaceHandler struct {
	svc *service.WorkspaceService
}

func NewWorkspaceHandler(svc *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{svc: svc}
}

// GET /v1/workspaces
func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	workspaces, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	if workspaces == nil {
		workspaces = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": workspaces})
}

// SessionHandler handles /v1/sessions routes.
type SessionHandler struct {
	svc *service.SessionService
}

func NewSessionHandler(svc *service.SessionService) *SessionHandler {
	return &SessionHandler{svc: svc}
}

// GET /v1/sessions?workspace_id=...&project_id=...
func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "project_id is required")
		return
	}
	sessions, err := h.svc.List(r.Context(), wsID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	if sessions == nil {
		sessions = []service.SessionSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

// POST /v1/sessions?workspace_id=...
func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	var in service.CreateSessionInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid body")
		return
	}
	sess, err := h.svc.Create(r.Context(), wsID, in)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "E_VALIDATION", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"session": sess})
}

// PATCH /v1/sessions/{session_id}?workspace_id=...
func (h *SessionHandler) Update(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	sessionID := chi.URLParam(r, "session_id")
	var in service.UpdateSessionInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid body")
		return
	}
	sess, err := h.svc.Update(r.Context(), wsID, sessionID, in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	if sess == nil {
		writeError(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": sess})
}

// DELETE /v1/sessions/{session_id}?workspace_id=... (physical delete)
func (h *SessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	sessionID := chi.URLParam(r, "session_id")
	if err := h.svc.Delete(r.Context(), wsID, sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
