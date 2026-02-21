package handler

import (
	"net/http"
	"strings"

	"github.com/goyais/hub/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// GET /v1/auth/bootstrap/status
func (h *AuthHandler) BootstrapStatus(w http.ResponseWriter, r *http.Request) {
	done, err := h.svc.IsBootstrapComplete(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"setup_completed": done})
}

// POST /v1/auth/bootstrap/admin
func (h *AuthHandler) BootstrapAdmin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"display_name"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid body")
		return
	}
	token, err := h.svc.BootstrapAdmin(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		writeError(w, http.StatusConflict, "E_ALREADY_BOOTSTRAPPED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"token": token})
}

// POST /v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid body")
		return
	}
	token, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "E_UNAUTHORIZED", "invalid credentials")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// POST /v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Token already validated by middleware; revoke it
	authHeader := r.Header.Get("Authorization")
	if err := h.svc.Logout(r.Context(), authHeader); err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /v1/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.svc.Me(r.Context())
	if err != nil {
		if strings.Contains(err.Error(), "not authenticated") {
			writeError(w, http.StatusUnauthorized, "E_UNAUTHORIZED", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// GET /v1/me/navigation
func (h *AuthHandler) Navigation(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	if strings.TrimSpace(wsID) == "" {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "workspace_id is required")
		return
	}
	nav, err := h.svc.Navigation(r.Context(), wsID)
	if err != nil {
		if strings.Contains(err.Error(), "not authenticated") {
			writeError(w, http.StatusUnauthorized, "E_UNAUTHORIZED", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nav)
}
