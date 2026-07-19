package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const identityUserFieldValuesMigration = "202607190038_identity_user_field_values.sql"

func TestIdentityUserFieldValuesMigrationDefinesHostOwnedJSONBStore(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityUserFieldValuesMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity user-field values migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE identity_user_field_values",
		"user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"field_id TEXT NOT NULL",
		"field_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'",
		"owner_extension_id TEXT NOT NULL",
		"field_contract_version TEXT NOT NULL",
		"field_contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'",
		"field_schema_digest TEXT NOT NULL",
		"field_schema_digest ~ '^[0-9a-f]{64}$'",
		"declaration_revision BIGINT NOT NULL CHECK (declaration_revision > 0)",
		"value_json JSONB",
		"state TEXT NOT NULL CHECK (state IN ('active', 'erased'))",
		"revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0)",
		"updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL",
		"updated_audit_event_id BIGINT NOT NULL CHECK (updated_audit_event_id > 0)",
		"erased_at TIMESTAMPTZ",
		"erased_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL",
		"erase_audit_event_id BIGINT",
		"PRIMARY KEY (user_id, field_id)",
		"CHECK (updated_at >= created_at)",
		"state = 'active' AND value_json IS NOT NULL AND erased_at IS NULL AND erased_by_user_id IS NULL AND erase_audit_event_id IS NULL",
		"state = 'erased' AND value_json IS NULL AND erased_at IS NOT NULL AND erase_audit_event_id IS NOT NULL",
		"CREATE INDEX identity_user_field_values_field_user_idx",
		"ON identity_user_field_values (field_id, user_id)",
		"CREATE INDEX identity_user_field_values_owner_field_user_idx",
		"ON identity_user_field_values (owner_extension_id, field_id, user_id)",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("identity user-field values migration missing %q", clause)
		}
	}

	lower := strings.ToLower(up)
	for _, forbidden := range []string{
		"entity_meta_values",
		"entity_type",
		"value_type",
		"field_key",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("identity user-field values migration must not contain %q", forbidden)
		}
	}
	if strings.Contains(up, "REFERENCES audit_events") {
		t.Fatal("identity user-field values migration must not FK audit_events")
	}
}

func TestIdentityUserFieldValuesMigrationEventsAreRedactedEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityUserFieldValuesMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity user-field values migration has no Down section")
	}
	upParts := strings.SplitN(parts[0], "CREATE TABLE identity_user_field_value_events", 2)
	if len(upParts) != 2 {
		t.Fatal("identity user-field value events table is missing")
	}
	eventSection := strings.Join(strings.Fields(upParts[1]), " ")
	for _, clause := range []string{
		"id BIGSERIAL PRIMARY KEY",
		"user_id BIGINT NOT NULL CHECK (user_id > 0)",
		"field_id TEXT NOT NULL",
		"owner_extension_id TEXT NOT NULL",
		"field_contract_version TEXT NOT NULL",
		"field_schema_digest TEXT NOT NULL",
		"declaration_revision BIGINT NOT NULL CHECK (declaration_revision > 0)",
		"action TEXT NOT NULL CHECK (action IN ('set', 'erase'))",
		"previous_revision BIGINT",
		"next_revision BIGINT NOT NULL CHECK (next_revision > 0)",
		"previous_value_digest TEXT",
		"previous_value_digest ~ '^[0-9a-f]{64}$'",
		"next_value_digest TEXT",
		"next_value_digest ~ '^[0-9a-f]{64}$'",
		"idempotency_key TEXT NOT NULL UNIQUE",
		"octet_length(idempotency_key) BETWEEN 1 AND 128",
		"idempotency_key !~ '[^!-~]'",
		"request_fingerprint TEXT NOT NULL",
		"request_fingerprint ~ '^[0-9a-f]{64}$'",
		"actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL",
		"audit_event_id BIGINT NOT NULL CHECK (audit_event_id > 0)",
		"action = 'set' AND next_value_digest IS NOT NULL AND next_revision = COALESCE(previous_revision + 1, 1)",
		"action = 'erase' AND previous_revision IS NOT NULL AND next_revision = previous_revision + 1 AND previous_value_digest IS NOT NULL AND next_value_digest IS NULL",
		"CREATE INDEX identity_user_field_value_events_user_field_idx",
		"CREATE INDEX identity_user_field_value_events_owner_field_user_idx",
	} {
		if !strings.Contains(eventSection, clause) {
			t.Fatalf("identity user-field value events missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"value_json",
		"raw_value",
		"REFERENCES users(id) ON DELETE CASCADE",
		"REFERENCES audit_events",
		"REFERENCES identity_user_field_values",
	} {
		if strings.Contains(eventSection, forbidden) {
			t.Fatalf("identity user-field value events must not contain %q", forbidden)
		}
	}
}

func TestIdentityUserFieldValuesMigrationDownFailsClosed(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityUserFieldValuesMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity user-field values migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"LOCK TABLE identity_user_field_values, identity_user_field_value_events IN ACCESS EXCLUSIVE MODE",
		"IF EXISTS (SELECT 1 FROM identity_user_field_value_events) OR EXISTS (SELECT 1 FROM identity_user_field_values)",
		"RAISE EXCEPTION 'cannot remove identity user-field value evidence'",
		"DROP TABLE IF EXISTS identity_user_field_value_events",
		"DROP TABLE IF EXISTS identity_user_field_values",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("identity user-field values Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE", "audit_events"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("identity user-field values Down contains %q", forbidden)
		}
	}

	full := strings.Join(strings.Fields(string(body)), " ")
	if strings.Contains(full, "REFERENCES audit_events") {
		t.Fatal("identity user-field values migration must not FK audit_events")
	}
	if strings.Contains(strings.ToLower(full), "entity_meta_values") {
		t.Fatal("identity user-field values migration must never use entity_meta_values")
	}
}
