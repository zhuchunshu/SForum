-- +goose Up
-- Host-owned external identity links. Current rows hold the live link state;
-- append-only events retain redacted transition evidence after privacy erase or
-- user deletion. Core stores only a keyed subject digest, never vendor tokens.
CREATE TABLE identity_external_links (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  -- Stable lowercase provider id (not a display name).
  provider_id TEXT NOT NULL
    CHECK (provider_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'),
  provider_contract_version TEXT NOT NULL
    CHECK (provider_contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),

  -- Exact declaration provenance. owner_extension_version_id is nullable for
  -- sealed Core providers that have no extension_versions row; when set it
  -- must be a positive id.
  owner_extension_id TEXT NOT NULL
    CHECK (owner_extension_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'),
  owner_extension_version_id BIGINT
    CHECK (owner_extension_version_id IS NULL OR owner_extension_version_id > 0),
  owner_extension_version TEXT NOT NULL
    CHECK (
      owner_extension_version = btrim(owner_extension_version)
      AND octet_length(owner_extension_version) BETWEEN 1 AND 100
    ),
  owner_package_digest TEXT NOT NULL
    CHECK (owner_package_digest ~ '^[0-9a-f]{64}$'),
  declaration_revision BIGINT NOT NULL CHECK (declaration_revision > 0),

  CHECK (
    (owner_extension_id ~ '^core[.]' AND owner_extension_version_id IS NULL)
    OR
    (owner_extension_id !~ '^core[.]' AND owner_extension_version_id IS NOT NULL)
  ),

  -- Keyed digest of the external subject. Required while active/unlinked so
  -- uniqueness and re-link checks remain possible; cleared on privacy erase.
  provider_subject_digest TEXT
    CHECK (
      provider_subject_digest IS NULL
      OR provider_subject_digest ~ '^[0-9a-f]{64}$'
    ),

  status TEXT NOT NULL CHECK (status IN ('active', 'unlinked', 'erased')),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),

  linked_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  unlinked_at TIMESTAMPTZ,
  erased_at TIMESTAMPTZ,

  -- Actor/audit provenance without an FK to audit_events so audit retention
  -- cleanup cannot block or erase link rows.
  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  audit_event_id BIGINT NOT NULL CHECK (audit_event_id > 0),

  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),

  CHECK (updated_at >= created_at),
  CHECK (updated_at >= linked_at),
  CHECK (
    (status = 'active'
      AND provider_subject_digest IS NOT NULL
      AND unlinked_at IS NULL
      AND erased_at IS NULL)
    OR
    (status = 'unlinked'
      AND provider_subject_digest IS NOT NULL
      AND unlinked_at IS NOT NULL
      AND erased_at IS NULL
      AND unlinked_at >= linked_at)
    OR
    (status = 'erased'
      AND provider_subject_digest IS NULL
      AND erased_at IS NOT NULL
      AND erased_at >= linked_at
      AND (unlinked_at IS NULL OR (
        unlinked_at >= linked_at AND erased_at >= unlinked_at
      )))
  )
);

-- One active link per provider subject. No (user_id, provider_id) uniqueness:
-- multi-account linking policy is a Host service concern, not a schema rule.
CREATE UNIQUE INDEX identity_external_links_active_provider_digest_uidx
  ON identity_external_links (provider_id, provider_subject_digest)
  WHERE status = 'active';

CREATE INDEX identity_external_links_user_status_provider_idx
  ON identity_external_links (user_id, status, provider_id);

CREATE INDEX identity_external_links_owner_status_provider_idx
  ON identity_external_links (owner_extension_id, status, provider_id);

-- Append-only transition evidence. link_id has no FK so privacy user deletion
-- can remove current PII while redacted evidence remains.
CREATE TABLE identity_external_link_events (
  id BIGSERIAL PRIMARY KEY,
  link_id BIGINT NOT NULL CHECK (link_id > 0),
  provider_id TEXT NOT NULL
    CHECK (provider_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'),
  provider_contract_version TEXT NOT NULL
    CHECK (provider_contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  owner_extension_id TEXT NOT NULL
    CHECK (owner_extension_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'),
  action TEXT NOT NULL CHECK (action IN ('link', 'unlink', 'erase')),
  idempotency_key TEXT NOT NULL UNIQUE
    CHECK (
      octet_length(idempotency_key) BETWEEN 1 AND 128
      AND idempotency_key !~ '[^!-~]'
    ),
  request_fingerprint TEXT NOT NULL
    CHECK (request_fingerprint ~ '^[0-9a-f]{64}$'),
  previous_revision BIGINT
    CHECK (previous_revision IS NULL OR previous_revision > 0),
  next_revision BIGINT NOT NULL CHECK (next_revision > 0),
  previous_status TEXT
    CHECK (previous_status IS NULL OR previous_status IN ('active', 'unlinked', 'erased')),
  next_status TEXT NOT NULL
    CHECK (next_status IN ('active', 'unlinked', 'erased')),
  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  -- Opaque audit correlation only; no FK so audit retention is independent.
  audit_event_id BIGINT NOT NULL CHECK (audit_event_id > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),

  CHECK (
    (action = 'link'
      AND previous_status IS NULL
      AND next_status = 'active'
      AND previous_revision IS NULL
      AND next_revision = 1)
    OR
    (action = 'unlink'
      AND previous_status = 'active'
      AND next_status = 'unlinked'
      AND previous_revision IS NOT NULL
      AND next_revision = previous_revision + 1)
    OR
    (action = 'erase'
      AND previous_status IN ('active', 'unlinked')
      AND next_status = 'erased'
      AND previous_revision IS NOT NULL
      AND next_revision = previous_revision + 1)
  )
);

CREATE INDEX identity_external_link_events_link_idx
  ON identity_external_link_events (link_id, created_at DESC, id DESC);

CREATE INDEX identity_external_link_events_provider_idx
  ON identity_external_link_events (provider_id, created_at DESC, id DESC);

-- +goose Down
-- Link rows and events are privacy/audit evidence. Rollback may drop an unused
-- schema only after ACCESS EXCLUSIVE confirmation that both tables are empty;
-- never erase retained rows.
-- +goose StatementBegin
DO $$
BEGIN
  LOCK TABLE identity_external_links, identity_external_link_events
    IN ACCESS EXCLUSIVE MODE;
  IF EXISTS (SELECT 1 FROM identity_external_link_events)
    OR EXISTS (SELECT 1 FROM identity_external_links) THEN
    RAISE EXCEPTION 'cannot remove identity external link evidence';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS identity_external_link_events_provider_idx;
DROP INDEX IF EXISTS identity_external_link_events_link_idx;
DROP TABLE IF EXISTS identity_external_link_events;
DROP INDEX IF EXISTS identity_external_links_owner_status_provider_idx;
DROP INDEX IF EXISTS identity_external_links_user_status_provider_idx;
DROP INDEX IF EXISTS identity_external_links_active_provider_digest_uidx;
DROP TABLE IF EXISTS identity_external_links;
