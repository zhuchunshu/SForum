package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const lifecycleOperationLedgerMigration = "202607140001_extension_lifecycle_operation_ledger.sql"

func TestFilesIncludesLifecycleOperationLedgerMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() == lifecycleOperationLedgerMigration {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", lifecycleOperationLedgerMigration)
}

func TestLifecycleOperationLedgerMigrationDefinesRecoveryContract(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleOperationLedgerMigration)
	if err != nil {
		t.Fatalf("read lifecycle operation ledger migration: %v", err)
	}
	sql := strings.Join(strings.Fields(string(body)), " ")
	for _, clause := range []string{
		"CREATE TABLE extension_lifecycle_operations",
		"UNIQUE (extension_id, idempotency_key)",
		"request_fingerprint TEXT NOT NULL",
		"trust_grant_id BIGINT REFERENCES extension_trust_grants(id) ON DELETE RESTRICT",
		"audit_event_id BIGINT",
		"authority_snapshot JSONB NOT NULL",
		"operation = 'uninstall' AND removal_mode IS NOT NULL",
		"operation <> 'uninstall' AND removal_mode IS NULL",
		"forced = FALSE OR operation = 'uninstall'",
		"terminal_result IN ('succeeded', 'failed', 'cancelled', 'skipped')",
		"completed_at IS NULL AND terminal_result IS NULL",
		"completed_at IS NOT NULL AND terminal_result IS NOT NULL",
		"terminal_result IS DISTINCT FROM 'failed' OR state = 'failed'",
		"CREATE UNIQUE INDEX extension_lifecycle_operations_one_open_idx",
		"CREATE TABLE extension_lifecycle_steps",
		"UNIQUE (operation_id, step_id, attempt)",
		"CREATE UNIQUE INDEX extension_lifecycle_steps_one_open_attempt_idx",
		"completed_units <= total_units",
		"updated_at TIMESTAMPTZ NOT NULL DEFAULT now()",
		"status NOT IN ('failed', 'cancelled') OR (error_code <> '' AND error_reason <> '')",
	} {
		if !strings.Contains(sql, clause) {
			t.Fatalf("lifecycle operation ledger migration missing %q", clause)
		}
	}
	if strings.Contains(sql, "REFERENCES audit_events") {
		t.Fatal("lifecycle ledger must not block audit retention with a foreign key")
	}
	for _, state := range []string{
		"planned", "migrating", "starting", "healthy", "registering",
		"enabled", "draining", "uninstalling", "failed", "recovery",
	} {
		if !strings.Contains(sql, "'"+state+"'") {
			t.Fatalf("lifecycle operation ledger migration missing state %q", state)
		}
	}
	for _, mode := range []string{"preserve", "export_then_remove", "complete_removal"} {
		if !strings.Contains(sql, "'"+mode+"'") {
			t.Fatalf("lifecycle operation ledger migration missing removal mode %q", mode)
		}
	}
}

func TestLifecycleOperationLedgerDownDoesNotDeletePluginData(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleOperationLedgerMigration)
	if err != nil {
		t.Fatalf("read lifecycle operation ledger migration: %v", err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle operation ledger migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, statement := range []string{
		"DROP TABLE IF EXISTS extension_lifecycle_steps",
		"DROP TABLE IF EXISTS extension_lifecycle_operations",
	} {
		if !strings.Contains(down, statement) {
			t.Fatalf("lifecycle operation ledger Down missing %q", statement)
		}
	}
	for _, forbidden := range []string{
		"DROP TABLE extensions", "DROP TABLE IF EXISTS extensions",
		"DROP TABLE extension_migration_ledger", "DROP TABLE IF EXISTS extension_migration_ledger",
		"DELETE FROM extensions", "DELETE FROM extension_migration_ledger",
	} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("lifecycle operation ledger Down crosses plugin data boundary with %q", forbidden)
		}
	}
}
