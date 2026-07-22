package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const forumRevisionViewPermissionsMigration = "202607220053_forum_revision_view_permissions.sql"

func TestForumRevisionViewPermissionsMigrationScopesDefaultGrants(t *testing.T) {
	body, err := fs.ReadFile(Files(), forumRevisionViewPermissionsMigration)
	if err != nil {
		t.Fatal(err)
	}
	normalized := strings.Join(strings.Fields(string(body)), " ")
	for _, fragment := range []string{
		"('topic.revision.view_any', 'forum', 'Inspect any topic revision history.')",
		"('post.revision.view_any', 'forum', 'Inspect any comment revision history.')",
		"roles.key = 'super_admin'",
		"roles.key = 'moderator'",
	} {
		if !strings.Contains(normalized, fragment) {
			t.Fatalf("revision permission migration missing %q", fragment)
		}
	}
	for _, forbidden := range []string{
		"roles.key = 'member'",
		"roles.key = 'operator'",
		"roles.key = 'tech_admin'",
	} {
		if strings.Contains(normalized, forbidden) {
			t.Fatalf("revision history permissions must not be granted by default through %s", forbidden)
		}
	}
}
