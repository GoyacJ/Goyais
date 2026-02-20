ALTER TABLE runs ADD COLUMN created_by TEXT;

UPDATE runs
SET created_by = COALESCE(created_by, 'user');

CREATE INDEX IF NOT EXISTS idx_runs_created_by ON runs(created_by);

ALTER TABLE audit_logs ADD COLUMN user_id TEXT;

UPDATE audit_logs
SET user_id = COALESCE(user_id, 'user');

CREATE INDEX IF NOT EXISTS idx_audit_user_id ON audit_logs(user_id);
