-- +goose Up
CREATE TABLE extensions (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL CHECK (type IN ('plugin', 'theme')),
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'installed' CHECK (status IN ('installed', 'enabled', 'disabled')),
  active_version_id BIGINT,
  installed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE extension_versions (
  id BIGSERIAL PRIMARY KEY,
  extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE CASCADE,
  version TEXT NOT NULL,
  manifest JSONB NOT NULL,
  package_path TEXT NOT NULL,
  installed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (extension_id, version)
);

ALTER TABLE extensions
  ADD CONSTRAINT extensions_active_version_fk
  FOREIGN KEY (active_version_id) REFERENCES extension_versions(id) ON DELETE SET NULL;

CREATE TABLE extension_settings (
  extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  value TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (extension_id, name)
);

CREATE TABLE extension_events (
  id BIGSERIAL PRIMARY KEY,
  extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE CASCADE,
  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  action TEXT NOT NULL,
  message TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX extensions_type_status_idx ON extensions (type, status);
CREATE INDEX extension_events_extension_created_idx ON extension_events (extension_id, created_at DESC, id DESC);

INSERT INTO permissions (key, module, description)
VALUES ('extension.manage', 'extension', 'Install and manage extensions and themes.')
ON CONFLICT (key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, 'extension.manage'
FROM roles
WHERE roles.key = 'super_admin'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions WHERE permission_key = 'extension.manage';
DELETE FROM permissions WHERE key = 'extension.manage';

DROP TABLE IF EXISTS extension_events;
DROP TABLE IF EXISTS extension_settings;
ALTER TABLE extensions DROP CONSTRAINT IF EXISTS extensions_active_version_fk;
DROP TABLE IF EXISTS extension_versions;
DROP INDEX IF EXISTS extensions_type_status_idx;
DROP TABLE IF EXISTS extensions;
