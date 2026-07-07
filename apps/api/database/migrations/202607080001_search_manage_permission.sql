-- +goose Up
INSERT INTO permissions (key, module, description)
VALUES ('search.manage', 'search', 'Rebuild and manage the search index.')
ON CONFLICT (key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, 'search.manage'
FROM roles
WHERE roles.key = 'super_admin'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions
WHERE permission_key = 'search.manage';

DELETE FROM permissions
WHERE key = 'search.manage';
