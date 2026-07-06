package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestFilesIncludesSQLMigrations(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sql") {
			return
		}
	}
	t.Fatal("expected embedded SQL migrations")
}

func TestFilesIncludesForumTaxonomyMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	const expected = "202607070003_forum_taxonomy.sql"
	for _, entry := range entries {
		if entry.Name() == expected {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", expected)
}
