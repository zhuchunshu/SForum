package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const stableCoreViewsMigration = "202607140016_stable_core_views.sql"

func TestFilesIncludesStableCoreViewsMigration(t *testing.T) {
	entries, err := fs.ReadDir(Files(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == stableCoreViewsMigration {
			return
		}
	}
	t.Fatalf("expected embedded migration %s", stableCoreViewsMigration)
}

func TestStableCoreViewsMigrationOwnsFilteredReadOnlyContracts(t *testing.T) {
	body, err := fs.ReadFile(Files(), stableCoreViewsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("stable core views migration has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"CREATE SCHEMA sforum_core_v1 AUTHORIZATION CURRENT_USER",
		"REVOKE ALL ON SCHEMA sforum_core_v1 FROM PUBLIC",
		"CREATE VIEW sforum_core_v1.safe_users WITH (security_barrier = true, security_invoker = false)",
		"CREATE VIEW sforum_core_v1.forum_topics WITH (security_barrier = true, security_invoker = false)",
		"CREATE VIEW sforum_core_v1.forum_comments WITH (security_barrier = true, security_invoker = false)",
		"CREATE VIEW sforum_core_v1.public_entity_meta WITH (security_barrier = true, security_invoker = false)",
		"CREATE VIEW sforum_core_v1.public_attachment_metadata WITH (security_barrier = true, security_invoker = false)",
		"WHERE users.status = 'active' OFFSET 0",
		"WHERE topics.status IN ('active', 'locked')",
		"AND categories.visibility = 'public' OFFSET 0",
		"WHERE comments.status = 'active'",
		"AND definitions.visibility = 'public' OFFSET 0",
		"AND attachments.visibility = 'public'",
		"REVOKE ALL ON ALL TABLES IN SCHEMA sforum_core_v1 FROM PUBLIC",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("stable core views migration missing %q", clause)
		}
	}
	for _, forbidden := range []string{
		"users.email", "users.email_lower", "users.current_token_version",
		"posts.raw_content", "topics.ip_address", "topics.last_edit_ip",
		"comments.ip_address", "comments.last_edit_ip", "moderation_triggers",
		"attachments.provider", "attachments.object_key", "updated_by_user_id",
	} {
		if strings.Contains(" "+up+" ", forbidden) {
			t.Fatalf("stable core views migration exposes or grants %q", forbidden)
		}
	}
	for _, line := range strings.Split(parts[0], "\n") {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(line)), "GRANT ") {
			t.Fatalf("stable core views migration must not grant plugin authority: %q", strings.TrimSpace(line))
		}
	}
}

func TestStableCoreViewsMigrationDownIsNarrowAndNonCascading(t *testing.T) {
	body, err := fs.ReadFile(Files(), stableCoreViewsMigration)
	if err != nil {
		t.Fatal(err)
	}
	down := strings.Join(strings.Fields(strings.SplitN(string(body), "-- +goose Down", 2)[1]), " ")
	for _, view := range []string{
		"public_attachment_metadata", "public_entity_meta", "forum_comments", "forum_topics", "safe_users",
	} {
		if !strings.Contains(down, "DROP VIEW IF EXISTS sforum_core_v1."+view) {
			t.Fatalf("stable core views Down does not drop %s", view)
		}
	}
	if !strings.Contains(down, "DROP SCHEMA IF EXISTS sforum_core_v1") {
		t.Fatal("stable core views Down does not drop its schema")
	}
	for _, forbidden := range []string{"CASCADE", "DROP TABLE", "DROP ROLE", "DELETE FROM", "TRUNCATE"} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("stable core views Down contains %q", forbidden)
		}
	}
}
