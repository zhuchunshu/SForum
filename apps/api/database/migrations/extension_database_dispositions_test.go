package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const extensionDatabaseDispositionsMigration = "202607140014_extension_database_dispositions.sql"

func TestFilesIncludesExtensionDatabaseDispositionsMigration(t *testing.T) {
	if _, err := fs.Stat(Files(), extensionDatabaseDispositionsMigration); err != nil {
		t.Fatalf("expected embedded migration %s: %v", extensionDatabaseDispositionsMigration, err)
	}
}

func TestExtensionDatabaseDispositionsPersistExactCleanupReceipts(t *testing.T) {
	body, err := fs.ReadFile(Files(), extensionDatabaseDispositionsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("extension database dispositions migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE extension_database_dispositions",
		"cleanup_id TEXT NOT NULL UNIQUE",
		"operation_id BIGINT NOT NULL UNIQUE",
		"cleanup_mode TEXT NOT NULL",
		"extension_version_id BIGINT NOT NULL",
		"package_digest TEXT NOT NULL",
		"export_evidence_digest TEXT",
		"status TEXT NOT NULL DEFAULT 'prepared'",
		"credential_revoked BOOLEAN NOT NULL DEFAULT FALSE",
		"schema_retained BOOLEAN",
		"roles_removed BOOLEAN NOT NULL DEFAULT FALSE",
		"receipt_id TEXT UNIQUE",
		"proof JSONB",
		"proof_digest TEXT",
		"CREATE INDEX extension_database_dispositions_prepared_idx",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("extension database dispositions migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DROP SCHEMA", "DROP ROLE", "DELETE FROM", "extension_migration_ledger"} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("extension database dispositions Up contains %q", forbidden)
		}
	}
}

func TestExtensionDatabaseDispositionsDownRetainsEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), extensionDatabaseDispositionsMigration)
	if err != nil {
		t.Fatal(err)
	}
	down := strings.Join(strings.Fields(strings.SplitN(string(body), "-- +goose Down", 2)[1]), " ")
	for _, clause := range []string{
		"IF EXISTS (SELECT 1 FROM extension_database_dispositions)",
		"RAISE EXCEPTION 'cannot remove extension database disposition evidence'",
		"DROP TABLE IF EXISTS extension_database_dispositions",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("extension database dispositions Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE", "DROP SCHEMA", "DROP ROLE"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("extension database dispositions Down contains %q", forbidden)
		}
	}
}
