package model

// AuthUser is the in-memory representation of an authenticated principal,
// populated at token validation time (from DB join of user + workspace_members + role_permissions).
type AuthUser struct {
	UserID      string
	Email       string
	DisplayName string
	// perms is a set of "workspace_id:perm_key" strings for O(1) lookup.
	perms map[string]struct{}
	// workspaces is the set of workspace IDs this user belongs to.
	workspaces map[string]struct{}
}

func NewAuthUser(userID, email, displayName string) *AuthUser {
	return &AuthUser{
		UserID:      userID,
		Email:       email,
		DisplayName: displayName,
		perms:       make(map[string]struct{}),
		workspaces:  make(map[string]struct{}),
	}
}

// GrantPerm adds a scoped permission (workspace_id + perm_key).
func (u *AuthUser) GrantPerm(workspaceID, permKey string) {
	u.perms[workspaceID+":"+permKey] = struct{}{}
	u.workspaces[workspaceID] = struct{}{}
}

// HasPerm checks for a global perm (any workspace). Caller should use
// HasPermIn for workspace-scoped checks.
func (u *AuthUser) HasPerm(permKey string) bool {
	for k := range u.perms {
		// k is "workspace_id:perm_key"
		if len(k) > len(permKey) && k[len(k)-len(permKey)-1:] == ":"+permKey {
			return true
		}
	}
	return false
}

// HasPermIn checks for a permission within a specific workspace.
func (u *AuthUser) HasPermIn(workspaceID, permKey string) bool {
	_, ok := u.perms[workspaceID+":"+permKey]
	return ok
}

// IsMember returns true if the user belongs to the given workspace.
func (u *AuthUser) IsMember(workspaceID string) bool {
	_, ok := u.workspaces[workspaceID]
	return ok
}
