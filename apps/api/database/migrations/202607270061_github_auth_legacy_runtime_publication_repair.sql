-- +goose Up
-- Migration 060 can restore an audited legacy active artifact. Its paired
-- runtime publication must be repaired as a new immutable full-set revision:
-- historical rows remain evidence and are never edited. The predicate is
-- intentionally the same conservative GitHub-only pre-lifecycle shape, plus a
-- proven mismatch between the latest desired runtime member and active_version.
-- +goose StatementBegin
DO $$
DECLARE
  repaired_revision BIGINT;
  prior_revision BIGINT;
  repaired_count INTEGER;
  repaired_digest TEXT;
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM extensions AS extension
    JOIN extension_versions AS active_version
      ON active_version.id = extension.active_version_id
    JOIN LATERAL (
      SELECT publication.*
      FROM extension_identity_registry_publications AS publication
      WHERE publication.owner_extension_id = extension.id
      ORDER BY publication.revision DESC
      LIMIT 1
    ) AS root ON root.registry_state = 'active'
      AND root.extension_version_id = active_version.id
      AND root.extension_version = active_version.version
      AND root.package_digest = active_version.package_digest
    JOIN audit_events AS audit
      ON audit.id = root.audit_event_id
      AND audit.action = 'extension.enable'
      AND audit.metadata ->> 'extensionId' = extension.id
    JOIN plugin_runtime_publication_members AS desired
      ON desired.publication_revision = (
        SELECT max(revision) FROM plugin_runtime_publications
      )
      AND desired.extension_id = extension.id
      AND (
        desired.extension_version_id <> active_version.id
       OR desired.extension_version <> active_version.version
       OR desired.package_digest <> active_version.package_digest
      )
    WHERE extension.id = 'sforum.auth-github'
      AND extension.type = 'plugin'
      AND extension.source = 'builtin'
      AND extension.status = 'enabled'
      AND NOT EXISTS (
        SELECT 1
        FROM extension_lifecycle_operations AS operation
        WHERE operation.extension_id = extension.id
          AND operation.operation IN ('install', 'enable', 'upgrade', 'rollback')
          AND operation.state = 'enabled'
          AND operation.terminal_result = 'succeeded'
      )
  ) THEN
    RETURN;
  END IF;

  SELECT max(revision) INTO prior_revision FROM plugin_runtime_publications;

  -- A recovery publication is a full set. Existing members are carried only
  -- from the latest immutable full set and only when their active artifact is
  -- an exact match. An enabled mutable row is not durable execution evidence.
  SELECT count(*)::INTEGER,
         encode(sha256(convert_to(coalesce(string_agg(
           octet_length(member.extension_id)::text || ':' || member.extension_id ||
           octet_length(member.extension_version_id::text)::text || ':' || member.extension_version_id::text ||
           octet_length(member.extension_version)::text || ':' || member.extension_version ||
           octet_length(member.package_digest)::text || ':' || member.package_digest,
           '' ORDER BY member.extension_id COLLATE "C"
         ), ''), 'UTF8')), 'hex')
  INTO repaired_count, repaired_digest
  FROM (
    SELECT member.extension_id, member.extension_version_id,
           member.extension_version, member.package_digest
    FROM plugin_runtime_publication_members AS member
    JOIN extensions AS extension ON extension.id = member.extension_id
    JOIN extension_versions AS version ON version.id = extension.active_version_id
    WHERE member.publication_revision = prior_revision
      AND extension.type = 'plugin'
      AND extension.status = 'enabled'
      AND member.extension_version_id = version.id
      AND member.extension_version = version.version
      AND member.package_digest = version.package_digest
    UNION ALL
    SELECT extension.id, active_version.id, active_version.version, active_version.package_digest
    FROM extensions AS extension
    JOIN extension_versions AS active_version ON active_version.id = extension.active_version_id
    WHERE extension.id = 'sforum.auth-github'
      AND extension.type = 'plugin'
      AND extension.source = 'builtin'
      AND extension.status = 'enabled'
      AND EXISTS (
        SELECT 1
        FROM extension_identity_registry_publications AS root
        JOIN audit_events AS audit
          ON audit.id = root.audit_event_id
          AND audit.action = 'extension.enable'
          AND audit.metadata ->> 'extensionId' = extension.id
        WHERE root.owner_extension_id = extension.id
          AND root.registry_state = 'active'
          AND root.extension_version_id = active_version.id
          AND root.extension_version = active_version.version
          AND root.package_digest = active_version.package_digest
      )
  ) AS member;

  INSERT INTO plugin_runtime_publications (member_count, members_digest, reason, actor_user_id)
  VALUES (repaired_count, repaired_digest, 'recovery', NULL)
  RETURNING revision INTO repaired_revision;

  INSERT INTO plugin_runtime_publication_members (
    publication_revision, extension_id, extension_version_id, extension_version, package_digest
  )
  SELECT repaired_revision, member.extension_id, member.extension_version_id,
         member.extension_version, member.package_digest
  FROM (
    SELECT runtime_member.extension_id, runtime_member.extension_version_id,
           runtime_member.extension_version, runtime_member.package_digest
    FROM plugin_runtime_publication_members AS runtime_member
    JOIN extensions AS extension ON extension.id = runtime_member.extension_id
    JOIN extension_versions AS version ON version.id = extension.active_version_id
    WHERE runtime_member.publication_revision = prior_revision
      AND extension.type = 'plugin'
      AND extension.status = 'enabled'
      AND runtime_member.extension_version_id = version.id
      AND runtime_member.extension_version = version.version
      AND runtime_member.package_digest = version.package_digest
    UNION ALL
    SELECT extension.id, active_version.id, active_version.version, active_version.package_digest
    FROM extensions AS extension
    JOIN extension_versions AS active_version ON active_version.id = extension.active_version_id
    WHERE extension.id = 'sforum.auth-github'
      AND extension.type = 'plugin'
      AND extension.source = 'builtin'
      AND extension.status = 'enabled'
      AND EXISTS (
        SELECT 1
        FROM extension_identity_registry_publications AS root
        JOIN audit_events AS audit
          ON audit.id = root.audit_event_id
          AND audit.action = 'extension.enable'
          AND audit.metadata ->> 'extensionId' = extension.id
        WHERE root.owner_extension_id = extension.id
          AND root.registry_state = 'active'
          AND root.extension_version_id = active_version.id
          AND root.extension_version = active_version.version
          AND root.package_digest = active_version.package_digest
      )
  ) AS member;
END $$;
-- +goose StatementEnd

-- +goose Down
-- Deliberately no-op. Runtime publication history is immutable evidence.
