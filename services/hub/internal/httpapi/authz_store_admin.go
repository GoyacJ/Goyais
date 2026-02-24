package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

func (s *authzStore) listUsers(workspaceID string) ([]AdminUser, error) {
	rows, err := s.db.Query(
		`SELECT id, workspace_id, username, display_name, role, enabled, created_at
		 FROM users
		 WHERE workspace_id=?
		 ORDER BY created_at ASC`,
		strings.TrimSpace(workspaceID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AdminUser, 0)
	for rows.Next() {
		var item AdminUser
		var roleRaw string
		var enabledInt int
		if scanErr := rows.Scan(&item.ID, &item.WorkspaceID, &item.Username, &item.DisplayName, &roleRaw, &enabledInt, &item.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		item.Role = parseRoleValue(roleRaw)
		item.Enabled = parseBoolInt(enabledInt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *authzStore) upsertUser(input AdminUser) (AdminUser, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	row := s.db.QueryRow(
		`SELECT id, enabled, created_at
		 FROM users
		 WHERE workspace_id=? AND username=?`,
		input.WorkspaceID,
		input.Username,
	)
	existingID := ""
	existingEnabled := 1
	existingCreatedAt := now
	err := row.Scan(&existingID, &existingEnabled, &existingCreatedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return AdminUser{}, err
	}

	if strings.TrimSpace(string(input.Role)) == "" {
		input.Role = RoleDeveloper
	}
	if strings.TrimSpace(input.DisplayName) == "" {
		input.DisplayName = input.Username
	}

	if errors.Is(err, sql.ErrNoRows) {
		input.ID = "u_" + randomHex(6)
		input.CreatedAt = now
		input.Enabled = true
		_, execErr := s.db.Exec(
			`INSERT INTO users(id, workspace_id, username, password_hash, display_name, role, enabled, created_at, updated_at)
			 VALUES(?,?,?,?,?,?,?,?,?)`,
			input.ID,
			input.WorkspaceID,
			input.Username,
			hashPassword("pw"),
			input.DisplayName,
			string(input.Role),
			1,
			now,
			now,
		)
		if execErr != nil {
			return AdminUser{}, execErr
		}
		return input, nil
	}

	_, err = s.db.Exec(
		`UPDATE users SET display_name=?, role=?, updated_at=? WHERE id=?`,
		input.DisplayName,
		string(ensureRoleKnown(input.Role)),
		now,
		existingID,
	)
	if err != nil {
		return AdminUser{}, err
	}
	return AdminUser{
		ID:          existingID,
		WorkspaceID: input.WorkspaceID,
		Username:    input.Username,
		DisplayName: input.DisplayName,
		Role:        ensureRoleKnown(input.Role),
		Enabled:     parseBoolInt(existingEnabled),
		CreatedAt:   existingCreatedAt,
	}, nil
}

func (s *authzStore) setUserEnabled(userID string, enabled bool) (AdminUser, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE users SET enabled=?, updated_at=? WHERE id=?`, boolToInt(enabled), now, strings.TrimSpace(userID))
	if err != nil {
		return AdminUser{}, err
	}
	return s.getUserByID(userID)
}

func (s *authzStore) setUserRole(userID string, role Role) (AdminUser, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE users SET role=?, updated_at=? WHERE id=?`, string(ensureRoleKnown(role)), now, strings.TrimSpace(userID))
	if err != nil {
		return AdminUser{}, err
	}
	return s.getUserByID(userID)
}

func (s *authzStore) getUserByID(userID string) (AdminUser, error) {
	row := s.db.QueryRow(
		`SELECT id, workspace_id, username, display_name, role, enabled, created_at
		 FROM users WHERE id=?`,
		strings.TrimSpace(userID),
	)
	item := AdminUser{}
	var roleRaw string
	var enabledInt int
	if err := row.Scan(&item.ID, &item.WorkspaceID, &item.Username, &item.DisplayName, &roleRaw, &enabledInt, &item.CreatedAt); err != nil {
		return AdminUser{}, err
	}
	item.Role = parseRoleValue(roleRaw)
	item.Enabled = parseBoolInt(enabledInt)
	return item, nil
}

func (s *authzStore) deleteUser(userID string) error {
	_, err := s.db.Exec(`DELETE FROM users WHERE id=?`, strings.TrimSpace(userID))
	return err
}

func (s *authzStore) listRoles(workspaceID string) ([]AdminRole, error) {
	rows, err := s.db.Query(
		`SELECT role_key, name, enabled
		 FROM roles
		 WHERE workspace_id=?
		 ORDER BY role_key ASC`,
		strings.TrimSpace(workspaceID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AdminRole, 0)
	for rows.Next() {
		var roleKeyRaw string
		var name string
		var enabledInt int
		if scanErr := rows.Scan(&roleKeyRaw, &name, &enabledInt); scanErr != nil {
			return nil, scanErr
		}
		permissions, loadErr := s.listRolePermissions(workspaceID, parseRoleValue(roleKeyRaw))
		if loadErr != nil {
			return nil, loadErr
		}
		items = append(items, AdminRole{
			Key:         parseRoleValue(roleKeyRaw),
			Name:        name,
			Permissions: permissions,
			Enabled:     parseBoolInt(enabledInt),
		})
	}
	return items, rows.Err()
}

func (s *authzStore) upsertRole(workspaceID string, input AdminRole) (AdminRole, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	roleKey := ensureRoleKnown(input.Key)
	_, err := s.db.Exec(
		`INSERT INTO roles(workspace_id, role_key, name, enabled, created_at, updated_at)
		 VALUES(?,?,?,?,?,?)
		 ON CONFLICT(workspace_id, role_key) DO UPDATE SET name=excluded.name, enabled=excluded.enabled, updated_at=excluded.updated_at`,
		workspaceID,
		string(roleKey),
		strings.TrimSpace(input.Name),
		boolToInt(input.Enabled),
		now,
		now,
	)
	if err != nil {
		return AdminRole{}, err
	}
	if _, err = s.db.Exec(`DELETE FROM role_grants WHERE workspace_id=? AND role_key=?`, workspaceID, string(roleKey)); err != nil {
		return AdminRole{}, err
	}
	for _, permissionKey := range input.Permissions {
		if _, err = s.db.Exec(`INSERT OR IGNORE INTO role_grants(workspace_id, role_key, permission_key) VALUES(?,?,?)`, workspaceID, string(roleKey), permissionKey); err != nil {
			return AdminRole{}, err
		}
	}
	return AdminRole{
		Key:         roleKey,
		Name:        strings.TrimSpace(input.Name),
		Permissions: append([]string{}, input.Permissions...),
		Enabled:     input.Enabled,
	}, nil
}

func (s *authzStore) setRoleEnabled(workspaceID string, roleKey Role, enabled bool) (AdminRole, error) {
	_, err := s.db.Exec(`UPDATE roles SET enabled=?, updated_at=? WHERE workspace_id=? AND role_key=?`, boolToInt(enabled), time.Now().UTC().Format(time.RFC3339), strings.TrimSpace(workspaceID), string(ensureRoleKnown(roleKey)))
	if err != nil {
		return AdminRole{}, err
	}
	items, err := s.listRoles(workspaceID)
	if err != nil {
		return AdminRole{}, err
	}
	for _, item := range items {
		if item.Key == ensureRoleKnown(roleKey) {
			return item, nil
		}
	}
	return AdminRole{}, sql.ErrNoRows
}

func (s *authzStore) deleteRole(workspaceID string, roleKey Role) error {
	normalizedRole := string(ensureRoleKnown(roleKey))
	if _, err := s.db.Exec(`DELETE FROM role_grants WHERE workspace_id=? AND role_key=?`, strings.TrimSpace(workspaceID), normalizedRole); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM permission_visibility WHERE workspace_id=? AND role_key=?`, strings.TrimSpace(workspaceID), normalizedRole); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM roles WHERE workspace_id=? AND role_key=?`, strings.TrimSpace(workspaceID), normalizedRole)
	return err
}

func (s *authzStore) listPermissions(workspaceID string) ([]AdminPermission, error) {
	rows, err := s.db.Query(`SELECT permission_key, label, enabled FROM permissions WHERE workspace_id=? ORDER BY permission_key ASC`, strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AdminPermission, 0)
	for rows.Next() {
		item := AdminPermission{}
		var enabledInt int
		if scanErr := rows.Scan(&item.Key, &item.Label, &enabledInt); scanErr != nil {
			return nil, scanErr
		}
		item.Enabled = parseBoolInt(enabledInt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *authzStore) upsertPermission(workspaceID string, input AdminPermission) (AdminPermission, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO permissions(workspace_id, permission_key, label, enabled, created_at, updated_at)
		 VALUES(?,?,?,?,?,?)
		 ON CONFLICT(workspace_id, permission_key) DO UPDATE SET label=excluded.label, enabled=excluded.enabled, updated_at=excluded.updated_at`,
		strings.TrimSpace(workspaceID),
		strings.TrimSpace(input.Key),
		strings.TrimSpace(input.Label),
		boolToInt(input.Enabled),
		now,
		now,
	)
	if err != nil {
		return AdminPermission{}, err
	}
	return AdminPermission{Key: strings.TrimSpace(input.Key), Label: strings.TrimSpace(input.Label), Enabled: input.Enabled}, nil
}

func (s *authzStore) deletePermission(workspaceID string, permissionKey string) error {
	if _, err := s.db.Exec(`DELETE FROM role_grants WHERE workspace_id=? AND permission_key=?`, strings.TrimSpace(workspaceID), strings.TrimSpace(permissionKey)); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM permissions WHERE workspace_id=? AND permission_key=?`, strings.TrimSpace(workspaceID), strings.TrimSpace(permissionKey))
	return err
}

func (s *authzStore) listMenus(workspaceID string) ([]AdminMenu, error) {
	rows, err := s.db.Query(`SELECT menu_key, label, enabled FROM menus WHERE workspace_id=? ORDER BY menu_key ASC`, strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AdminMenu, 0)
	for rows.Next() {
		item := AdminMenu{}
		var enabledInt int
		if scanErr := rows.Scan(&item.Key, &item.Label, &enabledInt); scanErr != nil {
			return nil, scanErr
		}
		item.Enabled = parseBoolInt(enabledInt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *authzStore) upsertMenu(workspaceID string, input AdminMenu) (AdminMenu, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO menus(workspace_id, menu_key, label, enabled, created_at, updated_at)
		 VALUES(?,?,?,?,?,?)
		 ON CONFLICT(workspace_id, menu_key) DO UPDATE SET label=excluded.label, enabled=excluded.enabled, updated_at=excluded.updated_at`,
		strings.TrimSpace(workspaceID),
		strings.TrimSpace(input.Key),
		strings.TrimSpace(input.Label),
		boolToInt(input.Enabled),
		now,
		now,
	)
	if err != nil {
		return AdminMenu{}, err
	}
	return AdminMenu{Key: strings.TrimSpace(input.Key), Label: strings.TrimSpace(input.Label), Enabled: input.Enabled}, nil
}

func (s *authzStore) deleteMenu(workspaceID string, menuKey string) error {
	if _, err := s.db.Exec(`DELETE FROM permission_visibility WHERE workspace_id=? AND menu_key=?`, strings.TrimSpace(workspaceID), strings.TrimSpace(menuKey)); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM menus WHERE workspace_id=? AND menu_key=?`, strings.TrimSpace(workspaceID), strings.TrimSpace(menuKey))
	return err
}

func (s *authzStore) getMenuVisibility(workspaceID string, role Role) (map[string]PermissionVisibility, error) {
	rows, err := s.db.Query(
		`SELECT menu_key, visibility
		 FROM permission_visibility
		 WHERE workspace_id=? AND role_key=?`,
		strings.TrimSpace(workspaceID),
		string(ensureRoleKnown(role)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := map[string]PermissionVisibility{}
	for rows.Next() {
		var menuKey string
		var visibilityRaw string
		if scanErr := rows.Scan(&menuKey, &visibilityRaw); scanErr != nil {
			return nil, scanErr
		}
		items[menuKey] = PermissionVisibility(visibilityRaw)
	}
	return items, rows.Err()
}

func (s *authzStore) setMenuVisibility(workspaceID string, role Role, items map[string]PermissionVisibility) (RoleMenuVisibility, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return RoleMenuVisibility{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM permission_visibility WHERE workspace_id=? AND role_key=?`, strings.TrimSpace(workspaceID), string(ensureRoleKnown(role))); err != nil {
		return RoleMenuVisibility{}, err
	}
	for menuKey, visibility := range items {
		if _, err = tx.Exec(
			`INSERT INTO permission_visibility(workspace_id, role_key, menu_key, visibility, created_at, updated_at) VALUES(?,?,?,?,?,?)`,
			strings.TrimSpace(workspaceID),
			string(ensureRoleKnown(role)),
			menuKey,
			string(visibility),
			now,
			now,
		); err != nil {
			return RoleMenuVisibility{}, err
		}
	}
	err = tx.Commit()
	if err != nil {
		return RoleMenuVisibility{}, err
	}
	return RoleMenuVisibility{RoleKey: ensureRoleKnown(role), Items: items}, nil
}

func (s *authzStore) listABACPolicies(workspaceID string) ([]ABACPolicy, error) {
	rows, err := s.db.Query(
		`SELECT id, workspace_id, name, effect, priority, enabled, subject_expr, resource_expr, action_expr, context_expr, created_at, updated_at
		 FROM abac_policies
		 WHERE workspace_id=?
		 ORDER BY priority ASC, id ASC`,
		strings.TrimSpace(workspaceID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ABACPolicy, 0)
	for rows.Next() {
		item := ABACPolicy{}
		var enabledInt int
		var subjectRaw string
		var resourceRaw string
		var actionRaw string
		var contextRaw string
		if scanErr := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.Name,
			&item.Effect,
			&item.Priority,
			&enabledInt,
			&subjectRaw,
			&resourceRaw,
			&actionRaw,
			&contextRaw,
			&item.CreatedAt,
			&item.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		item.Enabled = parseBoolInt(enabledInt)
		item.Effect = ensureABACEffectKnown(item.Effect)
		item.SubjectExpr = decodeExpression(subjectRaw)
		item.ResourceExpr = decodeExpression(resourceRaw)
		item.ActionExpr = decodeExpression(actionRaw)
		item.ContextExpr = decodeExpression(contextRaw)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *authzStore) upsertABACPolicy(input ABACPolicy) (ABACPolicy, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(input.ID) == "" {
		input.ID = "policy_" + randomHex(6)
	}
	if input.Priority == 0 {
		input.Priority = 100
	}
	input.Effect = ensureABACEffectKnown(input.Effect)
	_, err := s.db.Exec(
		`INSERT INTO abac_policies(id, workspace_id, name, effect, priority, enabled, subject_expr, resource_expr, action_expr, context_expr, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET workspace_id=excluded.workspace_id, name=excluded.name, effect=excluded.effect, priority=excluded.priority, enabled=excluded.enabled, subject_expr=excluded.subject_expr, resource_expr=excluded.resource_expr, action_expr=excluded.action_expr, context_expr=excluded.context_expr, updated_at=excluded.updated_at`,
		input.ID,
		strings.TrimSpace(input.WorkspaceID),
		strings.TrimSpace(input.Name),
		string(input.Effect),
		input.Priority,
		boolToInt(input.Enabled),
		mustEncodeExpression(input.SubjectExpr),
		mustEncodeExpression(input.ResourceExpr),
		mustEncodeExpression(input.ActionExpr),
		mustEncodeExpression(input.ContextExpr),
		now,
		now,
	)
	if err != nil {
		return ABACPolicy{}, err
	}
	input.CreatedAt = now
	input.UpdatedAt = now
	return input, nil
}

func (s *authzStore) getABACPolicyByID(policyID string) (ABACPolicy, error) {
	row := s.db.QueryRow(
		`SELECT id, workspace_id, name, effect, priority, enabled, subject_expr, resource_expr, action_expr, context_expr, created_at, updated_at
		 FROM abac_policies WHERE id=?`,
		strings.TrimSpace(policyID),
	)
	item := ABACPolicy{}
	var enabledInt int
	var subjectRaw string
	var resourceRaw string
	var actionRaw string
	var contextRaw string
	if err := row.Scan(
		&item.ID,
		&item.WorkspaceID,
		&item.Name,
		&item.Effect,
		&item.Priority,
		&enabledInt,
		&subjectRaw,
		&resourceRaw,
		&actionRaw,
		&contextRaw,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return ABACPolicy{}, err
	}
	item.Enabled = parseBoolInt(enabledInt)
	item.Effect = ensureABACEffectKnown(item.Effect)
	item.SubjectExpr = decodeExpression(subjectRaw)
	item.ResourceExpr = decodeExpression(resourceRaw)
	item.ActionExpr = decodeExpression(actionRaw)
	item.ContextExpr = decodeExpression(contextRaw)
	return item, nil
}

func (s *authzStore) deleteABACPolicy(policyID string) error {
	_, err := s.db.Exec(`DELETE FROM abac_policies WHERE id=?`, strings.TrimSpace(policyID))
	return err
}

func (s *authzStore) appendAudit(workspaceID string, actorUserID string, actionKey string, targetType string, targetID string, result string, details map[string]any, traceID string) error {
	payload, _ := json.Marshal(details)
	_, err := s.db.Exec(
		`INSERT INTO audit_logs(id, workspace_id, actor_user_id, action_key, target_type, target_id, result, details_json, trace_id, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)`,
		"audit_"+randomHex(6),
		strings.TrimSpace(workspaceID),
		strings.TrimSpace(actorUserID),
		strings.TrimSpace(actionKey),
		strings.TrimSpace(targetType),
		strings.TrimSpace(targetID),
		strings.TrimSpace(result),
		string(payload),
		strings.TrimSpace(traceID),
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *authzStore) listAudit(workspaceID string) ([]AdminAuditEvent, error) {
	rows, err := s.db.Query(
		`SELECT id, actor_user_id, action_key, target_type, target_id, result, trace_id, created_at
		 FROM audit_logs
		 WHERE workspace_id=?
		 ORDER BY created_at DESC`,
		strings.TrimSpace(workspaceID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AdminAuditEvent, 0)
	for rows.Next() {
		item := AdminAuditEvent{}
		var actorUserID string
		var targetType string
		var targetID string
		if scanErr := rows.Scan(&item.ID, &actorUserID, &item.Action, &targetType, &targetID, &item.Result, &item.TraceID, &item.Timestamp); scanErr != nil {
			return nil, scanErr
		}
		item.Actor = actorUserID
		item.Resource = targetType + ":" + targetID
		items = append(items, item)
	}
	return items, rows.Err()
}
