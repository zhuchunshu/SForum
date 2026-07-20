-- +goose Up
-- Host-owned one-use evidence for a prior exact session.evaluate step_up
-- disposition. Only the SHA-256 of the opaque token is durable; plaintext is
-- returned once at issue and never logged. Plugins cannot mint or consume this
-- evidence. Completing step_up never re-invokes the provider for that claim.
CREATE TABLE identity_session_policy_step_up_evidence (
  token_hash TEXT PRIMARY KEY
    CHECK (token_hash ~ '^[0-9a-f]{64}$'),

  user_id BIGINT NOT NULL CHECK (user_id > 0),
  token_version BIGINT NOT NULL CHECK (token_version >= 0),
  purpose TEXT NOT NULL CHECK (purpose IN ('issue', 'renew')),

  policy_id TEXT NOT NULL CHECK (policy_id = btrim(policy_id) AND octet_length(policy_id) BETWEEN 1 AND 200),
  selection_revision BIGINT NOT NULL CHECK (selection_revision >= 0),
  registry_revision BIGINT NOT NULL CHECK (registry_revision >= 0),
  registry_digest TEXT NOT NULL
    CHECK (registry_digest = btrim(registry_digest) AND octet_length(registry_digest) BETWEEN 1 AND 128),
  package_digest TEXT NOT NULL DEFAULT ''
    CHECK (package_digest = '' OR package_digest ~ '^[0-9a-f]{64}$'),
  owner_extension_id TEXT NOT NULL DEFAULT ''
    CHECK (owner_extension_id = btrim(owner_extension_id) AND octet_length(owner_extension_id) <= 200),

  correlation_id TEXT NOT NULL DEFAULT ''
    CHECK (octet_length(correlation_id) <= 128),
  device_fingerprint TEXT NOT NULL DEFAULT ''
    CHECK (octet_length(device_fingerprint) <= 256),

  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),

  CHECK (expires_at > created_at),
  CHECK (consumed_at IS NULL OR consumed_at >= created_at)
);

CREATE INDEX identity_session_policy_step_up_live_idx
  ON identity_session_policy_step_up_evidence (user_id, expires_at)
  WHERE consumed_at IS NULL;

-- +goose Down
-- Step-up evidence closes a replay window. Refuse rollback while any row exists.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM identity_session_policy_step_up_evidence) THEN
    RAISE EXCEPTION 'cannot remove identity session policy step-up evidence';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS identity_session_policy_step_up_live_idx;
DROP TABLE IF EXISTS identity_session_policy_step_up_evidence;
