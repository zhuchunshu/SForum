package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestExtensionPermissionLocalizationMigrationKeepsCopyExtensionOwned(t *testing.T) {
	body, err := fs.ReadFile(Files(), "202607231001_extension_permission_localization.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("extension permission localization migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"ADD COLUMN label TEXT NOT NULL DEFAULT ''",
		"ADD COLUMN label_locales JSONB NOT NULL DEFAULT '{}'::jsonb",
		"ADD COLUMN description_locales JSONB NOT NULL DEFAULT '{}'::jsonb",
		"FROM extension_permission_catalog AS catalog",
		"JOIN extension_versions AS version",
		"version.manifest -> 'permissionDefinitions'",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("extension permission localization migration missing %q", clause)
		}
	}
}
