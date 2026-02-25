package httpapi

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	errAuthInvalidCredentials = errors.New("invalid credentials")
	errAuthUserDisabled       = errors.New("user disabled")
	errAuthSessionExpired     = errors.New("session expired")
	errAuthSessionNotFound    = errors.New("session not found")
)

func (s *authzStore) authenticatePassword(workspaceID string, username string, password string, requestedRole Role, allowBootstrap bool) (AdminUser, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	normalizedWorkspace := strings.TrimSpace(workspaceID)
	normalizedUsername := strings.TrimSpace(username)
	if normalizedWorkspace == "" || normalizedUsername == "" || strings.TrimSpace(password) == "" {
		return AdminUser{}, errAuthInvalidCredentials
	}

	row := s.db.QueryRow(
		`SELECT id, workspace_id, username, display_name, role, enabled, created_at, password_hash
		 FROM users WHERE workspace_id=? AND username=?`,
		normalizedWorkspace,
		normalizedUsername,
	)
	user := AdminUser{}
	var roleRaw string
	var enabledInt int
	var passwordHash string
	err := row.Scan(&user.ID, &user.WorkspaceID, &user.Username, &user.DisplayName, &roleRaw, &enabledInt, &user.CreatedAt, &passwordHash)
	if err == nil {
		if enabledInt == 0 {
			return AdminUser{}, errAuthUserDisabled
		}
		if hashPassword(password) != passwordHash {
			return AdminUser{}, errAuthInvalidCredentials
		}
		user.Role = parseRole(roleRaw)
		user.Enabled = true
		return user, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return AdminUser{}, err
	}
	if !allowBootstrap {
		return AdminUser{}, errAuthInvalidCredentials
	}

	userCountRow := s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE workspace_id=?`, normalizedWorkspace)
	userCount := 0
	if countErr := userCountRow.Scan(&userCount); countErr != nil {
		return AdminUser{}, countErr
	}

	role := requestedRole
	if userCount == 0 {
		role = RoleAdmin
	} else if role == "" {
		role = RoleDeveloper
	}
	if role == "" {
		role = RoleDeveloper
	}

	created := AdminUser{
		ID:          "u_" + randomHex(6),
		WorkspaceID: normalizedWorkspace,
		Username:    normalizedUsername,
		DisplayName: normalizedUsername,
		Role:        role,
		Enabled:     true,
		CreatedAt:   now,
	}
	_, err = s.db.Exec(
		`INSERT INTO users(id, workspace_id, username, password_hash, display_name, role, enabled, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		created.ID,
		created.WorkspaceID,
		created.Username,
		hashPassword(password),
		created.DisplayName,
		string(created.Role),
		1,
		now,
		now,
	)
	if err != nil {
		return AdminUser{}, err
	}
	return created, nil
}

func (s *authzStore) createSessionFromUser(user AdminUser) (Session, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(defaultAccessTokenTTL)
	refreshExpiresAt := now.Add(defaultRefreshTokenTTL)

	session := Session{
		Token:            "at_" + randomHex(16),
		RefreshToken:     "rt_" + randomHex(16),
		WorkspaceID:      user.WorkspaceID,
		Role:             user.Role,
		UserID:           user.ID,
		DisplayName:      user.DisplayName,
		ExpiresAt:        expiresAt,
		RefreshExpiresAt: refreshExpiresAt,
		Revoked:          false,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	_, err := s.db.Exec(
		`INSERT INTO sessions(access_token, refresh_token, workspace_id, user_id, display_name, role, expires_at, refresh_expires_at, revoked, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		session.Token,
		session.RefreshToken,
		session.WorkspaceID,
		session.UserID,
		session.DisplayName,
		string(session.Role),
		session.ExpiresAt.Format(time.RFC3339),
		session.RefreshExpiresAt.Format(time.RFC3339),
		0,
		session.CreatedAt.Format(time.RFC3339),
		session.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *authzStore) createSessionWithRole(workspaceID string, username string, role Role) (Session, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	userID := "u_" + strings.TrimSpace(username)
	if strings.TrimSpace(username) == "" {
		userID = "u_remote_user"
	}
	user := AdminUser{
		ID:          userID,
		WorkspaceID: strings.TrimSpace(workspaceID),
		Username:    strings.TrimPrefix(userID, "u_"),
		DisplayName: strings.TrimPrefix(userID, "u_"),
		Role:        role,
		Enabled:     true,
		CreatedAt:   now,
	}
	return s.createSessionFromUser(user)
}

func (s *authzStore) getSession(accessToken string) (Session, bool, error) {
	row := s.db.QueryRow(
		`SELECT access_token, refresh_token, workspace_id, user_id, display_name, role, expires_at, refresh_expires_at, revoked, created_at, updated_at
		 FROM sessions
		 WHERE access_token=?`,
		strings.TrimSpace(accessToken),
	)
	session := Session{}
	var roleRaw string
	var revokedInt int
	var expiresAtRaw string
	var refreshExpiresAtRaw string
	var createdAtRaw string
	var updatedAtRaw string
	if err := row.Scan(
		&session.Token,
		&session.RefreshToken,
		&session.WorkspaceID,
		&session.UserID,
		&session.DisplayName,
		&roleRaw,
		&expiresAtRaw,
		&refreshExpiresAtRaw,
		&revokedInt,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, false, nil
		}
		return Session{}, false, err
	}
	session.Role = parseRole(roleRaw)
	session.Revoked = revokedInt == 1
	session.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAtRaw)
	session.RefreshExpiresAt, _ = time.Parse(time.RFC3339, refreshExpiresAtRaw)
	session.CreatedAt, _ = time.Parse(time.RFC3339, createdAtRaw)
	session.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtRaw)
	if session.Revoked || time.Now().UTC().After(session.ExpiresAt) {
		return Session{}, false, nil
	}
	return session, true, nil
}

func (s *authzStore) refreshSession(refreshToken string) (Session, bool, error) {
	now := time.Now().UTC()
	row := s.db.QueryRow(
		`SELECT access_token, workspace_id, user_id, display_name, role, refresh_expires_at, revoked
		 FROM sessions WHERE refresh_token=?`,
		strings.TrimSpace(refreshToken),
	)

	var oldAccessToken string
	var workspaceID string
	var userID string
	var displayName string
	var roleRaw string
	var refreshExpiresAtRaw string
	var revokedInt int
	if err := row.Scan(&oldAccessToken, &workspaceID, &userID, &displayName, &roleRaw, &refreshExpiresAtRaw, &revokedInt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, false, nil
		}
		return Session{}, false, err
	}
	if revokedInt == 1 {
		return Session{}, false, nil
	}
	refreshExpiresAt, err := time.Parse(time.RFC3339, refreshExpiresAtRaw)
	if err != nil {
		return Session{}, false, err
	}
	if now.After(refreshExpiresAt) {
		return Session{}, false, nil
	}

	newAccessToken := "at_" + randomHex(16)
	newRefreshToken := "rt_" + randomHex(16)
	expiresAt := now.Add(defaultAccessTokenTTL)
	refreshExpiresAt = now.Add(defaultRefreshTokenTTL)

	_, err = s.db.Exec(
		`UPDATE sessions
		 SET access_token=?, refresh_token=?, expires_at=?, refresh_expires_at=?, updated_at=?, revoked=0
		 WHERE access_token=?`,
		newAccessToken,
		newRefreshToken,
		expiresAt.Format(time.RFC3339),
		refreshExpiresAt.Format(time.RFC3339),
		now.Format(time.RFC3339),
		oldAccessToken,
	)
	if err != nil {
		return Session{}, false, err
	}

	return Session{
		Token:            newAccessToken,
		RefreshToken:     newRefreshToken,
		WorkspaceID:      workspaceID,
		UserID:           userID,
		DisplayName:      displayName,
		Role:             parseRole(roleRaw),
		ExpiresAt:        expiresAt,
		RefreshExpiresAt: refreshExpiresAt,
		Revoked:          false,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, true, nil
}

func (s *authzStore) revokeSession(accessToken string) error {
	_, err := s.db.Exec(`UPDATE sessions SET revoked=1, updated_at=? WHERE access_token=?`, time.Now().UTC().Format(time.RFC3339), strings.TrimSpace(accessToken))
	return err
}

func (s *authzStore) buildPermissionSnapshot(workspaceID string, role Role) (PermissionSnapshot, error) {
	permissions, err := s.listRolePermissions(workspaceID, role)
	if err != nil {
		return PermissionSnapshot{}, err
	}
	menuVisibility, err := s.getMenuVisibility(workspaceID, role)
	if err != nil {
		return PermissionSnapshot{}, err
	}
	knownPermissions, err := s.listPermissions(workspaceID)
	if err != nil {
		return PermissionSnapshot{}, err
	}

	granted := map[string]bool{}
	for _, permission := range permissions {
		granted[permission] = true
	}
	actionVisibility := map[string]PermissionVisibility{}
	for _, item := range knownPermissions {
		if granted["*"] || granted[item.Key] {
			actionVisibility[item.Key] = PermissionVisibilityEnabled
			continue
		}
		if strings.HasSuffix(item.Key, ".read") {
			actionVisibility[item.Key] = PermissionVisibilityReadonly
			continue
		}
		actionVisibility[item.Key] = PermissionVisibilityDisabled
	}

	sort.Strings(permissions)
	return PermissionSnapshot{
		Role:             role,
		Permissions:      permissions,
		MenuVisibility:   menuVisibility,
		ActionVisibility: actionVisibility,
		// PolicyVersion tracks auth model schema, not release build version.
		PolicyVersion: "v0.4.0-rbac-abac-json-1",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *authzStore) listRolePermissions(workspaceID string, role Role) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT rg.permission_key
		 FROM role_grants rg
		 INNER JOIN roles r ON r.workspace_id=rg.workspace_id AND r.role_key=rg.role_key
		 WHERE rg.workspace_id=? AND rg.role_key=? AND r.enabled=1`,
		strings.TrimSpace(workspaceID),
		string(role),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]string, 0)
	for rows.Next() {
		var permissionKey string
		if scanErr := rows.Scan(&permissionKey); scanErr != nil {
			return nil, scanErr
		}
		items = append(items, permissionKey)
	}
	return items, rows.Err()
}

func hashPassword(password string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(password)))
	return hex.EncodeToString(sum[:])
}

func decodeExpression(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	value := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return map[string]any{}
	}
	return value
}

func mustEncodeExpression(input map[string]any) string {
	if input == nil {
		input = map[string]any{}
	}
	raw, _ := json.Marshal(input)
	return string(raw)
}

func parseBoolInt(value int) bool {
	return value == 1
}

func parseRoleValue(raw string) Role {
	return parseRole(raw)
}

func ensureRoleKnown(role Role) Role {
	switch role {
	case RoleAdmin, RoleApprover, RoleDeveloper, RoleViewer:
		return role
	default:
		return RoleDeveloper
	}
}

func ensureABACEffectKnown(effect ABACEffect) ABACEffect {
	if effect == ABACEffectDeny {
		return ABACEffectDeny
	}
	return ABACEffectAllow
}
