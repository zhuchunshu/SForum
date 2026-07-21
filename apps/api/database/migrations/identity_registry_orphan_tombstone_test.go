package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const identityRegistryOrphanTombstoneMigration = "202607210044_identity_registry_orphan_tombstone.sql"

func TestIdentityRegistryOrphanTombstoneMigrationAllowsMissingOwnerRetirement(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityRegistryOrphanTombstoneMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("orphan tombstone migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE OR REPLACE FUNCTION validate_extension_identity_registry_declaration()",
		"CREATE OR REPLACE FUNCTION validate_extension_identity_registry_publication()",
		"NEW.registry_state IS DISTINCT FROM 'tombstone'",
		"EXISTS (SELECT 1 FROM extensions WHERE id = NEW.owner_extension_id)",
		"identity registry declaration exact artifact is invalid",
		"identity registry root publication exact artifact is invalid",
		"identity registry tombstone does not match the active artifact",
		"identity registry root tombstone does not match the active publication",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("up migration missing clause %q", clause)
		}
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"CREATE OR REPLACE FUNCTION validate_extension_identity_registry_declaration()",
		"CREATE OR REPLACE FUNCTION validate_extension_identity_registry_publication()",
		"IF NOT FOUND THEN",
		"identity registry declaration exact artifact is invalid",
		"identity registry root publication exact artifact is invalid",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("down migration missing clause %q", clause)
		}
	}
}
