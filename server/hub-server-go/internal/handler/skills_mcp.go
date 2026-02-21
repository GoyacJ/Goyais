package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/goyais/hub/internal/service"
)

// ─────────────────────────────────────────────────────────────────────────────
// Skill Sets Handler
// ─────────────────────────────────────────────────────────────────────────────

type SkillSetHandler struct {
	svc *service.SkillSetService
}

func NewSkillSetHandler(svc *service.SkillSetService) *SkillSetHandler {
	return &SkillSetHandler{svc: svc}
}

// GET /v1/skill-sets?workspace_id=...
func (h *SkillSetHandler) List(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	items, err := h.svc.List(r.Context(), wsID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	if items == nil {
		items = []service.SkillSetSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"skill_sets": items})
}

// POST /v1/skill-sets?workspace_id=...
func (h *SkillSetHandler) Create(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	var in service.CreateSkillSetInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid body")
		return
	}
	ss, err := h.svc.Create(r.Context(), wsID, in)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "E_VALIDATION", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"skill_set": ss})
}

// PUT /v1/skill-sets/{skill_set_id}?workspace_id=...
func (h *SkillSetHandler) Update(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	skillSetID := chi.URLParam(r, "skill_set_id")
	var in service.UpdateSkillSetInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid body")
		return
	}
	ss, err := h.svc.Update(r.Context(), wsID, skillSetID, in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	if ss == nil {
		writeError(w, http.StatusNotFound, "E_NOT_FOUND", "skill set not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"skill_set": ss})
}

// DELETE /v1/skill-sets/{skill_set_id}?workspace_id=...
func (h *SkillSetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	skillSetID := chi.URLParam(r, "skill_set_id")
	if err := h.svc.Delete(r.Context(), wsID, skillSetID); err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /v1/skill-sets/{skill_set_id}/skills?workspace_id=...
func (h *SkillSetHandler) ListSkills(w http.ResponseWriter, r *http.Request) {
	skillSetID := chi.URLParam(r, "skill_set_id")
	items, err := h.svc.ListSkills(r.Context(), skillSetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	if items == nil {
		items = []service.SkillSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"skills": items})
}

// POST /v1/skill-sets/{skill_set_id}/skills?workspace_id=...
func (h *SkillSetHandler) CreateSkill(w http.ResponseWriter, r *http.Request) {
	skillSetID := chi.URLParam(r, "skill_set_id")
	var in service.CreateSkillInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid body")
		return
	}
	sk, err := h.svc.CreateSkill(r.Context(), skillSetID, in)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "E_VALIDATION", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"skill": sk})
}

// DELETE /v1/skills/{skill_id}?workspace_id=...
func (h *SkillSetHandler) DeleteSkill(w http.ResponseWriter, r *http.Request) {
	skillID := chi.URLParam(r, "skill_id")
	if err := h.svc.DeleteSkill(r.Context(), skillID); err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─────────────────────────────────────────────────────────────────────────────
// MCP Connectors Handler
// ─────────────────────────────────────────────────────────────────────────────

type MCPConnectorHandler struct {
	svc *service.MCPConnectorService
}

func NewMCPConnectorHandler(svc *service.MCPConnectorService) *MCPConnectorHandler {
	return &MCPConnectorHandler{svc: svc}
}

// GET /v1/mcp-connectors?workspace_id=...
func (h *MCPConnectorHandler) List(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	items, err := h.svc.List(r.Context(), wsID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	if items == nil {
		items = []service.MCPConnectorSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"mcp_connectors": items})
}

// POST /v1/mcp-connectors?workspace_id=...
func (h *MCPConnectorHandler) Create(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	var in service.CreateMCPConnectorInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid body")
		return
	}
	mc, err := h.svc.Create(r.Context(), wsID, in)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "E_VALIDATION", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"mcp_connector": mc})
}

// PUT /v1/mcp-connectors/{connector_id}?workspace_id=...
func (h *MCPConnectorHandler) Update(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	connectorID := chi.URLParam(r, "connector_id")
	var in service.UpdateMCPConnectorInput
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid body")
		return
	}
	mc, err := h.svc.Update(r.Context(), wsID, connectorID, in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	if mc == nil {
		writeError(w, http.StatusNotFound, "E_NOT_FOUND", "connector not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"mcp_connector": mc})
}

// DELETE /v1/mcp-connectors/{connector_id}?workspace_id=...
func (h *MCPConnectorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	connectorID := chi.URLParam(r, "connector_id")
	if err := h.svc.Delete(r.Context(), wsID, connectorID); err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
