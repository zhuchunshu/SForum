-- +goose Up
-- Versioned provider slots use an exact desired binding. Runtime discovery may
-- only honor it while both the contract owner and candidate artifact still
-- match; append-only events retain selection/reset/invalidation evidence.
CREATE TABLE extension_provider_slot_selections (
  contract_id TEXT PRIMARY KEY CHECK (contract_id <> ''),
  contract_version TEXT NOT NULL
    CHECK (contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  slot TEXT NOT NULL CHECK (slot ~ '^[a-z][a-z0-9_.-]{0,127}$'),
  contract_extension_id TEXT NOT NULL CHECK (contract_extension_id <> ''),
  contract_extension_version_id BIGINT NOT NULL CHECK (contract_extension_version_id > 0),
  contract_extension_version TEXT NOT NULL CHECK (contract_extension_version <> ''),
  contract_package_digest TEXT NOT NULL CHECK (contract_package_digest ~ '^[0-9a-f]{64}$'),

  candidate_id TEXT NOT NULL CHECK (candidate_id <> ''),
  provider_extension_id TEXT NOT NULL CHECK (provider_extension_id <> ''),
  provider_extension_version_id BIGINT NOT NULL CHECK (provider_extension_version_id > 0),
  provider_extension_version TEXT NOT NULL CHECK (provider_extension_version <> ''),
  provider_package_digest TEXT NOT NULL CHECK (provider_package_digest ~ '^[0-9a-f]{64}$'),

  selected_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  selection_audit_event_id BIGINT NOT NULL CHECK (selection_audit_event_id > 0),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  selected_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  CHECK (updated_at >= selected_at)
);

CREATE UNIQUE INDEX extension_provider_slot_selections_slot_idx
  ON extension_provider_slot_selections (slot);
CREATE INDEX extension_provider_slot_selections_provider_idx
  ON extension_provider_slot_selections (
    provider_extension_id, provider_extension_version_id,
    provider_package_digest, candidate_id
  );
CREATE INDEX extension_provider_slot_selections_contract_owner_idx
  ON extension_provider_slot_selections (
    contract_extension_id, contract_extension_version_id, contract_package_digest
  );

CREATE TABLE extension_provider_slot_selection_events (
  id BIGSERIAL PRIMARY KEY,
  contract_id TEXT NOT NULL CHECK (contract_id <> ''),
  contract_version TEXT NOT NULL
    CHECK (contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  slot TEXT NOT NULL CHECK (slot ~ '^[a-z][a-z0-9_.-]{0,127}$'),
  action TEXT NOT NULL CHECK (action IN ('select', 'reset', 'invalidate')),
  previous_selection JSONB
    CHECK (previous_selection IS NULL OR jsonb_typeof(previous_selection) = 'object'),
  selected_selection JSONB
    CHECK (selected_selection IS NULL OR jsonb_typeof(selected_selection) = 'object'),
  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  audit_event_id BIGINT NOT NULL CHECK (audit_event_id > 0),
  reason_code TEXT NOT NULL DEFAULT ''
    CHECK (reason_code = '' OR reason_code ~ '^[a-z0-9][a-z0-9._-]{0,127}$'),
  selection_revision BIGINT NOT NULL CHECK (selection_revision > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  CHECK ((action = 'select' AND selected_selection IS NOT NULL)
      OR (action IN ('reset', 'invalidate') AND selected_selection IS NULL
          AND previous_selection IS NOT NULL)),
  CHECK (action <> 'invalidate' OR reason_code <> '')
);

CREATE INDEX extension_provider_slot_selection_events_contract_idx
  ON extension_provider_slot_selection_events (
    contract_id, created_at DESC, id DESC
  );
CREATE INDEX extension_provider_slot_selection_events_provider_idx
  ON extension_provider_slot_selection_events (
    (COALESCE(selected_selection, previous_selection) ->> 'providerExtensionId'),
    created_at DESC,
    id DESC
  );

-- +goose Down
-- Provider-selection history is audit evidence. Never erase it implicitly.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_provider_slot_selections)
    OR EXISTS (SELECT 1 FROM extension_provider_slot_selection_events) THEN
    RAISE EXCEPTION 'cannot remove provider slot selection evidence';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS extension_provider_slot_selection_events_provider_idx;
DROP INDEX IF EXISTS extension_provider_slot_selection_events_contract_idx;
DROP TABLE IF EXISTS extension_provider_slot_selection_events;
DROP INDEX IF EXISTS extension_provider_slot_selections_contract_owner_idx;
DROP INDEX IF EXISTS extension_provider_slot_selections_provider_idx;
DROP INDEX IF EXISTS extension_provider_slot_selections_slot_idx;
DROP TABLE IF EXISTS extension_provider_slot_selections;
