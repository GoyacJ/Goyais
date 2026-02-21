package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/goyais/hub/internal/service"
)

type ProjectHandler struct {
	svc     *service.ProjectService
	syncSvc *service.ProjectSyncService
}

func NewProjectHandler(svc *service.ProjectService, syncSvc *service.ProjectSyncService) *ProjectHandler {
	return &ProjectHandler{svc: svc, syncSvc: syncSvc}
}

// List handles GET /v1/projects?workspace_id=...
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_WORKSPACE_ID", "workspace_id is required")
		return
	}
	projects, err := h.svc.List(r.Context(), workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if projects == nil {
		projects = []service.ProjectSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

// Create handles POST /v1/projects?workspace_id=...
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_WORKSPACE_ID", "workspace_id is required")
		return
	}
	var in service.CreateProjectInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	project, err := h.svc.Create(r.Context(), workspaceID, in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	// Kick off async sync for git-backed projects
	if project.RepoURL != nil && *project.RepoURL != "" {
		_ = h.syncSvc.TriggerSync(r.Context(), workspaceID, project.ProjectID)
	}

	writeJSON(w, http.StatusCreated, map[string]any{"project": project})
}

// Delete handles DELETE /v1/projects/{project_id}?workspace_id=...
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_WORKSPACE_ID", "workspace_id is required")
		return
	}
	projectID := chi.URLParam(r, "project_id")
	if err := h.svc.Delete(r.Context(), workspaceID, projectID); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Sync handles POST /v1/projects/{project_id}/sync?workspace_id=...
// Returns 202 Accepted immediately; git operations run in background.
func (h *ProjectHandler) Sync(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace_id")
	if workspaceID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_WORKSPACE_ID", "workspace_id is required")
		return
	}
	projectID := chi.URLParam(r, "project_id")

	// Verify project exists and belongs to workspace
	project, err := h.svc.Get(r.Context(), workspaceID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "project not found")
		return
	}
	if project.RepoURL == nil || *project.RepoURL == "" {
		writeError(w, http.StatusBadRequest, "NO_REPO_URL", "project has no repo_url configured")
		return
	}

	if err := h.syncSvc.TriggerSync(r.Context(), workspaceID, projectID); err != nil {
		writeError(w, http.StatusInternalServerError, "SYNC_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "syncing"})
}
