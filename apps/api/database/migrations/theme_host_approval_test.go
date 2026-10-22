package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const themeHostApprovalMigration = "202610230001_theme_host_approval.sql"

func TestThemeHostApprovalMigrationKeepsApprovalProvenanceExplicit(t *testing.T) {
	body, err := fs.ReadFile(Files(), themeHostApprovalMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("theme host approval migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"-- +sforum OnlineSafe",
		"SET LOCAL lock_timeout = '5s'",
		"SET LOCAL statement_timeout = '1min'",
		"ALTER TABLE page_provider_bindings ADD COLUMN approval_source TEXT NOT NULL DEFAULT 'admin' CHECK (approval_source IN ('admin', 'host'))",
		"ADD COLUMN source_approval_source TEXT NOT NULL DEFAULT 'admin'",
		"DROP CONSTRAINT IF EXISTS theme_runtime_publications_source_approval_check",
		"pg_get_constraintdef(oid) LIKE '%core_replacements_approved%'",
		"pg_get_constraintdef(oid) LIKE '%actor_user_id IS NOT NULL%'",
		"RAISE EXCEPTION 'theme_runtime_publications approved=>actor constraint not found'",
		"ADD CONSTRAINT theme_runtime_publications_approval_check",
		"core_replacements_approved = FALSE OR (approval_source = 'admin' AND actor_user_id IS NOT NULL) OR (approval_source = 'host' AND actor_user_id IS NULL)",
		"source_core_replacements_approved = TRUE AND source_approval_source = 'host' AND source_actor_user_id IS NULL AND source_theme_id IS NOT NULL",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("theme host approval migration missing %q", clause)
		}
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"IF EXISTS (SELECT 1 FROM theme_runtime_publications WHERE approval_source = 'host')",
		"RAISE EXCEPTION 'cannot remove host-approved theme runtime history'",
		"DROP CONSTRAINT IF EXISTS theme_runtime_publications_approval_check",
		"ADD CONSTRAINT theme_runtime_publications_check2 CHECK (core_replacements_approved = FALSE OR actor_user_id IS NOT NULL)",
		"DROP COLUMN IF EXISTS source_approval_source",
		"DROP COLUMN IF EXISTS approval_source",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("theme host approval migration Down missing %q", clause)
		}
	}
}
