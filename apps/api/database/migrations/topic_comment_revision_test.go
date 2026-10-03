package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTopicCommentRevisionMigration(t *testing.T) {
	path := filepath.Join("202610030002_topic_comment_revision.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	up := strings.Split(string(raw), "-- +goose Down")[0]
	for _, fragment := range []string{
		"ALTER TABLE topics ADD COLUMN IF NOT EXISTS comment_revision BIGINT NOT NULL DEFAULT 0",
		"CREATE OR REPLACE FUNCTION sforum_bump_topic_comment_revision()",
		// goose 需要显式语句边界：$$ 函数体里的分号不能被当成迁移语句分隔符。
		"-- +goose StatementBegin",
		"-- +goose StatementEnd",
		"AFTER INSERT ON comments",
		"AFTER UPDATE ON comments",
		"AFTER DELETE ON comments",
		"REFERENCING NEW TABLE AS revision_inserted",
		"REFERENCING OLD TABLE AS revision_old NEW TABLE AS revision_new",
		"REFERENCING OLD TABLE AS revision_deleted",
		"FOR EACH STATEMENT EXECUTE FUNCTION sforum_bump_topic_comment_revision()",
		"pg_notify('sforum_forum_comment_revision'",
		"WHERE status = 'active'",
	} {
		if !strings.Contains(up, fragment) {
			t.Fatalf("topic comment revision migration missing %q", fragment)
		}
	}
	// 逐行触发器会在 seed/批量审核时产生写放大；语句级 + transition table 才是设计意图。
	if strings.Contains(up, "FOR EACH ROW") {
		t.Fatal("comment revision triggers must be statement-level with transition tables")
	}
	// 信号不得携带评论内容或作者身份。
	for _, forbidden := range []string{"plain_text", "raw_content", "author_user_id", "html_content"} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("comment revision migration must not touch content payloads: %s", forbidden)
		}
	}

	down := strings.Split(string(raw), "-- +goose Down")[1]
	for _, fragment := range []string{
		"DROP TRIGGER IF EXISTS comments_revision_insert ON comments",
		"DROP TRIGGER IF EXISTS comments_revision_update ON comments",
		"DROP TRIGGER IF EXISTS comments_revision_delete ON comments",
		"DROP FUNCTION IF EXISTS sforum_bump_topic_comment_revision()",
		"ALTER TABLE topics DROP COLUMN IF EXISTS comment_revision",
	} {
		if !strings.Contains(down, fragment) {
			t.Fatalf("topic comment revision down migration missing %q", fragment)
		}
	}
}
