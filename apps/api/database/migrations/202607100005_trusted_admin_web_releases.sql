-- +goose Up
CREATE SEQUENCE web_release_generation_seq
  AS BIGINT
  START WITH 1
  INCREMENT BY 1
  CACHE 1;

CREATE TABLE extension_frontend_trust_grants (
  id BIGSERIAL PRIMARY KEY,
  -- 授权记录是安全审计历史；扩展卸载后仍保留当时的稳定 ID 与摘要。
  extension_id TEXT NOT NULL,
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  api_version INTEGER NOT NULL CHECK (api_version > 0),
  contribution_points JSONB NOT NULL DEFAULT '[]'::jsonb
    CHECK (jsonb_typeof(contribution_points) = 'array'),
  component_ids JSONB NOT NULL DEFAULT '[]'::jsonb
    CHECK (jsonb_typeof(component_ids) = 'array'),
  granted_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  revocation_requested_at TIMESTAMPTZ,
  revocation_requested_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  revoked_at TIMESTAMPTZ,
  revoked_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  CHECK (revoked_at IS NULL OR revocation_requested_at IS NOT NULL)
);

CREATE UNIQUE INDEX extension_frontend_trust_grants_live_exact_idx
  ON extension_frontend_trust_grants (extension_id, extension_version, package_digest)
  WHERE revoked_at IS NULL;

CREATE INDEX extension_frontend_trust_grants_extension_created_idx
  ON extension_frontend_trust_grants (extension_id, granted_at DESC, id DESC);

CREATE TABLE web_releases (
  id BIGSERIAL PRIMARY KEY,
  desired_generation BIGINT NOT NULL DEFAULT nextval('web_release_generation_seq') UNIQUE,
  trigger_kind TEXT NOT NULL,
  trigger_extension_id TEXT NOT NULL DEFAULT '',
  composition_hash TEXT NOT NULL CHECK (composition_hash ~ '^[0-9a-f]{64}$'),
  composition_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(composition_snapshot) = 'object'),
  active_theme_id TEXT NOT NULL,
  theme_version TEXT NOT NULL,
  theme_layer_path TEXT NOT NULL,
  theme_package_digest TEXT NOT NULL DEFAULT ''
    CHECK (theme_package_digest = '' OR theme_package_digest ~ '^[0-9a-f]{64}$'),
  status TEXT NOT NULL CHECK (status IN (
    'queued',
    'resolving',
    'installing',
    'building',
    'verifying',
    'ready',
    'activating',
    'active',
    'inactive',
    'failed',
    'superseded',
    'rolled_back'
  )),
  activation_checkpoint TEXT NOT NULL DEFAULT 'pending',
  reload_mode TEXT NOT NULL DEFAULT 'prompt' CHECK (reload_mode IN ('prompt', 'force')),
  artifact_path TEXT NOT NULL DEFAULT '',
  artifact_digest TEXT NOT NULL DEFAULT ''
    CHECK (artifact_digest = '' OR artifact_digest ~ '^[0-9a-f]{64}$'),
  server_entry TEXT NOT NULL DEFAULT '',
  build_log TEXT NOT NULL DEFAULT '',
  public_reason TEXT NOT NULL DEFAULT '',
  public_message TEXT NOT NULL DEFAULT '',
  previous_release_id BIGINT REFERENCES web_releases(id) ON DELETE SET NULL,
  requested_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  activated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ready_at TIMESTAMPTZ,
  activation_started_at TIMESTAMPTZ,
  activated_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ
);

ALTER SEQUENCE web_release_generation_seq
  OWNED BY web_releases.desired_generation;

CREATE UNIQUE INDEX web_releases_single_active_idx
  ON web_releases ((status))
  WHERE status = 'active';

CREATE INDEX web_releases_generation_idx
  ON web_releases (desired_generation DESC, id DESC);

CREATE INDEX web_releases_status_generation_idx
  ON web_releases (status, desired_generation DESC, id DESC);

CREATE INDEX web_releases_live_composition_idx
  ON web_releases (composition_hash, desired_generation DESC, id DESC)
  WHERE status IN (
    'queued',
    'resolving',
    'installing',
    'building',
    'verifying',
    'ready',
    'activating',
    'active'
  );

CREATE TABLE web_release_extensions (
  web_release_id BIGINT NOT NULL REFERENCES web_releases(id) ON DELETE RESTRICT,
  extension_id TEXT NOT NULL,
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  frontend_root TEXT NOT NULL CHECK (frontend_root <> ''),
  component_map JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(component_map) = 'object'),
  api_version INTEGER NOT NULL CHECK (api_version > 0),
  trusted_components JSONB NOT NULL DEFAULT '[]'::jsonb
    CHECK (jsonb_typeof(trusted_components) = 'array'),
  locale_map JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(locale_map) = 'object'),
  locale_map_digest TEXT NOT NULL CHECK (locale_map_digest ~ '^[0-9a-f]{64}$'),
  lockfile_digest TEXT NOT NULL CHECK (lockfile_digest ~ '^[0-9a-f]{64}$'),
  resolved_dependencies JSONB NOT NULL DEFAULT '[]'::jsonb
    CHECK (jsonb_typeof(resolved_dependencies) = 'array'),
  resolved_dependency_snapshot_digest TEXT NOT NULL DEFAULT ''
    CHECK (
      resolved_dependency_snapshot_digest = ''
      OR resolved_dependency_snapshot_digest ~ '^[0-9a-f]{64}$'
    ),
  sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (web_release_id, extension_id)
);

CREATE INDEX web_release_extensions_order_idx
  ON web_release_extensions (web_release_id, sort_order, extension_id);

CREATE INDEX web_release_extensions_package_idx
  ON web_release_extensions (extension_id, extension_version, package_digest);

CREATE TABLE web_release_extension_effects (
  web_release_id BIGINT NOT NULL REFERENCES web_releases(id) ON DELETE RESTRICT,
  extension_id TEXT NOT NULL,
  previous_status TEXT NOT NULL CHECK (previous_status IN ('installed', 'enabled', 'disabled')),
  target_status TEXT NOT NULL CHECK (target_status IN ('installed', 'enabled', 'disabled')),
  activation_checkpoint TEXT NOT NULL DEFAULT 'pending',
  public_reason TEXT NOT NULL DEFAULT '',
  public_message TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  compensated_at TIMESTAMPTZ,
  PRIMARY KEY (web_release_id, extension_id)
);

CREATE INDEX web_release_extension_effects_checkpoint_idx
  ON web_release_extension_effects (activation_checkpoint, web_release_id, extension_id);

CREATE TABLE web_release_events (
  id BIGSERIAL PRIMARY KEY,
  web_release_id BIGINT NOT NULL REFERENCES web_releases(id) ON DELETE RESTRICT,
  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  previous_status TEXT CHECK (previous_status IS NULL OR previous_status IN (
    'queued',
    'resolving',
    'installing',
    'building',
    'verifying',
    'ready',
    'activating',
    'active',
    'inactive',
    'failed',
    'superseded',
    'rolled_back'
  )),
  next_status TEXT NOT NULL CHECK (next_status IN (
    'queued',
    'resolving',
    'installing',
    'building',
    'verifying',
    'ready',
    'activating',
    'active',
    'inactive',
    'failed',
    'superseded',
    'rolled_back'
  )),
  reason TEXT NOT NULL CHECK (reason <> ''),
  message TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX web_release_events_release_timeline_idx
  ON web_release_events (web_release_id, created_at, id);

ALTER TABLE extension_theme_releases
  ADD COLUMN web_release_id BIGINT REFERENCES web_releases(id) ON DELETE SET NULL;

CREATE INDEX extension_theme_releases_web_release_idx
  ON extension_theme_releases (web_release_id);

-- +goose Down
DROP INDEX IF EXISTS extension_theme_releases_web_release_idx;

ALTER TABLE extension_theme_releases
  DROP COLUMN IF EXISTS web_release_id;

DROP TABLE IF EXISTS web_release_events;
DROP TABLE IF EXISTS web_release_extension_effects;
DROP TABLE IF EXISTS web_release_extensions;
DROP TABLE IF EXISTS web_releases;
DROP TABLE IF EXISTS extension_frontend_trust_grants;
DROP SEQUENCE IF EXISTS web_release_generation_seq;
