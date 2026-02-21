package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/goyais/hub/internal/middleware"
	"github.com/goyais/hub/internal/service"
)

type ModelConfigHandler struct {
	svc        *service.ModelConfigService
	runtimeSvc *service.RuntimeGatewayService
}

func NewModelConfigHandler(svc *service.ModelConfigService, runtimeSvc *service.RuntimeGatewayService) *ModelConfigHandler {
	return &ModelConfigHandler{svc: svc, runtimeSvc: runtimeSvc}
}

// GET /v1/model-configs?workspace_id=...
func (h *ModelConfigHandler) List(w http.ResponseWriter, r *http.Request) {
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	items, err := h.svc.List(r.Context(), workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"model_configs": items})
}

// POST /v1/model-configs?workspace_id=...
func (h *ModelConfigHandler) Create(w http.ResponseWriter, r *http.Request) {
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	var body struct {
		Provider    string   `json:"provider"`
		Model       string   `json:"model"`
		BaseURL     *string  `json:"base_url"`
		Temperature *float64 `json:"temperature"`
		MaxTokens   *int     `json:"max_tokens"`
		APIKey      string   `json:"api_key"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid body")
		return
	}

	item, err := h.svc.Create(r.Context(), workspaceID, service.CreateModelConfigInput{
		Provider:    body.Provider,
		Model:       body.Model,
		BaseURL:     body.BaseURL,
		Temperature: body.Temperature,
		MaxTokens:   body.MaxTokens,
		APIKey:      body.APIKey,
	})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "E_VALIDATION", err.Error())
		return
	}
	if h.runtimeSvc != nil {
		user := middleware.UserFromCtx(r.Context())
		if user != nil {
			if err := h.runtimeSvc.UpsertRuntimeModelConfig(
				r.Context(),
				user.UserID,
				middleware.TraceIDFromCtx(r.Context()),
				*item,
			); err != nil {
				writeError(w, http.StatusBadGateway, "E_RUNTIME_UPSTREAM", err.Error())
				return
			}
		}
	}

	writeJSON(w, http.StatusCreated, map[string]any{"model_config": item})
}

// PUT /v1/model-configs/{model_config_id}?workspace_id=...
func (h *ModelConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	modelConfigID := chi.URLParam(r, "model_config_id")

	var body struct {
		Provider    *string  `json:"provider"`
		Model       *string  `json:"model"`
		BaseURL     *string  `json:"base_url"`
		Temperature *float64 `json:"temperature"`
		MaxTokens   *int     `json:"max_tokens"`
		APIKey      *string  `json:"api_key"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid body")
		return
	}

	item, err := h.svc.Update(r.Context(), workspaceID, modelConfigID, service.UpdateModelConfigInput{
		Provider:    body.Provider,
		Model:       body.Model,
		BaseURL:     body.BaseURL,
		Temperature: body.Temperature,
		MaxTokens:   body.MaxTokens,
		APIKey:      body.APIKey,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	if item == nil {
		writeError(w, http.StatusNotFound, "E_NOT_FOUND", "model config not found")
		return
	}
	if h.runtimeSvc != nil {
		user := middleware.UserFromCtx(r.Context())
		if user != nil {
			if err := h.runtimeSvc.UpsertRuntimeModelConfig(
				r.Context(),
				user.UserID,
				middleware.TraceIDFromCtx(r.Context()),
				*item,
			); err != nil {
				writeError(w, http.StatusBadGateway, "E_RUNTIME_UPSTREAM", err.Error())
				return
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"model_config": item})
}

// DELETE /v1/model-configs/{model_config_id}?workspace_id=...
func (h *ModelConfigHandler) Delete(w http.ResponseWriter, r *http.Request) {
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	modelConfigID := chi.URLParam(r, "model_config_id")
	ok, err := h.svc.Delete(r.Context(), workspaceID, modelConfigID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "E_NOT_FOUND", "model config not found")
		return
	}
	if h.runtimeSvc != nil {
		user := middleware.UserFromCtx(r.Context())
		if user != nil {
			err := h.runtimeSvc.DeleteRuntimeModelConfig(
				r.Context(),
				user.UserID,
				middleware.TraceIDFromCtx(r.Context()),
				modelConfigID,
			)
			if err != nil && !strings.Contains(err.Error(), " 404:") {
				writeError(w, http.StatusBadGateway, "E_RUNTIME_UPSTREAM", err.Error())
				return
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
