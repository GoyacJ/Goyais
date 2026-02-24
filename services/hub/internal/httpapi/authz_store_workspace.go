package httpapi

import (
	"database/sql"
	"strings"
	"time"
)

func (s *authzStore) listWorkspaces() ([]Workspace, error) {
	rows, err := s.db.Query(
		`SELECT id, name, mode, hub_url, is_default_local, created_at, login_disabled, auth_mode
		 FROM workspaces
		 ORDER BY is_default_local DESC, created_at ASC, lower(name) ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Workspace, 0)
	for rows.Next() {
		item, scanErr := scanWorkspace(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *authzStore) getWorkspace(workspaceID string) (Workspace, bool, error) {
	row := s.db.QueryRow(
		`SELECT id, name, mode, hub_url, is_default_local, created_at, login_disabled, auth_mode
		 FROM workspaces
		 WHERE id=?`,
		strings.TrimSpace(workspaceID),
	)
	item, err := scanWorkspace(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return Workspace{}, false, nil
		}
		return Workspace{}, false, err
	}
	return item, true, nil
}

func (s *authzStore) getWorkspaceConnection(workspaceID string) (WorkspaceConnection, bool, error) {
	row := s.db.QueryRow(
		`SELECT workspace_id, hub_url, username, connection_status, connected_at
		 FROM workspace_connections
		 WHERE workspace_id=?`,
		strings.TrimSpace(workspaceID),
	)
	item, err := scanWorkspaceConnection(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return WorkspaceConnection{}, false, nil
		}
		return WorkspaceConnection{}, false, err
	}
	return item, true, nil
}

func (s *authzStore) hasRemoteWorkspace() (bool, error) {
	row := s.db.QueryRow(`SELECT COUNT(1) FROM workspaces WHERE mode=?`, string(WorkspaceModeRemote))
	count := 0
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *authzStore) upsertWorkspace(input Workspace) (Workspace, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	workspace := normalizeWorkspace(input, now)
	_, err := s.db.Exec(
		`INSERT INTO workspaces(id, name, mode, hub_url, is_default_local, created_at, login_disabled, auth_mode)
		 VALUES(?,?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET
		   name=excluded.name,
		   mode=excluded.mode,
		   hub_url=excluded.hub_url,
		   is_default_local=excluded.is_default_local,
		   login_disabled=excluded.login_disabled,
		   auth_mode=excluded.auth_mode`,
		workspace.ID,
		workspace.Name,
		string(workspace.Mode),
		workspace.HubURL,
		boolToInt(workspace.IsDefaultLocal),
		workspace.CreatedAt,
		boolToInt(workspace.LoginDisabled),
		string(workspace.AuthMode),
	)
	if err != nil {
		return Workspace{}, err
	}
	return workspace, nil
}

func (s *authzStore) upsertWorkspaceConnection(input WorkspaceConnection) error {
	now := time.Now().UTC().Format(time.RFC3339)
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	if workspaceID == "" {
		return nil
	}
	connectedAt := strings.TrimSpace(input.ConnectedAt)
	if connectedAt == "" {
		connectedAt = now
	}
	connectionStatus := strings.TrimSpace(input.ConnectionStatus)
	if connectionStatus == "" {
		connectionStatus = "connected"
	}
	_, err := s.db.Exec(
		`INSERT INTO workspace_connections(workspace_id, hub_url, username, connection_status, connected_at, updated_at)
		 VALUES(?,?,?,?,?,?)
		 ON CONFLICT(workspace_id) DO UPDATE SET
		   hub_url=excluded.hub_url,
		   username=excluded.username,
		   connection_status=excluded.connection_status,
		   connected_at=excluded.connected_at,
		   updated_at=excluded.updated_at`,
		workspaceID,
		strings.TrimSpace(input.HubURL),
		strings.TrimSpace(input.Username),
		connectionStatus,
		connectedAt,
		now,
	)
	return err
}

type workspaceScanner interface {
	Scan(dest ...any) error
}

func scanWorkspace(scanner workspaceScanner) (Workspace, error) {
	item := Workspace{}
	var modeRaw string
	var hubURL sql.NullString
	var isDefaultLocalInt int
	var loginDisabledInt int
	var authModeRaw string

	if err := scanner.Scan(
		&item.ID,
		&item.Name,
		&modeRaw,
		&hubURL,
		&isDefaultLocalInt,
		&item.CreatedAt,
		&loginDisabledInt,
		&authModeRaw,
	); err != nil {
		return Workspace{}, err
	}

	item.Mode = parseWorkspaceMode(modeRaw, isDefaultLocalInt == 1)
	item.IsDefaultLocal = isDefaultLocalInt == 1
	item.LoginDisabled = parseBoolInt(loginDisabledInt)
	item.AuthMode = parseAuthMode(authModeRaw, item.Mode)
	if hubURL.Valid && strings.TrimSpace(hubURL.String) != "" {
		value := strings.TrimSpace(hubURL.String)
		item.HubURL = &value
	}
	return item, nil
}

func scanWorkspaceConnection(scanner workspaceScanner) (WorkspaceConnection, error) {
	item := WorkspaceConnection{}
	if err := scanner.Scan(
		&item.WorkspaceID,
		&item.HubURL,
		&item.Username,
		&item.ConnectionStatus,
		&item.ConnectedAt,
	); err != nil {
		return WorkspaceConnection{}, err
	}
	return item, nil
}

func normalizeWorkspace(input Workspace, now string) Workspace {
	workspace := input
	workspace.ID = strings.TrimSpace(workspace.ID)
	workspace.Name = strings.TrimSpace(workspace.Name)
	if workspace.Name == "" {
		workspace.Name = "Workspace"
	}
	if strings.TrimSpace(workspace.CreatedAt) == "" {
		workspace.CreatedAt = now
	}
	workspace.Mode = parseWorkspaceMode(string(workspace.Mode), workspace.IsDefaultLocal)
	workspace.AuthMode = parseAuthMode(string(workspace.AuthMode), workspace.Mode)
	if workspace.Mode == WorkspaceModeLocal {
		workspace.IsDefaultLocal = true
		workspace.HubURL = nil
		workspace.LoginDisabled = true
		workspace.AuthMode = AuthModeDisabled
		return workspace
	}
	if workspace.HubURL != nil {
		hubURL := strings.TrimSpace(*workspace.HubURL)
		if hubURL == "" {
			workspace.HubURL = nil
		} else {
			workspace.HubURL = &hubURL
		}
	}
	return workspace
}

func parseWorkspaceMode(raw string, localHint bool) WorkspaceMode {
	if localHint {
		return WorkspaceModeLocal
	}
	if strings.TrimSpace(raw) == string(WorkspaceModeRemote) {
		return WorkspaceModeRemote
	}
	return WorkspaceModeLocal
}

func parseAuthMode(raw string, workspaceMode WorkspaceMode) AuthMode {
	switch AuthMode(strings.TrimSpace(raw)) {
	case AuthModeDisabled:
		return AuthModeDisabled
	case AuthModeTokenOnly:
		return AuthModeTokenOnly
	case AuthModePasswordOrToken:
		return AuthModePasswordOrToken
	}
	if workspaceMode == WorkspaceModeLocal {
		return AuthModeDisabled
	}
	return AuthModePasswordOrToken
}
