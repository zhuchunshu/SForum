package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const identityRoleApprovalsSchemaRepairMigration = "202607170034_identity_role_approvals_schema_repair.sql"

func TestIdentityRoleApprovalsSchemaRepairMigrationBindsExactEvidence(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityRoleApprovalsSchemaRepairMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity role approvals schema repair migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"LOCK TABLE %s IN ACCESS EXCLUSIVE MODE NOWAIT",
		"timed out acquiring identity role approval repair fence",
		"extension_versions",
		"extension_identity_registry_publications",
		"extension_permission_role_grants",
		"extension_permission_catalog",
		"extension_permission_role_suggestions",
		"extension_identity_registry_declarations",
		"extension_identity_registry_owners",
		"cannot bind legacy identity role decisions without exact audit evidence",
		"ADD COLUMN declaration_revision BIGINT",
		"ADD COLUMN extension_version_id BIGINT",
		"ADD COLUMN extension_version TEXT",
		"ADD COLUMN package_digest TEXT",
		"ADD COLUMN contract_version TEXT",
		"ADD COLUMN declaration_digest TEXT",
		"ALTER COLUMN registered_suggestion_id DROP NOT NULL",
		"ALTER COLUMN registered_by_user_id DROP NOT NULL",
		"ALTER COLUMN registered_audit_event_id DROP NOT NULL",
		"cannot repair extension permission catalog without exact suggestion/declaration evidence",
		"candidate.created_at <= suggestion.created_at",
		"ORDER BY candidate.revision DESC",
		"existing extension_permission_role_grants schema is incompatible with migration 202607170034",
		"FROM pg_index AS index_info",
		"index_info.indpred IS NULL",
		"index_info.indexprs IS NULL",
		"CREATE TABLE IF NOT EXISTS extension_permission_role_grants",
		"CREATE OR REPLACE FUNCTION validate_extension_permission_catalog() RETURNS trigger",
		"CREATE OR REPLACE FUNCTION validate_extension_permission_role_grant() RETURNS trigger",
		"CREATE OR REPLACE FUNCTION extension_identity_actor_can_manage_roles(candidate_user_id BIGINT)",
		"CREATE OR REPLACE FUNCTION reject_extension_identity_decision_audit_mutation() RETURNS trigger",
		"CREATE OR REPLACE FUNCTION validate_extension_permission_role_suggestion_decision() RETURNS trigger",
		"CREATE TRIGGER extension_permission_catalog_immutable",
		"CREATE TRIGGER extension_permission_catalog_no_truncate",
		"CREATE TRIGGER extension_permission_role_grant_immutable",
		"CREATE TRIGGER extension_permission_role_grant_no_truncate",
		"CREATE TRIGGER extension_identity_decision_audit_immutable",
		"CREATE TRIGGER extension_permission_role_suggestion_update_valid",
		"CREATE INDEX IF NOT EXISTS extension_permission_catalog_declaration_idx",
		"CREATE INDEX IF NOT EXISTS extension_permission_catalog_audit_idx",
		"CREATE INDEX IF NOT EXISTS audit_events_created_at_id_idx",
		"permission role suggestion grant evidence is missing",
		"extension permission catalog declaration is invalid",
		"FOREIGN KEY (identity_kind, permission_key, declaration_revision)",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("identity role approvals schema repair migration missing %q", clause)
		}
	}

	lower := strings.ToLower(up)
	for _, forbidden := range []string{
		"insert into role_permissions",
		"delete from role_permissions",
		"insert into extension_permission_role_grants",
		"delete from extension_permission_catalog",
		"delete from audit_events",
		"on delete cascade",
		// Never invent declaration fields from non-exact sources.
		"md5(",
		"gen_random",
		"digest(",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("identity role approvals schema repair must not contain %q", forbidden)
		}
	}

	// Repair must not rewrite already-applied current 029 source.
	if strings.Contains(string(body), "202607160029_identity_role_approvals.sql") {
		t.Fatal("repair migration must not reference or edit 029 source file")
	}
}

func TestIdentityRoleApprovalsSchemaRepairMigrationDownIsForwardOnly(t *testing.T) {
	body, err := fs.ReadFile(Files(), identityRoleApprovalsSchemaRepairMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("identity role approvals schema repair migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"cannot reverse identity role approval schema repair 202607170034",
		"roll forward with a new repair migration",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("identity role approvals schema repair Down missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"DROP TABLE",
		"DELETE FROM",
		"TRUNCATE",
		"DROP COLUMN",
	} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("identity role approvals schema repair Down contains %q", forbidden)
		}
	}
}
