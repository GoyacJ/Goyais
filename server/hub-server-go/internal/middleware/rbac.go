package middleware

import (
	"net/http"
	"strings"

	"github.com/goyais/hub/internal/model"
)

// RequirePerm returns a middleware that checks the authenticated user has the
// given permission within the workspace specified by the workspace_id query param.
func RequirePerm(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromCtx(r.Context())
			if user == nil {
				http.Error(w, `{"error":{"code":"E_UNAUTHORIZED"}}`, http.StatusUnauthorized)
				return
			}
			wsID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
			if wsID == "" {
				http.Error(w, `{"error":{"code":"E_BAD_REQUEST","message":"workspace_id is required"}}`, http.StatusBadRequest)
				return
			}
			if !user.HasPermIn(wsID, perm) {
				http.Error(w, `{"error":{"code":"E_FORBIDDEN","message":"insufficient permissions"}}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireInternalSecret validates X-Hub-Auth header for worker→hub internal calls.
func RequireInternalSecret(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" || r.Header.Get("X-Hub-Auth") != secret {
				http.Error(w, `{"error":{"code":"E_FORBIDDEN","message":"invalid internal secret"}}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// WorkspaceFromQuery ensures workspace_id query param is present.
func WorkspaceFromQuery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsID := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
		if wsID == "" {
			http.Error(w, `{"error":{"code":"E_BAD_REQUEST","message":"workspace_id is required"}}`, http.StatusBadRequest)
			return
		}
		// Validate the user is a member of this workspace.
		user := UserFromCtx(r.Context())
		if user != nil && !user.IsMember(wsID) {
			http.Error(w, `{"error":{"code":"E_FORBIDDEN","message":"not a member of workspace"}}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// WorkspacePerms returns a model.AuthUser method check helper type.
// (Actual RBAC data loaded at auth token validation time.)
var _ = model.AuthUser{}
