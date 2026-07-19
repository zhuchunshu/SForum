package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const identitySessionPolicySelectionMigration = "202607190039_identity_session_policy_selection.sql"

func TestIdentitySessionPolicySelectionMigrationDefinesHostOwnedSingleton(t *testing.T) {
	body, err := fs.ReadFile(Files(), identitySessionPolicySelectionMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity session policy selection migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE identity_session_policy_selection",
		"singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton)",
		"policy_id TEXT NOT NULL",
		"policy_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'",
		"provider_contract_version TEXT",
		"provider_contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'",
		"owner_extension_id TEXT",
		"owner_extension_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'",
		"owner_extension_version_id BIGINT",
		"owner_extension_version_id IS NULL OR owner_extension_version_id > 0",
		"owner_extension_version TEXT",
		"owner_package_digest TEXT",
		"owner_package_digest ~ '^[0-9a-f]{64}$'",
		"declaration_revision BIGINT",
		"declaration_revision IS NULL OR declaration_revision > 0",
		// Exact Core tuple: recommended default with every plugin column NULL.
		"policy_id = 'core.session.default'",
		"provider_contract_version IS NULL",
		"owner_extension_id IS NULL",
		"owner_extension_version_id IS NULL",
		"owner_extension_version IS NULL",
		"owner_package_digest IS NULL",
		"declaration_revision IS NULL",
		// Exact plugin tuple: non-Core policy with full owner/provider binding.
		"policy_id <> 'core.session.default'",
		"provider_contract_version IS NOT NULL",
		"owner_extension_id IS NOT NULL",
		"owner_extension_id !~ '^core[.]'",
		"owner_extension_version_id IS NOT NULL",
		"owner_extension_version IS NOT NULL",
		"owner_package_digest IS NOT NULL",
		"declaration_revision IS NOT NULL",
		"revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0)",
		"selected_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL",
		"selection_audit_event_id BIGINT NOT NULL CHECK (selection_audit_event_id > 0)",
		"selected_at TIMESTAMPTZ NOT NULL",
		"updated_at TIMESTAMPTZ NOT NULL",
		"CHECK (updated_at >= selected_at)",
		"CREATE INDEX identity_session_policy_selection_owner_provider_idx",
		"ON identity_session_policy_selection ( owner_extension_id, owner_extension_version_id, owner_package_digest, provider_contract_version, policy_id )",
		"WHERE owner_extension_id IS NOT NULL",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("identity session policy selection migration missing %q", clause)
		}
	}

	// Implicit Core default: no bootstrap row so empty table remains usable
	// and unused Down can still drop the schema without erasing evidence.
	if strings.Contains(strings.ToUpper(up), "INSERT INTO") {
		t.Fatal("identity session policy selection migration must not INSERT a bootstrap row")
	}

	// Dedicated Host table; do not reuse web_options or generic provider selections.
	lower := strings.ToLower(up)
	for _, forbidden := range []string{
		"web_options",
		"extension_provider_slot_selections",
		"extension_route_provider_selections",
		"entity_meta",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("identity session policy selection must not reuse %q", forbidden)
		}
	}
	if strings.Contains(up, "REFERENCES audit_events") {
		t.Fatal("identity session policy selection migration must not FK audit_events")
	}
}

func TestIdentitySessionPolicySelectionMigrationEventsAreMetadataOnly(t *testing.T) {
	body, err := fs.ReadFile(Files(), identitySessionPolicySelectionMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity session policy selection migration has no Down section")
	}
	upParts := strings.SplitN(parts[0], "CREATE TABLE identity_session_policy_selection_events", 2)
	if len(upParts) != 2 {
		t.Fatal("identity session policy selection events table is missing")
	}
	eventSection := strings.Join(strings.Fields(upParts[1]), " ")
	for _, clause := range []string{
		"id BIGSERIAL PRIMARY KEY",
		"action TEXT NOT NULL CHECK (action IN ('select', 'reset', 'invalidate'))",
		"previous_selection JSONB",
		"previous_selection IS NULL OR jsonb_typeof(previous_selection) = 'object'",
		"selected_selection JSONB",
		"selected_selection IS NULL OR jsonb_typeof(selected_selection) = 'object'",
		"actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL",
		"audit_event_id BIGINT NOT NULL CHECK (audit_event_id > 0)",
		"reason_code TEXT NOT NULL DEFAULT ''",
		"reason_code = '' OR reason_code ~ '^[a-z0-9][a-z0-9._-]{0,127}$'",
		"selection_revision BIGINT NOT NULL CHECK (selection_revision > 0)",
		"created_at TIMESTAMPTZ NOT NULL",
		// First select from implicit Core may encode previous as NULL; reset/
		// invalidate require previous and encode Core default as selected NULL.
		"action = 'select' AND selected_selection IS NOT NULL",
		"action IN ('reset', 'invalidate') AND selected_selection IS NULL AND previous_selection IS NOT NULL",
		"previous_selection IS NOT NULL OR (action = 'select' AND selection_revision = 1)",
		"action <> 'invalidate' OR reason_code <> ''",
		"CREATE INDEX identity_session_policy_selection_events_created_idx",
		"CREATE INDEX identity_session_policy_selection_events_owner_idx",
	} {
		if !strings.Contains(eventSection, clause) {
			t.Fatalf("identity session policy selection events missing %q", clause)
		}
	}

	for _, forbidden := range []string{
		"REFERENCES audit_events",
		"REFERENCES identity_session_policy_selection",
		"REFERENCES users(id) ON DELETE CASCADE",
	} {
		if strings.Contains(eventSection, forbidden) {
			t.Fatalf("identity session policy selection events must not contain %q", forbidden)
		}
	}

	// Event payloads are selection metadata only; forbid secret-bearing names.
	eventLower := strings.ToLower(eventSection)
	for _, forbidden := range []string{
		"password",
		"cookie",
		"access_token",
		"refresh_token",
		"session_id",
		"csrf",
		"secret",
		"raw_token",
		"pat_plaintext",
	} {
		if strings.Contains(eventLower, forbidden) {
			t.Fatalf("identity session policy selection events must not store %q", forbidden)
		}
	}
}

func TestIdentitySessionPolicySelectionMigrationDownFailsClosed(t *testing.T) {
	body, err := fs.ReadFile(Files(), identitySessionPolicySelectionMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity session policy selection migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"LOCK TABLE identity_session_policy_selection, identity_session_policy_selection_events IN ACCESS EXCLUSIVE MODE",
		"IF EXISTS (SELECT 1 FROM identity_session_policy_selection_events) OR EXISTS (SELECT 1 FROM identity_session_policy_selection)",
		"RAISE EXCEPTION 'cannot remove identity session policy selection evidence'",
		"DROP TABLE IF EXISTS identity_session_policy_selection_events",
		"DROP TABLE IF EXISTS identity_session_policy_selection",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("identity session policy selection Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("identity session policy selection Down contains %q", forbidden)
		}
	}

	full := strings.Join(strings.Fields(string(body)), " ")
	if strings.Contains(full, "REFERENCES audit_events") {
		t.Fatal("identity session policy selection migration must not FK audit_events")
	}
	fullLower := strings.ToLower(full)
	for _, forbidden := range []string{
		"web_options",
		"extension_provider_slot_selections",
		"extension_route_provider_selections",
	} {
		if strings.Contains(fullLower, forbidden) {
			t.Fatalf("identity session policy selection migration must not reuse %q", forbidden)
		}
	}
	// Full file must never bootstrap a durable Core selection row.
	if strings.Contains(strings.ToUpper(full), "INSERT INTO") {
		t.Fatal("identity session policy selection migration must not INSERT bootstrap rows")
	}
}
