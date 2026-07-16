-- +goose Up
CREATE TABLE host_installation_identity (
  singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
  installation_id TEXT NOT NULL UNIQUE
    CHECK (installation_id ~ '^[0-9a-f]{64}$'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp()
);

-- The installation identity is Host authority. Runtime code may initialize the
-- empty singleton once, but ordinary SQL must never rotate or remove it.
-- +goose StatementBegin
CREATE FUNCTION reject_host_installation_identity_mutation() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION 'host installation identity is immutable';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER host_installation_identity_immutable
BEFORE UPDATE OR DELETE ON host_installation_identity
FOR EACH ROW EXECUTE FUNCTION reject_host_installation_identity_mutation();

CREATE TRIGGER host_installation_identity_no_truncate
BEFORE TRUNCATE ON host_installation_identity
FOR EACH STATEMENT EXECUTE FUNCTION reject_host_installation_identity_mutation();

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
  LOCK TABLE host_installation_identity IN ACCESS EXCLUSIVE MODE;
  IF EXISTS (SELECT 1 FROM host_installation_identity) THEN
    RAISE EXCEPTION 'cannot remove initialized host installation identity';
  END IF;
END;
$$;
-- +goose StatementEnd

DROP TABLE IF EXISTS host_installation_identity;
DROP FUNCTION IF EXISTS reject_host_installation_identity_mutation();
