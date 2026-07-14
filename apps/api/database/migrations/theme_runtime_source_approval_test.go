package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestThemeRuntimeSourceApprovalMigrationPreservesCompensationAuthority(t *testing.T) {
	body, err := fs.ReadFile(Files(), "202607150021_theme_runtime_source_approval.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("theme runtime source approval migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"ALTER TABLE theme_runtime_publications",
		"source_core_replacements_approved BOOLEAN NOT NULL DEFAULT FALSE",
		"CHECK (source_actor_user_id IS NULL OR source_actor_user_id > 0)",
		"source_core_replacements_approved = FALSE AND source_actor_user_id IS NULL",
		"source_core_replacements_approved = TRUE AND source_actor_user_id IS NOT NULL AND source_theme_id IS NOT NULL",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("theme source approval migration missing %q", clause)
		}
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"IF EXISTS (SELECT 1 FROM theme_runtime_publications)",
		"RAISE EXCEPTION 'cannot remove theme runtime source approval history'",
		"DROP COLUMN IF EXISTS source_actor_user_id",
		"DROP COLUMN IF EXISTS source_core_replacements_approved",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("theme source approval migration Down missing %q", clause)
		}
	}
}
