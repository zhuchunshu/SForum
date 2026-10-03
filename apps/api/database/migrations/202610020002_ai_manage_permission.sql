-- +goose Up
INSERT INTO permissions (key, module, description)
VALUES ('ai.manage', 'ai', 'Manage the AI gateway: provider profiles, base URLs, credential references, quotas, budgets, failure posture, and usage audit.')
ON CONFLICT (key) DO UPDATE
SET module = EXCLUDED.module, description = EXCLUDED.description;

INSERT INTO role_permissions (role_id, permission_key)
SELECT id, 'ai.manage' FROM roles WHERE roles.key = 'super_admin'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT id, 'ai.manage' FROM roles WHERE roles.key = 'tech_admin'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions WHERE permission_key = 'ai.manage';
DELETE FROM permissions WHERE key = 'ai.manage';
