package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

func defaultLocalWorkspace(now string) Workspace {
	return Workspace{
		ID:             localWorkspaceID,
		Name:           "Local",
		Mode:           WorkspaceModeLocal,
		HubURL:         nil,
		IsDefaultLocal: true,
		CreatedAt:      now,
		LoginDisabled:  true,
		AuthMode:       AuthModeDisabled,
	}
}

func (s *AppState) setWorkspaceCache(workspace Workspace) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setWorkspaceCacheLocked(workspace)
}

func (s *AppState) setWorkspaceCacheLocked(workspace Workspace) {
	s.workspaces[workspace.ID] = workspace
}

func (s *AppState) syncWorkspaceCache(workspaces []Workspace) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := make(map[string]Workspace, len(workspaces))
	for _, workspace := range workspaces {
		next[workspace.ID] = workspace
	}
	if _, exists := next[localWorkspaceID]; !exists {
		local := defaultLocalWorkspace(time.Now().UTC().Format(time.RFC3339))
		next[local.ID] = local
	}
	s.workspaces = next
}

func sortWorkspaces(items []Workspace) {
	sort.Slice(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		leftLocal := left.ID == localWorkspaceID || left.Mode == WorkspaceModeLocal || left.IsDefaultLocal
		rightLocal := right.ID == localWorkspaceID || right.Mode == WorkspaceModeLocal || right.IsDefaultLocal
		if leftLocal != rightLocal {
			return leftLocal
		}
		return strings.ToLower(left.Name) < strings.ToLower(right.Name)
	})
}

func defaultRoles() map[Role]AdminRole {
	return map[Role]AdminRole{
		RoleViewer: {
			Key:         RoleViewer,
			Name:        "Viewer",
			Permissions: []string{"read"},
			Enabled:     true,
		},
		RoleDeveloper: {
			Key:         RoleDeveloper,
			Name:        "Developer",
			Permissions: []string{"read", "write", "execute"},
			Enabled:     true,
		},
		RoleApprover: {
			Key:         RoleApprover,
			Name:        "Approver",
			Permissions: []string{"read", "approve"},
			Enabled:     true,
		},
		RoleAdmin: {
			Key:         RoleAdmin,
			Name:        "Admin",
			Permissions: []string{"*"},
			Enabled:     true,
		},
	}
}

func randomHex(bytes int) string {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return GenerateTraceID()
	}
	return hex.EncodeToString(buf)
}
