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
