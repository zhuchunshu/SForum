-- +goose Up
-- Actor delegation is a one-use Host capability, not a plugin-authored actor.
-- Store only the SHA-256 digest of its JWT id and immutable execution evidence;
-- the signed bearer token itself must never reach PostgreSQL or audit output.
CREATE TABLE extension_host_command_actor_delegation_consumptions (
  id BIGSERIAL PRIMARY KEY,
  delegation_id_digest TEXT NOT NULL UNIQUE
    CHECK (delegation_id_digest ~ '^[0-9a-f]{64}$'),

  extension_id TEXT NOT NULL CHECK (extension_id <> ''),
  extension_version_id BIGINT NOT NULL CHECK (extension_version_id > 0),
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  runtime_epoch BIGINT NOT NULL CHECK (runtime_epoch > 0),
  runtime_instance_id TEXT NOT NULL
    CHECK (
      runtime_instance_id = btrim(runtime_instance_id)
      AND octet_length(runtime_instance_id) BETWEEN 1 AND 512
    ),

  actor_user_id BIGINT NOT NULL CHECK (actor_user_id > 0),
  command_id TEXT NOT NULL
    CHECK (octet_length(command_id) BETWEEN 1 AND 200 AND command_id = btrim(command_id)),
  command_version TEXT NOT NULL
    CHECK (octet_length(command_version) BETWEEN 1 AND 64 AND command_version = btrim(command_version)),
  audience TEXT NOT NULL
    CHECK (audience = 'sforum.host-command.v2'),
  idempotency_key TEXT NOT NULL
    CHECK (
      octet_length(idempotency_key) BETWEEN 1 AND 128
      AND idempotency_key !~ '[^!-~]'
    ),
  request_fingerprint TEXT NOT NULL
    CHECK (request_fingerprint ~ '^[0-9a-f]{64}$'),

  issued_at TIMESTAMPTZ NOT NULL,
  not_before TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),

  UNIQUE (extension_id, command_id, command_version, idempotency_key),
  CHECK (not_before >= issued_at),
  CHECK (expires_at > not_before),
  CHECK (consumed_at >= issued_at AND consumed_at <= expires_at)
);

CREATE INDEX extension_host_command_actor_delegations_actor_idx
  ON extension_host_command_actor_delegation_consumptions (
    actor_user_id, consumed_at DESC, id DESC
  );
CREATE INDEX extension_host_command_actor_delegations_artifact_idx
  ON extension_host_command_actor_delegation_consumptions (
    extension_id, extension_version_id, package_digest, consumed_at DESC, id DESC
  );

-- +goose Down
-- Delegation consumption closes a replay window and remains useful after an
-- actor or extension is removed. Refuse rollback once evidence exists.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_host_command_actor_delegation_consumptions) THEN
    RAISE EXCEPTION 'cannot remove Host Command actor delegation evidence';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS extension_host_command_actor_delegations_artifact_idx;
DROP INDEX IF EXISTS extension_host_command_actor_delegations_actor_idx;
DROP TABLE IF EXISTS extension_host_command_actor_delegation_consumptions;
