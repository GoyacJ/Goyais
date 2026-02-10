package httpapi

import (
	"net/http"
	"strings"

	"goyais/internal/command"
	"goyais/internal/common/errorx"
)

const (
	headerTenantID    = "X-Tenant-Id"
	headerWorkspaceID = "X-Workspace-Id"
	headerUserID      = "X-User-Id"
)

func requireRequestContext(w http.ResponseWriter, r *http.Request) (command.RequestContext, bool) {
	ctx, missing := extractRequestContext(r)
	if len(missing) > 0 {
		errorx.Write(w, http.StatusBadRequest, "MISSING_CONTEXT", "error.context.missing", map[string]any{"missingHeaders": missing})
		return command.RequestContext{}, false
	}
	return ctx, true
}

func extractRequestContext(r *http.Request) (command.RequestContext, []string) {
	tenantID := strings.TrimSpace(r.Header.Get(headerTenantID))
	workspaceID := strings.TrimSpace(r.Header.Get(headerWorkspaceID))
	userID := strings.TrimSpace(r.Header.Get(headerUserID))

	missing := make([]string, 0, 3)
	if tenantID == "" {
		missing = append(missing, headerTenantID)
	}
	if workspaceID == "" {
		missing = append(missing, headerWorkspaceID)
	}
	if userID == "" {
		missing = append(missing, headerUserID)
	}

	return command.RequestContext{TenantID: tenantID, WorkspaceID: workspaceID, UserID: userID, OwnerID: userID}, missing
}
