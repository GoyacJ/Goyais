package httpapi

import (
	"database/sql"
	"strings"
	"time"
)

func (s *authzStore) listProjects(workspaceID string) ([]Project, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	query := `SELECT id, workspace_id, name, repo_path, is_git, default_model_config_id, default_mode, current_revision, created_at, updated_at
		FROM projects`
	args := []any{}
	if workspaceID != "" {
		query += ` WHERE workspace_id=?`
		args = append(args, workspaceID)
	}
	query += ` ORDER BY created_at DESC, updated_at DESC, id DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Project, 0)
	for rows.Next() {
		item, scanErr := scanProjectRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *authzStore) getProject(projectID string) (Project, bool, error) {
	row := s.db.QueryRow(
		`SELECT id, workspace_id, name, repo_path, is_git, default_model_config_id, default_mode, current_revision, created_at, updated_at
		 FROM projects
		 WHERE id=?`,
		strings.TrimSpace(projectID),
	)
	item, err := scanProjectRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return Project{}, false, nil
		}
		return Project{}, false, err
	}
	return item, true, nil
}

func (s *authzStore) upsertProject(input Project) (Project, error) {
	project := normalizeProjectForStorage(input)
	_, err := s.db.Exec(
		`INSERT INTO projects(id, workspace_id, name, repo_path, is_git, default_model_config_id, default_mode, current_revision, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET
		   workspace_id=excluded.workspace_id,
		   name=excluded.name,
		   repo_path=excluded.repo_path,
		   is_git=excluded.is_git,
		   default_model_config_id=excluded.default_model_config_id,
		   default_mode=excluded.default_mode,
		   current_revision=excluded.current_revision,
		   updated_at=excluded.updated_at`,
		project.ID,
		project.WorkspaceID,
		project.Name,
		project.RepoPath,
		boolToInt(project.IsGit),
		nullWhenEmpty(project.DefaultModelConfigID),
		string(project.DefaultMode),
		project.CurrentRevision,
		project.CreatedAt,
		project.UpdatedAt,
	)
	if err != nil {
		return Project{}, err
	}
	return project, nil
}

func (s *authzStore) deleteProject(projectID string) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return sql.ErrNoRows
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM project_configs WHERE project_id=?`, projectID); err != nil {
		return err
	}
	result, err := tx.Exec(`DELETE FROM projects WHERE id=?`, projectID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

type projectScanner interface {
	Scan(dest ...any) error
}

func scanProjectRow(scanner projectScanner) (Project, error) {
	item := Project{}
	var (
		isGitInt             int
		defaultModelConfigID sql.NullString
		defaultModeRaw       string
	)
	if err := scanner.Scan(
		&item.ID,
		&item.WorkspaceID,
		&item.Name,
		&item.RepoPath,
		&isGitInt,
		&defaultModelConfigID,
		&defaultModeRaw,
		&item.CurrentRevision,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return Project{}, err
	}
	item.IsGit = parseBoolInt(isGitInt)
	item.DefaultModelConfigID = strings.TrimSpace(defaultModelConfigID.String)
	if strings.TrimSpace(defaultModeRaw) == "" {
		item.DefaultMode = ConversationModeAgent
	} else {
		item.DefaultMode = ConversationMode(defaultModeRaw)
	}
	return normalizeProjectForStorage(item), nil
}

func normalizeProjectForStorage(input Project) Project {
	now := time.Now().UTC().Format(time.RFC3339)
	item := input
	item.ID = strings.TrimSpace(item.ID)
	item.WorkspaceID = strings.TrimSpace(item.WorkspaceID)
	item.Name = strings.TrimSpace(item.Name)
	item.RepoPath = strings.TrimSpace(item.RepoPath)
	item.DefaultModelConfigID = strings.TrimSpace(item.DefaultModelConfigID)
	if item.Name == "" {
		item.Name = "Project"
	}
	if item.RepoPath == "" {
		item.RepoPath = "."
	}
	if item.DefaultMode == "" {
		item.DefaultMode = ConversationModeAgent
	}
	if item.CurrentRevision < 0 {
		item.CurrentRevision = 0
	}
	if strings.TrimSpace(item.CreatedAt) == "" {
		item.CreatedAt = now
	}
	if strings.TrimSpace(item.UpdatedAt) == "" {
		item.UpdatedAt = now
	}
	return item
}

func nullWhenEmpty(input string) any {
	value := strings.TrimSpace(input)
	if value == "" {
		return nil
	}
	return value
}
