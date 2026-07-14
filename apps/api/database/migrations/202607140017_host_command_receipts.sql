-- +goose Up
-- P5 Host Command receipts are durable replay and audit evidence. They intentionally
-- snapshot extension identity instead of referencing extensions/extension_versions,
-- so uninstall cannot erase a committed result or reopen an idempotency key.
CREATE TABLE extension_host_command_receipts (
  id BIGSERIAL PRIMARY KEY,
  extension_id TEXT NOT NULL CHECK (extension_id <> ''),
  extension_version_id BIGINT NOT NULL CHECK (extension_version_id > 0),
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  authority_type TEXT NOT NULL CHECK (authority_type IN ('builtin', 'trust_grant')),
  trust_grant_id BIGINT
    REFERENCES extension_trust_grants(id) ON DELETE RESTRICT,

  command_id TEXT NOT NULL
    CHECK (octet_length(command_id) BETWEEN 1 AND 200 AND command_id = btrim(command_id)),
  command_version TEXT NOT NULL
    CHECK (octet_length(command_version) BETWEEN 1 AND 64 AND command_version = btrim(command_version)),
  idempotency_key TEXT NOT NULL
    CHECK (
      octet_length(idempotency_key) BETWEEN 1 AND 128
      AND idempotency_key !~ '[^!-~]'
    ),
  request_fingerprint TEXT NOT NULL
    CHECK (request_fingerprint ~ '^[0-9a-f]{64}$'),

  result JSONB NOT NULL CHECK (jsonb_typeof(result) = 'object'),
  transaction_id TEXT NOT NULL UNIQUE
    CHECK (octet_length(transaction_id) BETWEEN 1 AND 200),
  -- Audit rows have an independent retention policy. Keep the opaque numeric
  -- reference without an FK, matching the other durable extension ledgers.
  audit_event_id BIGINT NOT NULL UNIQUE CHECK (audit_event_id > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  committed_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),

  UNIQUE (extension_id, command_id, command_version, idempotency_key),
  CHECK (
    (authority_type = 'builtin' AND trust_grant_id IS NULL)
    OR (authority_type = 'trust_grant' AND trust_grant_id IS NOT NULL)
  ),
  CHECK (committed_at >= created_at)
);

CREATE INDEX extension_host_command_receipts_artifact_idx
  ON extension_host_command_receipts (
    extension_id, extension_version_id, package_digest, committed_at DESC, id DESC
  );
CREATE INDEX extension_host_command_receipts_command_idx
  ON extension_host_command_receipts (
    command_id, command_version, committed_at DESC, id DESC
  );

-- +goose Down
-- A receipt is the only authoritative replay result after an extension artifact
-- is removed. Refuse rollback instead of silently reopening committed keys.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_host_command_receipts) THEN
    RAISE EXCEPTION 'cannot remove Host Command receipt evidence';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS extension_host_command_receipts_command_idx;
DROP INDEX IF EXISTS extension_host_command_receipts_artifact_idx;
DROP TABLE IF EXISTS extension_host_command_receipts;
