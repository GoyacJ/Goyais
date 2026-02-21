PRAGMA foreign_keys = ON;

DELETE FROM projects
WHERE project_id LIKE 'project-sessions%'
   OR project_id LIKE 'project-rename%'
   OR project_id LIKE 'diag-project-diag-execution-%';
