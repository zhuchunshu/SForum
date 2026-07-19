-- +goose Up
-- Session-policy evidence is immutable after migration 040, so reject any
-- ambiguous or secret-bearing JSON shape before it can become permanent.
-- Core is one exact key; plugin evidence is the exact seven-field provenance
-- tuple already frozen by migration 039's singleton columns.
-- +goose StatementBegin
CREATE FUNCTION valid_identity_session_policy_evidence(input JSONB) RETURNS BOOLEAN
LANGUAGE plpgsql
IMMUTABLE
STRICT
PARALLEL SAFE
AS $$
DECLARE
  owner_version_id BIGINT;
  declaration_revision BIGINT;
BEGIN
  IF jsonb_typeof(input) <> 'object' THEN
    RETURN FALSE;
  END IF;

  IF input = jsonb_build_object('policyId', 'core.session.default') THEN
    RETURN TRUE;
  END IF;

  IF jsonb_typeof(input -> 'policyId') <> 'string'
     OR jsonb_typeof(input -> 'providerContractVersion') <> 'string'
     OR jsonb_typeof(input -> 'ownerExtensionId') <> 'string'
     OR jsonb_typeof(input -> 'ownerExtensionVersionId') <> 'number'
     OR jsonb_typeof(input -> 'ownerExtensionVersion') <> 'string'
     OR jsonb_typeof(input -> 'ownerPackageDigest') <> 'string'
     OR jsonb_typeof(input -> 'declarationRevision') <> 'number' THEN
    RETURN FALSE;
  END IF;

  owner_version_id := (input ->> 'ownerExtensionVersionId')::BIGINT;
  declaration_revision := (input ->> 'declarationRevision')::BIGINT;

  IF (input ->> 'policyId') = 'core.session.default'
     OR (input ->> 'policyId') !~ '^[a-z0-9][a-z0-9._-]{1,120}$'
     OR (input ->> 'providerContractVersion') !~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'
     OR (input ->> 'ownerExtensionId') !~ '^[a-z0-9][a-z0-9._-]{1,120}$'
     OR (input ->> 'ownerExtensionId') ~ '^core[.]'
     OR owner_version_id <= 0
     OR (input ->> 'ownerExtensionVersion') <> btrim(input ->> 'ownerExtensionVersion')
     OR octet_length(input ->> 'ownerExtensionVersion') NOT BETWEEN 1 AND 100
     OR (input ->> 'ownerPackageDigest') !~ '^[0-9a-f]{64}$'
     OR declaration_revision <= 0 THEN
    RETURN FALSE;
  END IF;

  RETURN input = jsonb_build_object(
    'policyId', input ->> 'policyId',
    'providerContractVersion', input ->> 'providerContractVersion',
    'ownerExtensionId', input ->> 'ownerExtensionId',
    'ownerExtensionVersionId', owner_version_id,
    'ownerExtensionVersion', input ->> 'ownerExtensionVersion',
    'ownerPackageDigest', input ->> 'ownerPackageDigest',
    'declarationRevision', declaration_revision
  );
EXCEPTION
  WHEN invalid_text_representation OR numeric_value_out_of_range THEN
    RETURN FALSE;
END;
$$;
-- +goose StatementEnd

ALTER TABLE identity_session_policy_selection_events
  ADD CONSTRAINT identity_session_policy_events_previous_evidence_check
  CHECK (
    previous_selection IS NULL
    OR valid_identity_session_policy_evidence(previous_selection)
  ) NOT VALID;

ALTER TABLE identity_session_policy_selection_events
  ADD CONSTRAINT identity_session_policy_events_selected_evidence_check
  CHECK (
    selected_selection IS NULL
    OR valid_identity_session_policy_evidence(selected_selection)
  ) NOT VALID;

ALTER TABLE identity_session_policy_selection_events
  VALIDATE CONSTRAINT identity_session_policy_events_previous_evidence_check;

ALTER TABLE identity_session_policy_selection_events
  VALIDATE CONSTRAINT identity_session_policy_events_selected_evidence_check;

-- +goose Down
-- Removing the exact evidence checks would weaken retained immutable rows.
-- Permit rollback only before this ledger has received evidence.
-- +goose StatementBegin
DO $$
BEGIN
  LOCK TABLE identity_session_policy_selection_events IN ACCESS EXCLUSIVE MODE;
  IF EXISTS (SELECT 1 FROM identity_session_policy_selection_events) THEN
    RAISE EXCEPTION 'cannot remove identity session policy evidence contract while evidence exists';
  END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE identity_session_policy_selection_events
  DROP CONSTRAINT IF EXISTS identity_session_policy_events_selected_evidence_check;

ALTER TABLE identity_session_policy_selection_events
  DROP CONSTRAINT IF EXISTS identity_session_policy_events_previous_evidence_check;

DROP FUNCTION IF EXISTS valid_identity_session_policy_evidence(JSONB);
