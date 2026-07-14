package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const lifecycleStatePublicationsMigration = "202607140009_extension_lifecycle_state_publications.sql"

func TestFilesIncludesLifecycleStatePublicationsMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == lifecycleStatePublicationsMigration {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", lifecycleStatePublicationsMigration)
}
func TestLifecycleStatePublicationsMigrationPersistsExactSourceAndTarget(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleStatePublicationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle state publication migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE extension_lifecycle_state_publications",
		"operation_id BIGINT NOT NULL",
		"operation TEXT NOT NULL",
		"step_id TEXT NOT NULL",
		"publication_mode TEXT NOT NULL",
		"source_status TEXT NOT NULL",
		"source_active_version_id BIGINT NOT NULL",
		"source_active_version TEXT NOT NULL",
		"source_active_package_digest TEXT NOT NULL",
		"source_staged_version_id BIGINT",
		"target_status TEXT NOT NULL",
		"target_active_version_id BIGINT NOT NULL",
		"target_active_version TEXT NOT NULL",
		"target_active_package_digest TEXT NOT NULL",
		"target_staged_version_id BIGINT",
		"transaction_state TEXT NOT NULL DEFAULT 'source'",
		"first_attempt INTEGER NOT NULL",
		"last_attempt INTEGER NOT NULL",
		"revision BIGINT NOT NULL DEFAULT 1",
		"UNIQUE (operation_id, step_id, publication_mode)",
		"FOREIGN KEY (operation_id, step_id, publication_mode) REFERENCES extension_lifecycle_publications(operation_id, step_id, publication_mode) ON DELETE CASCADE",
		"publication_mode = 'activate' AND target_status = 'enabled'",
		"publication_mode = 'deactivate' AND target_status = 'disabled'",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("lifecycle state publication migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"REFERENCES extensions", "REFERENCES extension_versions", "REFERENCES audit_events",
		"UPDATE extensions", "DELETE FROM extensions",
	} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("lifecycle state publication migration crosses retained-state boundary with %q", forbidden)
		}
	}
}

func TestLifecycleStatePublicationsDownRefusesToDeleteRecoveryFacts(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecycleStatePublicationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle state publication migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"IF EXISTS (SELECT 1 FROM extension_lifecycle_state_publications)",
		"RAISE EXCEPTION 'cannot remove lifecycle state publication history'",
		"DROP TABLE IF EXISTS extension_lifecycle_state_publications",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("lifecycle state publication Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE", "DROP TABLE IF EXISTS extensions", "DROP TABLE IF EXISTS extension_lifecycle_operations"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("lifecycle state publication Down crosses retention boundary with %q", forbidden)
		}
	}
}
