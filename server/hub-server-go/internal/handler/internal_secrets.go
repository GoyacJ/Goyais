package handler

import (
	"database/sql"
	"net/http"
	"strings"
)

type InternalSecretsHandler struct {
	db *sql.DB
}

func NewInternalSecretsHandler(db *sql.DB) *InternalSecretsHandler {
	return &InternalSecretsHandler{db: db}
}

// POST /internal/secrets/resolve
func (h *InternalSecretsHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WorkspaceID string `json:"workspace_id"`
		SecretRef   string `json:"secret_ref"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "invalid body")
		return
	}
	workspaceID := strings.TrimSpace(body.WorkspaceID)
	secretRef := strings.TrimSpace(body.SecretRef)
	if workspaceID == "" || secretRef == "" {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "workspace_id and secret_ref are required")
		return
	}

	var value string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT value_encrypted FROM secrets WHERE workspace_id = ? AND secret_ref = ?`,
		workspaceID, secretRef,
	).Scan(&value)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "E_NOT_FOUND", "secret not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"value": value})
}
