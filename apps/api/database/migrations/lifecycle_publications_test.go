package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const lifecyclePublicationsMigration = "202607140007_extension_lifecycle_publications.sql"

func TestFilesIncludesLifecyclePublicationsMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() == lifecyclePublicationsMigration {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", lifecyclePublicationsMigration)
}

func TestLifecyclePublicationsMigrationDefinesRetainedExactFence(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecyclePublicationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle publication migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE extension_lifecycle_publications",
		"REFERENCES extension_lifecycle_operations(id) ON DELETE CASCADE",
		"operation TEXT NOT NULL",
		"step_id TEXT NOT NULL",
		"publication_mode TEXT NOT NULL",
		"source_extension_id TEXT",
		"source_extension_version TEXT",
		"source_package_digest TEXT",
		"source_version_id BIGINT",
		"source_runtime_instance_id TEXT",
		"target_extension_id TEXT NOT NULL",
		"target_extension_version TEXT NOT NULL",
		"target_package_digest TEXT NOT NULL",
		"target_version_id BIGINT NOT NULL",
		"target_runtime_instance_id TEXT NOT NULL",
		"first_attempt INTEGER NOT NULL",
		"last_attempt INTEGER NOT NULL",
		"runtime_attempts JSONB NOT NULL DEFAULT '[]'::jsonb",
		"CHECK (jsonb_typeof(runtime_attempts) = 'array')",
		"committed_attempt INTEGER",
		"commit_marker BOOLEAN NOT NULL DEFAULT FALSE",
		"revision BIGINT NOT NULL DEFAULT 1",
		"UNIQUE (operation_id, step_id, publication_mode)",
		"commit_marker = FALSE AND committed_attempt IS NULL AND committed_at IS NULL",
		"commit_marker = TRUE AND committed_attempt BETWEEN first_attempt AND last_attempt",
		"CREATE INDEX extension_lifecycle_publications_uncommitted_idx",
		"WHERE commit_marker = FALSE",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("lifecycle publication migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"REFERENCES extensions", "REFERENCES extension_versions", "REFERENCES audit_events",
	} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("publication journal has retention-blocking foreign key %q", forbidden)
		}
	}
}

func TestLifecyclePublicationsDownRefusesToDeleteEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecyclePublicationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle publication migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"IF EXISTS (SELECT 1 FROM extension_lifecycle_publications)",
		"RAISE EXCEPTION 'cannot remove lifecycle publication history'",
		"DROP TABLE IF EXISTS extension_lifecycle_publications",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("lifecycle publication Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE", "DROP TABLE IF EXISTS extensions", "DROP TABLE IF EXISTS extension_lifecycle_operations"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("lifecycle publication Down crosses retention boundary with %q", forbidden)
		}
	}
}
