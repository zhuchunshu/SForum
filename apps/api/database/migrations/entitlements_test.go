package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const entitlementsMigration = "202607150025_entitlements.sql"

func TestEntitlementsMigrationKeepsProviderNeutralEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), entitlementsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("entitlements migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE entitlements",
		"subject_type TEXT NOT NULL",
		"subject_id TEXT NOT NULL",
		"scope_kind TEXT NOT NULL CHECK (scope_kind IN ('resource', 'capability'))",
		"resource_type TEXT",
		"resource_id TEXT",
		"capability TEXT",
		"status TEXT NOT NULL CHECK (status IN ('active', 'revoked', 'expired'))",
		"source_type TEXT NOT NULL",
		"source_id TEXT NOT NULL",
		"valid_from TIMESTAMPTZ NOT NULL",
		"valid_until TIMESTAMPTZ",
		"revision BIGINT NOT NULL DEFAULT 1",
		"CREATE TABLE entitlement_events",
		"entitlement_id BIGINT NOT NULL REFERENCES entitlements(id) ON DELETE RESTRICT",
		"action TEXT NOT NULL CHECK (action IN ('grant', 'revoke', 'expire'))",
		"idempotency_key TEXT NOT NULL UNIQUE",
		"request_fingerprint TEXT NOT NULL",
		"audit_event_id BIGINT NOT NULL",
		"CREATE INDEX entitlements_resource_effective_idx",
		"CREATE INDEX entitlements_capability_effective_idx",
		"CREATE INDEX entitlements_source_idx",
		"CREATE INDEX entitlement_events_entitlement_idx",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("entitlements migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"currency", "amount", "billing", "checkout", "gateway", "provider_transaction",
		"REFERENCES users(", "REFERENCES audit_events(", "ON DELETE CASCADE",
	} {
		if strings.Contains(strings.ToLower(up), strings.ToLower(forbidden)) {
			t.Fatalf("entitlements migration must not use %q", forbidden)
		}
	}
}

func TestEntitlementsMigrationDownFailsClosed(t *testing.T) {
	body, err := fs.ReadFile(Files(), entitlementsMigration)
	if err != nil {
		t.Fatal(err)
	}
	down := strings.Join(strings.Fields(strings.SplitN(string(body), "-- +goose Down", 2)[1]), " ")
	for _, clause := range []string{
		"IF EXISTS (SELECT 1 FROM entitlement_events) OR EXISTS (SELECT 1 FROM entitlements)",
		"RAISE EXCEPTION 'cannot remove entitlement evidence'",
		"DROP TABLE IF EXISTS entitlement_events",
		"DROP TABLE IF EXISTS entitlements",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("entitlements migration Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("entitlements migration Down contains %q", forbidden)
		}
	}
}
