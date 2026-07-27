-- +goose Up
-- A pre-T8D development build could leave the protected GitHub built-in marked
-- enabled without an active durable Identity Registry root *and* without any
-- surviving lifecycle/audit evidence of an operator-authorized activation.
-- This is deliberately conservative: a damaged/partial publication alone is
-- never proof that a legitimate enabled operator state is stale. Keep the
-- immutable package staged and require a normal admin enable only when every
-- authoritative evidence source is absent.
UPDATE extensions AS extension
SET status = 'installed',
    active_version_id = NULL,
    staged_version_id = COALESCE(extension.staged_version_id, extension.active_version_id),
    updated_at = now()
WHERE extension.id = 'sforum.auth-github'
  AND extension.type = 'plugin'
  AND extension.source = 'builtin'
  AND extension.status = 'enabled'
  AND extension.active_version_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM extension_identity_registry_publications AS publication
    WHERE publication.owner_extension_id = extension.id
      AND publication.registry_state = 'active'
      AND publication.revision = (
        SELECT MAX(current_publication.revision)
        FROM extension_identity_registry_publications AS current_publication
        WHERE current_publication.owner_extension_id = extension.id
      )
  )
  AND NOT EXISTS (
    SELECT 1
    FROM extension_lifecycle_operations AS operation
	JOIN extension_versions AS version
	  ON version.id = extension.active_version_id
    WHERE operation.extension_id = extension.id
      AND operation.extension_version = version.version
      AND operation.package_digest = version.package_digest
      AND operation.operation IN ('install', 'enable', 'upgrade', 'rollback')
      AND operation.state = 'enabled'
      AND operation.terminal_result = 'succeeded'
  )
  AND NOT EXISTS (
    SELECT 1
    FROM audit_events AS audit
    WHERE audit.action = 'extension.enable'
      AND audit.metadata ->> 'extensionId' = extension.id
  );

-- +goose Down
-- Deliberately no-op. Re-enabling an executable plugin must use the normal,
-- actor-bound lifecycle instead of restoring an unaudited enabled state.
