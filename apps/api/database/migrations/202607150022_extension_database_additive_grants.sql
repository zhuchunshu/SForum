-- +goose Up
-- V3 database powers are an exact additive set. The legacy authority column is
-- retained as compatibility evidence; new additive grants use a fail-closed
-- sentinel and must be resolved through extension_database_grant_powers.
ALTER TABLE extension_database_grants
  DROP CONSTRAINT extension_database_grants_authority_check,
  ADD CONSTRAINT extension_database_grants_authority_check
    CHECK (authority IN (
      'own_schema', 'core_views', 'host_commands', 'raw_core', 'kernel',
      'additive'
    )),
  ADD CONSTRAINT extension_database_grants_id_artifact_unique
    UNIQUE (id, extension_id, extension_version_id, extension_version, package_digest);

CREATE TABLE extension_database_grant_powers (
  grant_id BIGINT NOT NULL
    REFERENCES extension_database_grants(id) ON DELETE RESTRICT,
  power TEXT NOT NULL
    CHECK (power IN ('own_schema', 'core_views', 'host_commands', 'raw_core', 'kernel')),
  source TEXT NOT NULL
    CHECK (source IN ('legacy_authority', 'manifest_grants')),
  ordinal SMALLINT NOT NULL CHECK (ordinal BETWEEN 1 AND 5),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  PRIMARY KEY (grant_id, power),
  UNIQUE (grant_id, ordinal),
  CHECK (
    (power = 'own_schema' AND ordinal = 1)
    OR (power = 'core_views' AND ordinal = 2)
    OR (power = 'host_commands' AND ordinal = 3)
    OR (power = 'raw_core' AND ordinal = 4)
    OR (power = 'kernel' AND ordinal = 5)
  )
);

-- Existing single-choice declarations expand cumulatively in the frozen tier
-- order. This backfill is append-only authority evidence for exact artifacts.
INSERT INTO extension_database_grant_powers (grant_id, power, source, ordinal)
SELECT grants.id, powers.power, 'legacy_authority', powers.ordinal
FROM extension_database_grants AS grants
CROSS JOIN (
  VALUES
    ('own_schema'::TEXT, 1::SMALLINT),
    ('core_views'::TEXT, 2::SMALLINT),
    ('host_commands'::TEXT, 3::SMALLINT),
    ('raw_core'::TEXT, 4::SMALLINT),
    ('kernel'::TEXT, 5::SMALLINT)
) AS powers(power, ordinal)
WHERE powers.ordinal <= CASE grants.authority
  WHEN 'own_schema' THEN 1
  WHEN 'core_views' THEN 2
  WHEN 'host_commands' THEN 3
  WHEN 'raw_core' THEN 4
  WHEN 'kernel' THEN 5
  ELSE 0
END;

-- Source and target exact artifacts may both remain granted while their
-- runtimes overlap. One exact artifact still has at most one active grant.
DROP INDEX extension_database_grants_active_extension_idx;
CREATE UNIQUE INDEX extension_database_grants_active_artifact_idx
  ON extension_database_grants (
    extension_id, extension_version_id, extension_version, package_digest
  ) WHERE status = 'active';

-- Legacy credentials remain available during the compatibility window, but an
-- active credential is now scoped to its exact grant rather than the extension.
DROP INDEX extension_database_credentials_active_extension_idx;
CREATE UNIQUE INDEX extension_database_credentials_active_grant_idx
  ON extension_database_credentials (grant_id) WHERE status = 'active';

CREATE TABLE extension_database_runtime_leases (
  id BIGSERIAL PRIMARY KEY,
  lease_id TEXT NOT NULL UNIQUE CHECK (lease_id ~ '^[0-9a-f]{64}$'),
  grant_id BIGINT NOT NULL,
  extension_id TEXT NOT NULL
    REFERENCES extension_database_resources(extension_id) ON DELETE RESTRICT,
  extension_version_id BIGINT NOT NULL CHECK (extension_version_id > 0),
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  runtime_instance_id TEXT NOT NULL
    CHECK (runtime_instance_id = btrim(runtime_instance_id)
      AND octet_length(runtime_instance_id) BETWEEN 1 AND 512),
  role_name TEXT NOT NULL UNIQUE
    CHECK (role_name ~ '^[a-z_][a-z0-9_]{0,62}$'),
  credential_fingerprint TEXT NOT NULL
    CHECK (credential_fingerprint ~ '^[0-9a-f]{64}$'),
  status TEXT NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'draining', 'revoked', 'failed')),
  issued_by TEXT NOT NULL CHECK (issued_by IN ('actor', 'host')),
  issued_by_user_id BIGINT CHECK (issued_by_user_id IS NULL OR issued_by_user_id > 0),
  issue_audit_event_id BIGINT NOT NULL CHECK (issue_audit_event_id > 0),
  issued_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  last_heartbeat_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  lease_expires_at TIMESTAMPTZ NOT NULL,
  draining_at TIMESTAMPTZ,
  revoked_by TEXT CHECK (revoked_by IS NULL OR revoked_by IN ('actor', 'host')),
  revoked_by_user_id BIGINT CHECK (revoked_by_user_id IS NULL OR revoked_by_user_id > 0),
  revoke_audit_event_id BIGINT CHECK (revoke_audit_event_id IS NULL OR revoke_audit_event_id > 0),
  revoked_at TIMESTAMPTZ,
  failure_code TEXT NOT NULL DEFAULT ''
    CHECK (failure_code = '' OR failure_code ~ '^[a-z0-9][a-z0-9._-]{0,127}$'),
  lease_revision BIGINT NOT NULL DEFAULT 1 CHECK (lease_revision > 0),
  FOREIGN KEY (
    grant_id, extension_id, extension_version_id, extension_version, package_digest
  ) REFERENCES extension_database_grants (
    id, extension_id, extension_version_id, extension_version, package_digest
  ) ON DELETE RESTRICT,
  CHECK (
    (issued_by = 'actor' AND issued_by_user_id IS NOT NULL)
    OR (issued_by = 'host' AND issued_by_user_id IS NULL)
  ),
  CHECK (last_heartbeat_at >= issued_at AND lease_expires_at > issued_at),
  CHECK (draining_at IS NULL OR draining_at >= issued_at),
  CHECK (revoked_at IS NULL OR revoked_at >= issued_at),
  CHECK (
    (status = 'active' AND draining_at IS NULL AND revoked_at IS NULL AND failure_code = '')
    OR (status = 'draining' AND draining_at IS NOT NULL AND revoked_at IS NULL AND failure_code = '')
    OR (status = 'revoked' AND revoked_at IS NOT NULL AND failure_code = '')
    OR (status = 'failed' AND revoked_at IS NOT NULL AND failure_code <> '')
  ),
  CHECK (
    (revoked_at IS NULL AND revoked_by IS NULL
      AND revoked_by_user_id IS NULL AND revoke_audit_event_id IS NULL)
    OR
    (revoked_at IS NOT NULL AND revoked_by IS NOT NULL
      AND revoke_audit_event_id IS NOT NULL
      AND ((revoked_by = 'actor' AND revoked_by_user_id IS NOT NULL)
        OR (revoked_by = 'host' AND revoked_by_user_id IS NULL)))
  )
);

CREATE UNIQUE INDEX extension_database_runtime_leases_live_instance_idx
  ON extension_database_runtime_leases (extension_id, runtime_instance_id)
  WHERE status IN ('active', 'draining');
CREATE INDEX extension_database_runtime_leases_grant_idx
  ON extension_database_runtime_leases (grant_id, status, id DESC);
CREATE INDEX extension_database_runtime_leases_expiry_idx
  ON extension_database_runtime_leases (lease_expires_at, id)
  WHERE status IN ('active', 'draining');

-- +goose Down
-- Power rows and runtime leases are durable authority evidence. Downgrade is
-- allowed only when no V3 additive authority has ever been materialized.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_database_grant_powers)
    OR EXISTS (SELECT 1 FROM extension_database_runtime_leases)
    OR EXISTS (SELECT 1 FROM extension_database_grants WHERE authority = 'additive')
    OR EXISTS (
      SELECT extension_id
      FROM extension_database_grants
      WHERE status = 'active'
      GROUP BY extension_id
      HAVING count(*) > 1
    )
    OR EXISTS (
      SELECT credentials.extension_id
      FROM extension_database_credentials AS credentials
      JOIN extension_database_grants AS grants ON grants.id = credentials.grant_id
      WHERE credentials.status = 'active'
      GROUP BY credentials.extension_id
      HAVING count(*) > 1
    ) THEN
    RAISE EXCEPTION 'cannot remove additive extension database grant or runtime lease evidence';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS extension_database_runtime_leases_expiry_idx;
DROP INDEX IF EXISTS extension_database_runtime_leases_grant_idx;
DROP INDEX IF EXISTS extension_database_runtime_leases_live_instance_idx;
DROP TABLE IF EXISTS extension_database_runtime_leases;

DROP INDEX IF EXISTS extension_database_credentials_active_grant_idx;
CREATE UNIQUE INDEX extension_database_credentials_active_extension_idx
  ON extension_database_credentials (extension_id) WHERE status = 'active';

DROP INDEX IF EXISTS extension_database_grants_active_artifact_idx;
CREATE UNIQUE INDEX extension_database_grants_active_extension_idx
  ON extension_database_grants (extension_id) WHERE status = 'active';

DROP TABLE IF EXISTS extension_database_grant_powers;
ALTER TABLE extension_database_grants
  DROP CONSTRAINT IF EXISTS extension_database_grants_id_artifact_unique,
  DROP CONSTRAINT extension_database_grants_authority_check,
  ADD CONSTRAINT extension_database_grants_authority_check
    CHECK (authority IN ('own_schema', 'core_views', 'host_commands', 'raw_core', 'kernel'));
