package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const identityRegistryOwnershipMigration = "202607160028_identity_registry_ownership.sql"

func TestIdentityRegistryOwnershipMigrationKeepsAuthorityHostOwned(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityRegistryOwnershipMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity registry ownership migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE extension_identity_registry_owners",
		"identity_kind IN ('permission', 'user_field', 'provider')",
		"PRIMARY KEY (identity_kind, stable_id)",
		"CREATE INDEX extension_identity_registry_owners_extension_idx",
		"left(stable_id, length(owner_extension_id) + 1) = owner_extension_id || '.'",
		"CHECK (owner_extension_id !~ '^core([.]|$)')",
		"CREATE FUNCTION validate_extension_identity_registry_owner() RETURNS trigger",
		"WHERE id = NEW.owner_extension_id FOR NO KEY UPDATE",
		"identity registry owner must be an installed plugin",
		"CREATE FUNCTION reject_extension_identity_registry_history_mutation() RETURNS trigger",
		"CREATE TRIGGER extension_identity_registry_owner_no_truncate",
		"CREATE TRIGGER extension_identity_registry_owner_type_immutable",
		"CREATE TABLE extension_identity_registry_declarations",
		"registry_state IN ('active', 'tombstone')",
		"extension_version_id BIGINT NOT NULL CHECK (extension_version_id > 0)",
		"package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$')",
		"declaration_digest TEXT NOT NULL CHECK (declaration_digest ~ '^[0-9a-f]{64}$')",
		"PRIMARY KEY (identity_kind, stable_id, revision)",
		"CREATE FUNCTION validate_extension_identity_registry_declaration() RETURNS trigger",
		"FOR KEY SHARE OF extension_versions, extensions",
		"NEW.revision <> previous.revision + 1",
		"identity registry tombstone does not match the active artifact",
		"identity registry exact artifact cannot drift on reactivation",
		"CREATE TRIGGER extension_identity_registry_declaration_no_truncate",
		"CREATE TRIGGER extension_identity_registry_active_extension_delete_guard",
		"declarations must be tombstoned before uninstall",
		"CREATE TRIGGER extension_identity_registry_active_version_delete_guard",
		"active identity registry artifact cannot be removed before tombstone",
		"CREATE TABLE extension_permission_role_suggestions",
		"approval_state IN ('pending', 'approved', 'rejected')",
		"CHECK (role_key <> 'super_admin')",
		"permission role suggestion must begin pending",
		"permission role suggestion decision requires Host CAS evidence",
		"permission role suggestion decision actor is not active",
		"permission role suggestion approval target is unavailable",
		"CREATE TRIGGER extension_permission_role_suggestion_no_delete",
		"CREATE TRIGGER extension_permission_role_suggestion_no_truncate",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("identity registry ownership migration missing %q", clause)
		}
	}

	lower := strings.ToLower(up)
	for _, forbidden := range []string{
		"insert into role_permissions",
		"update role_permissions",
		"delete from role_permissions",
		"insert into user_permission_overrides",
		"update user_permission_overrides",
		"delete from user_permission_overrides",
		"entity_meta_values",
		"external_identity",
		"session_cookie",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("identity registry ownership migration must not contain %q", forbidden)
		}
	}
}

func TestIdentityRegistryOwnershipMigrationProtectsHistoryOnDown(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityRegistryOwnershipMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity registry ownership migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"LOCK TABLE extension_permission_role_suggestions, extension_identity_registry_declarations, extension_identity_registry_owners IN ACCESS EXCLUSIVE MODE",
		"IF EXISTS (SELECT 1 FROM extension_identity_registry_owners)",
		"OR EXISTS (SELECT 1 FROM extension_identity_registry_declarations)",
		"OR EXISTS (SELECT 1 FROM extension_permission_role_suggestions)",
		"RAISE EXCEPTION 'cannot remove extension identity registry ownership history'",
		"DROP TABLE IF EXISTS extension_permission_role_suggestions",
		"DROP TABLE IF EXISTS extension_identity_registry_declarations",
		"DROP TABLE IF EXISTS extension_identity_registry_owners",
		"DROP FUNCTION IF EXISTS validate_extension_identity_registry_declaration()",
		"DROP FUNCTION IF EXISTS reject_extension_identity_registry_active_version_delete()",
		"DROP FUNCTION IF EXISTS reject_extension_identity_registry_active_extension_delete()",
		"DROP FUNCTION IF EXISTS reject_extension_identity_registry_history_mutation()",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("identity registry ownership Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{"DELETE FROM", "TRUNCATE", "DROP TABLE IF EXISTS role_permissions", "DROP TABLE IF EXISTS permissions"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("identity registry ownership Down contains %q", forbidden)
		}
	}
}
