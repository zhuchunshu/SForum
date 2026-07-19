-- +goose Up
-- Identity user-field values are Host-owned PII state. Entity Meta has a
-- different policy and type system, so values live in this dedicated JSONB
-- table with exact Registry declaration provenance and privacy erase state.
CREATE TABLE identity_user_field_values (
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  field_id TEXT NOT NULL
    CHECK (field_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'),
  owner_extension_id TEXT NOT NULL
    CHECK (owner_extension_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'),
  field_contract_version TEXT NOT NULL
    CHECK (field_contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  field_schema_digest TEXT NOT NULL
    CHECK (field_schema_digest ~ '^[0-9a-f]{64}$'),
  declaration_revision BIGINT NOT NULL CHECK (declaration_revision > 0),

  value_json JSONB,
  state TEXT NOT NULL CHECK (state IN ('active', 'erased')),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),

  updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  -- Opaque audit correlation only; no FK so audit retention is independent.
  updated_audit_event_id BIGINT NOT NULL CHECK (updated_audit_event_id > 0),

  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  erased_at TIMESTAMPTZ,
  erased_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  erase_audit_event_id BIGINT CHECK (
    erase_audit_event_id IS NULL OR erase_audit_event_id > 0
  ),

  PRIMARY KEY (user_id, field_id),
  CHECK (updated_at >= created_at),
  CHECK (erased_at IS NULL OR erased_at >= created_at),
  CHECK (
    (state = 'active'
      AND value_json IS NOT NULL
      AND erased_at IS NULL
      AND erased_by_user_id IS NULL
      AND erase_audit_event_id IS NULL)
    OR
    (state = 'erased'
      AND value_json IS NULL
      AND erased_at IS NOT NULL
      AND erase_audit_event_id IS NOT NULL)
  )
);

CREATE INDEX identity_user_field_values_field_user_idx
  ON identity_user_field_values (field_id, user_id);

CREATE INDEX identity_user_field_values_owner_field_user_idx
  ON identity_user_field_values (owner_extension_id, field_id, user_id);

-- Append-only redacted transition evidence. user_id intentionally has no FK:
-- privacy deletion can remove current PII while preserving audit correlation.
-- This table must never store JSONB payloads or raw field values.
CREATE TABLE identity_user_field_value_events (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL CHECK (user_id > 0),
  field_id TEXT NOT NULL
    CHECK (field_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'),
  owner_extension_id TEXT NOT NULL
    CHECK (owner_extension_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'),
  field_contract_version TEXT NOT NULL
    CHECK (field_contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  field_schema_digest TEXT NOT NULL
    CHECK (field_schema_digest ~ '^[0-9a-f]{64}$'),
  declaration_revision BIGINT NOT NULL CHECK (declaration_revision > 0),

  action TEXT NOT NULL CHECK (action IN ('set', 'erase')),
  previous_revision BIGINT
    CHECK (previous_revision IS NULL OR previous_revision > 0),
  next_revision BIGINT NOT NULL CHECK (next_revision > 0),
  previous_value_digest TEXT
    CHECK (
      previous_value_digest IS NULL
      OR previous_value_digest ~ '^[0-9a-f]{64}$'
    ),
  next_value_digest TEXT
    CHECK (
      next_value_digest IS NULL
      OR next_value_digest ~ '^[0-9a-f]{64}$'
    ),
  idempotency_key TEXT NOT NULL UNIQUE
    CHECK (
      octet_length(idempotency_key) BETWEEN 1 AND 128
      AND idempotency_key !~ '[^!-~]'
    ),
  request_fingerprint TEXT NOT NULL
    CHECK (request_fingerprint ~ '^[0-9a-f]{64}$'),
  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  -- Opaque audit correlation only; no FK so audit retention is independent.
  audit_event_id BIGINT NOT NULL CHECK (audit_event_id > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),

  CHECK (
    (action = 'set'
      AND next_value_digest IS NOT NULL
      AND next_revision = COALESCE(previous_revision + 1, 1)
      AND (previous_revision IS NOT NULL OR previous_value_digest IS NULL))
    OR
    (action = 'erase'
      AND previous_revision IS NOT NULL
      AND next_revision = previous_revision + 1
      AND previous_value_digest IS NOT NULL
      AND next_value_digest IS NULL)
  )
);

CREATE INDEX identity_user_field_value_events_user_field_idx
  ON identity_user_field_value_events (user_id, field_id, created_at DESC, id DESC);

CREATE INDEX identity_user_field_value_events_owner_field_user_idx
  ON identity_user_field_value_events (
    owner_extension_id, field_id, user_id, created_at DESC, id DESC
  );

-- +goose Down
-- User-field rows and events are privacy/audit evidence. Rollback may drop an
-- unused schema only after ACCESS EXCLUSIVE confirmation that both tables are
-- empty; retained rows require an explicit privacy/audit handling plan.
-- +goose StatementBegin
DO $$
BEGIN
  LOCK TABLE identity_user_field_values, identity_user_field_value_events
    IN ACCESS EXCLUSIVE MODE;
  IF EXISTS (SELECT 1 FROM identity_user_field_value_events)
    OR EXISTS (SELECT 1 FROM identity_user_field_values) THEN
    RAISE EXCEPTION 'cannot remove identity user-field value evidence';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS identity_user_field_value_events_owner_field_user_idx;
DROP INDEX IF EXISTS identity_user_field_value_events_user_field_idx;
DROP TABLE IF EXISTS identity_user_field_value_events;
DROP INDEX IF EXISTS identity_user_field_values_owner_field_user_idx;
DROP INDEX IF EXISTS identity_user_field_values_field_user_idx;
DROP TABLE IF EXISTS identity_user_field_values;
