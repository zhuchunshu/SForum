package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const lifecyclePluginRuntimeBindingsMigration = "202607160030_lifecycle_plugin_runtime_bindings.sql"

func TestLifecyclePluginRuntimeBindingsMigrationBindsImmutableDesiredRevision(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecyclePluginRuntimeBindingsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle plugin runtime binding migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"ALTER TABLE extension_lifecycle_publications ADD COLUMN plugin_runtime_publication_revision BIGINT",
		"FOREIGN KEY (plugin_runtime_publication_revision)",
		"REFERENCES plugin_runtime_publications(revision) ON DELETE RESTRICT",
		"CHECK (commit_marker = TRUE OR plugin_runtime_publication_revision IS NULL)",
		"CREATE INDEX extension_lifecycle_publications_plugin_runtime_idx",
		"WHERE plugin_runtime_publication_revision IS NOT NULL",
		"CREATE FUNCTION enforce_lifecycle_plugin_runtime_binding() RETURNS trigger",
		"lifecycle plugin runtime binding requires committed marker",
		"OLD.plugin_runtime_publication_revision IS NOT NULL",
		"NEW.plugin_runtime_publication_revision IS DISTINCT FROM OLD.plugin_runtime_publication_revision",
		"lifecycle plugin runtime binding is immutable",
		"CREATE TRIGGER extension_lifecycle_publications_plugin_runtime_valid",
		"BEFORE INSERT OR UPDATE ON extension_lifecycle_publications",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("lifecycle plugin runtime binding migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"UPDATE extension_lifecycle_publications SET plugin_runtime_publication_revision",
		"UPDATE plugin_runtime_publications",
		"DELETE FROM",
	} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("lifecycle plugin runtime binding migration must not contain %q", forbidden)
		}
	}
}

func TestLifecyclePluginRuntimeBindingsMigrationProtectsEvidenceOnDown(t *testing.T) {
	body, err := fs.ReadFile(Files(), lifecyclePluginRuntimeBindingsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("lifecycle plugin runtime binding migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"WHERE plugin_runtime_publication_revision IS NOT NULL",
		"RAISE EXCEPTION 'cannot remove lifecycle plugin runtime binding history'",
		"DROP TRIGGER IF EXISTS extension_lifecycle_publications_plugin_runtime_valid",
		"DROP FUNCTION IF EXISTS enforce_lifecycle_plugin_runtime_binding()",
		"DROP COLUMN IF EXISTS plugin_runtime_publication_revision",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("lifecycle plugin runtime binding Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE", "DROP TABLE IF EXISTS plugin_runtime_publications"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("lifecycle plugin runtime binding Down destroys evidence with %q", forbidden)
		}
	}
}
