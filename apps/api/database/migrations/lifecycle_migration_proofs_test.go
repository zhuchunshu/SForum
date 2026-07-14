package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const lifecycleMigrationProofsMigration = "202607140012_extension_lifecycle_migration_proofs.sql"

func TestFilesIncludesLifecycleMigrationProofsMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == lifecycleMigrationProofsMigration {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", lifecycleMigrationProofsMigration)
}

func TestLifecycleMigrationProofsAreExactAndIgnoreLegacyLedger(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleMigrationProofsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle migration proof migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE extension_lifecycle_migration_proofs",
		"REFERENCES extension_lifecycle_operations(id) ON DELETE RESTRICT",
		"operation TEXT NOT NULL",
		"migration_mode TEXT NOT NULL",
		"step_id TEXT NOT NULL",
		"source_package_digest TEXT",
		"source_migrations_digest TEXT",
		"target_package_digest TEXT NOT NULL",
		"target_migrations_digest TEXT NOT NULL",
		"plan_digest TEXT NOT NULL",
		"first_attempt INTEGER NOT NULL DEFAULT 0",
		"last_attempt INTEGER NOT NULL DEFAULT 0",
		"status TEXT NOT NULL DEFAULT 'not_started'",
		"target_ready BOOLEAN NOT NULL DEFAULT FALSE",
		"source_resume_safe BOOLEAN NOT NULL DEFAULT TRUE",
		"status <> 'executing' OR source_resume_safe = FALSE",
		"status = 'not_started' OR first_attempt > 0",
		"proof_kind NOT IN ('host_noop', 'reused') OR target_ready = TRUE",
		"proof_kind TEXT",
		"proof_id TEXT",
		"proof_digest TEXT",
		"UNIQUE (operation_id, migration_mode)",
		"CREATE INDEX extension_lifecycle_migration_proofs_target_ready_idx",
		"WHERE target_ready = TRUE",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("lifecycle migration proof migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"REFERENCES extension_migration_ledger", "FROM extension_migration_ledger",
		"JOIN extension_migration_ledger", "INSERT INTO extension_migration_ledger",
	} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("V3 proof table must not use the checksum-only v1 ledger through %q", forbidden)
		}
	}
}

func TestLifecycleMigrationProofsDownRetainsEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleMigrationProofsMigration)
	if err != nil {
		t.Fatal(err)
	}
	down := strings.Join(strings.Fields(strings.SplitN(string(body), "-- +goose Down", 2)[1]), " ")
	for _, clause := range []string{
		"IF EXISTS (SELECT 1 FROM extension_lifecycle_migration_proofs)",
		"RAISE EXCEPTION 'cannot remove lifecycle migration proofs'",
		"DROP TABLE IF EXISTS extension_lifecycle_migration_proofs",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("lifecycle migration proof Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE", "DROP TABLE IF EXISTS extension_migration_ledger"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("lifecycle migration proof Down contains %q", forbidden)
		}
	}
}
