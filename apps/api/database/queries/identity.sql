-- name: AnyUserExists :one
SELECT EXISTS (SELECT 1 FROM users LIMIT 1)::boolean;

-- name: CreateUser :one
INSERT INTO users (username, username_lower, email, email_lower, display_name, locale, is_initial_super_admin)
VALUES ($1, lower($1), $2, lower($2), $3, $4, $5)
RETURNING id, username, display_name, locale, status, is_initial_super_admin;

-- name: CreateUserCredential :exec
INSERT INTO user_credentials (user_id, password_hash)
VALUES ($1, $2);

-- name: GetUserCredentialByLogin :one
SELECT users.id, users.username, users.display_name, users.locale, users.status, users.is_initial_super_admin, user_credentials.password_hash
FROM users
JOIN user_credentials ON user_credentials.user_id = users.id
WHERE users.username_lower = lower($1) OR users.email_lower = lower($1);

-- name: GetDefaultRole :one
SELECT id, key, alias, description, is_system, is_default, is_deletable, is_enabled
FROM roles
WHERE is_default = TRUE AND is_enabled = TRUE;

-- name: GetRoleByKey :one
SELECT id, key, alias, description, is_system, is_default, is_deletable, is_enabled
FROM roles
WHERE key = $1;

-- name: AssignRoleToUser :exec
INSERT INTO user_roles (user_id, role_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ListUserRoleKeys :many
SELECT roles.key
FROM user_roles
JOIN roles ON roles.id = user_roles.role_id
WHERE user_roles.user_id = $1 AND roles.is_enabled = TRUE
ORDER BY roles.key;

-- name: ListUserPermissions :many
SELECT DISTINCT permissions.key
FROM user_roles
JOIN roles ON roles.id = user_roles.role_id
JOIN role_permissions ON role_permissions.role_id = roles.id
JOIN permissions ON permissions.key = role_permissions.permission_key
WHERE user_roles.user_id = $1 AND roles.is_enabled = TRUE
ORDER BY permissions.key;

-- name: GetCurrentUser :one
SELECT id, username, display_name, locale, status, is_initial_super_admin
FROM users
WHERE id = $1;

-- name: ListRoles :many
SELECT id, key, alias, description, is_system, is_default, is_deletable, is_enabled
FROM roles
ORDER BY is_system DESC, key ASC;

-- name: CreateRole :one
INSERT INTO roles (key, alias, description, is_system, is_default, is_deletable, is_enabled)
VALUES ($1, $2, $3, FALSE, FALSE, TRUE, TRUE)
RETURNING id, key, alias, description, is_system, is_default, is_deletable, is_enabled;

-- name: UpdateRoleAlias :one
UPDATE roles
SET alias = $2, description = $3, updated_at = now()
WHERE key = $1
RETURNING id, key, alias, description, is_system, is_default, is_deletable, is_enabled;

-- name: DeleteRoleByKey :exec
DELETE FROM roles
WHERE key = $1 AND is_deletable = TRUE AND is_system = FALSE;

-- name: DeleteRolePermissions :exec
DELETE FROM role_permissions
WHERE role_id = $1;

-- name: AddRolePermission :exec
INSERT INTO role_permissions (role_id, permission_key)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: CreateAuditEvent :exec
INSERT INTO audit_events (actor_user_id, target_user_id, action, metadata)
VALUES ($1, $2, $3, $4);
