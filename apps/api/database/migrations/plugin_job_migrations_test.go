package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const pluginJobMigrationsMigration = "202607140003_extension_plugin_job_migrations.sql"

func TestFilesIncludesPluginJobMigrationsMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() == pluginJobMigrationsMigration {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", pluginJobMigrationsMigration)
}

func TestPluginJobMigrationsMigrationDefinesExactReplacementLedger(t *testing.T) {
	body, err := fs.ReadFile(Files(), pluginJobMigrationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.Join(strings.Fields(string(body)), " ")
	for _, clause := range []string{
		"old_job_id BIGINT PRIMARY KEY",
		"extension_id TEXT NOT NULL",
		"migration_id TEXT NOT NULL",
		"source_contract JSONB NOT NULL",
		"source_trust_grant_id TEXT NOT NULL",
		"target_contract JSONB NOT NULL",
		"target_trust_grant_id TEXT NOT NULL",
		"new_job_id BIGINT UNIQUE",
		"new_job_id IS NULL AND completed_at IS NULL",
		"new_job_id IS NOT NULL AND completed_at IS NOT NULL",
		"extension_plugin_job_migrations_extension_created_idx",
	} {
		if !strings.Contains(sql, clause) {
			t.Fatalf("plugin job migration ledger missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"REFERENCES river_job", "REFERENCES extensions", "UPDATE river_job", "DELETE FROM river_job",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("plugin job migration ledger crosses retained boundary with %q", forbidden)
		}
	}
}

func TestPluginJobMigrationsDownOnlyDropsHostLedger(t *testing.T) {
	body, err := fs.ReadFile(Files(), pluginJobMigrationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("plugin job migration ledger has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	if !strings.Contains(down, "DROP TABLE IF EXISTS extension_plugin_job_migrations") {
		t.Fatal("plugin job migration Down does not remove its ledger")
	}
	for _, forbidden := range []string{"river_job", "extensions", "extension_lifecycle_", "DELETE FROM", "TRUNCATE"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("plugin job migration Down crosses ownership boundary with %q", forbidden)
		}
	}
}
