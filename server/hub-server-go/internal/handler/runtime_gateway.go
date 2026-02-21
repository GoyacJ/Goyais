package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/goyais/hub/internal/middleware"
	"github.com/goyais/hub/internal/service"
)

type RuntimeGatewayHandler struct {
	runtimeSvc     *service.RuntimeGatewayService
	modelConfigSvc *service.ModelConfigService
}

func NewRuntimeGatewayHandler(
	runtimeSvc *service.RuntimeGatewayService,
	modelConfigSvc *service.ModelConfigService,
) *RuntimeGatewayHandler {
	return &RuntimeGatewayHandler{
		runtimeSvc:     runtimeSvc,
		modelConfigSvc: modelConfigSvc,
	}
}

// GET /v1/runtime/model-configs/{model_config_id}/models?workspace_id=...
func (h *RuntimeGatewayHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	modelConfigID := chi.URLParam(r, "model_config_id")
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "E_UNAUTHORIZED", "not authenticated")
		return
	}

	existing, err := h.modelConfigSvc.Get(r.Context(), workspaceID, modelConfigID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "E_NOT_FOUND", "model config not found")
		return
	}

	payload, err := h.runtimeSvc.RuntimeModelCatalog(
		r.Context(),
		user.UserID,
		middleware.TraceIDFromCtx(r.Context()),
		modelConfigID,
		r.Header.Get("X-Api-Key-Override"),
	)
	if err != nil {
		writeError(w, http.StatusBadGateway, "E_RUNTIME_UPSTREAM", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

// GET /v1/runtime/health?workspace_id=...
func (h *RuntimeGatewayHandler) Health(w http.ResponseWriter, r *http.Request) {
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "E_UNAUTHORIZED", "not authenticated")
		return
	}

	upstream, err := h.runtimeSvc.RuntimeHealth(
		r.Context(),
		user.UserID,
		middleware.TraceIDFromCtx(r.Context()),
	)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"workspace_id":   workspaceID,
			"runtime_status": "offline",
			"error":          err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"workspace_id":     workspaceID,
		"runtime_base_url": h.runtimeSvc.RuntimeBaseURL(),
		"runtime_status":   "online",
		"upstream":         upstream,
	})
}
