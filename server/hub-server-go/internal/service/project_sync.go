package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/goyais/hub/internal/middleware"
)

// ─────────────────────────────────────────────────────────────────────────────
// Project model
// ─────────────────────────────────────────────────────────────────────────────

type ProjectSummary struct {
	ProjectID      string  `json:"project_id"`
	WorkspaceID    string  `json:"workspace_id"`
	Name           string  `json:"name"`
	RootURI        *string `json:"root_uri,omitempty"`
	RepoURL        *string `json:"repo_url,omitempty"`
	Branch         string  `json:"branch"`
	AuthRef        *string `json:"auth_ref,omitempty"`
	RepoCachePath  *string `json:"repo_cache_path,omitempty"`
	SyncStatus     string  `json:"sync_status"`
	SyncError      *string `json:"sync_error,omitempty"`
	LastSyncedAt   *string `json:"last_synced_at,omitempty"`
	CreatedBy      string  `json:"created_by"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type CreateProjectInput struct {
	Name    string  `json:"name"`
	RootURI *string `json:"root_uri"`
	RepoURL *string `json:"repo_url"`
	Branch  *string `json:"branch"`
	AuthRef *string `json:"auth_ref"`
}

// ─────────────────────────────────────────────────────────────────────────────
// ProjectService — CRUD
// ─────────────────────────────────────────────────────────────────────────────

type ProjectService struct {
	db *sql.DB
}

func NewProjectService(db *sql.DB) *ProjectService {
	return &ProjectService{db: db}
}

func (s *ProjectService) List(ctx context.Context, workspaceID string) ([]ProjectSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT project_id, workspace_id, name, root_uri, repo_url, branch, auth_ref,
		       repo_cache_path, sync_status, sync_error, last_synced_at, created_by, created_at, updated_at
		FROM projects WHERE workspace_id = ?
		ORDER BY created_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProjectSummary
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (s *ProjectService) Get(ctx context.Context, workspaceID, projectID string) (*ProjectSummary, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT project_id, workspace_id, name, root_uri, repo_url, branch, auth_ref,
		       repo_cache_path, sync_status, sync_error, last_synced_at, created_by, created_at, updated_at
		FROM projects WHERE project_id = ? AND workspace_id = ?`, projectID, workspaceID)
	p, err := scanProject(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (s *ProjectService) Create(ctx context.Context, workspaceID string, in CreateProjectInput) (*ProjectSummary, error) {
	user := middleware.UserFromCtx(ctx)
	if user == nil {
		return nil, fmt.Errorf("unauthenticated")
	}
	if in.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	branch := "main"
	if in.Branch != nil && *in.Branch != "" {
		branch = *in.Branch
	}

	// Git-backed projects start as pending; local projects are ready.
	syncStatus := "ready"
	if in.RepoURL != nil && *in.RepoURL != "" {
		syncStatus = "pending"
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO projects
			(project_id, workspace_id, name, root_uri, repo_url, branch, auth_ref,
			 sync_status, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, workspaceID, in.Name, in.RootURI, in.RepoURL, branch, in.AuthRef,
		syncStatus, user.UserID, now, now)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, workspaceID, id)
}

func (s *ProjectService) Delete(ctx context.Context, workspaceID, projectID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM projects WHERE project_id = ? AND workspace_id = ?`,
		projectID, workspaceID)
	return err
}

// setSyncing atomically marks a project as 'syncing'. Returns repo_url, branch, auth_ref.
func (s *ProjectService) setSyncing(ctx context.Context, projectID string) (repoURL, branch string, authRef *string, err error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(ctx, `
		UPDATE projects SET sync_status = 'syncing', sync_error = NULL, updated_at = ?
		WHERE project_id = ?`, now, projectID)
	if err != nil {
		return "", "", nil, err
	}

	var repoURLVal, branchVal sql.NullString
	var authRefVal sql.NullString
	err = s.db.QueryRowContext(ctx, `
		SELECT repo_url, branch, auth_ref FROM projects WHERE project_id = ?`, projectID).
		Scan(&repoURLVal, &branchVal, &authRefVal)
	if err != nil {
		return "", "", nil, err
	}
	if !repoURLVal.Valid || repoURLVal.String == "" {
		return "", "", nil, fmt.Errorf("project has no repo_url")
	}
	if authRefVal.Valid && authRefVal.String != "" {
		v := authRefVal.String
		authRef = &v
	}
	return repoURLVal.String, branchVal.String, authRef, nil
}

func (s *ProjectService) markReady(projectID, cachePath string) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = s.db.ExecContext(context.Background(), `
		UPDATE projects SET sync_status = 'ready', repo_cache_path = ?,
		       last_synced_at = ?, sync_error = NULL, updated_at = ?
		WHERE project_id = ?`, cachePath, now, now, projectID)
}

func (s *ProjectService) markError(projectID, errMsg string) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = s.db.ExecContext(context.Background(), `
		UPDATE projects SET sync_status = 'error', sync_error = ?, updated_at = ?
		WHERE project_id = ?`, errMsg, now, projectID)
}

// ─────────────────────────────────────────────────────────────────────────────
// ProjectSyncService — async git clone / pull
// ─────────────────────────────────────────────────────────────────────────────

// DefaultRepoCacheBase is where cloned repos live on the server.
// Override via GOYAIS_REPO_CACHE_DIR env var.
const DefaultRepoCacheBase = "/var/goyais/repo-cache"

type ProjectSyncService struct {
	svc          *ProjectService
	repoCacheDir string
}

func NewProjectSyncService(svc *ProjectService) *ProjectSyncService {
	base := os.Getenv("GOYAIS_REPO_CACHE_DIR")
	if base == "" {
		base = DefaultRepoCacheBase
	}
	return &ProjectSyncService{svc: svc, repoCacheDir: base}
}

// TriggerSync starts a background goroutine that clones or updates the repo.
// It returns immediately after marking the project as 'syncing'.
func (s *ProjectSyncService) TriggerSync(ctx context.Context, workspaceID, projectID string) error {
	repoURL, branch, _, err := s.svc.setSyncing(ctx, projectID)
	if err != nil {
		return fmt.Errorf("prepare sync: %w", err)
	}

	go func() {
		if err := s.run(projectID, repoURL, branch); err != nil {
			log.Printf("sync error project=%s: %v", projectID, err)
			s.svc.markError(projectID, err.Error())
		}
	}()
	return nil
}

// run performs the actual git operations. Called from a goroutine.
func (s *ProjectSyncService) run(projectID, repoURL, branch string) error {
	// Sanitize projectID for use as directory name (strip path separators).
	safeID := strings.ReplaceAll(projectID, "/", "-")
	safeID = strings.ReplaceAll(safeID, "..", "")
	cloneDir := filepath.Join(s.repoCacheDir, safeID)

	if err := os.MkdirAll(s.repoCacheDir, 0o755); err != nil {
		return fmt.Errorf("create cache dir %s: %w", s.repoCacheDir, err)
	}

	if _, statErr := os.Stat(filepath.Join(cloneDir, ".git")); os.IsNotExist(statErr) {
		// First time: clone
		if err := s.gitClone(repoURL, branch, cloneDir); err != nil {
			return err
		}
	} else {
		// Subsequent: fetch + checkout + pull
		if err := s.gitPull(cloneDir, branch); err != nil {
			return err
		}
	}

	s.svc.markReady(projectID, cloneDir)
	log.Printf("sync complete project=%s path=%s", projectID, cloneDir)
	return nil
}

func (s *ProjectSyncService) gitClone(repoURL, branch, dest string) error {
	args := []string{"clone", "--depth", "1", "--branch", branch, "--", repoURL, dest}
	return s.runGit("", args...)
}

func (s *ProjectSyncService) gitPull(repoDir, branch string) error {
	if err := s.runGit(repoDir, "fetch", "--depth", "1", "origin", branch); err != nil {
		return err
	}
	return s.runGit(repoDir, "checkout", branch)
}

func (s *ProjectSyncService) runGit(dir string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // args sanitized by caller
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, string(out))
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// scan helper
// ─────────────────────────────────────────────────────────────────────────────

type projectScanner interface {
	Scan(dest ...any) error
}

func scanProject(row projectScanner) (*ProjectSummary, error) {
	var p ProjectSummary
	var rootURI, repoURL, authRef, repoCachePath, syncError, lastSyncedAt sql.NullString
	if err := row.Scan(
		&p.ProjectID, &p.WorkspaceID, &p.Name,
		&rootURI, &repoURL, &p.Branch, &authRef,
		&repoCachePath, &p.SyncStatus, &syncError, &lastSyncedAt,
		&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if rootURI.Valid { p.RootURI = &rootURI.String }
	if repoURL.Valid { p.RepoURL = &repoURL.String }
	if authRef.Valid { p.AuthRef = &authRef.String }
	if repoCachePath.Valid { p.RepoCachePath = &repoCachePath.String }
	if syncError.Valid { p.SyncError = &syncError.String }
	if lastSyncedAt.Valid { p.LastSyncedAt = &lastSyncedAt.String }
	return &p, nil
}
