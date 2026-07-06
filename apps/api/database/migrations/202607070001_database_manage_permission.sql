-- +goose Up
INSERT INTO permissions (key, module, description)
VALUES ('database.manage', 'admin', 'Browse database tables and rows.')
ON CONFLICT (key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, 'database.manage'
FROM roles
WHERE roles.key = 'super_admin'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions
WHERE permission_key = 'database.manage';

DELETE FROM permissions
WHERE key = 'database.manage';
