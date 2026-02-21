-- name: ListSessions :many
SELECT session_id, workspace_id, project_id, title, mode, model_config_id,
       skill_set_ids, mcp_connector_ids, use_worktree,
       active_execution_id, status, created_by, created_at, updated_at, archived_at
FROM sessions
WHERE project_id = ? AND workspace_id = ? AND archived_at IS NULL
ORDER BY updated_at DESC;

-- name: GetSession :one
SELECT session_id, workspace_id, project_id, title, mode, model_config_id,
       skill_set_ids, mcp_connector_ids, use_worktree,
       active_execution_id, status, created_by, created_at, updated_at, archived_at
FROM sessions
WHERE session_id = ? AND workspace_id = ?;

-- name: CreateSession :one
INSERT INTO sessions (
    session_id, workspace_id, project_id, title, mode,
    model_config_id, skill_set_ids, mcp_connector_ids, use_worktree,
    status, created_by, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?,
    'idle', ?, datetime('now'), datetime('now')
)
RETURNING *;

-- name: UpdateSession :one
UPDATE sessions
SET title = COALESCE(?, title),
    mode = COALESCE(?, mode),
    model_config_id = COALESCE(?, model_config_id),
    skill_set_ids = COALESCE(?, skill_set_ids),
    mcp_connector_ids = COALESCE(?, mcp_connector_ids),
    use_worktree = COALESCE(?, use_worktree),
    updated_at = datetime('now')
WHERE session_id = ? AND workspace_id = ?
RETURNING *;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE session_id = ? AND workspace_id = ?;
