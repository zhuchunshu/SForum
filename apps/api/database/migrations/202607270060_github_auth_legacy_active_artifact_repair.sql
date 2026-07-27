-- +goose Up
-- Before the lifecycle-owned built-in synchronizer existed, development builds
-- could move an already-enabled GitHub package's active_version_id while its
-- immutable Identity Registry root still named the artifact authorized by the
-- original extension.enable audit event. Preserve that legitimate enable by
-- restoring the active pointer only when the evidence is unambiguous. Any
-- lifecycle evidence makes the row operator-actionable and leaves it closed by
-- the normal exact-artifact startup fence.
WITH current_root AS (
  SELECT DISTINCT ON (publication.owner_extension_id)
    publication.owner_extension_id,
    publication.extension_version_id,
    publication.extension_version,
    publication.package_digest,
    publication.audit_event_id
  FROM extension_identity_registry_publications AS publication
  WHERE publication.owner_extension_id = 'sforum.auth-github'
  ORDER BY publication.owner_extension_id, publication.revision DESC
), authorized_root AS (
  SELECT root.owner_extension_id, root.extension_version_id
  FROM current_root AS root
  JOIN extension_versions AS version
    ON version.id = root.extension_version_id
   AND version.extension_id = root.owner_extension_id
   AND version.version = root.extension_version
   AND version.package_digest = root.package_digest
  JOIN audit_events AS audit
    ON audit.id = root.audit_event_id
   AND audit.action = 'extension.enable'
   AND audit.metadata ->> 'extensionId' = root.owner_extension_id
  WHERE (
    SELECT publication.registry_state
    FROM extension_identity_registry_publications AS publication
    WHERE publication.owner_extension_id = root.owner_extension_id
    ORDER BY publication.revision DESC
    LIMIT 1
  ) = 'active'
)
UPDATE extensions AS extension
SET active_version_id = root.extension_version_id,
    updated_at = now()
FROM authorized_root AS root
WHERE extension.id = root.owner_extension_id
  AND extension.id = 'sforum.auth-github'
  AND extension.type = 'plugin'
  AND extension.source = 'builtin'
  AND extension.status = 'enabled'
  AND extension.active_version_id IS DISTINCT FROM root.extension_version_id
  AND NOT EXISTS (
    SELECT 1
    FROM extension_lifecycle_operations AS operation
    WHERE operation.extension_id = extension.id
      AND operation.operation IN ('install', 'enable', 'upgrade', 'rollback')
      AND operation.state = 'enabled'
      AND operation.terminal_result = 'succeeded'
  );

-- +goose Down
-- Deliberately no-op. Reversing this recovery would reintroduce an artifact
-- mismatch; executable artifact changes must use the normal lifecycle.
