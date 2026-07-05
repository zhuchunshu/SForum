-- +goose Up
INSERT INTO permissions (key, module, description)
VALUES ('seo.manage', 'admin', 'Manage search engine optimization settings.')
ON CONFLICT (key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, 'seo.manage'
FROM roles
WHERE roles.key = 'super_admin'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions
WHERE permission_key = 'seo.manage';

DELETE FROM permissions
WHERE key = 'seo.manage';
