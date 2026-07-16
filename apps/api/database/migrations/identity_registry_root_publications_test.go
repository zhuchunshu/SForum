package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const identityRegistryRootPublicationsMigration = "202607160033_identity_registry_root_publications.sql"

func TestIdentityRegistryRootPublicationsMigrationBindsCompleteArtifact(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityRegistryRootPublicationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity registry root publications migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE extension_identity_registry_publications",
		"PRIMARY KEY (owner_extension_id, revision)",
		"registry_state IN ('active', 'tombstone')",
		"schema_version = 'sforum.identity-registry@1'",
		"publication_digest TEXT NOT NULL",
		"publication_json JSONB NOT NULL",
		"CREATE FUNCTION validate_extension_identity_registry_publication() RETURNS trigger",
		"FOR KEY SHARE OF extension_versions, extensions",
		"identity registry root tombstone does not match the active publication",
		"identity registry root exact artifact cannot drift on reactivation",
		"CREATE TRIGGER extension_identity_registry_publication_no_truncate",
		"CREATE TRIGGER extension_identity_registry_publication_owner_type_immutable",
		"CREATE TRIGGER extension_identity_registry_publication_active_extension_delete_guard",
		"root publication must be tombstoned before uninstall",
		"CREATE TRIGGER extension_identity_registry_publication_active_version_delete_guard",
		"active identity registry root artifact cannot be removed before tombstone",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("identity registry root publications migration missing %q", clause)
		}
	}
	if strings.Contains(up, "ALTER TABLE extension_identity_registry_owners") ||
		strings.Contains(up, "identity_kind IN ('permission', 'user_field', 'provider',") {
		t.Fatal("root publication migration must not change the frozen identity_kind catalog")
	}
}

func TestIdentityRegistryRootPublicationsMigrationProtectsHistoryOnDown(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityRegistryRootPublicationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity registry root publications migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"LOCK TABLE extension_identity_registry_publications IN ACCESS EXCLUSIVE MODE",
		"IF EXISTS (SELECT 1 FROM extension_identity_registry_publications)",
		"cannot remove extension identity registry root publication history",
		"DROP TABLE IF EXISTS extension_identity_registry_publications",
		"DROP FUNCTION IF EXISTS validate_extension_identity_registry_publication()",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("identity registry root publications Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE", "DROP TABLE IF EXISTS extension_identity_registry_declarations"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("identity registry root publications Down contains %q", forbidden)
		}
	}
}
