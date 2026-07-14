package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const lifecycleCleanupRecordsMigration = "202607140008_extension_lifecycle_cleanup_records.sql"

func TestFilesIncludesLifecycleCleanupRecordsMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() == lifecycleCleanupRecordsMigration {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", lifecycleCleanupRecordsMigration)
}

func TestLifecycleCleanupRecordsMigrationDefinesRetainedTombstoneContract(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleCleanupRecordsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle cleanup migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE extension_lifecycle_cleanup_records",
		"REFERENCES extension_lifecycle_operations(id) ON DELETE RESTRICT",
		"UNIQUE (operation_id, step_id, cleanup_mode)",
		"cleanup_mode TEXT NOT NULL",
		"'disable', 'retired_source', 'uninstall_preserve'",
		"'uninstall_export_then_remove', 'uninstall_complete_removal'",
		"status IN ('retained', 'pending', 'finalized')",
		"retained_extension_version TEXT NOT NULL",
		"retained_package_digest TEXT NOT NULL",
		"retained_version_id BIGINT NOT NULL",
		"retained_runtime_instance_id TEXT NOT NULL",
		"retained_package_path TEXT NOT NULL",
		"identity_snapshot JSONB NOT NULL",
		"package_snapshot JSONB NOT NULL",
		"runtime_recovery_snapshot JSONB NOT NULL",
		"runtime_recovery_attempts JSONB NOT NULL DEFAULT '[]'::jsonb",
		"target_package_digest TEXT NOT NULL",
		"identity_recovery_evidence_retained BOOLEAN NOT NULL DEFAULT TRUE",
		"package_recovery_evidence_retained BOOLEAN NOT NULL DEFAULT TRUE",
		"runtime_recovery_evidence_retained BOOLEAN NOT NULL DEFAULT TRUE",
		"physical_identity_present BOOLEAN NOT NULL DEFAULT TRUE",
		"physical_package_present BOOLEAN NOT NULL DEFAULT TRUE",
		"physical_runtime_recovery_present BOOLEAN NOT NULL DEFAULT TRUE",
		"status = 'finalized' AND NOT physical_identity_present",
		"retention_marker TEXT",
		"export_artifact_id TEXT",
		"export_artifact_id ~ '^[A-Za-z0-9._-]{1,200}$'",
		"export_digest TEXT",
		"export_evidence_action TEXT",
		"export_evidence JSONB",
		"export_evidence_digest TEXT",
		"finalized_operation_revision BIGINT",
		"finalized_operation_completed_at TIMESTAMPTZ",
		"purge_receipt_id TEXT",
		"purge_receipt_id ~ '^[A-Za-z0-9._-]{1,200}$'",
		"purge_proof JSONB",
		"purge_proof_digest TEXT",
		"purge_proof_digest ~ '^[0-9a-f]{64}$'",
		"CREATE INDEX extension_lifecycle_cleanup_records_pending_idx",
		"CREATE UNIQUE INDEX extension_lifecycle_cleanup_records_one_tombstone_idx",
		"WHERE record_kind = 'uninstall_tombstone' AND status = 'pending'",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("lifecycle cleanup migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"REFERENCES extensions", "REFERENCES extension_versions", "REFERENCES audit_events",
		"DELETE FROM extensions", "DELETE FROM extension_versions",
	} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("cleanup record crosses retained identity/package boundary with %q", forbidden)
		}
	}
}

func TestLifecycleCleanupRecordsDownRefusesToDeleteEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleCleanupRecordsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle cleanup migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"-- +goose StatementBegin",
		"IF EXISTS (SELECT 1 FROM extension_lifecycle_cleanup_records)",
		"RAISE EXCEPTION 'cannot remove lifecycle cleanup history'",
		"-- +goose StatementEnd",
		"DROP TABLE IF EXISTS extension_lifecycle_cleanup_records",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("lifecycle cleanup Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"DELETE FROM", "TRUNCATE", "DROP TABLE IF EXISTS extensions",
		"DROP TABLE IF EXISTS extension_versions", "DROP TABLE IF EXISTS extension_lifecycle_operations",
	} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("lifecycle cleanup Down crosses retention boundary with %q", forbidden)
		}
	}
}
