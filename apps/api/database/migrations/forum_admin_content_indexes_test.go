package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForumAdminContentIndexesMigration(t *testing.T) {
	path := filepath.Join("202607231000_forum_admin_content_indexes.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	up := strings.Split(string(raw), "-- +goose Down")[0]
	for _, fragment := range []string{
		"-- +goose NO TRANSACTION",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS topics_admin_content_updated_idx",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS topics_admin_content_status_updated_idx",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS comments_admin_content_updated_idx",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS comments_admin_content_status_updated_idx",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS comments_admin_content_topic_updated_idx",
	} {
		if !strings.Contains(up, fragment) {
			t.Fatalf("admin content index migration missing %q", fragment)
		}
	}
	if strings.Contains(strings.ToLower(up), "plain_text") || strings.Contains(strings.ToLower(up), "raw_content") {
		t.Fatal("admin content indexes must not index content payloads")
	}
}
