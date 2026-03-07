package sqlite

import (
	"context"
	"database/sql"

	"goyais/services/hub/internal/domain"
)

type RunRepository struct {
	db *sql.DB
}

func NewRunRepository(db *sql.DB) RunRepository {
	return RunRepository{db: db}
}

func (r RunRepository) GetByID(ctx context.Context, id domain.RunID) (domain.Run, bool, error) {
	if r.db == nil {
		return domain.Run{}, false, nil
	}
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, session_id, workspace_id, state, input_text, working_dir,
		        additional_directories_json, created_at, updated_at
		   FROM domain_runs
		  WHERE id = ?`,
		string(id),
	)
	var (
		run                       domain.Run
		additionalDirectoriesJSON string
	)
	err := row.Scan(
		&run.ID,
		&run.SessionID,
		&run.WorkspaceID,
		&run.State,
		&run.InputText,
		&run.WorkingDir,
		&additionalDirectoriesJSON,
		&run.CreatedAt,
		&run.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return domain.Run{}, false, nil
	}
	if err != nil {
		return domain.Run{}, false, err
	}
	if err := decodeStringSlice(additionalDirectoriesJSON, &run.AdditionalDirectories); err != nil {
		return domain.Run{}, false, err
	}
	return run, true, nil
}

func (r RunRepository) Save(ctx context.Context, run domain.Run) error {
	if r.db == nil {
		return nil
	}
	additionalDirectoriesJSON, err := encodeStringSlice(run.AdditionalDirectories)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(
		ctx,
		`INSERT INTO domain_runs(
			id, session_id, workspace_id, state, input_text, working_dir,
			additional_directories_json, created_at, updated_at
		) VALUES(?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			session_id=excluded.session_id,
			workspace_id=excluded.workspace_id,
			state=excluded.state,
			input_text=excluded.input_text,
			working_dir=excluded.working_dir,
			additional_directories_json=excluded.additional_directories_json,
			updated_at=excluded.updated_at`,
		string(run.ID),
		string(run.SessionID),
		string(run.WorkspaceID),
		string(run.State),
		run.InputText,
		run.WorkingDir,
		additionalDirectoriesJSON,
		run.CreatedAt,
		run.UpdatedAt,
	)
	return err
}

func (r RunRepository) ListBySession(ctx context.Context, sessionID domain.SessionID) ([]domain.Run, error) {
	if r.db == nil {
		return []domain.Run{}, nil
	}
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id
		   FROM domain_runs
		  WHERE session_id = ?
		  ORDER BY created_at ASC, id ASC`,
		string(sessionID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Run{}
	for rows.Next() {
		var runID domain.RunID
		if err := rows.Scan(&runID); err != nil {
			return nil, err
		}
		item, exists, err := r.GetByID(ctx, runID)
		if err != nil {
			return nil, err
		}
		if exists {
			items = append(items, item)
		}
	}
	return items, rows.Err()
}
