-- +goose Up
-- Host-owned effective session-policy selection. The recommended default is
-- core.session.default and is intentionally implicit: an empty singleton table
-- resolves in service code to Core, so this migration inserts no bootstrap row
-- and unused Down remains possible without deleting evidence. A Manifest
-- sessionPolicy is only a candidate; install/enable/upgrade never select it.
CREATE TABLE identity_session_policy_selection (
  singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),

  -- Stable session-policy id. Core is exactly core.session.default.
  policy_id TEXT NOT NULL
    CHECK (policy_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'),

  -- Exact plugin declaration provenance. All NULL for the Core tuple; all
  -- non-NULL for a plugin tuple (owner must not be a core.* extension).
  provider_contract_version TEXT
    CHECK (
      provider_contract_version IS NULL
      OR provider_contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'
    ),
  owner_extension_id TEXT
    CHECK (
      owner_extension_id IS NULL
      OR owner_extension_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'
    ),
  owner_extension_version_id BIGINT
    CHECK (owner_extension_version_id IS NULL OR owner_extension_version_id > 0),
  owner_extension_version TEXT
    CHECK (
      owner_extension_version IS NULL
      OR (
        owner_extension_version = btrim(owner_extension_version)
        AND octet_length(owner_extension_version) BETWEEN 1 AND 100
      )
    ),
  owner_package_digest TEXT
    CHECK (
      owner_package_digest IS NULL
      OR owner_package_digest ~ '^[0-9a-f]{64}$'
    ),
  declaration_revision BIGINT
    CHECK (declaration_revision IS NULL OR declaration_revision > 0),

  -- Core: exact id + every plugin-tuple column NULL.
  -- Plugin: non-Core id + full exact owner/provider tuple, owner not core.
  CHECK (
    (
      policy_id = 'core.session.default'
      AND provider_contract_version IS NULL
      AND owner_extension_id IS NULL
      AND owner_extension_version_id IS NULL
      AND owner_extension_version IS NULL
      AND owner_package_digest IS NULL
      AND declaration_revision IS NULL
    )
    OR
    (
      policy_id <> 'core.session.default'
      AND provider_contract_version IS NOT NULL
      AND owner_extension_id IS NOT NULL
      AND owner_extension_id !~ '^core[.]'
      AND owner_extension_version_id IS NOT NULL
      AND owner_extension_version IS NOT NULL
      AND owner_package_digest IS NOT NULL
      AND declaration_revision IS NOT NULL
    )
  ),

  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  selected_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  -- Opaque audit correlation only; no FK so audit retention is independent.
  selection_audit_event_id BIGINT NOT NULL CHECK (selection_audit_event_id > 0),
  selected_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  CHECK (updated_at >= selected_at)
);

-- Lifecycle invalidation looks up the live selection by exact plugin owner and
-- policy provider identity when an extension is disabled/upgraded/uninstalled.
CREATE INDEX identity_session_policy_selection_owner_provider_idx
  ON identity_session_policy_selection (
    owner_extension_id,
    owner_extension_version_id,
    owner_package_digest,
    provider_contract_version,
    policy_id
  )
  WHERE owner_extension_id IS NOT NULL;

-- Append-only selection/reset/invalidation evidence. Event JSON is metadata
-- only (policy id, owner/provider tuple, revision); never tokens, cookies,
-- passwords, session ids, or secrets. NULL previous_selection encodes the
-- first select from the implicit Core default; NULL selected_selection on
-- reset/invalidate means the effective policy returns to Core default.
CREATE TABLE identity_session_policy_selection_events (
  id BIGSERIAL PRIMARY KEY,
  action TEXT NOT NULL CHECK (action IN ('select', 'reset', 'invalidate')),
  previous_selection JSONB
    CHECK (previous_selection IS NULL OR jsonb_typeof(previous_selection) = 'object'),
  selected_selection JSONB
    CHECK (selected_selection IS NULL OR jsonb_typeof(selected_selection) = 'object'),
  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  -- Opaque audit correlation only; no FK so audit retention is independent.
  audit_event_id BIGINT NOT NULL CHECK (audit_event_id > 0),
  reason_code TEXT NOT NULL DEFAULT ''
    CHECK (reason_code = '' OR reason_code ~ '^[a-z0-9][a-z0-9._-]{0,127}$'),
  selection_revision BIGINT NOT NULL CHECK (selection_revision > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),

  -- select requires a selected object (Core or plugin). reset/invalidate clear
  -- to Core (selected NULL) and must record the previous durable selection.
  -- previous_selection is nullable only on the first select from implicit Core.
  CHECK (
    (action = 'select' AND selected_selection IS NOT NULL)
    OR (
      action IN ('reset', 'invalidate')
      AND selected_selection IS NULL
      AND previous_selection IS NOT NULL
    )
  ),
  CHECK (
    previous_selection IS NOT NULL
    OR (action = 'select' AND selection_revision = 1)
  ),
  CHECK (action <> 'invalidate' OR reason_code <> '')
);

CREATE INDEX identity_session_policy_selection_events_created_idx
  ON identity_session_policy_selection_events (created_at DESC, id DESC);

CREATE INDEX identity_session_policy_selection_events_owner_idx
  ON identity_session_policy_selection_events (
    (COALESCE(selected_selection, previous_selection) ->> 'ownerExtensionId'),
    created_at DESC,
    id DESC
  );

-- +goose Down
-- Selection rows and events are Host audit evidence. Rollback may drop an
-- unused schema only after ACCESS EXCLUSIVE confirmation that both tables are
-- empty; never erase retained rows. Empty tables are possible because Up does
-- not insert a bootstrap Core row.
-- +goose StatementBegin
DO $$
BEGIN
  LOCK TABLE identity_session_policy_selection, identity_session_policy_selection_events
    IN ACCESS EXCLUSIVE MODE;
  IF EXISTS (SELECT 1 FROM identity_session_policy_selection_events)
    OR EXISTS (SELECT 1 FROM identity_session_policy_selection) THEN
    RAISE EXCEPTION 'cannot remove identity session policy selection evidence';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS identity_session_policy_selection_events_owner_idx;
DROP INDEX IF EXISTS identity_session_policy_selection_events_created_idx;
DROP TABLE IF EXISTS identity_session_policy_selection_events;
DROP INDEX IF EXISTS identity_session_policy_selection_owner_provider_idx;
DROP TABLE IF EXISTS identity_session_policy_selection;
