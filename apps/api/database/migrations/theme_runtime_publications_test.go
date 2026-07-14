package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const themeRuntimePublicationsMigration = "202607150020_theme_runtime_publications.sql"

func TestThemeRuntimePublicationsMigrationPersistsDesiredNodeAndAckState(t *testing.T) {
	body, err := fs.ReadFile(Files(), themeRuntimePublicationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("theme runtime publication migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE TABLE theme_runtime_publications",
		"revision BIGSERIAL PRIMARY KEY",
		"CHECK (desired_state IN ('active', 'none'))",
		"package_digest ~ '^[0-9a-f]{64}$'",
		"core_replacements_approved BOOLEAN NOT NULL DEFAULT FALSE",
		"CHECK (actor_user_id IS NULL OR actor_user_id > 0)",
		"CHECK (core_replacements_approved = FALSE OR actor_user_id IS NOT NULL)",
		"CREATE FUNCTION reject_theme_runtime_publication_mutation() RETURNS trigger",
		"RAISE EXCEPTION 'theme runtime publications are append-only'",
		"CREATE TRIGGER theme_runtime_publication_immutable",
		"BEFORE UPDATE OR DELETE ON theme_runtime_publications",
		"CREATE TRIGGER theme_runtime_publication_no_truncate",
		"BEFORE TRUNCATE ON theme_runtime_publications",
		"CREATE TABLE theme_runtime_nodes",
		"PRIMARY KEY (node_id, boot_id)",
		"lease_expires_at > last_seen_at",
		"CREATE TABLE theme_runtime_publication_acks",
		"PRIMARY KEY (publication_revision, node_id, boot_id)",
		"REFERENCES theme_runtime_publications(revision) ON DELETE RESTRICT",
		"REFERENCES theme_runtime_nodes(node_id, boot_id) ON DELETE RESTRICT",
		"CHECK (status IN ('applying', 'applied', 'failed'))",
		"CREATE FUNCTION notify_theme_runtime_publication() RETURNS trigger",
		"pg_notify('sforum_theme_runtime_publication', NEW.revision::text)",
		"CREATE TRIGGER theme_runtime_publication_notify",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("theme runtime publication migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{"REFERENCES users(id)", "ON DELETE CASCADE", "DELETE FROM theme_runtime_publications"} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("theme runtime publication migration must not contain %q", forbidden)
		}
	}
}

func TestThemeRuntimePublicationsMigrationProtectsDurableHistoryOnDown(t *testing.T) {
	body, err := fs.ReadFile(Files(), themeRuntimePublicationsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("theme runtime publication migration has no Down section")
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	for _, clause := range []string{
		"-- +goose StatementBegin",
		"IF EXISTS (SELECT 1 FROM theme_runtime_publications)",
		"OR EXISTS (SELECT 1 FROM theme_runtime_publication_acks)",
		"RAISE EXCEPTION 'cannot remove theme runtime publication history'",
		"DROP TRIGGER IF EXISTS theme_runtime_publication_notify ON theme_runtime_publications",
		"DROP TRIGGER IF EXISTS theme_runtime_publication_immutable ON theme_runtime_publications",
		"DROP TRIGGER IF EXISTS theme_runtime_publication_no_truncate ON theme_runtime_publications",
		"DROP FUNCTION IF EXISTS notify_theme_runtime_publication()",
		"DROP FUNCTION IF EXISTS reject_theme_runtime_publication_mutation()",
		"DROP TABLE IF EXISTS theme_runtime_publication_acks",
		"DROP TABLE IF EXISTS theme_runtime_nodes",
		"DROP TABLE IF EXISTS theme_runtime_publications",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("theme runtime publication migration Down missing %q", clause)
		}
	}
}
