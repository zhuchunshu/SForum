-- +goose Up
INSERT INTO permissions (key, module, description) VALUES
  ('jobs.view', 'jobs', 'View background jobs, queues, failures, and worker activity.'),
  ('jobs.manage', 'jobs', 'Retry, cancel, pause, and resume background job processing.')
ON CONFLICT (key) DO UPDATE SET module = EXCLUDED.module, description = EXCLUDED.description;

INSERT INTO role_permissions (role_id, permission_id)
SELECT roles.id, permissions.id
FROM roles CROSS JOIN permissions
WHERE roles.key = 'super_admin' AND permissions.key IN ('jobs.view', 'jobs.manage')
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE key IN ('jobs.view', 'jobs.manage'));
DELETE FROM permissions WHERE key IN ('jobs.view', 'jobs.manage');
