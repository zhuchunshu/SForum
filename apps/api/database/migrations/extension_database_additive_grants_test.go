package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const extensionDatabaseAdditiveGrantsMigration = "202607150022_extension_database_additive_grants.sql"

func TestExtensionDatabaseAdditiveGrantsMigrationOwnsExactPowersAndRuntimeLeases(t *testing.T) {
	body, err := fs.ReadFile(Files(), extensionDatabaseAdditiveGrantsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("additive database grant migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"'additive'",
		"CREATE TABLE extension_database_grant_powers",
		"source IN ('legacy_authority', 'manifest_grants')",
		"power = 'raw_core' AND ordinal = 4",
		"INSERT INTO extension_database_grant_powers",
		"WHEN 'raw_core' THEN 4",
		"CREATE UNIQUE INDEX extension_database_grants_active_artifact_idx",
		"CREATE UNIQUE INDEX extension_database_credentials_active_grant_idx",
		"CREATE TABLE extension_database_runtime_leases",
		"runtime_instance_id TEXT NOT NULL",
		"credential_fingerprint TEXT NOT NULL",
		"lease_expires_at TIMESTAMPTZ NOT NULL",
		"issued_by IN ('actor', 'host')",
		"status IN ('active', 'draining', 'revoked', 'failed')",
		"CREATE UNIQUE INDEX extension_database_runtime_leases_live_instance_idx",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("additive database grant migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{"credential_password", "password TEXT", "DROP ROLE", "DROP SCHEMA"} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("additive database grant migration must not contain %q", forbidden)
		}
	}
}

func TestExtensionDatabaseAdditiveGrantsDownProtectsAuthorityEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), extensionDatabaseAdditiveGrantsMigration)
	if err != nil {
		t.Fatal(err)
	}
	down := strings.Join(strings.Fields(strings.SplitN(string(body), "-- +goose Down", 2)[1]), " ")
	for _, clause := range []string{
		"IF EXISTS (SELECT 1 FROM extension_database_grant_powers)",
		"OR EXISTS (SELECT 1 FROM extension_database_runtime_leases)",
		"WHERE authority = 'additive'",
		"RAISE EXCEPTION 'cannot remove additive extension database grant or runtime lease evidence'",
		"DROP TABLE IF EXISTS extension_database_runtime_leases",
		"DROP TABLE IF EXISTS extension_database_grant_powers",
		"CREATE UNIQUE INDEX extension_database_grants_active_extension_idx",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("additive database grant migration Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE", "DROP ROLE", "DROP SCHEMA"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("additive database grant migration Down contains %q", forbidden)
		}
	}
}
