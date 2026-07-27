-- +goose Up
-- External social login host state: credential-less users, provider activation,
-- session-bound recent-auth for sensitive unlink/password setup, and password
-- setup support.
--
-- See plans/2026-07-27-github-social-login-builtin-plugin.md and
-- decisions/2026-07-27-github-social-login-builtin-v1.md. Core owns all of
-- this; the plugin only verifies an external assertion.
--
-- T1D: external-only users are represented by absence of a user_credentials
-- row (password_hash stays NOT NULL on existing rows). Recent-auth is bound to
-- (user_id, session_fingerprint), not user_id alone.

-- 1. Credential method marker (password rows only). password_hash remains
--    NOT NULL; external-only accounts simply have no credential row.
ALTER TABLE user_credentials
  ADD COLUMN IF NOT EXISTS method TEXT NOT NULL DEFAULT 'password'
    CHECK (method IN ('password'));

-- 2. Session-bound recent-auth marker for sensitive account-security
--    operations (unlink external identity, first password setup).
--    Fingerprint is a non-reversible SID digest; cross-session isolation is
--    required so another device of the same user cannot satisfy step-up.
CREATE TABLE IF NOT EXISTS user_recent_auth (
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  session_fingerprint TEXT NOT NULL
    CHECK (session_fingerprint ~ '^[a-f0-9]{64}$'),
  auth_method TEXT NOT NULL CHECK (auth_method IN ('password','external')),
  auth_provider_id TEXT,
  authenticated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, session_fingerprint)
);
CREATE INDEX IF NOT EXISTS user_recent_auth_expires_idx
  ON user_recent_auth (expires_at);

-- 3. External auth provider Host activation catalog.
--    Core owns durable, revisioned, audited activation. Built-in discovery
--    never auto-activates. Each operation (login/registration/link) is off
--    by default and must be explicitly enabled by a super_admin/operator.
CREATE TABLE IF NOT EXISTS identity_provider_activations (
  provider_id TEXT PRIMARY KEY
    CHECK (provider_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'),
  owner_extension_id TEXT NOT NULL,
  owner_extension_version_id BIGINT,
  owner_package_digest TEXT,
  login_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  registration_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  link_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  priority INT NOT NULL DEFAULT 0,
  revision BIGINT NOT NULL DEFAULT 1,
  last_probe_at TIMESTAMPTZ,
  last_probe_ok BOOLEAN,
  last_probe_reason TEXT,
  settings_revision BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS identity_provider_activations_owner_idx
  ON identity_provider_activations (owner_extension_id);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_name = 'identity_provider_activations'
  ) THEN
    DECLARE row_count INT;
    BEGIN
      SELECT COUNT(*) INTO row_count FROM identity_provider_activations;
      IF row_count > 0 THEN
        RAISE EXCEPTION 'Refusing to drop identity_provider_activations with % rows', row_count
          USING HINT = 'External provider activations exist; remove them manually first.';
      END IF;
    END;
  END IF;
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_name = 'user_recent_auth'
  ) THEN
    DECLARE recent_count INT;
    BEGIN
      SELECT COUNT(*) INTO recent_count FROM user_recent_auth;
      IF recent_count > 0 THEN
        RAISE EXCEPTION 'Refusing to drop user_recent_auth with % rows', recent_count
          USING HINT = 'Recent auth markers exist; clear them manually first.';
      END IF;
    END;
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS identity_provider_activations_owner_idx;
DROP TABLE IF EXISTS identity_provider_activations;
DROP INDEX IF EXISTS user_recent_auth_expires_idx;
DROP TABLE IF EXISTS user_recent_auth;

ALTER TABLE user_credentials DROP COLUMN IF EXISTS method;
