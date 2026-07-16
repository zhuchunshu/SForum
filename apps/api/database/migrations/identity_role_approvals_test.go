package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const identityRoleApprovalsMigration = "202607160029_identity_role_approvals.sql"

func TestIdentityRoleApprovalsMigrationBindsAdditiveHostGrant(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityRoleApprovalsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity role approvals migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"LOCK TABLE extension_permission_role_suggestions, extension_identity_registry_declarations, extension_identity_registry_owners, permissions, role_permissions, roles, users, user_roles, user_permission_overrides, audit_events IN ACCESS EXCLUSIVE MODE",
		"CREATE TABLE extension_permission_catalog",
		"declaration_revision BIGINT NOT NULL CHECK (declaration_revision > 0)",
		"FOREIGN KEY (identity_kind, permission_key, declaration_revision)",
		"REFERENCES extension_identity_registry_declarations( identity_kind, stable_id, revision )",
		"cannot backfill extension permission owner without declaration history",
		"cannot claim untracked Host permission for extension catalog",
		"CREATE TABLE extension_permission_role_grants",
		"suggestion_id BIGINT PRIMARY KEY",
		"applied_by_user_id BIGINT NOT NULL CHECK (applied_by_user_id > 0)",
		"applied_audit_event_id BIGINT NOT NULL REFERENCES audit_events(id) ON DELETE RESTRICT",
		"REFERENCES permissions(key) ON DELETE RESTRICT",
		"FOREIGN KEY (identity_kind, permission_key, owner_extension_id)",
		"CREATE TRIGGER extension_permission_catalog_immutable",
		"CREATE TRIGGER extension_permission_catalog_no_truncate",
		"CREATE TRIGGER extension_permission_role_grant_immutable",
		"CREATE INDEX extension_permission_catalog_audit_idx",
		"ADD CONSTRAINT extension_permission_role_suggestions_decision_audit_fk",
		"REFERENCES audit_events(id) ON DELETE RESTRICT",
		"CREATE INDEX extension_permission_role_suggestions_decision_audit_idx",
		"CREATE FUNCTION extension_identity_actor_can_manage_roles(candidate_user_id BIGINT)",
		"permission_key = 'role.manage'",
		"roles.key = 'super_admin'",
		"CREATE TRIGGER extension_identity_decision_audit_immutable",
		"identity role decision audit evidence is immutable",
		"OLD.actor_user_id IS NOT NULL AND NEW.actor_user_id IS NULL",
		"OLD.target_user_id IS NOT NULL AND NEW.target_user_id IS NULL",
		"FROM extension_permission_role_grants WHERE applied_audit_event_id = OLD.id",
		"CREATE INDEX IF NOT EXISTS audit_events_created_at_id_idx ON audit_events (created_at, id)",
		"CREATE FUNCTION validate_extension_permission_role_suggestion_decision() RETURNS trigger",
		"extension.status = 'enabled'",
		"extension.active_version_id = version.id",
		"FOR KEY SHARE",
		"permission role suggestion decision actor lacks role.manage",
		"permission role suggestion decision audit evidence is invalid",
		"permission role suggestion Host catalog is unavailable",
		"permission role suggestion additive grant is missing",
		"permission role suggestion grant evidence is missing",
		"AND key <> 'super_admin'",
		"AND is_enabled = TRUE",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("identity role approvals migration missing %q", clause)
		}
	}

	// Catalog must not require a suggestion or decision audit for registration.
	for _, forbidden := range []string{
		"registered_suggestion_id BIGINT NOT NULL",
		"registered_by_user_id BIGINT NOT NULL",
		"registered_audit_event_id BIGINT NOT NULL",
		"cannot migrate legacy identity approval without explicit reconciliation",
	} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("identity role approvals migration must not require %q", forbidden)
		}
	}

	lower := strings.ToLower(up)
	for _, forbidden := range []string{
		"insert into role_permissions",
		"delete from role_permissions",
		"update role_permissions",
		"insert into extension_permission_role_grants",
		"delete from permissions",
		"update permissions",
		"delete from audit_events",
		"on delete cascade",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("identity role approvals migration must not contain %q", forbidden)
		}
	}

	permissionAddedTypeCheck := "jsonb_typeof(event.metadata -> 'rolePermissionAdded') IS DISTINCT FROM 'boolean'"
	if count := strings.Count(up, permissionAddedTypeCheck); count != 2 {
		t.Fatalf("rolePermissionAdded must be boolean in both grant and decision triggers; checks=%d", count)
	}
	strictGrantCheck := "event.metadata -> 'roleGrantApplied' IS DISTINCT FROM 'true'::jsonb"
	if count := strings.Count(up, strictGrantCheck); count != 2 {
		t.Fatalf("roleGrantApplied must stay true in both grant and approval triggers; checks=%d", count)
	}
	if strings.Contains(up, "event.metadata -> 'rolePermissionAdded' IS DISTINCT FROM 'true'::jsonb") {
		t.Fatal("rolePermissionAdded must allow false when the Host mapping already exists")
	}
}

func TestIdentityRoleApprovalsMigrationProtectsAuthorityOnDown(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityRoleApprovalsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity role approvals migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"LOCK TABLE extension_permission_role_grants, extension_permission_catalog, extension_permission_role_suggestions, audit_events IN ACCESS EXCLUSIVE MODE",
		"IF EXISTS (SELECT 1 FROM extension_permission_role_grants)",
		"OR EXISTS (SELECT 1 FROM extension_permission_catalog)",
		"WHERE approval_state <> 'pending'",
		"RAISE EXCEPTION 'cannot remove identity role approval authority history'",
		"CREATE TRIGGER extension_permission_role_suggestion_update_valid",
		"EXECUTE FUNCTION validate_extension_permission_role_suggestion_update()",
		"DROP CONSTRAINT IF EXISTS extension_permission_role_suggestions_decision_audit_fk",
		"DROP TABLE IF EXISTS extension_permission_role_grants",
		"DROP TABLE IF EXISTS extension_permission_catalog",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("identity role approvals Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"DELETE FROM",
		"TRUNCATE",
		"DROP TABLE IF EXISTS role_permissions",
		"DROP TABLE IF EXISTS permissions",
		"DROP TABLE IF EXISTS audit_events",
		"DROP TABLE IF EXISTS extension_identity_registry_declarations",
		"DROP TABLE IF EXISTS extension_identity_registry_owners",
	} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("identity role approvals Down contains %q", forbidden)
		}
	}
}
