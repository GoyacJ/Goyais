-- name: ListWorkspaces :many
SELECT workspace_id, name, slug, kind, created_at
FROM workspaces
WHERE workspace_id IN (
    SELECT workspace_id FROM workspace_members
    WHERE user_id = ? AND status = 'active'
)
ORDER BY name;

-- name: GetWorkspace :one
SELECT workspace_id, name, slug, kind, created_at
FROM workspaces
WHERE workspace_id = ?;

-- name: ListProjects :many
SELECT project_id, workspace_id, name, root_uri, repo_url, branch,
       sync_status, sync_error, last_synced_at, created_by, created_at, updated_at
FROM projects
WHERE workspace_id = ?
ORDER BY created_at DESC;

-- name: GetProject :one
SELECT project_id, workspace_id, name, root_uri, repo_url, branch,
       sync_status, sync_error, last_synced_at, created_by, created_at, updated_at
FROM projects
WHERE project_id = ? AND workspace_id = ?;

-- name: CreateProject :one
INSERT INTO projects (project_id, workspace_id, name, root_uri, created_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects WHERE project_id = ? AND workspace_id = ?;
