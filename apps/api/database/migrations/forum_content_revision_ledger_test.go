package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const forumContentRevisionLedgerMigration = "202607220052_forum_content_revision_ledger.sql"

func TestForumContentRevisionLedgerMigrationIsOnlineAdditive(t *testing.T) {
	body, err := fs.ReadFile(Files(), forumContentRevisionLedgerMigration)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	up := strings.SplitN(text, "-- +goose Down", 2)[0]
	normalized := strings.Join(strings.Fields(up), " ")

	for _, fragment := range []string{
		"-- +goose NO TRANSACTION",
		"ADD COLUMN IF NOT EXISTS current_revision BIGINT NOT NULL DEFAULT 0",
		"ADD COLUMN IF NOT EXISTS revision_no BIGINT",
		"ADD COLUMN IF NOT EXISTS actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL",
		"ADD COLUMN IF NOT EXISTS changed_fields TEXT[] NOT NULL DEFAULT '{}'",
		"ADD COLUMN IF NOT EXISTS attachment_ids BIGINT[] NOT NULL DEFAULT '{}'",
		"CREATE TABLE IF NOT EXISTS topic_revision_snapshots",
		"CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS post_revisions_post_revision_no_unique_idx",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS post_revisions_post_revision_desc_idx",
	} {
		if !strings.Contains(normalized, fragment) && !strings.Contains(up, fragment) {
			t.Fatalf("revision ledger migration missing %q", fragment)
		}
	}
	if strings.Contains(normalized, "INSERT INTO post_revisions SELECT") ||
		strings.Contains(normalized, "UPDATE post_revisions SET raw_content") ||
		strings.Contains(normalized, "UPDATE posts SET current_revision") {
		t.Fatalf("schema migration must not bulk-copy revision payloads or mark backfill complete:\n%s", up)
	}
}

func TestForumContentRevisionLedgerMigrationKeepsAuditSeparate(t *testing.T) {
	body, err := fs.ReadFile(Files(), forumContentRevisionLedgerMigration)
	if err != nil {
		t.Fatal(err)
	}
	normalized := strings.Join(strings.Fields(string(body)), " ")
	if strings.Contains(normalized, "REFERENCES audit_events") {
		t.Fatalf("revision ledger migration must not add a restrictive audit_events FK")
	}
}
