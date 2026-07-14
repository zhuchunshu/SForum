package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const extensionDatabaseRegistryMigration = "202607140013_extension_database_registry.sql"

func TestFilesIncludesExtensionDatabaseRegistryMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == extensionDatabaseRegistryMigration {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", extensionDatabaseRegistryMigration)
}

func TestExtensionDatabaseRegistryOwnsIndependentExactArtifactEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), extensionDatabaseRegistryMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("extension database registry migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE extension_database_resources",
		"schema_name TEXT NOT NULL UNIQUE",
		"owner_role_name TEXT NOT NULL UNIQUE",
		"runtime_role_name TEXT NOT NULL UNIQUE",
		"CREATE TABLE extension_database_grants",
		"extension_version_id BIGINT NOT NULL",
		"package_digest TEXT NOT NULL",
		"database_contract_version TEXT NOT NULL",
		"active_credential_revision BIGINT NOT NULL DEFAULT 0",
		"granted_by_user_id BIGINT NOT NULL",
		"grant_audit_event_id BIGINT NOT NULL",
		"revoked_by_user_id BIGINT",
		"revoke_audit_event_id BIGINT",
		"CREATE TABLE extension_database_credentials",
		"credential_fingerprint TEXT NOT NULL",
		"issued_by_user_id BIGINT NOT NULL",
		"issue_audit_event_id BIGINT NOT NULL",
		"CREATE TABLE extension_database_migration_plans",
		"plan_digest TEXT NOT NULL UNIQUE",
		"dry_run_digest TEXT NOT NULL",
		"source_resume_safe BOOLEAN NOT NULL DEFAULT TRUE",
		"CREATE TABLE extension_database_migration_steps",
		"transaction_policy TEXT NOT NULL",
		"execution_mode TEXT NOT NULL",
		"CREATE TABLE extension_database_migration_state",
		"CREATE TABLE extension_database_migration_proofs",
		"proof_digest TEXT NOT NULL",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("extension database registry migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"REFERENCES extension_migration_ledger", "FROM extension_migration_ledger",
		"JOIN extension_migration_ledger", "goose_db_version", "DROP SCHEMA", "DROP ROLE",
	} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("extension database registry Up must not use %q", forbidden)
		}
	}
}

func TestExtensionDatabaseRegistryDownRetainsEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), extensionDatabaseRegistryMigration)
	if err != nil {
		t.Fatal(err)
	}
	down := strings.Join(strings.Fields(strings.SplitN(string(body), "-- +goose Down", 2)[1]), " ")
	for _, clause := range []string{
		"IF EXISTS (SELECT 1 FROM extension_database_resources)",
		"OR EXISTS (SELECT 1 FROM extension_database_migration_proofs)",
		"RAISE EXCEPTION 'cannot remove extension database registry evidence'",
		"DROP TABLE IF EXISTS extension_database_migration_proofs",
		"DROP TABLE IF EXISTS extension_database_resources",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("extension database registry Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE", "DROP SCHEMA", "DROP ROLE"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("extension database registry Down contains %q", forbidden)
		}
	}
}
