-- +goose Up
-- Route runtime incidents are durable, payload-free evidence for failures that
-- close one exact executable artifact. This ledger records the local quarantine
-- result; it does not claim cross-node or restart-persistent admission authority.
CREATE TABLE extension_route_runtime_incidents (
  id BIGSERIAL PRIMARY KEY,
  incident_key TEXT NOT NULL UNIQUE CHECK (incident_key ~ '^[0-9a-f]{64}$'),
  route_revision BIGINT NOT NULL CHECK (route_revision > 0),
  step_index INTEGER NOT NULL CHECK (step_index >= 0),
  phase TEXT NOT NULL CHECK (phase IN ('filter', 'wrap', 'handler', 'after')),
  invocation_stage TEXT NOT NULL CHECK (invocation_stage IN ('handler', 'response')),
  action TEXT NOT NULL CHECK (action IN ('add', 'after', 'filter', 'wrap', 'replace')),
  mode TEXT NOT NULL CHECK (mode IN ('http', 'multipart', 'stream', 'sse', 'websocket')),
  route_id TEXT NOT NULL
    CHECK (route_id = btrim(route_id) AND octet_length(route_id) BETWEEN 1 AND 200),
  contract_version TEXT NOT NULL
    CHECK (contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  method TEXT NOT NULL
    CHECK (method ~ '^[A-Z][A-Z0-9!#$%&''*+.^_`|~-]{0,31}$'),
  path_signature TEXT NOT NULL
    CHECK (octet_length(path_signature) BETWEEN 1 AND 1024),
  failure_code TEXT NOT NULL
    CHECK (failure_code IN ('transport_failed', 'response_schema_rejected')),
  cause_class TEXT NOT NULL CHECK (cause_class IN (
    'runtime_transport', 'host_budget', 'invalid_preflight',
    'missing_terminal', 'response_schema'
  )),
  runtime_execution_observed BOOLEAN NOT NULL
    CHECK (runtime_execution_observed),
  actor_user_id BIGINT CHECK (actor_user_id IS NULL OR actor_user_id > 0),
  response_status SMALLINT
    CHECK (response_status IS NULL OR response_status BETWEEN 100 AND 599),
  commit_state TEXT NOT NULL
    CHECK (commit_state IN ('response_started', 'side_effect_started', 'committed')),
  extension_id TEXT NOT NULL
    CHECK (extension_id = btrim(extension_id) AND octet_length(extension_id) BETWEEN 1 AND 200),
  -- Snapshot the immutable numeric version id without an FK so uninstall cannot
  -- erase incident evidence or become blocked by the operational ledger.
  extension_version_id BIGINT NOT NULL CHECK (extension_version_id > 0),
  extension_version TEXT NOT NULL
    CHECK (extension_version = btrim(extension_version) AND octet_length(extension_version) BETWEEN 1 AND 100),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  runtime_instance_id TEXT NOT NULL
    CHECK (runtime_instance_id = btrim(runtime_instance_id) AND octet_length(runtime_instance_id) BETWEEN 1 AND 200),
  audit_event_id BIGINT NOT NULL UNIQUE CHECK (audit_event_id > 0),
  local_quarantine_result TEXT NOT NULL DEFAULT 'pending' CHECK (local_quarantine_result IN (
    'pending', 'quarantined', 'stale_missing', 'stale_artifact', 'failed'
  )),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  resolved_at TIMESTAMPTZ,

  CHECK (
    (local_quarantine_result = 'pending' AND resolved_at IS NULL)
    OR (local_quarantine_result <> 'pending' AND resolved_at IS NOT NULL)
  ),

  CHECK (
    (
      invocation_stage = 'handler'
      AND phase = 'handler'
      AND action IN ('add', 'replace')
      AND mode IN ('multipart', 'stream', 'sse', 'websocket')
      AND failure_code = 'transport_failed'
      AND cause_class <> 'response_schema'
      AND commit_state IN ('response_started', 'side_effect_started')
    )
    OR
    (
      invocation_stage = 'response'
      AND phase IN ('filter', 'wrap', 'after')
      AND action IN ('filter', 'wrap', 'after')
      AND mode = 'http'
      AND (
        (failure_code = 'transport_failed' AND cause_class = 'runtime_transport')
        OR (failure_code = 'response_schema_rejected' AND cause_class = 'response_schema')
      )
      AND commit_state = 'committed'
    )
  )
);

CREATE INDEX extension_route_runtime_incidents_artifact_idx
  ON extension_route_runtime_incidents (
    extension_id, extension_version_id, package_digest, runtime_instance_id,
    created_at DESC, id DESC
  );

CREATE INDEX extension_route_runtime_incidents_route_idx
  ON extension_route_runtime_incidents (
    route_id, method, path_signature, created_at DESC, id DESC
  );

CREATE INDEX extension_route_runtime_incidents_pending_idx
  ON extension_route_runtime_incidents (created_at, id)
  WHERE local_quarantine_result = 'pending';

-- +goose Down
-- Runtime incident rows are operational safety evidence. A rollback may remove
-- an unused schema, but it must never silently erase existing incidents.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_route_runtime_incidents) THEN
    RAISE EXCEPTION 'cannot remove route runtime incident evidence';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS extension_route_runtime_incidents_route_idx;
DROP INDEX IF EXISTS extension_route_runtime_incidents_artifact_idx;
DROP INDEX IF EXISTS extension_route_runtime_incidents_pending_idx;
DROP TABLE IF EXISTS extension_route_runtime_incidents;
