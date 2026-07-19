package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const identitySessionPolicyEvidenceContractMigration = "202607190041_identity_session_policy_evidence_contract.sql"

func TestIdentitySessionPolicyEvidenceContractMigrationIsAdditiveAndFailsClosed(t *testing.T) {
	body, err := fs.ReadFile(Files(), identitySessionPolicyEvidenceContractMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity session policy evidence contract migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE FUNCTION valid_identity_session_policy_evidence(input JSONB)",
		"LANGUAGE plpgsql IMMUTABLE STRICT PARALLEL SAFE",
		"input = jsonb_build_object('policyId', 'core.session.default')",
		"jsonb_typeof(input -> 'providerContractVersion') <> 'string'",
		"jsonb_typeof(input -> 'ownerExtensionVersionId') <> 'number'",
		"jsonb_typeof(input -> 'declarationRevision') <> 'number'",
		"owner_version_id := (input ->> 'ownerExtensionVersionId')::BIGINT",
		"declaration_revision := (input ->> 'declarationRevision')::BIGINT",
		"ownerPackageDigest', input ->> 'ownerPackageDigest'",
		"ADD CONSTRAINT identity_session_policy_events_previous_evidence_check",
		"ADD CONSTRAINT identity_session_policy_events_selected_evidence_check",
		"VALIDATE CONSTRAINT identity_session_policy_events_previous_evidence_check",
		"VALIDATE CONSTRAINT identity_session_policy_events_selected_evidence_check",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("identity session policy evidence contract missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"CREATE TABLE",
		"DROP TABLE",
		"DELETE FROM",
		"UPDATE identity_session_policy_selection_events",
	} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("identity session policy evidence contract Up contains %q", forbidden)
		}
	}

	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"LOCK TABLE identity_session_policy_selection_events IN ACCESS EXCLUSIVE MODE",
		"IF EXISTS (SELECT 1 FROM identity_session_policy_selection_events)",
		"RAISE EXCEPTION 'cannot remove identity session policy evidence contract while evidence exists'",
		"DROP CONSTRAINT IF EXISTS identity_session_policy_events_selected_evidence_check",
		"DROP CONSTRAINT IF EXISTS identity_session_policy_events_previous_evidence_check",
		"DROP FUNCTION IF EXISTS valid_identity_session_policy_evidence(JSONB)",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("identity session policy evidence contract Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("identity session policy evidence contract Down contains %q", forbidden)
		}
	}
}
