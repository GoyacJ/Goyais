package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"goyais/services/hub/internal/domain"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) SessionRepository {
	return SessionRepository{db: db}
}

func (r SessionRepository) GetByID(ctx context.Context, id domain.SessionID) (domain.Session, bool, error) {
	if r.db == nil {
		return domain.Session{}, false, nil
	}
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, workspace_id, project_id, name, default_mode, model_config_id, working_dir,
		        additional_directories_json, rule_ids_json, skill_ids_json, mcp_ids_json,
		        next_sequence, active_run_id, created_at, updated_at
		   FROM domain_sessions
		  WHERE id = ?`,
		string(id),
	)

	var (
		session                     domain.Session
		additionalDirectoriesJSON   string
		ruleIDsJSON                 string
		skillIDsJSON                string
		mcpIDsJSON                  string
		activeRunID                 sql.NullString
	)
	err := row.Scan(
		&session.ID,
		&session.WorkspaceID,
		&session.ProjectID,
		&session.Name,
		&session.DefaultMode,
		&session.ModelConfigID,
		&session.WorkingDir,
		&additionalDirectoriesJSON,
		&ruleIDsJSON,
		&skillIDsJSON,
		&mcpIDsJSON,
		&session.NextSequence,
		&activeRunID,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return domain.Session{}, false, nil
	}
	if err != nil {
		return domain.Session{}, false, err
	}
	if err := decodeStringSlice(additionalDirectoriesJSON, &session.AdditionalDirectories); err != nil {
		return domain.Session{}, false, err
	}
	if err := decodeStringSlice(ruleIDsJSON, &session.RuleIDs); err != nil {
		return domain.Session{}, false, err
	}
	if err := decodeStringSlice(skillIDsJSON, &session.SkillIDs); err != nil {
		return domain.Session{}, false, err
	}
	if err := decodeStringSlice(mcpIDsJSON, &session.MCPIDs); err != nil {
		return domain.Session{}, false, err
	}
	if activeRunID.Valid {
		value := domain.RunID(activeRunID.String)
		session.ActiveRunID = &value
	}
	return session, true, nil
}

func (r SessionRepository) Save(ctx context.Context, session domain.Session) error {
	if r.db == nil {
		return nil
	}
	additionalDirectoriesJSON, err := encodeStringSlice(session.AdditionalDirectories)
	if err != nil {
		return err
	}
	ruleIDsJSON, err := encodeStringSlice(session.RuleIDs)
	if err != nil {
		return err
	}
	skillIDsJSON, err := encodeStringSlice(session.SkillIDs)
	if err != nil {
		return err
	}
	mcpIDsJSON, err := encodeStringSlice(session.MCPIDs)
	if err != nil {
		return err
	}
	var activeRunID any
	if session.ActiveRunID != nil {
		activeRunID = string(*session.ActiveRunID)
	}
	_, err = r.db.ExecContext(
		ctx,
		`INSERT INTO domain_sessions(
			id, workspace_id, project_id, name, default_mode, model_config_id, working_dir,
			additional_directories_json, rule_ids_json, skill_ids_json, mcp_ids_json,
			next_sequence, active_run_id, created_at, updated_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			workspace_id=excluded.workspace_id,
			project_id=excluded.project_id,
			name=excluded.name,
			default_mode=excluded.default_mode,
			model_config_id=excluded.model_config_id,
			working_dir=excluded.working_dir,
			additional_directories_json=excluded.additional_directories_json,
			rule_ids_json=excluded.rule_ids_json,
			skill_ids_json=excluded.skill_ids_json,
			mcp_ids_json=excluded.mcp_ids_json,
			next_sequence=excluded.next_sequence,
			active_run_id=excluded.active_run_id,
			updated_at=excluded.updated_at`,
		string(session.ID),
		string(session.WorkspaceID),
		session.ProjectID,
		session.Name,
		session.DefaultMode,
		session.ModelConfigID,
		session.WorkingDir,
		additionalDirectoriesJSON,
		ruleIDsJSON,
		skillIDsJSON,
		mcpIDsJSON,
		session.NextSequence,
		activeRunID,
		session.CreatedAt,
		session.UpdatedAt,
	)
	return err
}

func (r SessionRepository) ListByWorkspace(ctx context.Context, workspaceID domain.WorkspaceID) ([]domain.Session, error) {
	if r.db == nil {
		return []domain.Session{}, nil
	}
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id
		   FROM domain_sessions
		  WHERE workspace_id = ?
		  ORDER BY created_at ASC, id ASC`,
		string(workspaceID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Session{}
	for rows.Next() {
		var sessionID domain.SessionID
		if err := rows.Scan(&sessionID); err != nil {
			return nil, err
		}
		item, exists, err := r.GetByID(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		if exists {
			items = append(items, item)
		}
	}
	return items, rows.Err()
}

func encodeStringSlice(values []string) (string, error) {
	if len(values) == 0 {
		return "[]", nil
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func decodeStringSlice(raw string, target *[]string) error {
	if strings.TrimSpace(raw) == "" {
		*target = []string{}
		return nil
	}
	return json.Unmarshal([]byte(raw), target)
}
