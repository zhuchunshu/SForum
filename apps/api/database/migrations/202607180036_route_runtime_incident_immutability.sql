-- +goose Up
-- Repair development databases that applied the unreleased 035 draft before
-- its payload-free evidence contract was finalized. Fresh installs already
-- have these columns, so every operation below is additive/idempotent.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'extension_route_runtime_incidents'
      AND column_name = 'quarantine_result'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'extension_route_runtime_incidents'
      AND column_name = 'local_quarantine_result'
  ) THEN
    ALTER TABLE extension_route_runtime_incidents
      RENAME COLUMN quarantine_result TO local_quarantine_result;
  END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE extension_route_runtime_incidents
  ADD COLUMN IF NOT EXISTS incident_key TEXT,
  ADD COLUMN IF NOT EXISTS extension_version_id BIGINT,
  ADD COLUMN IF NOT EXISTS audit_event_id BIGINT,
  ADD COLUMN IF NOT EXISTS local_quarantine_result TEXT DEFAULT 'pending',
  ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ;

ALTER TABLE extension_route_runtime_incidents
  DROP CONSTRAINT IF EXISTS extension_route_runtime_incidents_cause_class_check,
  DROP CONSTRAINT IF EXISTS extension_route_runtime_incidents_check,
  DROP CONSTRAINT IF EXISTS extension_route_runtime_incidents_check1,
  DROP CONSTRAINT IF EXISTS extension_route_runtime_incidents_check2,
  DROP CONSTRAINT IF EXISTS extension_route_runtime_incidents_quarantine_result_check,
  DROP CONSTRAINT IF EXISTS extension_route_runtime_incidents_local_quarantine_result_check,
  DROP CONSTRAINT IF EXISTS extension_route_runtime_incidents_local_resolution_check,
  DROP CONSTRAINT IF EXISTS extension_route_runtime_incidents_state_check,
  DROP CONSTRAINT IF EXISTS extension_route_runtime_incidents_incident_key_check,
  DROP CONSTRAINT IF EXISTS extension_route_runtime_incidents_extension_version_id_check,
  DROP CONSTRAINT IF EXISTS extension_route_runtime_incidents_audit_event_id_check;

-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM extension_route_runtime_incidents
    WHERE cause_class = 'force_cancel'
  ) THEN
    RAISE EXCEPTION 'legacy force-cancel route incidents require manual review';
  END IF;
END $$;
-- +goose StatementEnd

UPDATE extension_route_runtime_incidents
SET incident_key =
  md5('legacy-route-runtime-incident:' || id::text) ||
  md5('legacy-route-runtime-audit:' || id::text)
WHERE incident_key IS NULL;

UPDATE extension_route_runtime_incidents AS incident
SET extension_version_id = version.id
FROM extension_versions AS version
JOIN extensions AS extension
  ON extension.id = version.extension_id AND extension.type = 'plugin'
WHERE incident.extension_version_id IS NULL
  AND version.extension_id = incident.extension_id
  AND version.version = incident.extension_version
  AND version.package_digest = incident.package_digest;

-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM extension_route_runtime_incidents
    WHERE extension_version_id IS NULL
  ) THEN
    RAISE EXCEPTION 'legacy route incident exact extension version is unavailable';
  END IF;
END $$;
-- +goose StatementEnd

WITH missing AS (
  SELECT incident.*, CASE WHEN actor.id IS NULL THEN NULL ELSE incident.actor_user_id END AS stored_actor
  FROM extension_route_runtime_incidents AS incident
  LEFT JOIN users AS actor ON actor.id = incident.actor_user_id
  WHERE incident.audit_event_id IS NULL
), inserted AS (
  INSERT INTO audit_events (actor_user_id, action, metadata)
  SELECT stored_actor, 'routes.runtime_incident', jsonb_build_object(
    'incidentKey', incident_key,
    'revision', route_revision,
    'stepIndex', step_index,
    'phase', phase,
    'invocationStage', invocation_stage,
    'action', action,
    'mode', mode,
    'routeId', route_id,
    'contractVersion', contract_version,
    'method', method,
    'pathSignature', path_signature,
    'failureCode', failure_code,
    'causeClass', cause_class,
    'runtimeExecutionObserved', runtime_execution_observed,
    'responseStatus', response_status,
    'commitState', commit_state,
    'extensionId', extension_id,
    'extensionVersion', extension_version,
    'packageDigest', package_digest,
    'runtimeInstanceId', runtime_instance_id,
    'legacyRepair', true
  )
  FROM missing
  RETURNING id, metadata ->> 'incidentKey' AS incident_key
)
UPDATE extension_route_runtime_incidents AS incident
SET audit_event_id = inserted.id
FROM inserted
WHERE incident.incident_key = inserted.incident_key;

UPDATE extension_route_runtime_incidents
SET resolved_at = created_at
WHERE local_quarantine_result <> 'pending' AND resolved_at IS NULL;

ALTER TABLE extension_route_runtime_incidents
  ALTER COLUMN incident_key SET NOT NULL,
  ALTER COLUMN extension_version_id SET NOT NULL,
  ALTER COLUMN audit_event_id SET NOT NULL,
  ALTER COLUMN local_quarantine_result SET DEFAULT 'pending',
  ALTER COLUMN local_quarantine_result SET NOT NULL;

-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = 'extension_route_runtime_incidents'::regclass
      AND conname = 'extension_route_runtime_incidents_incident_key_key'
  ) THEN
    ALTER TABLE extension_route_runtime_incidents
      ADD CONSTRAINT extension_route_runtime_incidents_incident_key_key UNIQUE (incident_key);
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = 'extension_route_runtime_incidents'::regclass
      AND conname = 'extension_route_runtime_incidents_audit_event_id_key'
  ) THEN
    ALTER TABLE extension_route_runtime_incidents
      ADD CONSTRAINT extension_route_runtime_incidents_audit_event_id_key UNIQUE (audit_event_id);
  END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE extension_route_runtime_incidents
  ADD CONSTRAINT extension_route_runtime_incidents_incident_key_check
    CHECK (incident_key ~ '^[0-9a-f]{64}$'),
  ADD CONSTRAINT extension_route_runtime_incidents_extension_version_id_check
    CHECK (extension_version_id > 0),
  ADD CONSTRAINT extension_route_runtime_incidents_audit_event_id_check
    CHECK (audit_event_id > 0),
  ADD CONSTRAINT extension_route_runtime_incidents_cause_class_check
    CHECK (cause_class IN (
      'runtime_transport', 'host_budget', 'invalid_preflight',
      'missing_terminal', 'response_schema'
    )),
  ADD CONSTRAINT extension_route_runtime_incidents_local_quarantine_result_check
    CHECK (local_quarantine_result IN (
      'pending', 'quarantined', 'stale_missing', 'stale_artifact', 'failed'
    )),
  ADD CONSTRAINT extension_route_runtime_incidents_local_resolution_check CHECK (
    (local_quarantine_result = 'pending' AND resolved_at IS NULL)
    OR (local_quarantine_result <> 'pending' AND resolved_at IS NOT NULL)
  ),
  ADD CONSTRAINT extension_route_runtime_incidents_state_check CHECK (
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
  );

DROP INDEX IF EXISTS extension_route_runtime_incidents_artifact_idx;
CREATE INDEX extension_route_runtime_incidents_artifact_idx
  ON extension_route_runtime_incidents (
    extension_id, extension_version_id, package_digest, runtime_instance_id,
    created_at DESC, id DESC
  );

CREATE INDEX IF NOT EXISTS extension_route_runtime_incidents_pending_idx
  ON extension_route_runtime_incidents (created_at, id)
  WHERE local_quarantine_result = 'pending';

ALTER TABLE extension_route_runtime_incidents
  ADD CONSTRAINT extension_route_runtime_incidents_resolution_time_check
  CHECK (resolved_at IS NULL OR resolved_at >= created_at),
  ADD CONSTRAINT extension_route_runtime_incidents_response_stage_status_check
  CHECK (invocation_stage <> 'response' OR response_status IS NOT NULL);

-- The numeric audit correlation intentionally has no FK because generic audit
-- retention may delete it. At insertion time it must nevertheless identify the
-- matching runtime-incident audit row.
-- +goose StatementBegin
CREATE FUNCTION validate_route_runtime_incident_audit() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  stored_action TEXT;
  stored_incident_key TEXT;
BEGIN
  SELECT action, metadata ->> 'incidentKey'
    INTO stored_action, stored_incident_key
  FROM audit_events
  WHERE id = NEW.audit_event_id
  FOR KEY SHARE;
  IF stored_action IS DISTINCT FROM 'routes.runtime_incident'
     OR stored_incident_key IS DISTINCT FROM NEW.incident_key THEN
    RAISE EXCEPTION 'route runtime incident audit evidence is invalid';
  END IF;
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_route_runtime_incident_audit_valid
BEFORE INSERT ON extension_route_runtime_incidents
FOR EACH ROW EXECUTE FUNCTION validate_route_runtime_incident_audit();

-- Evidence is append-only except for one pending -> local-result transition.
-- resolved_at means that the local quarantine attempt completed; it is not an
-- operator clearance and does not reopen admission.
-- +goose StatementBegin
CREATE FUNCTION enforce_route_runtime_incident_immutability() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF TG_OP IN ('DELETE', 'TRUNCATE') THEN
    RAISE EXCEPTION 'route runtime incident evidence is append-only';
  END IF;
  IF OLD.local_quarantine_result <> 'pending'
     OR NEW.local_quarantine_result = 'pending'
     OR NEW.resolved_at IS NULL
     OR NEW.resolved_at < OLD.created_at THEN
    RAISE EXCEPTION 'route runtime incident resolution transition is invalid';
  END IF;
  IF (to_jsonb(NEW) - ARRAY['local_quarantine_result', 'resolved_at'])
      IS DISTINCT FROM
     (to_jsonb(OLD) - ARRAY['local_quarantine_result', 'resolved_at']) THEN
    RAISE EXCEPTION 'route runtime incident identity is immutable';
  END IF;
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_route_runtime_incident_resolve_once
BEFORE UPDATE OR DELETE ON extension_route_runtime_incidents
FOR EACH ROW EXECUTE FUNCTION enforce_route_runtime_incident_immutability();

CREATE TRIGGER extension_route_runtime_incident_no_truncate
BEFORE TRUNCATE ON extension_route_runtime_incidents
FOR EACH STATEMENT EXECUTE FUNCTION enforce_route_runtime_incident_immutability();

-- +goose Down
-- Never remove the append-only guard while evidence exists.
-- +goose StatementBegin
DO $$
BEGIN
  LOCK TABLE extension_route_runtime_incidents IN ACCESS EXCLUSIVE MODE;
  IF EXISTS (SELECT 1 FROM extension_route_runtime_incidents) THEN
    RAISE EXCEPTION 'cannot remove route runtime incident immutability';
  END IF;
END $$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS extension_route_runtime_incident_no_truncate
  ON extension_route_runtime_incidents;
DROP TRIGGER IF EXISTS extension_route_runtime_incident_resolve_once
  ON extension_route_runtime_incidents;
DROP FUNCTION IF EXISTS enforce_route_runtime_incident_immutability();
DROP TRIGGER IF EXISTS extension_route_runtime_incident_audit_valid
  ON extension_route_runtime_incidents;
DROP FUNCTION IF EXISTS validate_route_runtime_incident_audit();

ALTER TABLE extension_route_runtime_incidents
  DROP CONSTRAINT IF EXISTS extension_route_runtime_incidents_response_stage_status_check,
  DROP CONSTRAINT IF EXISTS extension_route_runtime_incidents_resolution_time_check;
