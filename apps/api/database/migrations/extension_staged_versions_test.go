package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const extensionStagedVersionsMigration = "202607140005_extension_staged_versions.sql"

func TestFilesIncludesExtensionStagedVersionsMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == extensionStagedVersionsMigration {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", extensionStagedVersionsMigration)
}

func TestExtensionStagedVersionsMigrationIsAdditive(t *testing.T) {
	body, err := fs.ReadFile(Files(), extensionStagedVersionsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("staged version migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"ADD COLUMN staged_version_id BIGINT",
		"ADD CONSTRAINT extension_versions_extension_id_id_key UNIQUE (extension_id, id)",
		"ADD CONSTRAINT extensions_staged_version_fk",
		"FOREIGN KEY (id, staged_version_id)",
		"REFERENCES extension_versions(extension_id, id)",
		"ON DELETE SET NULL (staged_version_id)",
		"WHERE staged_version_id IS NOT NULL",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("staged version migration Up missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"UPDATE extensions",
		"DELETE FROM",
		"DROP COLUMN active_version_id",
	} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("staged version migration must not mutate active rows with %q", forbidden)
		}
	}
}
