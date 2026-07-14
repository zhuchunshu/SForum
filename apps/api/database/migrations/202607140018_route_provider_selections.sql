-- +goose Up
-- P6 replace providers are never selected by priority. The current row is an
-- exact desired binding; append-only events retain selection/reset evidence.
CREATE TABLE extension_route_provider_selections (
  target_route_id TEXT NOT NULL CHECK (target_route_id <> ''),
  target_contract_version TEXT NOT NULL
    CHECK (target_contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  method TEXT NOT NULL CHECK (method ~ '^[A-Z*][A-Z0-9!#$%&''*+.^_`|~-]{0,31}$'),
  path_signature TEXT NOT NULL CHECK (octet_length(path_signature) BETWEEN 1 AND 1024),

  provider_route_id TEXT NOT NULL CHECK (provider_route_id <> ''),
  provider_contract_version TEXT NOT NULL
    CHECK (provider_contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  provider_extension_id TEXT NOT NULL CHECK (provider_extension_id <> ''),
  provider_extension_version_id BIGINT NOT NULL CHECK (provider_extension_version_id > 0),
  provider_extension_version TEXT NOT NULL CHECK (provider_extension_version <> ''),
  provider_package_digest TEXT NOT NULL CHECK (provider_package_digest ~ '^[0-9a-f]{64}$'),

  selected_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  selection_audit_event_id BIGINT NOT NULL CHECK (selection_audit_event_id > 0),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  selected_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),

  PRIMARY KEY (target_route_id, method, path_signature),
  CHECK (updated_at >= selected_at)
);

CREATE INDEX extension_route_provider_selections_provider_idx
  ON extension_route_provider_selections (
    provider_extension_id, provider_extension_version_id,
    provider_package_digest, provider_route_id
  );

CREATE TABLE extension_route_provider_selection_events (
  id BIGSERIAL PRIMARY KEY,
  target_route_id TEXT NOT NULL CHECK (target_route_id <> ''),
  target_contract_version TEXT NOT NULL
    CHECK (target_contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  method TEXT NOT NULL CHECK (method ~ '^[A-Z*][A-Z0-9!#$%&''*+.^_`|~-]{0,31}$'),
  path_signature TEXT NOT NULL CHECK (octet_length(path_signature) BETWEEN 1 AND 1024),
  action TEXT NOT NULL CHECK (action IN ('select', 'reset', 'invalidate')),
  previous_provider JSONB
    CHECK (previous_provider IS NULL OR jsonb_typeof(previous_provider) = 'object'),
  selected_provider JSONB
    CHECK (selected_provider IS NULL OR jsonb_typeof(selected_provider) = 'object'),
  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  audit_event_id BIGINT NOT NULL CHECK (audit_event_id > 0),
  reason_code TEXT NOT NULL DEFAULT ''
    CHECK (reason_code = '' OR reason_code ~ '^[a-z0-9][a-z0-9._-]{0,127}$'),
  selection_revision BIGINT NOT NULL CHECK (selection_revision > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  CHECK ((action = 'select' AND selected_provider IS NOT NULL)
      OR (action IN ('reset', 'invalidate') AND selected_provider IS NULL
          AND previous_provider IS NOT NULL)),
  CHECK (action <> 'invalidate' OR reason_code <> '')
);

CREATE INDEX extension_route_provider_selection_events_target_idx
  ON extension_route_provider_selection_events (
    target_route_id, method, path_signature, created_at DESC, id DESC
  );
CREATE INDEX extension_route_provider_selection_events_provider_idx
  ON extension_route_provider_selection_events (
    (COALESCE(selected_provider, previous_provider) ->> 'extensionId'),
    created_at DESC,
    id DESC
  );

-- +goose Down
-- Provider-selection history is audit evidence. Never erase it implicitly.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_route_provider_selections)
    OR EXISTS (SELECT 1 FROM extension_route_provider_selection_events) THEN
    RAISE EXCEPTION 'cannot remove route provider selection evidence';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS extension_route_provider_selection_events_provider_idx;
DROP INDEX IF EXISTS extension_route_provider_selection_events_target_idx;
DROP TABLE IF EXISTS extension_route_provider_selection_events;
DROP INDEX IF EXISTS extension_route_provider_selections_provider_idx;
DROP TABLE IF EXISTS extension_route_provider_selections;
