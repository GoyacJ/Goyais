package sqlite

import (
	"context"
	"database/sql"

	"goyais/services/hub/internal/domain"
)

type WorkspaceRepository struct {
	db *sql.DB
}

func NewWorkspaceRepository(db *sql.DB) WorkspaceRepository {
	return WorkspaceRepository{db: db}
}

func (r WorkspaceRepository) GetByID(ctx context.Context, id domain.WorkspaceID) (domain.Workspace, bool, error) {
	if r.db == nil {
		return domain.Workspace{}, false, nil
	}
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, name, mode, hub_url, is_default_local, created_at, login_disabled, auth_mode
		 FROM workspaces
		 WHERE id = ?`,
		string(id),
	)

	var (
		workspace      domain.Workspace
		hubURL         sql.NullString
		isDefaultLocal int
		loginDisabled  int
	)
	err := row.Scan(
		&workspace.ID,
		&workspace.Name,
		&workspace.Mode,
		&hubURL,
		&isDefaultLocal,
		&workspace.CreatedAt,
		&loginDisabled,
		&workspace.AuthMode,
	)
	if err == sql.ErrNoRows {
		return domain.Workspace{}, false, nil
	}
	if err != nil {
		return domain.Workspace{}, false, err
	}
	if hubURL.Valid {
		value := hubURL.String
		workspace.HubURL = &value
	}
	workspace.IsDefaultLocal = isDefaultLocal == 1
	workspace.LoginDisabled = loginDisabled == 1
	return workspace, true, nil
}

func (r WorkspaceRepository) Save(ctx context.Context, workspace domain.Workspace) error {
	if r.db == nil {
		return nil
	}
	var hubURL any
	if workspace.HubURL != nil {
		hubURL = *workspace.HubURL
	}
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO workspaces(id, name, mode, hub_url, is_default_local, created_at, login_disabled, auth_mode)
		 VALUES(?,?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET
		 	name=excluded.name,
		 	mode=excluded.mode,
		 	hub_url=excluded.hub_url,
		 	is_default_local=excluded.is_default_local,
		 	login_disabled=excluded.login_disabled,
		 	auth_mode=excluded.auth_mode`,
		string(workspace.ID),
		workspace.Name,
		workspace.Mode,
		hubURL,
		boolToInt(workspace.IsDefaultLocal),
		workspace.CreatedAt,
		boolToInt(workspace.LoginDisabled),
		workspace.AuthMode,
	)
	return err
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
