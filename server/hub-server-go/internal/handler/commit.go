package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/goyais/hub/internal/middleware"
)

// CommitHandler handles /v1/executions/{id}/commit and /v1/executions/{id}/patch.
type CommitHandler struct {
	db        *sql.DB
	workerURL string
}

func NewCommitHandler(db *sql.DB, workerURL string) *CommitHandler {
	return &CommitHandler{db: db, workerURL: workerURL}
}

// POST /v1/executions/{execution_id}/commit?workspace_id=...
func (h *CommitHandler) Commit(w http.ResponseWriter, r *http.Request) {
	executionID := chi.URLParam(r, "execution_id")
	wsID := r.URL.Query().Get("workspace_id")
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "E_UNAUTHORIZED", "not authenticated")
		return
	}

	var body struct {
		Message string `json:"message"`
	}
	_ = decodeBody(r, &body)
	if body.Message == "" {
		body.Message = "feat: apply goyais execution changes"
	}

	// Verify execution belongs to workspace
	var state, worktreeRoot, sessionID string
	if err := h.db.QueryRowContext(r.Context(), `
		SELECT state, COALESCE(worktree_root,''), session_id
		FROM executions WHERE execution_id = ? AND workspace_id = ?`,
		executionID, wsID).Scan(&state, &worktreeRoot, &sessionID); err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "E_NOT_FOUND", "execution not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}

	if worktreeRoot == "" {
		writeError(w, http.StatusUnprocessableEntity, "E_NO_WORKTREE", "execution did not use worktree")
		return
	}

	// Fetch git_name / git_email from user profile
	var gitName, gitEmail sql.NullString
	_ = h.db.QueryRowContext(r.Context(),
		`SELECT COALESCE(git_name, display_name), COALESCE(git_email, email) FROM users WHERE user_id = ?`,
		user.UserID).Scan(&gitName, &gitEmail)

	// Dispatch commit to worker
	commitReq := map[string]string{
		"execution_id":  executionID,
		"worktree_root": worktreeRoot,
		"message":       body.Message,
		"git_name":      gitName.String,
		"git_email":     gitEmail.String,
	}

	var resp struct {
		CommitSHA string `json:"commit_sha"`
	}
	if err := workerPostJSON(h.workerURL+"/internal/executions/"+executionID+"/commit", commitReq, &resp); err != nil {
		writeError(w, http.StatusBadGateway, "E_WORKER_ERROR", err.Error())
		return
	}

	// Audit log
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = h.db.ExecContext(r.Context(), `
		INSERT INTO audit_logs
			(audit_id, workspace_id, session_id, execution_id, user_id, action, parameters_summary, outcome, created_at)
		VALUES (?, ?, ?, ?, ?, 'git.commit', ?, 'success', ?)`,
		fmt.Sprintf("%d", time.Now().UnixNano()),
		wsID, sessionID, executionID, user.UserID,
		resp.CommitSHA, now)

	writeJSON(w, http.StatusOK, map[string]string{"commit_sha": resp.CommitSHA})
}

// GET /v1/executions/{execution_id}/patch?workspace_id=...
func (h *CommitHandler) Patch(w http.ResponseWriter, r *http.Request) {
	executionID := chi.URLParam(r, "execution_id")
	wsID := r.URL.Query().Get("workspace_id")
	if wsID == "" {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "workspace_id is required")
		return
	}

	var exists int
	if err := h.db.QueryRowContext(
		r.Context(),
		`SELECT 1 FROM executions WHERE execution_id = ? AND workspace_id = ?`,
		executionID,
		wsID,
	).Scan(&exists); err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "E_NOT_FOUND", "execution not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}

	// Aggregate all patch events from execution_events
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT ee.payload_json
		FROM execution_events ee
		JOIN executions e ON e.execution_id = ee.execution_id
		WHERE ee.execution_id = ? AND e.workspace_id = ? AND ee.type = 'patch'
		ORDER BY ee.seq ASC`, executionID, wsID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}
	defer rows.Close()

	var patches []string
	for rows.Next() {
		var payloadJSON string
		if err := rows.Scan(&payloadJSON); err != nil {
			continue
		}
		if diff := extractJSONStringField(payloadJSON, "unified_diff"); diff != "" {
			patches = append(patches, diff)
		}
	}

	combined := strings.Join(patches, "\n")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="changes.patch"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(combined))
}

// DELETE /v1/executions/{execution_id}/discard?workspace_id=...
func (h *CommitHandler) Discard(w http.ResponseWriter, r *http.Request) {
	executionID := chi.URLParam(r, "execution_id")
	wsID := r.URL.Query().Get("workspace_id")
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "E_UNAUTHORIZED", "not authenticated")
		return
	}
	if wsID == "" {
		writeError(w, http.StatusBadRequest, "E_BAD_REQUEST", "workspace_id is required")
		return
	}

	var repoRoot, sessionID, worktreeRoot string
	if err := h.db.QueryRowContext(
		r.Context(),
		`SELECT COALESCE(repo_root,''), session_id, COALESCE(worktree_root,'')
		 FROM executions
		 WHERE execution_id = ? AND workspace_id = ?`,
		executionID,
		wsID,
	).Scan(&repoRoot, &sessionID, &worktreeRoot); err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "E_NOT_FOUND", "execution not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "E_INTERNAL", err.Error())
		return
	}

	if worktreeRoot == "" || repoRoot == "" {
		writeError(w, http.StatusUnprocessableEntity, "E_NO_WORKTREE", "execution did not use worktree")
		return
	}

	discardReq := map[string]string{
		"execution_id": executionID,
		"repo_root":    repoRoot,
	}
	if err := workerPostJSON(h.workerURL+"/internal/executions/"+executionID+"/discard", discardReq, nil); err != nil {
		writeError(w, http.StatusBadGateway, "E_WORKER_ERROR", err.Error())
		return
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = h.db.ExecContext(r.Context(), `
		INSERT INTO audit_logs
			(audit_id, workspace_id, session_id, execution_id, user_id, action, parameters_summary, outcome, created_at)
		VALUES (?, ?, ?, ?, ?, 'git.discard', ?, 'success', ?)`,
		fmt.Sprintf("%d", time.Now().UnixNano()),
		wsID, sessionID, executionID, user.UserID,
		"discard_worktree",
		now,
	)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// workerPostJSON POSTs JSON to the worker and optionally decodes the response.
func workerPostJSON(url string, body, dst any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("post %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("worker returned %d", resp.StatusCode)
	}
	if dst != nil {
		_ = json.NewDecoder(resp.Body).Decode(dst)
	}
	return nil
}

// extractJSONStringField extracts a JSON string value for the given key.
func extractJSONStringField(jsonStr, key string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}
