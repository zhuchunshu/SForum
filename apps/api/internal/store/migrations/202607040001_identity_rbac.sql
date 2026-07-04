-- +goose Up
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  username TEXT NOT NULL,
  username_lower TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL,
  email_lower TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  locale TEXT NOT NULL DEFAULT 'zh-CN',
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'banned')),
  is_initial_super_admin BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_one_initial_super_admin
  ON users (is_initial_super_admin)
  WHERE is_initial_super_admin;

CREATE TABLE user_credentials (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  password_hash TEXT NOT NULL,
  password_changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE roles (
  id BIGSERIAL PRIMARY KEY,
  key TEXT NOT NULL UNIQUE,
  alias TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  is_system BOOLEAN NOT NULL DEFAULT FALSE,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  is_deletable BOOLEAN NOT NULL DEFAULT TRUE,
  is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX roles_one_default_role
  ON roles (is_default)
  WHERE is_default;

CREATE TABLE permissions (
  key TEXT PRIMARY KEY,
  module TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE role_permissions (
  role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  permission_key TEXT NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (role_id, permission_key)
);

CREATE TABLE user_roles (
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, role_id)
);

CREATE TABLE audit_events (
  id BIGSERIAL PRIMARY KEY,
  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  target_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  action TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO roles (key, alias, description, is_system, is_default, is_deletable)
VALUES
  ('super_admin', '超级管理员', '拥有所有权限并用于系统恢复。', TRUE, FALSE, FALSE),
  ('member', '普通会员', '开放注册用户的默认用户组。', TRUE, TRUE, FALSE);

INSERT INTO permissions (key, module, description)
VALUES
  ('admin.access', 'admin', 'Access the admin area.'),
  ('role.manage', 'identity', 'Create and update roles and role permissions.'),
  ('user.manage', 'identity', 'Manage user accounts and assignments.'),
  ('user.ban', 'identity', 'Ban users from participating.'),
  ('category.manage', 'forum', 'Create and update categories.'),
  ('topic.create', 'forum', 'Create topics.'),
  ('topic.edit_any', 'forum', 'Edit any topic.'),
  ('topic.delete_any', 'forum', 'Delete any topic.'),
  ('topic.lock', 'forum', 'Lock or unlock topics.'),
  ('topic.pin', 'forum', 'Pin or unpin topics.'),
  ('post.create', 'forum', 'Create posts.'),
  ('post.edit_own', 'forum', 'Edit own posts.'),
  ('post.edit_any', 'forum', 'Edit any post.'),
  ('post.delete_own', 'forum', 'Delete own posts.'),
  ('post.delete_any', 'forum', 'Delete any post.'),
  ('moderation.report_review', 'moderation', 'Review moderation reports.'),
  ('settings.manage', 'admin', 'Manage system settings.');

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, permissions.key
FROM roles
CROSS JOIN permissions
WHERE roles.key = 'super_admin';

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, permissions.key
FROM roles
JOIN permissions ON permissions.key IN ('topic.create', 'post.create', 'post.edit_own', 'post.delete_own')
WHERE roles.key = 'member';

-- +goose Down
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP INDEX IF EXISTS roles_one_default_role;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS user_credentials;
DROP INDEX IF EXISTS users_one_initial_super_admin;
DROP TABLE IF EXISTS users;
