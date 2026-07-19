-- +goose Up
-- Identity transition evidence is one event per durable aggregate revision.
-- Existing duplicate evidence makes this migration fail instead of choosing a
-- winner and silently weakening the audit trail.
ALTER TABLE identity_external_link_events
  ADD CONSTRAINT identity_external_link_events_link_revision_key
  UNIQUE (link_id, next_revision);

ALTER TABLE identity_user_field_value_events
  ADD CONSTRAINT identity_user_field_value_events_user_field_revision_key
  UNIQUE (user_id, field_id, next_revision);

ALTER TABLE identity_session_policy_selection_events
  ADD CONSTRAINT identity_session_policy_selection_events_revision_key
  UNIQUE (selection_revision);

-- Event rows are append-only. The only permitted UPDATE is PostgreSQL's
-- ON DELETE SET NULL action for actor provenance after the referenced user has
-- actually been deleted; every other column must remain identical.
-- +goose StatementBegin
CREATE FUNCTION enforce_identity_event_ledger_immutability() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF TG_OP = 'UPDATE'
     AND pg_trigger_depth() > 1
     AND OLD.actor_user_id IS NOT NULL
     AND NEW.actor_user_id IS NULL
     AND (to_jsonb(NEW) - 'actor_user_id') IS NOT DISTINCT FROM
         (to_jsonb(OLD) - 'actor_user_id')
     AND NOT EXISTS (
       SELECT 1 FROM users WHERE id = OLD.actor_user_id
     ) THEN
    RETURN NEW;
  END IF;

  RAISE EXCEPTION 'identity event ledger % is append-only', TG_TABLE_NAME;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER identity_external_link_events_append_only
BEFORE UPDATE OR DELETE ON identity_external_link_events
FOR EACH ROW EXECUTE FUNCTION enforce_identity_event_ledger_immutability();

CREATE TRIGGER identity_external_link_events_no_truncate
BEFORE TRUNCATE ON identity_external_link_events
FOR EACH STATEMENT EXECUTE FUNCTION enforce_identity_event_ledger_immutability();

CREATE TRIGGER identity_user_field_value_events_append_only
BEFORE UPDATE OR DELETE ON identity_user_field_value_events
FOR EACH ROW EXECUTE FUNCTION enforce_identity_event_ledger_immutability();

CREATE TRIGGER identity_user_field_value_events_no_truncate
BEFORE TRUNCATE ON identity_user_field_value_events
FOR EACH STATEMENT EXECUTE FUNCTION enforce_identity_event_ledger_immutability();

CREATE TRIGGER identity_session_policy_selection_events_append_only
BEFORE UPDATE OR DELETE ON identity_session_policy_selection_events
FOR EACH ROW EXECUTE FUNCTION enforce_identity_event_ledger_immutability();

CREATE TRIGGER identity_session_policy_selection_events_no_truncate
BEFORE TRUNCATE ON identity_session_policy_selection_events
FOR EACH STATEMENT EXECUTE FUNCTION enforce_identity_event_ledger_immutability();

-- +goose Down
-- Removing these guards would make retained evidence mutable. Rollback is
-- allowed only while all three ledgers are still unused.
-- +goose StatementBegin
DO $$
BEGIN
  LOCK TABLE
    identity_external_link_events,
    identity_user_field_value_events,
    identity_session_policy_selection_events
    IN ACCESS EXCLUSIVE MODE;

  IF EXISTS (SELECT 1 FROM identity_external_link_events)
     OR EXISTS (SELECT 1 FROM identity_user_field_value_events)
     OR EXISTS (SELECT 1 FROM identity_session_policy_selection_events) THEN
    RAISE EXCEPTION 'cannot remove identity event ledger hardening while evidence exists';
  END IF;
END $$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS identity_session_policy_selection_events_no_truncate
  ON identity_session_policy_selection_events;
DROP TRIGGER IF EXISTS identity_session_policy_selection_events_append_only
  ON identity_session_policy_selection_events;
DROP TRIGGER IF EXISTS identity_user_field_value_events_no_truncate
  ON identity_user_field_value_events;
DROP TRIGGER IF EXISTS identity_user_field_value_events_append_only
  ON identity_user_field_value_events;
DROP TRIGGER IF EXISTS identity_external_link_events_no_truncate
  ON identity_external_link_events;
DROP TRIGGER IF EXISTS identity_external_link_events_append_only
  ON identity_external_link_events;

DROP FUNCTION IF EXISTS enforce_identity_event_ledger_immutability();

ALTER TABLE identity_session_policy_selection_events
  DROP CONSTRAINT IF EXISTS identity_session_policy_selection_events_revision_key;
ALTER TABLE identity_user_field_value_events
  DROP CONSTRAINT IF EXISTS identity_user_field_value_events_user_field_revision_key;
ALTER TABLE identity_external_link_events
  DROP CONSTRAINT IF EXISTS identity_external_link_events_link_revision_key;
